package config

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsPathDefault(t *testing.T) {
	t.Setenv("NATS_SETTINGS_FILE", "")
	if got := SettingsPath(); got != "/data/nats-settings.json" {
		t.Fatalf("SettingsPath() = %q", got)
	}
}

func TestSettingsSaveAndLoadFromEnvPath(t *testing.T) {
	// A nested dir that doesn't exist yet: Save must create it, which is what
	// failed outside Docker where /data doesn't exist.
	path := filepath.Join(t.TempDir(), "nested", "nats-settings.json")
	t.Setenv("NATS_SETTINGS_FILE", path)

	s := &NATSSettings{URL: "nats://example:4222", AuthType: "token", Token: "secret"}
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("settings file mode = %o, want 600", perm)
	}

	got := LoadNATSSettings(&Config{NatsURL: "nats://default:4222"})
	if got.URL != "nats://example:4222" || got.Token != "secret" {
		t.Fatalf("loaded %+v", got)
	}
}

func TestSettingsSecretsEncryptedWithKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nats-settings.json")
	t.Setenv("NATS_SETTINGS_FILE", path)
	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32)))

	s := &NATSSettings{URL: "nats://example:4222", AuthType: "userpass", Username: "app", Password: "hunter2-long-password", ClientKey: "-----BEGIN PRIVATE KEY-----x"}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "hunter2") || strings.Contains(string(raw), "PRIVATE KEY") {
		t.Fatalf("secrets written in plain text: %s", raw)
	}
	if !strings.Contains(string(raw), `"username": "app"`) {
		t.Fatalf("non-secret fields should stay readable: %s", raw)
	}
	if s.Password != "hunter2-long-password" {
		t.Fatal("Save must not modify the in-memory settings")
	}

	got := LoadNATSSettings(&Config{NatsURL: "nats://default:4222"})
	if got.Password != "hunter2-long-password" || got.ClientKey != "-----BEGIN PRIVATE KEY-----x" {
		t.Fatalf("round trip failed: %+v", got)
	}

	// Wrong key: fall back to NATS_URL rather than connect with garbage.
	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{4}, 32)))
	if got := LoadNATSSettings(&Config{NatsURL: "nats://default:4222"}); got.URL != "nats://default:4222" {
		t.Fatalf("wrong key should fall back to defaults, got %+v", got)
	}
}

func TestCorruptSettingsFallBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nats-settings.json")
	t.Setenv("NATS_SETTINGS_FILE", path)
	os.WriteFile(path, []byte(`{"url":"nats://half:4222", "auth_type": `), 0600)
	got := LoadNATSSettings(&Config{NatsURL: "nats://default:4222"})
	if got.URL != "nats://default:4222" || got.AuthType != "none" {
		t.Fatalf("got %+v", got)
	}
}

func TestEnvNATSAuth(t *testing.T) {
	t.Setenv("NATS_SETTINGS_FILE", filepath.Join(t.TempDir(), "none.json"))
	cfg := &Config{NatsURL: "nats://host:4222"}

	t.Setenv("NATS_USER", "dash")
	t.Setenv("NATS_PASSWORD", "pw-from-env")
	if s := LoadNATSSettings(cfg); s.AuthType != "userpass" || s.Username != "dash" || s.Password != "pw-from-env" {
		t.Fatalf("user/pass from env: %+v", s)
	}

	t.Setenv("NATS_USER", "")
	t.Setenv("NATS_TOKEN", "tok-from-env")
	if s := LoadNATSSettings(cfg); s.AuthType != "token" || s.Token != "tok-from-env" {
		t.Fatalf("token from env: %+v", s)
	}

	t.Setenv("NATS_TOKEN", "")
	if s := LoadNATSSettings(cfg); s.AuthType != "none" || s.URL != "nats://host:4222" {
		t.Fatalf("no auth: %+v", s)
	}
}
