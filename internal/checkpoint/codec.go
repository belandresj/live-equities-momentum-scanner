package checkpoint

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	EnvelopeV1          = "scanner-checkpoint-v1"
	ManifestByteLimit   = int64(64 << 10)
	AbsoluteByteCeiling = int64(4 << 30)
	MaximumSymbols      = 100_000
)

var (
	ErrCanceled     = errors.New("checkpoint operation canceled")
	ErrLimit        = errors.New("checkpoint byte limit")
	ErrInvalid      = errors.New("invalid checkpoint")
	ErrIncompatible = errors.New("incompatible checkpoint")
)

type envelope struct {
	Format        string          `json:"format"`
	Payload       json.RawMessage `json:"payload"`
	PayloadSHA256 string          `json:"payload_sha256"`
}

// Encode writes one deterministic fixed-schema envelope. The digest covers the
// exact raw semantic payload bytes, not a re-marshaled value.
func Encode(ctx context.Context, output io.Writer, image Image, limit int64) (int64, string, error) {
	if ctx == nil || output == nil || limit <= 0 || limit > AbsoluteByteCeiling {
		return 0, "", ErrLimit
	}
	if err := preflightImage(image); err != nil {
		return 0, "", err
	}
	if err := validateSemanticImage(ctx, image); err != nil {
		return 0, "", err
	}
	bounded := &contextLimitWriter{ctx: ctx, output: output, limit: limit}
	if _, err := bounded.Write([]byte(`{"format":"scanner-checkpoint-v1","payload":`)); err != nil {
		return bounded.written, "", err
	}
	digest := sha256.New()
	hold := &holdLastWriter{output: io.MultiWriter(bounded, digest)}
	encoder := json.NewEncoder(hold)
	if err := encoder.Encode(image); err != nil {
		return bounded.written, "", fmt.Errorf("%w: payload: %v", ErrInvalid, err)
	}
	if err := hold.finishJSON(); err != nil {
		return bounded.written, "", err
	}
	checksum := hex.EncodeToString(digest.Sum(nil))
	if _, err := bounded.Write([]byte(`,"payload_sha256":"` + checksum + `"}`)); err != nil {
		return bounded.written, "", err
	}
	return bounded.written, checksum, nil
}

type contextLimitWriter struct {
	ctx            context.Context
	output         io.Writer
	limit, written int64
}

func (w *contextLimitWriter) Write(value []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, ErrCanceled
	}
	if int64(len(value)) > w.limit-w.written {
		return 0, ErrLimit
	}
	n, err := w.output.Write(value)
	w.written += int64(n)
	if err == nil && n != len(value) {
		err = io.ErrShortWrite
	}
	return n, err
}

// json.Encoder appends one newline. Holding one byte lets the envelope digest
// cover exactly the JSON value bytes while retaining streaming behavior.
type holdLastWriter struct {
	output io.Writer
	last   byte
	have   bool
}

func (w *holdLastWriter) Write(value []byte) (int, error) {
	if len(value) == 0 {
		return 0, nil
	}
	original := len(value)
	if w.have {
		if _, err := w.output.Write([]byte{w.last}); err != nil {
			return 0, err
		}
	}
	if len(value) > 1 {
		if _, err := w.output.Write(value[:len(value)-1]); err != nil {
			return 0, err
		}
	}
	w.last, w.have = value[len(value)-1], true
	return original, nil
}

func (w *holdLastWriter) finishJSON() error {
	if !w.have || w.last != '\n' {
		return ErrInvalid
	}
	w.have = false
	return nil
}

// Decode returns no candidate unless the bounded envelope, raw checksum,
// strict JSON vocabulary, duplicate-key scan, and structural preflight pass.
func Decode(ctx context.Context, input io.Reader, limit int64) (Candidate, int64, error) {
	if ctx == nil || input == nil || limit <= 0 || limit > AbsoluteByteCeiling {
		return Candidate{}, 0, ErrLimit
	}
	spool, err := os.CreateTemp("", ".scanner-checkpoint-decode-")
	if err != nil {
		return Candidate{}, 0, err
	}
	spoolPath := spool.Name()
	defer os.Remove(spoolPath)
	defer spool.Close()
	if err := spool.Chmod(0600); err != nil {
		return Candidate{}, 0, err
	}
	reader := &boundedByteReader{ctx: ctx, input: bufio.NewReaderSize(input, 32<<10), limit: limit}
	if err := reader.expect([]byte(`{"format":"scanner-checkpoint-v1","payload":`)); err != nil {
		return Candidate{}, reader.count, err
	}
	digest := sha256.New()
	if err := copyJSONObject(reader, io.MultiWriter(spool, digest)); err != nil {
		return Candidate{}, reader.count, err
	}
	if err := reader.expect([]byte(`,"payload_sha256":"`)); err != nil {
		return Candidate{}, reader.count, err
	}
	checksumBytes := make([]byte, 64)
	for index := range checksumBytes {
		value, err := reader.readByte()
		if err != nil {
			return Candidate{}, reader.count, err
		}
		checksumBytes[index] = value
	}
	checksum := string(checksumBytes)
	if !lowerHex(checksum) || checksum != hex.EncodeToString(digest.Sum(nil)) {
		return Candidate{}, reader.count, ErrInvalid
	}
	if err := reader.expect([]byte(`"}`)); err != nil {
		return Candidate{}, reader.count, err
	}
	for {
		value, err := reader.readByte()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Candidate{}, reader.count, err
		}
		if !isJSONSpace(value) {
			return Candidate{}, reader.count, ErrInvalid
		}
	}
	if _, err := spool.Seek(0, io.SeekStart); err != nil {
		return Candidate{}, reader.count, err
	}
	if err := rejectDuplicateKeysReader(&contextReader{ctx: ctx, input: spool}); err != nil {
		if errors.Is(err, ErrCanceled) {
			return Candidate{}, reader.count, ErrCanceled
		}
		return Candidate{}, reader.count, ErrInvalid
	}
	if _, err := spool.Seek(0, io.SeekStart); err != nil {
		return Candidate{}, reader.count, err
	}
	if err := validateArrayLengthsReader(&contextReader{ctx: ctx, input: spool}); err != nil {
		if errors.Is(err, ErrCanceled) {
			return Candidate{}, reader.count, ErrCanceled
		}
		return Candidate{}, reader.count, err
	}
	if _, err := spool.Seek(0, io.SeekStart); err != nil {
		return Candidate{}, reader.count, err
	}
	var image Image
	if err := strictJSONReader(&contextReader{ctx: ctx, input: spool}, &image); err != nil {
		return Candidate{}, reader.count, err
	}
	if image.SchemaVersion != SchemaV1 || image.ProducerMode != ProducerLive {
		return Candidate{}, reader.count, ErrIncompatible
	}
	if err := preflightImage(image); err != nil {
		return Candidate{}, reader.count, err
	}
	if err := validateSemanticImage(ctx, image); err != nil {
		return Candidate{}, reader.count, err
	}
	return Candidate{Image: image, Checksum: checksum, Integrity: true}, reader.count, nil
}

type boundedByteReader struct {
	ctx          context.Context
	input        *bufio.Reader
	limit, count int64
}

type contextReader struct {
	ctx   context.Context
	input io.Reader
}

func (r *contextReader) Read(value []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, ErrCanceled
	}
	return r.input.Read(value)
}

func (r *boundedByteReader) readByte() (byte, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, ErrCanceled
	}
	if r.count == r.limit {
		if _, err := r.input.Peek(1); err == nil {
			return 0, ErrLimit
		}
	}
	value, err := r.input.ReadByte()
	if err == nil {
		r.count++
	}
	return value, err
}

func (r *boundedByteReader) expect(expected []byte) error {
	for _, want := range expected {
		got, err := r.readByte()
		if err != nil {
			return err
		}
		if got != want {
			return ErrInvalid
		}
	}
	return nil
}

func copyJSONObject(reader *boundedByteReader, output io.Writer) error {
	depth, quoted, escaped := 0, false, false
	for {
		value, err := reader.readByte()
		if err != nil {
			return err
		}
		if _, err := output.Write([]byte{value}); err != nil {
			return err
		}
		if quoted {
			if escaped {
				escaped = false
				continue
			}
			if value == '\\' {
				escaped = true
			} else if value == '"' {
				quoted = false
			}
			continue
		}
		if value == '"' {
			quoted = true
			continue
		}
		if value == '{' {
			depth++
		} else if value == '}' {
			depth--
			if depth == 0 {
				return nil
			}
		}
		if depth < 0 || depth == 0 && value != '{' {
			return ErrInvalid
		}
	}
}

func isJSONSpace(value byte) bool {
	return value == ' ' || value == '\n' || value == '\r' || value == '\t'
}

func validateArrayLengths(data []byte) error {
	return validateArrayLengthsReader(bytes.NewReader(data))
}

func validateArrayLengthsReader(input io.Reader) error {
	limits := map[string]int{
		"Symbols": MaximumSymbols, "Symbols[].Tail": 961, "Symbols[].Presence": 900, "Symbols[].ProvenAbsent": 900, "Symbols[].HistoricalConflict": 900,
		"Symbols[].PriceRange.Highs": 3600, "Symbols[].PriceRange.Lows": 3600, "Symbols[].PriceRange.SessionHighs": 57600, "Symbols[].PriceRange.SessionLows": 57600,
		"Symbols[].Activity.References": 1920, "Symbols[].Activity.References[].TransactionSum": 34, "Symbols[].Activity.Mutable": 33,
		"Symbols[].Activity.Mutable[].Folded.TransactionSum": 34, "Symbols[].Activity.Mutable[].Current.TransactionSum": 34,
		"Symbols[].Activity.FoldedTargets": 1920, "Symbols[].Activity.FoldedTargets[].Transactions": 30, "Symbols[].Activity.FoldedTargets[].Highs": 30, "Symbols[].Activity.FoldedTargets[].Lows": 30,
		"Symbols[].Qualification.FinalizedGateBars": 57601, "Symbols[].Qualification.Proofs": 961, "Symbols[].Qualification.Dirty": 961,
	}
	exact := map[string]int{
		"Symbols[].Activity.References[].TransactionSum":      34,
		"Symbols[].Activity.Mutable[].Folded.TransactionSum":  34,
		"Symbols[].Activity.Mutable[].Current.TransactionSum": 34,
		"Symbols[].Activity.FoldedTargets[].Transactions":     30,
		"Symbols[].Activity.FoldedTargets[].Highs":            30,
		"Symbols[].Activity.FoldedTargets[].Lows":             30,
	}
	decoder := json.NewDecoder(input)
	var walk func(string) error
	walk = func(path string) error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			if _, fixed := exact[path]; fixed {
				return fmt.Errorf("%w: %s array", ErrInvalid, path)
			}
			return nil
		}
		switch delim {
		case '{':
			required, present := requiredFixedFieldMask(path), uint8(0)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return ErrInvalid
				}
				present |= fixedFieldMask(path, key)
				child := key
				if path != "" {
					child = path + "." + key
				}
				if err := walk(child); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			if err == nil && present&required != required {
				return fmt.Errorf("%w: %s required fixed array", ErrInvalid, path)
			}
			return err
		case '[':
			count := 0
			for decoder.More() {
				count++
				if limit, exists := limits[path]; exists && count > limit {
					return fmt.Errorf("%w: %s length", ErrInvalid, path)
				}
				if err := walk(path + "[]"); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			if err == nil {
				if wanted, exists := exact[path]; exists && count != wanted {
					return fmt.Errorf("%w: %s length", ErrInvalid, path)
				}
			}
			return err
		default:
			return ErrInvalid
		}
	}
	return walk("")
}

func requiredFixedFieldMask(path string) uint8 {
	switch path {
	case "Symbols[].Activity.References[]", "Symbols[].Activity.Mutable[].Folded", "Symbols[].Activity.Mutable[].Current":
		return 1
	case "Symbols[].Activity.FoldedTargets[]":
		return 1 | 2 | 4
	default:
		return 0
	}
}

func fixedFieldMask(path, key string) uint8 {
	switch path {
	case "Symbols[].Activity.References[]", "Symbols[].Activity.Mutable[].Folded", "Symbols[].Activity.Mutable[].Current":
		if key == "TransactionSum" {
			return 1
		}
	case "Symbols[].Activity.FoldedTargets[]":
		switch key {
		case "Transactions":
			return 1
		case "Highs":
			return 2
		case "Lows":
			return 4
		}
	}
	return 0
}

func readLimitedContext(ctx context.Context, input io.Reader, limit int64) ([]byte, error) {
	reader := io.LimitReader(input, limit+1)
	buffer := bytes.NewBuffer(make([]byte, 0, min(limit, 64<<10)))
	chunk := make([]byte, 32<<10)
	for {
		if err := ctx.Err(); err != nil {
			return buffer.Bytes(), ErrCanceled
		}
		n, err := reader.Read(chunk)
		if n != 0 {
			buffer.Write(chunk[:n])
			if int64(buffer.Len()) > limit {
				return buffer.Bytes(), ErrLimit
			}
		}
		if errors.Is(err, io.EOF) {
			return buffer.Bytes(), nil
		}
		if err != nil {
			return buffer.Bytes(), err
		}
	}
}

func strictJSON(data []byte, target any) error {
	if err := rejectDuplicateKeys(data); err != nil {
		return ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, ErrCanceled) {
			return ErrCanceled
		}
		return ErrInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing value", ErrInvalid)
	}
	return nil
}

func strictJSONReader(input io.Reader, target any) error {
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, ErrCanceled) {
			return ErrCanceled
		}
		return ErrInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); errors.Is(err, ErrCanceled) {
		return ErrCanceled
	} else if !errors.Is(err, io.EOF) {
		return ErrInvalid
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	return rejectDuplicateKeysReader(bytes.NewReader(data))
}

func rejectDuplicateKeysReader(input io.Reader) error {
	decoder := json.NewDecoder(input)
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("object key")
				}
				if _, exists := seen[key]; exists {
					return errors.New("duplicate key")
				}
				seen[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return errors.New("unexpected delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing value")
	}
	return nil
}

func preflightImage(image Image) error {
	if image.SchemaVersion != SchemaV1 || image.ProducerMode != ProducerLive || image.Binding.Identity == "" || image.Sequence == 0 || !validWholeSecondUTC(image.T0) || !validUTCInstant(image.CreatedAt) || image.CreatedAt.Before(image.T0) {
		return fmt.Errorf("%w: header", ErrInvalid)
	}
	if image.Population < 0 || image.Population > MaximumSymbols || image.Population != len(image.Symbols) || image.Counts.RecordsWithState+image.Counts.EmptyStateRecords != image.Population {
		return fmt.Errorf("%w: population", ErrInvalid)
	}
	var tail, presence, absent, conflict, extrema, refs, mutable, blocks, contributions, bars, proofs, dirty, invalidMarks, coverage, withState int
	lastSymbol := ""
	for _, symbol := range image.Symbols {
		if len(symbol.Symbol) == 0 || len(symbol.Symbol) > 64 || lastSymbol != "" && symbol.Symbol <= lastSymbol || len(symbol.Tail) > 961 || len(symbol.Presence) > 900 || len(symbol.ProvenAbsent) > 900 || len(symbol.HistoricalConflict) > 900 {
			return fmt.Errorf("%w: symbol bounds", ErrInvalid)
		}
		lastSymbol = symbol.Symbol
		if symbol.HasState {
			withState++
		}
		tail += len(symbol.Tail)
		presence += len(symbol.Presence)
		absent += len(symbol.ProvenAbsent)
		conflict += len(symbol.HistoricalConflict)
		if symbol.PriceRange != nil {
			if len(symbol.PriceRange.Highs) > 3600 || len(symbol.PriceRange.Lows) > 3600 || len(symbol.PriceRange.SessionHighs) > 57600 || len(symbol.PriceRange.SessionLows) > 57600 {
				return fmt.Errorf("%w: price extrema bounds", ErrInvalid)
			}
			extrema += len(symbol.PriceRange.Highs) + len(symbol.PriceRange.Lows) + len(symbol.PriceRange.SessionHighs) + len(symbol.PriceRange.SessionLows)
		}
		if symbol.Activity != nil {
			if len(symbol.Activity.References) > 1920 || len(symbol.Activity.Mutable) > 33 || len(symbol.Activity.FoldedTargets) > 1920 || symbol.Activity.FoldedTargetContributions < 0 || symbol.Activity.FoldedTargetContributions > 57600 {
				return fmt.Errorf("%w: activity bounds", ErrInvalid)
			}
			refs += len(symbol.Activity.References)
			mutable += len(symbol.Activity.Mutable)
			blocks += len(symbol.Activity.FoldedTargets)
			contributions += symbol.Activity.FoldedTargetContributions
		}
		if symbol.Qualification != nil {
			if len(symbol.Qualification.Proofs) > 961 || len(symbol.Qualification.Dirty) > 961 || len(symbol.Qualification.FinalizedGateBars) > 57601 {
				return fmt.Errorf("%w: qualification bounds", ErrInvalid)
			}
			bars += len(symbol.Qualification.FinalizedGateBars)
			proofs += len(symbol.Qualification.Proofs)
			dirty += len(symbol.Qualification.Dirty)
		}
		if symbol.InvalidMarkStart != nil {
			invalidMarks++
		}
		if symbol.Coverage != nil {
			coverage++
		}
	}
	want := StructureCounts{RecordsWithState: withState, EmptyStateRecords: len(image.Symbols) - withState, TailRecords: tail, PresenceWords: presence, ProvenAbsentWords: absent, ConflictWords: conflict, PriceExtremaPoints: extrema, ActivityReferences: refs, ActivityMutable: mutable, ActivityTargetBlocks: blocks, ActivityTargetContributions: contributions, QualificationGateBars: bars, QualificationProofs: proofs, QualificationDirty: dirty, InvalidMarks: invalidMarks, CoverageConsequences: coverage}
	if image.Counts != want {
		return fmt.Errorf("%w: structure counts", ErrInvalid)
	}
	return nil
}

func validWholeSecondUTC(value time.Time) bool {
	return !value.IsZero() && value == value.UTC() && value.Nanosecond() == 0
}

func validUTCInstant(value time.Time) bool {
	return !value.IsZero() && value == value.UTC()
}
