package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

type diagnosticLog struct {
	file     io.WriteCloser
	encoder  *json.Encoder
	warnings io.Writer
	disabled bool
	warned   bool
}

func openDiagnosticLog(path string, warnings io.Writer) (*diagnosticLog, error) {
	if path == "" {
		return nil, nil
	}
	if !filepath.IsAbs(path) {
		return nil, errors.New("diagnostic log path must be absolute")
	}
	descriptor, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_APPEND|syscall.O_CREAT|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, errors.New("open diagnostic log safely")
	}
	file := os.NewFile(uintptr(descriptor), path)
	if file == nil {
		_ = syscall.Close(descriptor)
		return nil, errors.New("open diagnostic log handle")
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		_ = file.Close()
		return nil, errors.New("diagnostic log must be a regular 0600 file")
	}
	return newDiagnosticLog(file, warnings), nil
}

func newDiagnosticLog(file io.WriteCloser, warnings io.Writer) *diagnosticLog {
	return &diagnosticLog{file: file, encoder: json.NewEncoder(file), warnings: warnings}
}

func (d *diagnosticLog) Write(status operations.Status, metrics operations.Metrics) {
	if d == nil || d.disabled {
		return
	}
	if err := d.encoder.Encode(struct {
		Status  operations.Status
		Metrics operations.Metrics
	}{status, metrics}); err != nil {
		d.disabled = true
		_ = d.file.Close()
		if !d.warned {
			d.warned = true
			_, _ = fmt.Fprintln(d.warnings, "warning: diagnostic log write failed; private diagnostics disabled")
		}
	}
}

func (d *diagnosticLog) Close() error {
	if d == nil || d.file == nil || d.disabled {
		return nil
	}
	d.disabled = true
	return d.file.Close()
}
