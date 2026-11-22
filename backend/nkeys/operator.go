package nkeys

import (
	"fmt"
	"os"
	"sync"

	"github.com/nats-io/nkeys"
)

// OperatorManager manages the operator NKey
type OperatorManager struct {
	kp        nkeys.KeyPair
	publicKey string
	mu        sync.RWMutex
}

var operatorManager *OperatorManager
var once sync.Once

// GetOperatorManager returns the singleton operator manager
func GetOperatorManager() (*OperatorManager, error) {
	var err error
	once.Do(func() {
		operatorManager, err = NewOperatorManager()
	})
	return operatorManager, err
}

// NewOperatorManager creates a new operator manager from environment
func NewOperatorManager() (*OperatorManager, error) {
	seed := os.Getenv("OPERATOR_NKEY_SEED")
	if seed == "" {
		return nil, fmt.Errorf("OPERATOR_NKEY_SEED environment variable not set")
	}

	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		return nil, fmt.Errorf("invalid operator seed: %w", err)
	}

	publicKey, err := kp.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get operator public key: %w", err)
	}

	return &OperatorManager{
		kp:        kp,
		publicKey: publicKey,
	}, nil
}

// GetKeyPair returns the operator key pair
func (om *OperatorManager) GetKeyPair() nkeys.KeyPair {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return om.kp
}

// GetPublicKey returns the operator public key
func (om *OperatorManager) GetPublicKey() string {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return om.publicKey
}

// SignAccountJWT signs an account JWT with the operator key
func (om *OperatorManager) SignAccountJWT(accountPubKey string) (string, error) {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return GenerateAccountJWT(accountPubKey, om.kp)
}
