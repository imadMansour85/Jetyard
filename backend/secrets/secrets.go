// Package secrets encrypts small values (nkey seeds, passwords, tokens) before
// they are written to disk.
//
// Encryption is optional: when ENCRYPTION_KEY is unset, a nil *Box stores
// values as plain text (the files are still written 0600). When it is set,
// values are sealed with AES-256-GCM and stored as "enc:v1:<base64>".
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

const prefix = "enc:v1:"

// ErrNoKey is returned when an encrypted value is read but no key is configured.
var ErrNoKey = errors.New("value is encrypted but ENCRYPTION_KEY is not set")

// Box seals and opens values. A nil *Box is valid and stores values as-is.
type Box struct {
	aead cipher.AEAD
}

// FromEnv builds a Box from ENCRYPTION_KEY (32 bytes, standard base64).
// It returns (nil, nil) when the variable is unset.
func FromEnv() (*Box, error) {
	raw := strings.TrimSpace(os.Getenv("ENCRYPTION_KEY"))
	if raw == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("ENCRYPTION_KEY is not valid base64: %w", err)
	}
	return New(key)
}

// New builds a Box from a 32-byte key.
func New(key []byte) (*Box, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

// Enabled reports whether values are encrypted.
func (b *Box) Enabled() bool { return b != nil }

// Seal encrypts plain. Empty strings and a nil Box return plain unchanged.
func (b *Box) Seal(plain string) (string, error) {
	if b == nil || plain == "" {
		return plain, nil
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := b.aead.Seal(nonce, nonce, []byte(plain), nil)
	return prefix + base64.StdEncoding.EncodeToString(out), nil
}

// Open decrypts a value written by Seal. Values without the "enc:v1:" prefix
// are returned unchanged, so files written before encryption was enabled keep
// working (they are re-sealed on the next save).
func (b *Box) Open(stored string) (string, error) {
	if !IsSealed(stored) {
		return stored, nil
	}
	if b == nil {
		return "", ErrNoKey
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, prefix))
	if err != nil {
		return "", fmt.Errorf("encrypted value is corrupt: %w", err)
	}
	n := b.aead.NonceSize()
	if len(data) < n {
		return "", errors.New("encrypted value is too short")
	}
	plain, err := b.aead.Open(nil, data[:n], data[n:], nil)
	if err != nil {
		return "", errors.New("cannot decrypt value: wrong ENCRYPTION_KEY or corrupt data")
	}
	return string(plain), nil
}

// IsSealed reports whether a stored value was produced by Seal.
func IsSealed(stored string) bool {
	return strings.HasPrefix(stored, prefix)
}
