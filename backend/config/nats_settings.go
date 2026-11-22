package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/imadMansour85/Jetyard/backend/secrets"
)

const defaultSettingsFile = "/data/nats-settings.json"

// SettingsPath is where NATS connection settings are persisted:
// NATS_SETTINGS_FILE, defaulting to the container data volume (like
// AUTH_USERS_FILE and NATS_STORE_FILE).
func SettingsPath() string {
	if p := os.Getenv("NATS_SETTINGS_FILE"); p != "" {
		return p
	}
	return defaultSettingsFile
}

// NATSSettings holds all configurable NATS connection parameters.
type NATSSettings struct {
	URL        string `json:"url"`
	AuthType   string `json:"auth_type"` // "none" | "userpass" | "token"
	Username   string `json:"username"`
	Password   string `json:"password"`
	Token      string `json:"token"`
	TLSEnabled bool   `json:"tls_enabled"`
	CACert     string `json:"ca_cert"`     // PEM string
	ClientCert string `json:"client_cert"` // PEM string
	ClientKey  string `json:"client_key"`  // PEM string
}

// secretFields lists the fields sealed with ENCRYPTION_KEY on disk.
func (s *NATSSettings) secretFields() []*string {
	return []*string{&s.Password, &s.Token, &s.ClientKey}
}

// envNATSSettings builds connection settings from the environment:
// NATS_URL plus either NATS_USER/NATS_PASSWORD or NATS_TOKEN. This is what
// `docker run -e ...` configures; settings saved in the UI override it.
func envNATSSettings(cfg *Config) *NATSSettings {
	s := &NATSSettings{URL: cfg.NatsURL, AuthType: "none"}
	if user := os.Getenv("NATS_USER"); user != "" {
		s.AuthType = "userpass"
		s.Username = user
		s.Password = os.Getenv("NATS_PASSWORD")
	} else if token := os.Getenv("NATS_TOKEN"); token != "" {
		s.AuthType = "token"
		s.Token = token
	}
	return s
}

// LoadNATSSettings returns settings from the persisted file when present,
// falling back to the values already populated in cfg. Password, token and
// client key are decrypted when they were saved with ENCRYPTION_KEY.
func LoadNATSSettings(cfg *Config) *NATSSettings {
	defaults := envNATSSettings(cfg)

	path := SettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return defaults // no saved settings yet
	}

	var s NATSSettings
	if err := json.Unmarshal(data, &s); err != nil {
		slog.Warn("saved NATS settings are corrupt, using NATS_URL", "file", path, "error", err)
		return defaults
	}

	box, err := secrets.FromEnv()
	if err == nil {
		for _, f := range s.secretFields() {
			if *f, err = box.Open(*f); err != nil {
				break
			}
		}
	}
	if err != nil {
		slog.Warn("cannot decrypt saved NATS settings, using NATS_URL", "file", path, "error", err)
		return defaults
	}

	return &s
}

// Save persists the settings to the settings file, sealing the password,
// token and client key when ENCRYPTION_KEY is set.
func (s *NATSSettings) Save() error {
	box, err := secrets.FromEnv()
	if err != nil {
		return err
	}
	onDisk := *s
	for _, f := range onDisk.secretFields() {
		if *f, err = box.Seal(*f); err != nil {
			return fmt.Errorf("encrypting NATS settings: %w", err)
		}
	}

	data, err := json.MarshalIndent(onDisk, "", "  ")
	if err != nil {
		return err
	}
	path := SettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// SafeCopy returns a copy of the settings with sensitive fields redacted.
func (s *NATSSettings) SafeCopy() NATSSettings {
	cp := *s
	if cp.Password != "" {
		cp.Password = "••••••••"
	}
	if cp.Token != "" {
		cp.Token = "••••••••"
	}
	if cp.ClientKey != "" {
		cp.ClientKey = "••••••••"
	}
	return cp
}
