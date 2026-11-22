package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	natsgo "github.com/nats-io/nats.go"
)

func TestNATSErrorStatus(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		// The client wraps SDK errors with %w, e.g. "failed to delete stream: ...".
		{"stream not found", fmt.Errorf("failed to delete stream: %w", natsgo.ErrStreamNotFound), http.StatusNotFound},
		{"consumer not found", fmt.Errorf("failed to delete consumer: %w", natsgo.ErrConsumerNotFound), http.StatusNotFound},
		{"bucket not found", fmt.Errorf("bucket not found: %w", natsgo.ErrBucketNotFound), http.StatusNotFound},
		{"key not found", fmt.Errorf("key not found: %w", natsgo.ErrKeyNotFound), http.StatusNotFound},
		{"message not found", fmt.Errorf("message not found: %w", natsgo.ErrMsgNotFound), http.StatusNotFound},
		{"raw API 404", &natsgo.APIError{Code: 404, ErrorCode: natsgo.JSErrCodeStreamNotFound, Description: "stream not found"}, http.StatusNotFound},
		{"invalid stream name", fmt.Errorf("failed to create stream: %w", natsgo.ErrInvalidStreamName), http.StatusBadRequest},
		{"invalid kv key", fmt.Errorf("failed to put value: %w", natsgo.ErrInvalidKey), http.StatusBadRequest},
		{"bad stream config", fmt.Errorf("failed to create stream: %w", &natsgo.APIError{Code: 400, Description: "subjects overlap"}), http.StatusBadRequest},
		{"connection error", fmt.Errorf("failed to publish: %w", natsgo.ErrConnectionClosed), http.StatusInternalServerError},
		{"plain error", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := natsErrorStatus(tc.err); got != tc.want {
				t.Fatalf("natsErrorStatus(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func TestWriteErrorIncludesRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("X-Request-Id", "req-42")
	writeError(w, http.StatusNotFound, "stream not found")
	if !strings.Contains(w.Body.String(), `"request_id":"req-42"`) {
		t.Fatalf("body = %s", w.Body)
	}
}
