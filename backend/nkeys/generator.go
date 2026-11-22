package nkeys

import (
	"fmt"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

// NKeyPair represents a generated NKey pair
type NKeyPair struct {
	PublicKey string
	Seed      string // Private key - only returned once
}

// GenerateAccountNKey generates a new account NKey pair
func GenerateAccountNKey() (*NKeyPair, error) {
	kp, err := nkeys.CreateAccount()
	if err != nil {
		return nil, fmt.Errorf("failed to create account nkey: %w", err)
	}

	publicKey, err := kp.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	seed, err := kp.Seed()
	if err != nil {
		return nil, fmt.Errorf("failed to get seed: %w", err)
	}

	return &NKeyPair{
		PublicKey: publicKey,
		Seed:      string(seed),
	}, nil
}

// GenerateUserNKey generates a new user NKey pair
func GenerateUserNKey() (*NKeyPair, error) {
	kp, err := nkeys.CreateUser()
	if err != nil {
		return nil, fmt.Errorf("failed to create user nkey: %w", err)
	}

	publicKey, err := kp.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	seed, err := kp.Seed()
	if err != nil {
		return nil, fmt.Errorf("failed to get seed: %w", err)
	}

	return &NKeyPair{
		PublicKey: publicKey,
		Seed:      string(seed),
	}, nil
}

// GenerateAccountJWT creates a signed account JWT
func GenerateAccountJWT(accountPubKey string, operatorKP nkeys.KeyPair) (string, error) {
	claims := jwt.NewAccountClaims(accountPubKey)
	claims.Name = accountPubKey // Can be customized with account name
	claims.IssuedAt = time.Now().Unix()
	claims.Expires = 0 // Never expires

	// Sign with operator key
	token, err := claims.Encode(operatorKP)
	if err != nil {
		return "", fmt.Errorf("failed to encode account JWT: %w", err)
	}

	return token, nil
}

// GenerateUserJWT creates a signed user JWT with permissions
func GenerateUserJWT(userPubKey, accountPubKey string, accountKP nkeys.KeyPair, permissions *jwt.Permissions) (string, error) {
	claims := jwt.NewUserClaims(userPubKey)
	claims.Name = userPubKey // Can be customized with user name
	claims.IssuedAt = time.Now().Unix()
	claims.Expires = 0 // Never expires

	if permissions != nil {
		claims.Permissions = *permissions
	}

	// Sign with account key
	token, err := claims.Encode(accountKP)
	if err != nil {
		return "", fmt.Errorf("failed to encode user JWT: %w", err)
	}

	return token, nil
}

// GenerateCredentialsFile creates a .creds file content (JWT + seed)
func GenerateCredentialsFile(jwt, seed string) string {
	return fmt.Sprintf(`-----BEGIN NATS USER JWT-----
%s
------END NATS USER JWT------

************************* IMPORTANT *************************
NKEY Seed printed below can be used to sign and prove identity.
NKEYs are sensitive and should be treated as secrets.

-----BEGIN USER NKEY SEED-----
%s
------END USER NKEY SEED------

*************************************************************
`, jwt, seed)
}
