package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/imadMansour85/Jetyard/backend/models"
	"github.com/imadMansour85/Jetyard/backend/nkeys"
	"github.com/imadMansour85/Jetyard/backend/store"
	nkeyspkg "github.com/nats-io/nkeys"
)

type UserHandler struct {
	store *store.Store
}

func NewUserHandler(s *store.Store) *UserHandler {
	return &UserHandler{
		store: s,
	}
}

// CreateUser creates a new user with NKey and JWT
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "user name is required")
		return
	}

	if req.AccountName == "" {
		writeError(w, http.StatusBadRequest, "account name is required")
		return
	}

	// Get account to verify it exists and to get its keypair for signing
	account, err := h.store.GetAccount(req.AccountName)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("account not found: %v", err))
		return
	}

	if !account.CanSignUsers() {
		writeError(w, http.StatusConflict, fmt.Sprintf(
			"account %q has no stored signing key (it was created before account seeds were persisted). "+
				"Delete its users and re-create the account to issue new users.", account.Name))
		return
	}

	accountKP, err := nkeyspkg.FromSeed([]byte(account.Seed))
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load account key: %v", err))
		return
	}

	// Generate user NKey
	userKey, err := nkeys.GenerateUserNKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate user key: %v", err))
		return
	}

	userJWT, err := nkeys.GenerateUserJWT(userKey.PublicKey, account.PublicKey, accountKP, req.Permissions.ToJWTPermissions())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to sign user JWT: %v", err))
		return
	}

	// Store user (only public key and JWT, never the seed)
	user, err := h.store.CreateUser(req.Name, userKey.PublicKey, req.AccountName, userJWT, req.Permissions)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	// Generate credentials file content
	credsFile := nkeys.GenerateCredentialsFile(userJWT, userKey.Seed)

	// Return user WITH seed and creds file (only once)
	response := models.UserWithSeed{
		User:            *user,
		Seed:            userKey.Seed,
		CredentialsFile: credsFile,
	}

	writeJSON(w, http.StatusCreated, response)
}

// ListUsers returns all users, optionally filtered by account and paginated
// via ?limit=&offset=
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	accountName := r.URL.Query().Get("account")
	users := h.store.ListUsers(accountName)
	writeJSON(w, http.StatusOK, paginate(r, users))
}

// GetUser returns a specific user
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "user name is required")
		return
	}

	user, err := h.store.GetUser(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// DeleteUser deletes a user
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "user name is required")
		return
	}

	if err := h.store.DeleteUser(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

// UpdatePermissions updates user permissions
func (h *UserHandler) UpdatePermissions(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "user name is required")
		return
	}

	var permissions models.Permissions
	if err := json.NewDecoder(r.Body).Decode(&permissions); err != nil {
		writeError(w, http.StatusBadRequest, "invalid permissions")
		return
	}

	user, err := h.store.UpdateUserPermissions(name, &permissions)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// DownloadCredentials returns error since credentials are only available at creation
func (h *UserHandler) DownloadCredentials(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusForbidden, "Credentials were provided at creation time only. If lost, please delete and recreate the user.")
}
