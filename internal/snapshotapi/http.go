package snapshotapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

const maximumResponseBytes = 1 << 20

type CaptureSource interface {
	CaptureSnapshot() (operations.SnapshotCapture, error)
}

type HandlerConfig struct {
	AllowedOrigins []string
	Diagnostics    *MappingDiagnostics
}

type handler struct {
	source      CaptureSource
	origins     map[string]struct{}
	diagnostics *MappingDiagnostics
}

func NewHandler(source CaptureSource, config HandlerConfig) (http.Handler, error) {
	if source == nil || nilValue(source) {
		return nil, errors.New("snapshot source is required")
	}
	origins := make(map[string]struct{}, len(config.AllowedOrigins))
	for _, origin := range config.AllowedOrigins {
		if !validOrigin(origin) {
			return nil, errors.New("invalid allowed origin")
		}
		if _, exists := origins[origin]; exists {
			return nil, errors.New("duplicate allowed origin")
		}
		origins[origin] = struct{}{}
	}
	return &handler{source: source, origins: origins, diagnostics: config.Diagnostics}, nil
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	origin, allowed, preflight := h.authorizeOrigin(request)
	if !allowed {
		h.write(writer, request.Method, http.StatusForbidden, errorResponse{Error: "forbidden"}, "")
		return
	}
	if preflight {
		if !productRoute(request.URL.Path) || request.URL.RawQuery != "" || request.ContentLength > 0 {
			h.write(writer, request.Method, http.StatusForbidden, errorResponse{Error: "forbidden"}, "")
			return
		}
		h.preflight(writer, origin)
		return
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		h.write(writer, request.Method, http.StatusMethodNotAllowed, errorResponse{Error: "method_not_allowed"}, origin)
		return
	}
	if request.URL.RawQuery != "" || request.ContentLength > 0 {
		h.write(writer, request.Method, http.StatusBadRequest, errorResponse{Error: "invalid_request"}, origin)
		return
	}
	if !productRoute(request.URL.Path) {
		h.write(writer, request.Method, http.StatusNotFound, errorResponse{Error: "not_found"}, origin)
		return
	}
	if request.Context().Err() != nil {
		return
	}
	capture, err := h.source.CaptureSnapshot()
	if err != nil {
		h.write(writer, request.Method, http.StatusServiceUnavailable, errorResponse{Error: "capture_unavailable"}, origin)
		return
	}
	view, ok := operations.InspectSnapshotCapture(capture)
	if !ok {
		h.write(writer, request.Method, http.StatusServiceUnavailable, errorResponse{Error: "capture_unavailable"}, origin)
		return
	}
	switch request.URL.Path {
	case "/livez":
		reason := ""
		status := http.StatusOK
		if !view.ProcessLive {
			reason, status = "runtime_unavailable", http.StatusServiceUnavailable
		}
		h.write(writer, request.Method, status, livenessResponse{SchemaVersion: "scanner.liveness.v1", SampleID: decimal(view.SampleID), SampledAt: timestamp(view.SampledAt), ProcessLive: view.ProcessLive, Reason: reason}, origin)
	case "/readyz":
		snapshot, mapErr := Map(capture)
		response := readinessResponse{SchemaVersion: "scanner.readiness.v1", SampleID: decimal(view.SampleID), SampledAt: timestamp(view.SampledAt), ProcessLive: view.ProcessLive}
		status := http.StatusServiceUnavailable
		if mapErr != nil {
			h.recordMappingFailure(request.URL.Path, view, mapErr)
			response.Reason = "publication_unavailable"
		} else {
			response.PublicationID, response.BindingIdentity = &snapshot.Publication.ID, &snapshot.Publication.BindingIdentity
			response.BackendReady, response.Reason = snapshot.Status.BackendReady, snapshot.Status.ReadinessReason
			if response.BackendReady {
				status = http.StatusOK
			}
		}
		h.write(writer, request.Method, status, response, origin)
	default:
		snapshot, mapErr := Map(capture)
		if mapErr != nil {
			h.recordMappingFailure(request.URL.Path, view, mapErr)
			h.write(writer, request.Method, http.StatusServiceUnavailable, errorResponse{Error: "snapshot_unavailable"}, origin)
			return
		}
		h.write(writer, request.Method, http.StatusOK, snapshot, origin)
	}
}

func (h *handler) recordMappingFailure(route string, capture operations.SnapshotCaptureView, err error) {
	publication := capture.Engine.Publication
	h.diagnostics.record(MappingFailure{Invariant: mappingInvariant(err), Route: route, PublicationID: decimal(publication.PublicationID),
		LastEngineSequence: decimal(publication.LastEngineSequence), Lifecycle: publication.Lifecycle, RankingMode: publication.AggregateEvaluation.Mode})
}

func productRoute(path string) bool {
	return path == "/api/v2/snapshot" || path == "/livez" || path == "/readyz"
}

func (h *handler) authorizeOrigin(request *http.Request) (origin string, allowed, preflight bool) {
	values := request.Header.Values("Origin")
	if len(values) == 0 {
		return "", request.Method != http.MethodOptions, false
	}
	if len(values) != 1 || !validOrigin(values[0]) {
		return "", false, false
	}
	origin = values[0]
	if _, exists := h.origins[origin]; !exists {
		return "", false, false
	}
	if request.Method != http.MethodOptions {
		return origin, true, false
	}
	methods := request.Header.Values("Access-Control-Request-Method")
	if len(methods) != 1 || methods[0] != http.MethodGet && methods[0] != http.MethodHead || len(request.Header.Values("Access-Control-Request-Headers")) != 0 {
		return "", false, false
	}
	return origin, true, true
}

func (h *handler) preflight(writer http.ResponseWriter, origin string) {
	headers := writer.Header()
	standardHeaders(headers)
	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Set("Access-Control-Allow-Methods", "GET, HEAD")
	headers.Set("Vary", "Origin")
	headers.Set("Content-Length", "0")
	writer.WriteHeader(http.StatusNoContent)
}

func (h *handler) write(writer http.ResponseWriter, method string, status int, value any, origin string) {
	body, err := marshalBounded(value)
	if err != nil {
		status = http.StatusServiceUnavailable
		body = []byte(`{"error":"response_unavailable"}`)
	}
	headers := writer.Header()
	standardHeaders(headers)
	if origin != "" {
		headers.Set("Access-Control-Allow-Origin", origin)
		headers.Set("Vary", "Origin")
	}
	headers.Set("Content-Length", strconv.Itoa(len(body)))
	writer.WriteHeader(status)
	if method != http.MethodHead {
		_, _ = writer.Write(body)
	}
}

func standardHeaders(headers http.Header) {
	headers.Set("Content-Type", "application/json")
	headers.Set("Cache-Control", "no-store")
	headers.Set("X-Content-Type-Options", "nosniff")
}

func marshalBounded(value any) ([]byte, error) {
	body, err := json.Marshal(value)
	if err != nil || len(body) > maximumResponseBytes {
		return nil, errors.New("response exceeds bound")
	}
	return body, nil
}

func validOrigin(value string) bool {
	if value == "" || value == "null" || strings.TrimSpace(value) != value {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Opaque != "" || parsed.Path != "" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	if parsed.Hostname() == "" || strings.HasSuffix(parsed.Host, ":") {
		return false
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.ParseUint(port, 10, 16)
		if err != nil || value == 0 || value > 65535 {
			return false
		}
	}
	return parsed.Scheme+"://"+parsed.Host == value
}

func nilValue(value any) bool {
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

type livenessResponse struct {
	SchemaVersion string `json:"schema_version"`
	SampleID      string `json:"sample_id"`
	SampledAt     string `json:"sampled_at"`
	ProcessLive   bool   `json:"process_live"`
	Reason        string `json:"reason"`
}

type readinessResponse struct {
	SchemaVersion   string  `json:"schema_version"`
	SampleID        string  `json:"sample_id"`
	SampledAt       string  `json:"sampled_at"`
	ProcessLive     bool    `json:"process_live"`
	PublicationID   *string `json:"publication_id"`
	BindingIdentity *string `json:"binding_identity"`
	BackendReady    bool    `json:"backend_ready"`
	Reason          string  `json:"reason"`
}
