package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/imadMansour85/Jetyard/backend/auth"
	"github.com/imadMansour85/Jetyard/backend/models"
	"github.com/imadMansour85/Jetyard/backend/store"
)

type AuthHandler struct {
	store  *store.AuthStore
	signer *auth.Signer
}

func NewAuthHandler(s *store.AuthStore, signer *auth.Signer) *AuthHandler {
	return &AuthHandler{store: s, signer: signer}
}

// Register creates the first admin account. Once any user exists the endpoint
// is closed, so a fresh deployment is claimed by its operator rather than
// staying open to whoever reaches it next.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if h.store.Count() > 0 {
		writeError(w, http.StatusForbidden, "registration is closed; ask an admin to create your account")
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateCredentials(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.store.CreateUser(req.Username, req.Password, "admin")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	token, err := h.signer.Issue(user.Username, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	writeJSON(w, http.StatusCreated, models.LoginResponse{Token: token, User: user.Public()})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.store.Authenticate(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.signer.Issue(user.Username, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	writeJSON(w, http.StatusOK, models.LoginResponse{Token: token, User: user.Public()})
}

// Me returns the caller identity, letting the frontend validate a stored token
// on load without a dedicated refresh flow.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.store.GetUser(claims.Username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, user.Public())
}

// ChangePassword lets an authenticated user change their own password,
// verifying the current one first. Available to any role — not gated behind
// RequireAdmin since it only ever acts on the caller's own account.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateCredentials(models.LoginRequest{Username: claims.Username, Password: req.NewPassword}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.ChangePassword(claims.Username, req.CurrentPassword, req.NewPassword); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password changed"})
}

// SetupState tells the login screen whether to offer registration.
func (h *AuthHandler) SetupState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"needs_setup": h.store.Count() == 0})
}

func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users := h.store.ListUsers()
	out := make([]models.AuthUserPublic, 0, len(users))
	for _, u := range users {
		out = append(out, u.Public())
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		models.LoginRequest
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateCredentials(req.LoginRequest); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	role := req.Role
	if role == "" {
		role = "viewer"
	}
	if role != "admin" && role != "viewer" {
		writeError(w, http.StatusBadRequest, "role must be admin or viewer")
		return
	}

	user, err := h.store.CreateUser(req.Username, req.Password, role)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, user.Public())
}

func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")

	claims, _ := auth.ClaimsFrom(r.Context())
	if claims.Username == username {
		writeError(w, http.StatusBadRequest, "cannot delete your own account")
		return
	}

	if err := h.store.DeleteUser(username); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateCredentials(req models.LoginRequest) error {
	if len(req.Username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	if len(req.Password) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}
	if len(req.Password) > 72 {
		// bcrypt silently truncates beyond 72 bytes.
		return fmt.Errorf("password must be at most 72 characters")
	}
	return nil
}
