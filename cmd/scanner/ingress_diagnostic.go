package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

const ingressDiagnosticSchema = 1

func persistIngressIncident(directory string, incident *operations.IngressIncident) (string, error) {
	if directory == "" || incident == nil {
		return "", errors.New("ingress diagnostic persistence requires directory and incident")
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", errors.New("create ingress diagnostic directory")
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return "", errors.New("protect ingress diagnostic directory")
	}
	at := incident.EngineCapturedAt
	if at.IsZero() {
		at = incident.CapturedAt
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	name := fmt.Sprintf("live-ingress-%s.json", at.UTC().Format("20060102T150405.000000000Z"))
	path := filepath.Join(directory, name)
	file, err := os.CreateTemp(directory, ".live-ingress-*.tmp")
	if err != nil {
		return "", errors.New("create ingress diagnostic temporary file")
	}
	temporaryPath := file.Name()
	defer os.Remove(temporaryPath)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return "", errors.New("protect ingress diagnostic temporary file")
	}
	encodeErr := json.NewEncoder(file).Encode(struct {
		Schema   int                         `json:"schema"`
		Incident *operations.IngressIncident `json:"incident"`
	}{Schema: ingressDiagnosticSchema, Incident: incident})
	syncErr := file.Sync()
	closeErr := file.Close()
	if encodeErr != nil || syncErr != nil || closeErr != nil {
		return "", errors.Join(errors.New("persist ingress diagnostic"), encodeErr, syncErr, closeErr)
	}
	// A hard link publishes the fully written file in one namespace operation
	// and, unlike rename, refuses to replace an existing incident.
	if err := os.Link(temporaryPath, path); err != nil {
		return "", errors.New("publish ingress diagnostic without overwrite")
	}
	if err := os.Remove(temporaryPath); err != nil {
		return "", errors.New("remove published ingress diagnostic temporary file")
	}
	directoryFile, err := os.Open(directory)
	if err != nil {
		return "", errors.New("open ingress diagnostic directory for sync")
	}
	directorySyncErr := directoryFile.Sync()
	directoryCloseErr := directoryFile.Close()
	if directorySyncErr != nil || directoryCloseErr != nil {
		return "", errors.Join(errors.New("sync ingress diagnostic directory"), directorySyncErr, directoryCloseErr)
	}
	return path, nil
}
