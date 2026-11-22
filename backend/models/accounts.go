package models

import (
	"time"

	"github.com/nats-io/jwt/v2"
)

// Account represents a NATS account
type Account struct {
	Name      string    `json:"name"`
	PublicKey string    `json:"public_key"`
	JWT       string    `json:"jwt"`
	CreatedAt time.Time `json:"created_at"`

	// Seed is the account's private nkey seed, needed to sign user JWTs
	// issued under this account. It is persisted internally so users can be
	// created later, but must never be serialized back out over the API.
	Seed string `json:"-"`
}

// CanSignUsers reports whether the account's seed is available. Accounts
// created before seeds were persisted lost it on the first restart.
func (a *Account) CanSignUsers() bool {
	return a.Seed != ""
}

// AccountResponse is an account as returned by the list/get endpoints.
type AccountResponse struct {
	*Account
	// SigningKeyMissing is true when the seed was lost, so the account can no
	// longer issue users and has to be re-created.
	SigningKeyMissing bool `json:"signing_key_missing,omitempty"`
}

// NewAccountResponse wraps an account for the API.
func NewAccountResponse(a *Account) AccountResponse {
	return AccountResponse{Account: a, SigningKeyMissing: !a.CanSignUsers()}
}

// AccountWithSeed includes the seed for one-time download
type AccountWithSeed struct {
	Account
	Seed string `json:"seed"` // Only returned once at creation
}

// CreateAccountRequest is the request to create an account
type CreateAccountRequest struct {
	Name string `json:"name"`
}

// User represents a NATS user
type User struct {
	Name        string       `json:"name"`
	PublicKey   string       `json:"public_key"`
	AccountName string       `json:"account_name"`
	JWT         string       `json:"jwt"`
	Permissions *Permissions `json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

// UserWithSeed includes the seed for one-time download
type UserWithSeed struct {
	User
	Seed            string `json:"seed"` // Only returned once at creation
	CredentialsFile string `json:"credentials_file"`
}

// CreateUserRequest is the request to create a user
type CreateUserRequest struct {
	Name        string       `json:"name"`
	AccountName string       `json:"account_name"`
	Permissions *Permissions `json:"permissions,omitempty"`
}

// Permissions defines user permissions
type Permissions struct {
	Pub  *Permission         `json:"pub,omitempty"`
	Sub  *Permission         `json:"sub,omitempty"`
	Resp *ResponsePermission `json:"resp,omitempty"`
}

// Permission defines allow/deny patterns
type Permission struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// ResponsePermission defines response permissions
type ResponsePermission struct {
	MaxMsgs int `json:"max_msgs,omitempty"`
	Expires int `json:"expires,omitempty"` // seconds
}

// ToJWTPermissions converts our permissions to JWT permissions
func (p *Permissions) ToJWTPermissions() *jwt.Permissions {
	if p == nil {
		return nil
	}

	jwtPerms := &jwt.Permissions{}

	if p.Pub != nil {
		jwtPerms.Pub.Allow = p.Pub.Allow
		jwtPerms.Pub.Deny = p.Pub.Deny
	}

	if p.Sub != nil {
		jwtPerms.Sub.Allow = p.Sub.Allow
		jwtPerms.Sub.Deny = p.Sub.Deny
	}

	if p.Resp != nil {
		jwtPerms.Resp = &jwt.ResponsePermission{
			MaxMsgs: p.Resp.MaxMsgs,
			Expires: time.Duration(p.Resp.Expires) * time.Second,
		}
	}

	return jwtPerms
}
