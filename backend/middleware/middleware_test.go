package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedactedURIStripsAccessToken(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/events?access_token=secret-value&stream=orders", nil)
	got := redactedURI(r)

	if strings.Contains(got, "secret-value") {
		t.Fatalf("redactedURI leaked the token: %s", got)
	}
	if !strings.Contains(got, "access_token=REDACTED") {
		t.Fatalf("redactedURI did not redact access_token: %s", got)
	}
	if !strings.Contains(got, "stream=orders") {
		t.Fatalf("redactedURI dropped an unrelated query param: %s", got)
	}
}

func TestRedactedURILeavesPlainRequestsAlone(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/streams", nil)
	if got := redactedURI(r); got != "/api/streams" {
		t.Fatalf("redactedURI = %q, want /api/streams", got)
	}
}

func TestRequestIDGeneratedWhenMissing(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFrom(r.Context())
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/streams", nil))

	if seen == "" {
		t.Fatal("no request ID in context")
	}
	if got := w.Header().Get(RequestIDHeader); got != seen {
		t.Fatalf("response header %q != context ID %q", got, seen)
	}
}

func TestRequestIDReusesValidIncomingID(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set(RequestIDHeader, "proxy-abc.123")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if got := w.Header().Get(RequestIDHeader); got != "proxy-abc.123" {
		t.Fatalf("got %q, want the caller's ID", got)
	}
}

func TestRequestIDReplacesUnsafeIncomingID(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	for _, bad := range []string{"has space", "new\nline", strings.Repeat("a", 129)} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set(RequestIDHeader, bad)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if got := w.Header().Get(RequestIDHeader); got == bad || got == "" {
			t.Fatalf("unsafe ID %q was not replaced (got %q)", bad, got)
		}
	}
}

func TestRecoverReturnsJSONWithRequestID(t *testing.T) {
	h := RequestID(Logger(Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/streams", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %s", w.Body)
	}
	if body["request_id"] == "" || body["request_id"] != w.Header().Get(RequestIDHeader) {
		t.Fatalf("request_id missing or mismatched: %v", body)
	}
}

func TestLoggerKeepsFlusherForSSE(t *testing.T) {
	var flushable bool
	h := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, flushable = w.(http.Flusher)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/events", nil))
	if !flushable {
		t.Fatal("Logger hides http.Flusher, SSE would stop streaming")
	}
}
