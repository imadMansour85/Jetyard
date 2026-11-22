package store

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imadMansour85/Jetyard/backend/secrets"
)

const testSeed = "SAAFAKESEEDFORTESTSONLY"

func TestAccountSeedSurvivesReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nats-store.json")

	s, err := NewStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAccount("orders", "APUB", "jwt", testSeed); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	a, err := reloaded.GetAccount("orders")
	if err != nil {
		t.Fatal(err)
	}
	if a.Seed != testSeed {
		t.Fatalf("seed after reload = %q, want %q", a.Seed, testSeed)
	}
	if !a.CanSignUsers() {
		t.Fatal("reloaded account cannot sign users")
	}
}

func TestAccountSeedEncryptedAtRest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nats-store.json")
	box, err := secrets.New(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewStoreWithBox(path, box)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAccount("orders", "APUB", "jwt", testSeed); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), testSeed) {
		t.Fatal("seed written to disk in plain text while ENCRYPTION_KEY is set")
	}

	reloaded, err := NewStoreWithBox(path, box)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := reloaded.GetAccount("orders")
	if a.Seed != testSeed {
		t.Fatalf("seed after encrypted reload = %q", a.Seed)
	}

	// Without the key the store must refuse to load rather than silently
	// dropping the seed.
	if _, err := NewStoreAt(path); err == nil {
		t.Fatal("loading an encrypted store without a key should fail")
	}
}

func TestLegacyStoreWithoutSeedLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nats-store.json")
	legacy := `{"accounts":[{"name":"old","public_key":"APUB","jwt":"jwt","created_at":"2026-01-01T00:00:00Z"}],"users":[]}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}

	s, err := NewStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.GetAccount("old")
	if err != nil {
		t.Fatal(err)
	}
	if a.CanSignUsers() {
		t.Fatal("legacy account without a seed should report it cannot sign users")
	}
}
