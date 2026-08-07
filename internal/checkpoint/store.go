package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"
)

const manifestVersion = "scanner-checkpoint-manifest-v1"

var generationName = regexp.MustCompile(`^checkpoint-[0-9]{20}\.json$`)

type Manifest struct {
	Version  string         `json:"version"`
	Latest   *ManifestEntry `json:"latest"`
	Previous *ManifestEntry `json:"previous,omitempty"`
}

type ManifestEntry struct {
	Filename, PayloadSHA256, BindingIdentity string
	FileBytes                                int64
	T0                                       time.Time
	Sequence                                 uint64
}

type LoadDisposition string

const (
	LoadedLatest     LoadDisposition = "loaded_latest"
	LoadedPrevious   LoadDisposition = "loaded_previous"
	LoadUnavailable  LoadDisposition = "unavailable"
	LoadIncompatible LoadDisposition = "incompatible"
	LoadInvalid      LoadDisposition = "invalid"
	LoadCanceled     LoadDisposition = "canceled"
)

type LoadResult struct {
	Disposition LoadDisposition
	Candidate   Candidate
	Entry       ManifestEntry
	LatestError error
	Candidates  [2]CandidateLoadDisposition
}

type CandidateAuthority uint8

const (
	CandidateLatest CandidateAuthority = iota
	CandidatePrevious
)

// LoadCandidate decodes exactly one manifest-authorized generation. The
// coordinator asks for previous only after latest fails storage or engine
// validation, so two large detached graphs are never retained together.
func (s *Store) LoadCandidate(ctx context.Context, authority CandidateAuthority) LoadResult {
	if ctx == nil {
		return LoadResult{Disposition: LoadCanceled}
	}
	operation, cancel := context.WithTimeout(ctx, s.deadline)
	defer cancel()
	manifestData, err := readSafeFile(filepath.Join(s.directory, "manifest.json"), ManifestByteLimit)
	if err != nil {
		return LoadResult{Disposition: LoadUnavailable}
	}
	var manifest Manifest
	if strictJSON(manifestData, &manifest) != nil || !validManifest(manifest) {
		return LoadResult{Disposition: LoadUnavailable}
	}
	if authority != CandidateLatest && authority != CandidatePrevious {
		return LoadResult{Disposition: LoadInvalid}
	}
	entry := manifest.Latest
	disposition := LoadedLatest
	if authority == CandidatePrevious {
		entry, disposition = manifest.Previous, LoadedPrevious
	}
	if entry == nil {
		return LoadResult{Disposition: LoadUnavailable}
	}
	candidate, loadErr := s.loadEntry(operation, *entry)
	result := LoadResult{Entry: *entry, LatestError: loadErr}
	if loadErr == nil {
		result.Disposition, result.Candidate = disposition, candidate
		return result
	}
	switch {
	case errors.Is(loadErr, ErrCanceled):
		result.Disposition = LoadCanceled
	case errors.Is(loadErr, ErrIncompatible):
		result.Disposition = LoadIncompatible
	default:
		result.Disposition = LoadInvalid
	}
	return result
}

type CandidateLoadDisposition string

const (
	CandidateLoaded       CandidateLoadDisposition = "loaded"
	CandidateIncompatible CandidateLoadDisposition = "incompatible"
	CandidateInvalid      CandidateLoadDisposition = "invalid"
	CandidateUnavailable  CandidateLoadDisposition = "unavailable"
	CandidateCanceled     CandidateLoadDisposition = "canceled"
	CandidateUnattempted  CandidateLoadDisposition = "unattempted_after_success"
)

type WriteStep string

const (
	StepTempCreate        WriteStep = "temp_create"
	StepPayloadEncode     WriteStep = "payload_encode"
	StepFileSync          WriteStep = "file_sync"
	StepFileClose         WriteStep = "file_close"
	StepReopenValidation  WriteStep = "reopen_validation"
	StepGenerationRename  WriteStep = "generation_rename"
	StepGenerationDirSync WriteStep = "generation_directory_sync"
	StepManifestCreate    WriteStep = "manifest_create"
	StepManifestEncode    WriteStep = "manifest_encode"
	StepManifestSync      WriteStep = "manifest_sync"
	StepManifestClose     WriteStep = "manifest_close"
	StepManifestReopen    WriteStep = "manifest_reopen_validation"
	StepManifestRename    WriteStep = "manifest_rename"
	StepManifestDirSync   WriteStep = "manifest_directory_sync"
	StepCleanup           WriteStep = "cleanup"
)

type WriteDisposition string

const (
	WriteCompleted       WriteDisposition = "completed"
	WriteCleanupDeferred WriteDisposition = "completed_cleanup_deferred"
	WriteFailed          WriteDisposition = "failed"
	WriteCanceled        WriteDisposition = "canceled"
)

type StoreConfig struct {
	Directory, BindingIdentity string
	ArtifactByteLimit          int64
	OperationDeadline          time.Duration
}

type Store struct {
	directory, binding string
	byteLimit          int64
	deadline           time.Duration
	stepHook           func(WriteStep) error
	writeMu            sync.Mutex
}

type WriteResult struct {
	Disposition WriteDisposition
	Step        WriteStep
	Entry       ManifestEntry
	Err         error
}

func NewStore(config StoreConfig) (*Store, error) {
	if config.Directory == "" || config.BindingIdentity == "" || !filepath.IsAbs(config.Directory) || config.ArtifactByteLimit <= 0 || config.ArtifactByteLimit > AbsoluteByteCeiling || config.OperationDeadline <= 0 || config.OperationDeadline > time.Minute {
		return nil, errors.New("complete bounded absolute checkpoint store configuration required")
	}
	clean := filepath.Clean(config.Directory)
	if info, err := os.Lstat(clean); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, errors.New("checkpoint directory must be a real directory")
		}
		resolved, err := filepath.EvalSymlinks(clean)
		if err != nil {
			return nil, err
		}
		clean = resolved
	} else if errors.Is(err, os.ErrNotExist) {
		parent, err := filepath.EvalSymlinks(filepath.Dir(clean))
		if err != nil {
			return nil, err
		}
		clean = filepath.Join(parent, filepath.Base(clean))
		if err := os.Mkdir(clean, 0700); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	if err := os.Chmod(clean, 0700); err != nil {
		return nil, err
	}
	return &Store{directory: clean, binding: config.BindingIdentity, byteLimit: config.ArtifactByteLimit, deadline: config.OperationDeadline}, nil
}

func (s *Store) SetStepHookForTest(hook func(WriteStep) error) { s.stepHook = hook }

func (s *Store) Directory() string { return s.directory }

func (s *Store) Load(ctx context.Context) LoadResult {
	if ctx == nil {
		return LoadResult{Disposition: LoadCanceled}
	}
	operation, cancel := context.WithTimeout(ctx, s.deadline)
	defer cancel()
	manifestData, err := readSafeFile(filepath.Join(s.directory, "manifest.json"), ManifestByteLimit)
	if err != nil {
		return LoadResult{Disposition: LoadUnavailable}
	}
	var manifest Manifest
	if err := strictJSON(manifestData, &manifest); err != nil || !validManifest(manifest) {
		return LoadResult{Disposition: LoadUnavailable}
	}
	entries := []*ManifestEntry{manifest.Latest, manifest.Previous}
	outcomes := [2]CandidateLoadDisposition{CandidateUnavailable, CandidateUnavailable}
	var latestErr error
	var incompatible bool
	for index, entry := range entries {
		if entry == nil {
			continue
		}
		candidate, err := s.loadEntry(operation, *entry)
		if err == nil {
			outcomes[index] = CandidateLoaded
			if index == 0 {
				outcomes[1] = CandidateUnattempted
			}
			disposition := LoadedLatest
			if index == 1 {
				disposition = LoadedPrevious
			}
			return LoadResult{Disposition: disposition, Candidate: candidate, Entry: *entry, LatestError: latestErr, Candidates: outcomes}
		}
		if index == 0 {
			latestErr = err
		}
		if errors.Is(err, ErrCanceled) {
			outcomes[index] = CandidateCanceled
			return LoadResult{Disposition: LoadCanceled, LatestError: latestErr, Candidates: outcomes}
		}
		if errors.Is(err, ErrIncompatible) {
			outcomes[index] = CandidateIncompatible
		} else {
			outcomes[index] = CandidateInvalid
		}
		incompatible = incompatible || errors.Is(err, ErrIncompatible)
	}
	if incompatible {
		return LoadResult{Disposition: LoadIncompatible, LatestError: latestErr, Candidates: outcomes}
	}
	return LoadResult{Disposition: LoadInvalid, LatestError: latestErr, Candidates: outcomes}
}

func (s *Store) loadEntry(ctx context.Context, entry ManifestEntry) (Candidate, error) {
	if !validManifestEntry(entry) || entry.BindingIdentity != s.binding {
		return Candidate{}, ErrIncompatible
	}
	return s.loadPath(ctx, filepath.Join(s.directory, entry.Filename), entry)
}

func (s *Store) loadPath(ctx context.Context, path string, entry ManifestEntry) (Candidate, error) {
	file, info, err := openSafeRegular(path)
	if err != nil {
		return Candidate{}, err
	}
	defer file.Close()
	if info.Size() != entry.FileBytes || info.Size() > s.byteLimit {
		return Candidate{}, ErrInvalid
	}
	candidate, bytesRead, err := Decode(ctx, file, s.byteLimit)
	if err != nil {
		return Candidate{}, err
	}
	if bytesRead != entry.FileBytes || candidate.Checksum != entry.PayloadSHA256 || candidate.Image.Binding.Identity != entry.BindingIdentity || candidate.Image.T0 != entry.T0 || candidate.Image.Sequence != entry.Sequence {
		return Candidate{}, ErrInvalid
	}
	return candidate, nil
}

func (s *Store) Write(ctx context.Context, image Image) WriteResult {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if ctx == nil {
		return WriteResult{Disposition: WriteCanceled, Err: ErrCanceled}
	}
	operation, cancel := context.WithTimeout(ctx, s.deadline)
	defer cancel()
	if image.Binding.Identity != s.binding || image.Sequence == 0 {
		return WriteResult{Disposition: WriteFailed, Step: StepPayloadEncode, Err: ErrIncompatible}
	}
	name := fmt.Sprintf("checkpoint-%020d.json", image.Sequence)
	entry := ManifestEntry{Filename: name, BindingIdentity: s.binding, T0: image.T0, Sequence: image.Sequence}
	tempPath, manifestTemp := "", ""
	fail := func(step WriteStep, err error) WriteResult {
		if tempPath != "" {
			_ = os.Remove(tempPath)
		}
		if manifestTemp != "" {
			_ = os.Remove(manifestTemp)
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrCanceled) {
			return WriteResult{Disposition: WriteCanceled, Step: step, Entry: entry, Err: err}
		}
		return WriteResult{Disposition: WriteFailed, Step: step, Entry: entry, Err: err}
	}
	step := func(value WriteStep) error {
		if err := operation.Err(); err != nil {
			return err
		}
		if s.stepHook != nil {
			return s.stepHook(value)
		}
		return nil
	}
	if err := step(StepTempCreate); err != nil {
		return fail(StepTempCreate, err)
	}
	file, err := os.CreateTemp(s.directory, ".checkpoint-temp-")
	if err != nil {
		return fail(StepTempCreate, err)
	}
	tempPath = file.Name()
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return fail(StepTempCreate, err)
	}
	if err := step(StepPayloadEncode); err != nil {
		file.Close()
		return fail(StepPayloadEncode, err)
	}
	entry.FileBytes, entry.PayloadSHA256, err = Encode(operation, file, image, s.byteLimit)
	if err != nil {
		file.Close()
		return fail(StepPayloadEncode, err)
	}
	if err := step(StepFileSync); err != nil {
		file.Close()
		return fail(StepFileSync, err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fail(StepFileSync, err)
	}
	if err := step(StepFileClose); err != nil {
		file.Close()
		return fail(StepFileClose, err)
	}
	if err := file.Close(); err != nil {
		return fail(StepFileClose, err)
	}
	if err := step(StepReopenValidation); err != nil {
		return fail(StepReopenValidation, err)
	}
	if candidate, err := s.loadPath(operation, tempPath, entry); err != nil || candidate.Image.Sequence != image.Sequence {
		return fail(StepReopenValidation, errors.Join(err, ErrInvalid))
	}
	if err := step(StepGenerationRename); err != nil {
		return fail(StepGenerationRename, err)
	}
	generationPath := filepath.Join(s.directory, name)
	if _, err := os.Lstat(generationPath); err == nil || !errors.Is(err, os.ErrNotExist) {
		return fail(StepGenerationRename, errors.New("immutable generation already exists"))
	}
	if err := os.Rename(tempPath, generationPath); err != nil {
		return fail(StepGenerationRename, err)
	}
	if err := step(StepGenerationDirSync); err != nil {
		return fail(StepGenerationDirSync, err)
	}
	if err := syncDirectory(s.directory); err != nil {
		return fail(StepGenerationDirSync, err)
	}
	old := s.readManifest()
	manifest := Manifest{Version: manifestVersion, Latest: &entry}
	if old.Latest != nil {
		copy := *old.Latest
		manifest.Previous = &copy
	}
	if err := step(StepManifestCreate); err != nil {
		return fail(StepManifestCreate, err)
	}
	mf, err := os.CreateTemp(s.directory, ".manifest-temp-")
	if err != nil {
		return fail(StepManifestCreate, err)
	}
	manifestTemp = mf.Name()
	if err := mf.Chmod(0600); err != nil {
		mf.Close()
		return fail(StepManifestCreate, err)
	}
	if err := step(StepManifestEncode); err != nil {
		mf.Close()
		return fail(StepManifestEncode, err)
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil || int64(len(manifestBytes)) > ManifestByteLimit {
		mf.Close()
		return fail(StepManifestEncode, ErrLimit)
	}
	if _, err := mf.Write(manifestBytes); err != nil {
		mf.Close()
		return fail(StepManifestEncode, err)
	}
	if err := step(StepManifestSync); err != nil {
		mf.Close()
		return fail(StepManifestSync, err)
	}
	if err := mf.Sync(); err != nil {
		mf.Close()
		return fail(StepManifestSync, err)
	}
	if err := step(StepManifestClose); err != nil {
		mf.Close()
		return fail(StepManifestClose, err)
	}
	if err := mf.Close(); err != nil {
		return fail(StepManifestClose, err)
	}
	if err := step(StepManifestReopen); err != nil {
		return fail(StepManifestReopen, err)
	}
	check, err := readSafeFile(manifestTemp, ManifestByteLimit)
	var reopened Manifest
	if err != nil || strictJSON(check, &reopened) != nil || !manifestEqual(manifest, reopened) {
		return fail(StepManifestReopen, ErrInvalid)
	}
	if err := step(StepManifestRename); err != nil {
		return fail(StepManifestRename, err)
	}
	if err := os.Rename(manifestTemp, filepath.Join(s.directory, "manifest.json")); err != nil {
		return fail(StepManifestRename, err)
	}
	if err := step(StepManifestDirSync); err != nil {
		return WriteResult{Disposition: WriteFailed, Step: StepManifestDirSync, Entry: entry, Err: err}
	}
	if err := syncDirectory(s.directory); err != nil {
		return WriteResult{Disposition: WriteFailed, Step: StepManifestDirSync, Entry: entry, Err: err}
	}
	if err := step(StepCleanup); err != nil {
		return WriteResult{Disposition: WriteCleanupDeferred, Step: StepCleanup, Entry: entry, Err: err}
	}
	if err := s.cleanup(manifest); err != nil {
		return WriteResult{Disposition: WriteCleanupDeferred, Step: StepCleanup, Entry: entry, Err: err}
	}
	return WriteResult{Disposition: WriteCompleted, Entry: entry}
}

func (s *Store) readManifest() Manifest {
	data, err := readSafeFile(filepath.Join(s.directory, "manifest.json"), ManifestByteLimit)
	if err != nil {
		return Manifest{Version: manifestVersion}
	}
	var manifest Manifest
	if strictJSON(data, &manifest) != nil || !validManifest(manifest) {
		return Manifest{Version: manifestVersion}
	}
	return manifest
}

func (s *Store) cleanup(manifest Manifest) error {
	directory, err := os.Open(s.directory)
	if err != nil {
		return err
	}
	defer directory.Close()
	referenced := map[string]bool{"manifest.json": true}
	if manifest.Latest != nil {
		referenced[manifest.Latest.Filename] = true
	}
	if manifest.Previous != nil {
		referenced[manifest.Previous.Filename] = true
	}
	inspected, removed := 0, 0
	for {
		entries, readErr := directory.ReadDir(32)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		if inspected+len(entries) > 256 {
			return errors.New("cleanup inspection bound")
		}
		inspected += len(entries)
		for _, item := range entries {
			if !generationName.MatchString(item.Name()) && !bytes.HasPrefix([]byte(item.Name()), []byte(".checkpoint-temp-")) && !bytes.HasPrefix([]byte(item.Name()), []byte(".manifest-temp-")) {
				continue
			}
			if referenced[item.Name()] || removed == 16 {
				continue
			}
			path := filepath.Join(s.directory, item.Name())
			info, err := os.Lstat(path)
			stat, linked := infoSysStat(info)
			if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || !linked || stat.Nlink != 1 {
				return errors.New("unsafe cleanup entry")
			}
			if err := os.Remove(path); err != nil {
				return err
			}
			removed++
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
	}
	return nil
}

func infoSysStat(info os.FileInfo) (*syscall.Stat_t, bool) {
	if info == nil {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return stat, ok
}

func validManifest(manifest Manifest) bool {
	return manifest.Version == manifestVersion && manifest.Latest != nil && validManifestEntry(*manifest.Latest) &&
		(manifest.Previous == nil || validManifestEntry(*manifest.Previous) && manifest.Previous.Filename != manifest.Latest.Filename)
}

func validManifestEntry(entry ManifestEntry) bool {
	return generationName.MatchString(entry.Filename) && entry.Filename == fmt.Sprintf("checkpoint-%020d.json", entry.Sequence) && entry.FileBytes > 0 && lowerHex(entry.PayloadSHA256) && entry.BindingIdentity != "" && !entry.T0.IsZero() && entry.T0.Nanosecond() == 0 && entry.Sequence > 0
}

func lowerHex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !('0' <= r && r <= '9') && !('a' <= r && r <= 'f') {
			return false
		}
	}
	return true
}

func manifestEqual(a, b Manifest) bool {
	dataA, _ := json.Marshal(a)
	dataB, _ := json.Marshal(b)
	return bytes.Equal(dataA, dataB)
}

func openSafeRegular(path string) (*os.File, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return nil, nil, ErrInvalid
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok || stat.Nlink != 1 {
		return nil, nil, ErrInvalid
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		file.Close()
		return nil, nil, ErrInvalid
	}
	return file, opened, nil
}

func readSafeFile(path string, limit int64) ([]byte, error) {
	file, info, err := openSafeRegular(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if info.Size() > limit {
		return nil, ErrLimit
	}
	return io.ReadAll(io.LimitReader(file, limit+1))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
