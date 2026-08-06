package reference

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maximumCacheBytes            = 16 << 20
	maximumCacheDates            = 14
	maximumCacheDirectoryEntries = 64

	persistenceCachePrepareFailed     = "cache_prepare_failed"
	persistenceCachePublicationFailed = "cache_publication_failed"
	persistenceCachePruneFailed       = "cache_prune_failed"
)

func inspectCacheDirectory(root, class string) (string, error) {
	if root == "" {
		return "", errors.New("reference data directory is required")
	}
	directory := filepath.Join(root, class)
	if err := validateNoSymlinkPath(directory); err != nil {
		return "", err
	}
	if err := validatePrivateDirectory(root); err != nil {
		return "", err
	}
	if err := validatePrivateDirectory(directory); err != nil {
		return "", err
	}
	return directory, nil
}

func prepareCacheDirectory(root, class string) (string, error) {
	if root == "" {
		return "", errors.New("reference data directory is required")
	}
	if err := createPrivateDirectory(root); err != nil {
		return "", err
	}
	directory := filepath.Join(root, class)
	if err := createPrivateDirectory(directory); err != nil {
		return "", err
	}
	return directory, nil
}

// createPrivateDirectory creates missing path components one at a time after
// proving that no existing component is a symlink. It never uses MkdirAll,
// whose traversal would follow an ancestor symlink.
func createPrivateDirectory(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve cache directory: %w", err)
	}
	if err := validateNoSymlinkPath(absolute); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	missing := make([]string, 0, 2)
	cursor := absolute
	for {
		info, statErr := os.Lstat(cursor)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return errors.New("cache path component must be a non-symlink directory")
			}
			break
		}
		if !errors.Is(statErr, fs.ErrNotExist) {
			return fmt.Errorf("inspect cache path component: %w", statErr)
		}
		parent := filepath.Dir(cursor)
		if parent == cursor {
			return errors.New("cache directory has no existing directory ancestor")
		}
		missing = append(missing, filepath.Base(cursor))
		cursor = parent
	}
	for index := len(missing) - 1; index >= 0; index-- {
		cursor = filepath.Join(cursor, missing[index])
		if err := os.Mkdir(cursor, 0o700); err != nil {
			return fmt.Errorf("create private cache directory: %w", err)
		}
		if err := validatePrivateDirectory(cursor); err != nil {
			return err
		}
	}
	return validatePrivateDirectory(absolute)
}

func validateNoSymlinkPath(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve cache path: %w", err)
	}
	volume := filepath.VolumeName(absolute)
	remainder := strings.TrimPrefix(absolute, volume)
	current := volume + string(filepath.Separator)
	for _, component := range strings.Split(strings.TrimPrefix(remainder, string(filepath.Separator)), string(filepath.Separator)) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, fs.ErrNotExist) {
			return fs.ErrNotExist
		}
		if statErr != nil {
			return fmt.Errorf("inspect cache path component: %w", statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("cache path contains a symlink")
		}
	}
	return nil
}

func validatePrivateDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return errors.New("cache directory must be a non-symlink 0700 directory")
	}
	return nil
}

func validatePrivateRegularFile(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return nil, errors.New("cache file must be a non-symlink 0600 regular file")
	}
	return info, nil
}

func readCacheDirectoryBounded(directory string) ([]os.DirEntry, error) {
	if err := validateNoSymlinkPath(directory); err != nil {
		return nil, err
	}
	if err := validatePrivateDirectory(directory); err != nil {
		return nil, err
	}
	handle, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	entries, err := handle.ReadDir(maximumCacheDirectoryEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > maximumCacheDirectoryEntries {
		return nil, errors.New("cache directory exceeds 64 entries")
	}
	return entries, nil
}

type cacheFileOps struct {
	syncFile      func(*os.File) error
	rename        func(string, string) error
	syncDirectory func(string) error
}

func publishPrivateCacheFile(directory, filename, temporaryPrefix string, body []byte, operations *cacheFileOps) error {
	if len(body) > maximumCacheBytes {
		return errors.New("derived cache exceeds 16 MiB")
	}
	if err := validateNoSymlinkPath(directory); err != nil {
		return err
	}
	if err := validatePrivateDirectory(directory); err != nil {
		return err
	}
	path := filepath.Join(directory, filename)
	entries, err := readCacheDirectoryBounded(directory)
	if err != nil {
		return err
	}
	targetExists := false
	if _, err := os.Lstat(path); err == nil {
		targetExists = true
		if _, err := validatePrivateRegularFile(path); err != nil {
			return err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if !targetExists && len(entries) == maximumCacheDirectoryEntries {
		return errors.New("cache directory has no bounded slot for publication")
	}

	temporary, err := os.CreateTemp(directory, temporaryPrefix+"*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(body); err != nil {
		_ = temporary.Close()
		return err
	}
	syncFile := (*os.File).Sync
	if operations != nil && operations.syncFile != nil {
		syncFile = operations.syncFile
	}
	if err := syncFile(temporary); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	rename := os.Rename
	if operations != nil && operations.rename != nil {
		rename = operations.rename
	}
	if err := rename(temporaryPath, path); err != nil {
		return err
	}
	renamed = true
	directorySync := syncDirectory
	if operations != nil && operations.syncDirectory != nil {
		directorySync = operations.syncDirectory
	}
	return directorySync(directory)
}

func pruneCacheDirectory(directory, temporaryPrefix string, now time.Time) error {
	entries, err := readCacheDirectoryBounded(directory)
	if err != nil {
		return err
	}
	dates := make([]string, 0, maximumCacheDates+1)
	removed := false
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("refuse to inspect symlink in cache directory")
		}
		if validCacheFilename(entry.Name()) {
			if _, err := validatePrivateRegularFile(path); err != nil {
				return err
			}
			dates = append(dates, entry.Name())
			continue
		}
		if recognizedTemporary(entry.Name(), temporaryPrefix) {
			info, err := validatePrivateRegularFile(path)
			if err != nil {
				return err
			}
			if now.Sub(info.ModTime()) > operationTimeout {
				if err := os.Remove(path); err != nil {
					return err
				}
				removed = true
			}
		}
	}
	sort.Strings(dates)
	for len(dates) > maximumCacheDates {
		if err := os.Remove(filepath.Join(directory, dates[0])); err != nil {
			return err
		}
		dates = dates[1:]
		removed = true
	}
	if removed {
		return syncDirectory(directory)
	}
	return nil
}

func recognizedTemporary(name, prefix string) bool {
	return strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".tmp") && len(name) > len(prefix)+len(".tmp")
}

func syncDirectory(directory string) error {
	handle, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer handle.Close()
	return handle.Sync()
}

func canonicalCacheTime(instant time.Time) string {
	return instant.UTC().Format(time.RFC3339Nano)
}

func parseCanonicalCacheTime(encoded string) (time.Time, error) {
	if !strings.HasSuffix(encoded, "Z") {
		return time.Time{}, errors.New("cache timestamp is not zero-offset")
	}
	parsed, err := time.Parse(time.RFC3339Nano, encoded)
	if err != nil || canonicalCacheTime(parsed) != encoded {
		return time.Time{}, errors.New("cache timestamp is not canonical RFC3339 nanosecond form")
	}
	return parsed, nil
}

func decodeStrictCacheJSON(body []byte, target any) error {
	if err := rejectDuplicateJSONMembers(body); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func rejectDuplicateJSONMembers(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := inspectJSONValue(decoder); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return err
	}
	return nil
}

func inspectJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			nameToken, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := nameToken.(string)
			if !ok {
				return errors.New("JSON object member name is not a string")
			}
			if _, duplicate := seen[name]; duplicate {
				return fmt.Errorf("duplicate JSON member %q", name)
			}
			seen[name] = struct{}{}
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return errors.New("unexpected JSON delimiter")
	}
}
