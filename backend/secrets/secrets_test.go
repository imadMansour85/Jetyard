package secrets

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func testBox(t *testing.T, fill byte) *Box {
	t.Helper()
	b, err := New(bytes.Repeat([]byte{fill}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestSealOpenRoundTrip(t *testing.T) {
	b := testBox(t, 1)
	sealed, err := b.Seal("SAABC123")
	if err != nil {
		t.Fatal(err)
	}
	if !IsSealed(sealed) || strings.Contains(sealed, "SAABC123") {
		t.Fatalf("value not sealed: %q", sealed)
	}
	got, err := b.Open(sealed)
	if err != nil || got != "SAABC123" {
		t.Fatalf("Open = %q, %v", got, err)
	}
}

func TestNilBoxIsPassthrough(t *testing.T) {
	var b *Box
	sealed, err := b.Seal("plain")
	if err != nil || sealed != "plain" {
		t.Fatalf("Seal = %q, %v", sealed, err)
	}
	got, err := b.Open("plain")
	if err != nil || got != "plain" {
		t.Fatalf("Open = %q, %v", got, err)
	}
}

func TestOpenPlainWithKey(t *testing.T) {
	got, err := testBox(t, 1).Open("legacy-plain")
	if err != nil || got != "legacy-plain" {
		t.Fatalf("Open = %q, %v", got, err)
	}
}

func TestOpenSealedWithoutKey(t *testing.T) {
	sealed, _ := testBox(t, 1).Seal("x")
	var b *Box
	if _, err := b.Open(sealed); !errors.Is(err, ErrNoKey) {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}

func TestOpenWithWrongKey(t *testing.T) {
	sealed, _ := testBox(t, 1).Seal("x")
	if _, err := testBox(t, 2).Open(sealed); err == nil {
		t.Fatal("expected an error with the wrong key")
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "")
	if b, err := FromEnv(); b != nil || err != nil {
		t.Fatalf("unset: got %v, %v", b, err)
	}

	t.Setenv("ENCRYPTION_KEY", "not base64!")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error for invalid base64")
	}

	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString([]byte("short")))
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error for a short key")
	}

	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))
	if b, err := FromEnv(); b == nil || err != nil {
		t.Fatalf("valid key: got %v, %v", b, err)
	}
}
