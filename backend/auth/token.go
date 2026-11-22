package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const TokenTTL = 12 * time.Hour

type Claims struct {
	Username string `json:"sub"`
	Role     string `json:"role"`
	IssuedAt int64  `json:"iat"`
	Expires  int64  `json:"exp"`
}

type Signer struct {
	secret []byte
}

// NewSigner reads JWT_SECRET, generating an ephemeral one when unset. An
// ephemeral secret invalidates every issued token on restart, so deployments
// are expected to set it.
func NewSigner() (*Signer, bool, error) {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		if len(s) < 32 {
			return nil, false, fmt.Errorf("JWT_SECRET must be at least 32 characters")
		}
		return &Signer{secret: []byte(s)}, true, nil
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, false, err
	}
	return &Signer{secret: []byte(hex.EncodeToString(buf))}, false, nil
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func (s *Signer) sign(msg string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(msg))
	return b64(mac.Sum(nil))
}

func (s *Signer) Issue(username, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		Username: username,
		Role:     role,
		IssuedAt: now.Unix(),
		Expires:  now.Add(TokenTTL).Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	header := b64([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := header + "." + b64(payload)
	return body + "." + s.sign(body), nil
}

func (s *Signer) Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token")
	}

	body := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(s.sign(body)), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("malformed payload")
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("malformed claims")
	}

	if time.Now().Unix() > claims.Expires {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}
