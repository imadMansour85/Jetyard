package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/imadMansour85/Jetyard/backend/models"
	"github.com/imadMansour85/Jetyard/backend/secrets"
)

const storeFile = "/data/nats-store.json"

// storedAccount is an account as written to disk. models.Account tags Seed
// `json:"-"` so the API never returns it; this wrapper adds it back for the
// store file, otherwise the seed is lost on restart and the account can never
// sign user JWTs again.
type storedAccount struct {
	models.Account
	Seed string `json:"seed,omitempty"` // sealed with the store's Box when ENCRYPTION_KEY is set
}

// storeFileFormat is the on-disk representation of Store's state.
type storeFileFormat struct {
	Accounts []storedAccount `json:"accounts"`
	Users    []*models.User  `json:"users"`
}

// Store manages accounts and users, persisted to disk so they survive
// backend restarts.
type Store struct {
	accounts map[string]*models.Account
	users    map[string]*models.User
	path     string
	box      *secrets.Box
	mu       sync.RWMutex
}

// NewStore loads accounts/users from NATS_STORE_FILE, defaulting to the
// container data volume. Account seeds are encrypted with ENCRYPTION_KEY when
// it is set.
func NewStore() (*Store, error) {
	path := os.Getenv("NATS_STORE_FILE")
	if path == "" {
		path = storeFile
	}
	box, err := secrets.FromEnv()
	if err != nil {
		return nil, err
	}
	return NewStoreWithBox(path, box)
}

// NewStoreAt loads a store from path without encrypting seeds.
func NewStoreAt(path string) (*Store, error) {
	return NewStoreWithBox(path, nil)
}

// NewStoreWithBox loads a store from path, sealing seeds with box (nil means
// plain text in a 0600 file).
func NewStoreWithBox(path string, box *secrets.Box) (*Store, error) {
	s := &Store{
		accounts: make(map[string]*models.Account),
		users:    make(map[string]*models.User),
		path:     path,
		box:      box,
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}

	var file storeFileFormat
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("nats store file is corrupt: %w", err)
	}
	for _, sa := range file.Accounts {
		seed, err := box.Open(sa.Seed)
		if err != nil {
			return nil, fmt.Errorf("account %s: %w", sa.Name, err)
		}
		a := sa.Account
		a.Seed = seed
		s.accounts[a.Name] = &a
	}
	for _, u := range file.Users {
		s.users[u.Name] = u
	}

	return s, nil
}

// save writes the current accounts/users to disk. Callers must hold the lock.
func (s *Store) save() error {
	file := storeFileFormat{
		Accounts: make([]storedAccount, 0, len(s.accounts)),
		Users:    make([]*models.User, 0, len(s.users)),
	}
	for _, a := range s.accounts {
		seed, err := s.box.Seal(a.Seed)
		if err != nil {
			return fmt.Errorf("sealing seed for account %s: %w", a.Name, err)
		}
		file.Accounts = append(file.Accounts, storedAccount{Account: *a, Seed: seed})
	}
	for _, u := range s.users {
		file.Users = append(file.Users, u)
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

// CreateAccount stores a new account
func (s *Store) CreateAccount(name, publicKey, jwt, seed string) (*models.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.accounts[name]; exists {
		return nil, fmt.Errorf("account %s already exists", name)
	}

	account := &models.Account{
		Name:      name,
		PublicKey: publicKey,
		JWT:       jwt,
		Seed:      seed,
		CreatedAt: time.Now(),
	}

	s.accounts[name] = account
	if err := s.save(); err != nil {
		delete(s.accounts, name)
		return nil, err
	}

	return account, nil
}

// GetAccount retrieves an account by name
func (s *Store) GetAccount(name string) (*models.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, exists := s.accounts[name]
	if !exists {
		return nil, fmt.Errorf("account %s not found", name)
	}

	return account, nil
}

// ListAccounts returns all accounts
func (s *Store) ListAccounts() []*models.Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]*models.Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		accounts = append(accounts, account)
	}

	return accounts
}

// DeleteAccount removes an account
func (s *Store) DeleteAccount(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	account, exists := s.accounts[name]
	if !exists {
		return fmt.Errorf("account %s not found", name)
	}

	// Check if account has users
	for _, user := range s.users {
		if user.AccountName == name {
			return fmt.Errorf("cannot delete account %s: has users", name)
		}
	}

	delete(s.accounts, name)
	if err := s.save(); err != nil {
		s.accounts[name] = account
		return err
	}
	return nil
}

// CreateUser stores a new user
func (s *Store) CreateUser(name, publicKey, accountName, jwt string, permissions *models.Permissions) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[name]; exists {
		return nil, fmt.Errorf("user %s already exists", name)
	}

	// Verify account exists
	if _, exists := s.accounts[accountName]; !exists {
		return nil, fmt.Errorf("account %s not found", accountName)
	}

	user := &models.User{
		Name:        name,
		PublicKey:   publicKey,
		AccountName: accountName,
		JWT:         jwt,
		Permissions: permissions,
		CreatedAt:   time.Now(),
	}

	s.users[name] = user
	if err := s.save(); err != nil {
		delete(s.users, name)
		return nil, err
	}

	return user, nil
}

// GetUser retrieves a user by name
func (s *Store) GetUser(name string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[name]
	if !exists {
		return nil, fmt.Errorf("user %s not found", name)
	}

	return user, nil
}

// ListUsers returns all users, optionally filtered by account
func (s *Store) ListUsers(accountName string) []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*models.User, 0)
	for _, user := range s.users {
		if accountName == "" || user.AccountName == accountName {
			users = append(users, user)
		}
	}

	return users
}

// DeleteUser removes a user
func (s *Store) DeleteUser(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[name]
	if !exists {
		return fmt.Errorf("user %s not found", name)
	}

	delete(s.users, name)
	if err := s.save(); err != nil {
		s.users[name] = user
		return err
	}
	return nil
}

// UpdateUserPermissions updates user permissions
func (s *Store) UpdateUserPermissions(name string, permissions *models.Permissions) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[name]
	if !exists {
		return nil, fmt.Errorf("user %s not found", name)
	}

	user.Permissions = permissions
	if err := s.save(); err != nil {
		return nil, err
	}
	return user, nil
}
