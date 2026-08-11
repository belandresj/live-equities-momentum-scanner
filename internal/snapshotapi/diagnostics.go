package snapshotapi

import (
	"errors"
	"sync"
)

// MappingInvariant is a closed, bounded identifier for the exact mapper gate
// that rejected one captured publication. It is intentionally private
// operational evidence; public HTTP responses remain generic.
type MappingInvariant string

type mappingError struct {
	invariant MappingInvariant
}

func (failure *mappingError) Error() string {
	return "snapshot mapping rejected: " + string(failure.invariant)
}

func rejectMapping(invariant string) error {
	return &mappingError{invariant: MappingInvariant(invariant)}
}

func mappingInvariant(err error) MappingInvariant {
	var failure *mappingError
	if errors.As(err, &failure) {
		return failure.invariant
	}
	return "unclassified"
}

// MappingFailure contains only fixed-size counters and closed engine/mapper
// labels. It deliberately excludes arbitrary error text and payload data.
type MappingFailure struct {
	Invariant          MappingInvariant `json:"invariant"`
	Route              string           `json:"route"`
	PublicationID      string           `json:"publication_id"`
	LastEngineSequence string           `json:"last_engine_sequence"`
	Lifecycle          string           `json:"lifecycle"`
	RankingMode        string           `json:"ranking_mode"`
}

// MappingDiagnostics retains one latest failure and one pending notification.
// Repeated failed requests therefore consume constant memory.
type MappingDiagnostics struct {
	mu      sync.Mutex
	latest  MappingFailure
	present bool
	updates chan MappingFailure
}

func NewMappingDiagnostics() *MappingDiagnostics {
	return &MappingDiagnostics{updates: make(chan MappingFailure, 1)}
}

func (diagnostics *MappingDiagnostics) record(failure MappingFailure) {
	if diagnostics == nil {
		return
	}
	diagnostics.mu.Lock()
	defer diagnostics.mu.Unlock()
	diagnostics.latest, diagnostics.present = failure, true
	select {
	case diagnostics.updates <- failure:
		return
	default:
	}
	select {
	case <-diagnostics.updates:
	default:
	}
	select {
	case diagnostics.updates <- failure:
	default:
	}
}

func (diagnostics *MappingDiagnostics) Latest() (MappingFailure, bool) {
	if diagnostics == nil {
		return MappingFailure{}, false
	}
	diagnostics.mu.Lock()
	defer diagnostics.mu.Unlock()
	return diagnostics.latest, diagnostics.present
}

func (diagnostics *MappingDiagnostics) Updates() <-chan MappingFailure {
	if diagnostics == nil {
		return nil
	}
	return diagnostics.updates
}
