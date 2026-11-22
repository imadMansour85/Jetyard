package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func exportDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"index.html":                "home",
		"streams.html":              "streams page",
		"kv/bucket.html":            "bucket page",
		"404.html":                  "not found page",
		"_next/static/chunks/a1.js": "js",
	}
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func get(t *testing.T, h http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, target, nil))
	return w
}

func TestHandlerRoutes(t *testing.T) {
	h := Handler(exportDir(t))
	cases := []struct {
		target, body string
		status       int
	}{
		{"/", "home", http.StatusOK},
		{"/streams", "streams page", http.StatusOK},
		{"/kv/bucket?name=cfg", "bucket page", http.StatusOK},
		{"/_next/static/chunks/a1.js", "js", http.StatusOK},
		{"/nope", "not found page", http.StatusNotFound},
		{"/../../etc/passwd", "not found page", http.StatusNotFound},
	}
	for _, c := range cases {
		w := get(t, h, "GET", c.target)
		if w.Code != c.status || !strings.Contains(w.Body.String(), c.body) {
			t.Errorf("GET %s = %d %q, want %d containing %q", c.target, w.Code, w.Body, c.status, c.body)
		}
	}
}

func TestHandlerCaching(t *testing.T) {
	h := Handler(exportDir(t))
	if cc := get(t, h, "GET", "/_next/static/chunks/a1.js").Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("hashed asset Cache-Control = %q", cc)
	}
	if cc := get(t, h, "GET", "/streams").Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("page Cache-Control = %q", cc)
	}
}

func TestHandlerRejectsWrites(t *testing.T) {
	if w := get(t, Handler(exportDir(t)), "POST", "/streams"); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d", w.Code)
	}
}

func TestAvailable(t *testing.T) {
	if !Available(exportDir(t)) {
		t.Fatal("export dir not detected")
	}
	if Available(t.TempDir()) {
		t.Fatal("empty dir detected as an export")
	}
}
