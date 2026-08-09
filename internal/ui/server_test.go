package ui

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestC11S1IndependentLoopbackServer(t *testing.T) {
	root := t.TempDir()
	for name, value := range map[string]string{"index.html": "INDEX", "styles.css": "CSS", "app.js": "APP", "model.js": "MODEL", "render.js": "RENDER"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, config := range []Config{
		{Address: "0.0.0.0:4173", AssetRoot: root, APIOrigin: DefaultAPIOrigin},
		{Address: "127.0.0.1:4173", AssetRoot: root, APIOrigin: "http://example.com"},
		{Address: "127.0.0.1:4173", AssetRoot: root, APIOrigin: "http://127.0.0.1:8080/path"},
	} {
		if server, err := Listen(config); err == nil {
			_ = server.Shutdown(context.Background())
			t.Fatalf("unsafe config accepted: %+v", config)
		}
	}
	server, err := Listen(Config{Address: "127.0.0.1:0", AssetRoot: root, APIOrigin: "http://127.0.0.1:8080"})
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + server.Address()
	for _, test := range []struct{ path, want, contentType string }{
		{"/", "INDEX", "text/html"}, {"/styles.css", "CSS", "text/css"}, {"/app.js", "APP", "application/javascript"},
		{"/model.js", "MODEL", "application/javascript"}, {"/render.js", "RENDER", "application/javascript"}, {"/config.js", `"apiOrigin":"http://127.0.0.1:8080"`, "application/javascript"},
	} {
		response, requestErr := http.Get(base + test.path)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK || !strings.Contains(string(body), test.want) || !strings.HasPrefix(response.Header.Get("Content-Type"), test.contentType) ||
			!strings.Contains(response.Header.Get("Content-Security-Policy"), "connect-src http://127.0.0.1:8080") || response.Header.Get("Cache-Control") != "no-store" {
			t.Fatalf("asset %s status=%d headers=%v body=%s", test.path, response.StatusCode, response.Header, body)
		}
	}
	request, _ := http.NewRequest(http.MethodHead, base+"/index.html", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || len(body) != 0 || response.ContentLength != 5 {
		t.Fatalf("HEAD parity status=%d len=%d content=%d", response.StatusCode, len(body), response.ContentLength)
	}
	if response, err = http.Post(base+"/", "text/plain", strings.NewReader("x")); err != nil || response.StatusCode != http.StatusMethodNotAllowed || response.Header.Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST response=%v err=%v", response, err)
	}
	_ = response.Body.Close()
	if response, err = http.Get(base + "/missing"); err != nil || response.StatusCode != http.StatusNotFound {
		t.Fatalf("missing response=%v err=%v", response, err)
	}
	_ = response.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-server.Done(); err != nil {
		t.Fatal(err)
	}
	if _, err := http.Get(base + "/"); err == nil {
		t.Fatal("listener remained reachable after joined shutdown")
	}
}

func TestC11S1RejectsSymlinkEscapeAndOversizeAsset(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	for _, name := range []string{"styles.css", "app.js", "model.js", "render.js"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "index.html")); err != nil {
		t.Fatal(err)
	}
	server, err := Listen(Config{Address: "127.0.0.1:0", AssetRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Shutdown(context.Background())
	response, err := http.Get("http://" + server.Address() + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound || strings.Contains(string(body), "secret") {
		t.Fatalf("symlink escaped status=%d body=%s", response.StatusCode, body)
	}
	if err := os.Remove(filepath.Join(root, "index.html")); err != nil {
		t.Fatal(err)
	}
	oversized := make([]byte, maximumAssetSize+1)
	if err := os.WriteFile(filepath.Join(root, "index.html"), oversized, 0o600); err != nil {
		t.Fatal(err)
	}
	response, err = http.Get("http://" + server.Address() + "/")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("oversize asset status=%d", response.StatusCode)
	}
}
