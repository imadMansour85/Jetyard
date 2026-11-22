package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(&buf, "json", "debug")
	if err != nil {
		t.Fatal(err)
	}
	logger.Debug("hello", "request_id", "abc")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("not JSON: %s", buf.String())
	}
	if line["msg"] != "hello" || line["request_id"] != "abc" || line["level"] != "DEBUG" {
		t.Fatalf("unexpected line: %v", line)
	}
}

func TestNewRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	logger, _ := New(&buf, "text", "warn")
	logger.Info("dropped")
	logger.Warn("kept")
	if strings.Contains(buf.String(), "dropped") || !strings.Contains(buf.String(), "kept") {
		t.Fatalf("level filtering wrong: %s", buf.String())
	}
}

func TestNewRejectsBadConfig(t *testing.T) {
	if _, err := New(&bytes.Buffer{}, "xml", ""); err == nil {
		t.Fatal("expected error for LOG_FORMAT=xml")
	}
	if _, err := New(&bytes.Buffer{}, "", "loud"); err == nil {
		t.Fatal("expected error for LOG_LEVEL=loud")
	}
}
