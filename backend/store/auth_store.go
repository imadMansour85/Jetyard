package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/imadMansour85/Jetyard/backend/models"
	"golang.org/x/crypto/bcrypt"
)

const authUsersFile = "/data/auth-users.json"

type AuthStore struct {
	users map[string]*models.AuthUser
	path  string
	mu    sync.RWMutex
}

// NewAuthStore loads users from AUTH_USERS_FILE, defaulting to the container
// data volume.
func NewAuthStore() (*AuthStore, error) {
	path := os.Getenv("AUTH_USERS_FILE")
	if path == "" {
		path = authUsersFile
	}
	return NewAuthStoreAt(path)
}

func NewAuthStoreAt(path string) (*AuthStore, error) {
	s := &AuthStore{
		users: make(map[string]*models.AuthUser),
		path:  path,
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}

	var users []*models.AuthUser
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("auth user file is corrupt: %w", err)
	}
	for _, u := range users {
		s.users[u.Username] = u
	}

	return s, nil
}

// save writes the current users to disk. Callers must hold the lock.
func (s *AuthStore) save() error {
	users := make([]*models.AuthUser, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *AuthStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

func (s *AuthStore) CreateUser(username, password, role string) (*models.AuthUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, fmt.Errorf("user %s already exists", username)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.AuthUser{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    time.Now(),
	}

	s.users[username] = user
	if err := s.save(); err != nil {
		delete(s.users, username)
		return nil, err
	}

	return user, nil
}

// Authenticate verifies credentials, returning the user on success.
func (s *AuthStore) Authenticate(username, password string) (*models.AuthUser, error) {
	s.mu.RLock()
	user, exists := s.users[username]
	s.mu.RUnlock()

	if !exists {
		// Compare against a dummy hash so a missing user costs the same as a
		// wrong password, keeping response time from leaking which it was.
		bcrypt.CompareHashAndPassword([]byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"), []byte(password))
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

func (s *AuthStore) GetUser(username string) (*models.AuthUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, fmt.Errorf("user %s not found", username)
	}
	return user, nil
}

func (s *AuthStore) ListUsers() []*models.AuthUser {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*models.AuthUser, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return users
}

// ChangePassword verifies currentPassword against the stored hash, then
// replaces it with a hash of newPassword.
func (s *AuthStore) ChangePassword(username, currentPassword, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[username]
	if !exists {
		return fmt.Errorf("user %s not found", username)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	oldHash := user.PasswordHash
	user.PasswordHash = string(newHash)
	if err := s.save(); err != nil {
		user.PasswordHash = oldHash
		return err
	}

	return nil
}

func (s *AuthStore) DeleteUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[username]
	if !exists {
		return fmt.Errorf("user %s not found", username)
	}

	delete(s.users, username)
	if err := s.save(); err != nil {
		s.users[username] = user
		return err
	}
	return nil
}
