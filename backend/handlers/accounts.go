package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/imadMansour85/Jetyard/backend/models"
	"github.com/imadMansour85/Jetyard/backend/nkeys"
	"github.com/imadMansour85/Jetyard/backend/store"
)

type AccountHandler struct {
	store    *store.Store
	operator *nkeys.OperatorManager
}

func NewAccountHandler(s *store.Store, op *nkeys.OperatorManager) *AccountHandler {
	return &AccountHandler{
		store:    s,
		operator: op,
	}
}

// CreateAccount creates a new account with NKey and JWT
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "account name is required")
		return
	}

	// Generate account NKey
	accountKey, err := nkeys.GenerateAccountNKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate account key: %v", err))
		return
	}

	// Sign account JWT with operator
	accountJWT, err := h.operator.SignAccountJWT(accountKey.PublicKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to sign account JWT: %v", err))
		return
	}

	// Store account, including the seed so we can sign user JWTs later
	account, err := h.store.CreateAccount(req.Name, accountKey.PublicKey, accountJWT, accountKey.Seed)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	// Return account WITH seed (only once)
	response := models.AccountWithSeed{
		Account: *account,
		Seed:    accountKey.Seed,
	}

	writeJSON(w, http.StatusCreated, response)
}

// ListAccounts returns all accounts, optionally paginated via ?limit=&offset=
func (h *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts := h.store.ListAccounts()
	out := make([]models.AccountResponse, len(accounts))
	for i, a := range accounts {
		out[i] = models.NewAccountResponse(a)
	}
	writeJSON(w, http.StatusOK, paginate(r, out))
}

// GetAccount returns a specific account
func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "account name is required")
		return
	}

	account, err := h.store.GetAccount(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.NewAccountResponse(account))
}

// DeleteAccount deletes an account
func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "account name is required")
		return
	}

	if err := h.store.DeleteAccount(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "account deleted"})
}

// GetAccountJWT returns the account JWT for NATS server configuration
func (h *AccountHandler) GetAccountJWT(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "account name is required")
		return
	}

	account, err := h.store.GetAccount(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(account.JWT))
}
