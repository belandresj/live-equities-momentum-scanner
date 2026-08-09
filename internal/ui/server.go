package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultAddress   = "127.0.0.1:4173"
	DefaultAPIOrigin = "http://127.0.0.1:8080"
	maximumAssetSize = 1 << 20
)

type Config struct {
	Address, AssetRoot, APIOrigin string
}

type Server struct {
	listener net.Listener
	http     *http.Server
	done     chan error
}

func Listen(config Config) (*Server, error) {
	if config.Address == "" {
		config.Address = DefaultAddress
	}
	if config.APIOrigin == "" {
		config.APIOrigin = DefaultAPIOrigin
	}
	if !loopbackAddress(config.Address) {
		return nil, errors.New("dashboard address must be an explicit loopback IP and port")
	}
	if !loopbackOrigin(config.APIOrigin) {
		return nil, errors.New("API origin must be an exact loopback http(s) origin")
	}
	root, err := filepath.Abs(config.AssetRoot)
	if err != nil || config.AssetRoot == "" {
		return nil, errors.New("dashboard asset root is required")
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve asset root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, errors.New("dashboard asset root must be a directory")
	}
	handler := &assetHandler{root: root, apiOrigin: config.APIOrigin}
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		return nil, err
	}
	server := &Server{listener: listener, done: make(chan error, 1)}
	server.http = &http.Server{Handler: handler, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 3 * time.Second,
		WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() {
		err := server.http.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		server.done <- err
		close(server.done)
	}()
	return server, nil
}

func (s *Server) Address() string                    { return s.listener.Addr().String() }
func (s *Server) Done() <-chan error                 { return s.done }
func (s *Server) Shutdown(ctx context.Context) error { return s.http.Shutdown(ctx) }

type assetHandler struct {
	root, apiOrigin string
}

var assets = map[string]string{
	"/":           "index.html",
	"/index.html": "index.html",
	"/styles.css": "styles.css",
	"/app.js":     "app.js",
	"/model.js":   "model.js",
	"/render.js":  "render.js",
}

func (h *assetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setHeaders(w.Header(), h.apiOrigin)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.RawQuery != "" || r.ContentLength > 0 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if r.URL.Path == "/config.js" {
		body, _ := json.Marshal(map[string]any{"apiOrigin": h.apiOrigin, "pollMilliseconds": 1000, "requestTimeoutMilliseconds": 3000})
		h.serve(w, r, "application/javascript; charset=utf-8", append([]byte("globalThis.SCANNER_UI_CONFIG = Object.freeze("), append(body, []byte(");\n")...)...))
		return
	}
	name, ok := assets[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(h.root, name)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Dir(resolved) != h.root {
		http.Error(w, "asset unavailable", http.StatusNotFound)
		return
	}
	file, err := os.Open(resolved)
	if err != nil {
		http.Error(w, "asset unavailable", http.StatusNotFound)
		return
	}
	defer file.Close()
	limited := io.LimitReader(file, maximumAssetSize+1)
	body, err := io.ReadAll(limited)
	if err != nil || len(body) > maximumAssetSize {
		http.Error(w, "asset unavailable", http.StatusServiceUnavailable)
		return
	}
	contentType := "application/octet-stream"
	switch filepath.Ext(name) {
	case ".html":
		contentType = "text/html; charset=utf-8"
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".js":
		contentType = "application/javascript; charset=utf-8"
	}
	h.serve(w, r, contentType, body)
}

func (h *assetHandler) serve(w http.ResponseWriter, r *http.Request, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(body)
	}
}

func setHeaders(header http.Header, apiOrigin string) {
	header.Set("Cache-Control", "no-store")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src "+apiOrigin+"; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
}

func loopbackAddress(address string) bool {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port == "" {
		return false
	}
	parsedPort, err := strconv.ParseUint(port, 10, 16)
	ip := net.ParseIP(host)
	return err == nil && parsedPort <= 65535 && ip != nil && ip.IsLoopback()
}

func loopbackOrigin(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	ip := net.ParseIP(parsed.Hostname())
	if ip == nil || !ip.IsLoopback() || strings.HasSuffix(parsed.Host, ":") {
		return false
	}
	if port := parsed.Port(); port != "" {
		parsedPort, err := strconv.ParseUint(port, 10, 16)
		if err != nil || parsedPort == 0 {
			return false
		}
	}
	return parsed.Scheme+"://"+parsed.Host == value
}
