package reference

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var (
	errAcquisitionConfiguration = errors.New("reference acquisition configuration")
	errSourceAmbiguous          = errors.New("reference source ambiguous")
)

// TerminalReason is a fixed, bounded classification of a failed reference
// acquisition. It deliberately carries no provider text, URL, payload, or
// symbol.
type TerminalReason string

const (
	TerminalReasonNone              TerminalReason = "none"
	TerminalReasonInvalidRequest    TerminalReason = "invalid_request"
	TerminalReasonConfiguration     TerminalReason = "configuration_failed"
	TerminalReasonSourceUnavailable TerminalReason = "source_unavailable"
	TerminalReasonSourceAmbiguous   TerminalReason = "source_ambiguous"
	TerminalReasonCanceled          TerminalReason = "canceled"
	TerminalReasonDeadline          TerminalReason = "deadline_exceeded"
)

// AcquisitionDiagnostics is the immutable, scalar-only account of work
// actually attempted by one resolver operation.
type AcquisitionDiagnostics struct {
	Source                  Source
	RequestCount            int
	PageCount               int
	AttemptCount            int
	UnattributablePriorRows int
	TerminalReason          TerminalReason
}

// AcquisitionError preserves a bounded terminal classification and exact work
// counts while retaining the implementation error only for immediate callers.
type AcquisitionError struct {
	diagnostics AcquisitionDiagnostics
	cause       error
}

func (e *AcquisitionError) Error() string {
	return fmt.Sprintf("reference acquisition failed: %s", e.diagnostics.TerminalReason)
}

func (e *AcquisitionError) Unwrap() error { return e.cause }

func (e *AcquisitionError) Diagnostics() AcquisitionDiagnostics { return e.diagnostics }

type acquisitionTracker struct {
	diagnostics AcquisitionDiagnostics
}

func newAcquisitionTracker() *acquisitionTracker {
	return &acquisitionTracker{diagnostics: AcquisitionDiagnostics{
		Source:         SourceNone,
		TerminalReason: TerminalReasonNone,
	}}
}

func (t *acquisitionTracker) beginRequest(page bool) {
	t.diagnostics.RequestCount++
	if page {
		t.diagnostics.PageCount++
	}
}

func (t *acquisitionTracker) beginAttempt() { t.diagnostics.AttemptCount++ }

func (t *acquisitionTracker) snapshot() AcquisitionDiagnostics { return t.diagnostics }

func acquisitionFailure(tracker *acquisitionTracker, reason TerminalReason, cause error) error {
	diagnostics := tracker.snapshot()
	diagnostics.TerminalReason = reason
	return &AcquisitionError{diagnostics: diagnostics, cause: cause}
}

func classifyTerminalReason(err error, fallback TerminalReason) TerminalReason {
	switch {
	case errors.Is(err, context.Canceled):
		return TerminalReasonCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return TerminalReasonDeadline
	case errors.Is(err, errAcquisitionConfiguration):
		return TerminalReasonConfiguration
	case errors.Is(err, errSourceAmbiguous):
		return TerminalReasonSourceAmbiguous
	default:
		return fallback
	}
}

// objectMembers decodes exactly one JSON object while retaining every member
// occurrence. Callers can therefore reject duplicate contract-bearing members
// instead of inheriting encoding/json's last-value behavior.
func objectMembers(raw []byte) (map[string][]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '{' {
		return nil, errors.New("JSON value is not an object")
	}
	members := make(map[string][]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		name, ok := token.(string)
		if !ok {
			return nil, errors.New("JSON object member name is not a string")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		members[name] = append(members[name], value)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("trailing JSON value")
		}
		return nil, err
	}
	return members, nil
}

func requiredMember(members map[string][]json.RawMessage, name string) (json.RawMessage, error) {
	values := members[name]
	if len(values) != 1 {
		return nil, fmt.Errorf("member %q must occur exactly once", name)
	}
	return values[0], nil
}

func optionalMember(members map[string][]json.RawMessage, name string) (json.RawMessage, bool, error) {
	values := members[name]
	if len(values) > 1 {
		return nil, false, fmt.Errorf("member %q occurs more than once", name)
	}
	if len(values) == 0 {
		return nil, false, nil
	}
	return values[0], true, nil
}

func decodeMember[T any](raw json.RawMessage) (T, error) {
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, err
	}
	return value, nil
}
