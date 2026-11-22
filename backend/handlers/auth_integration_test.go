package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imadMansour85/Jetyard/backend/auth"
	"github.com/imadMansour85/Jetyard/backend/store"
)

// buildRouter mirrors the public/protected split in main.go.
func buildRouter(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	signer, _, err := auth.NewSigner()
	if err != nil {
		t.Fatal(err)
	}
	as, err := store.NewAuthStoreAt(t.TempDir() + "/auth-users.json")
	if err != nil {
		t.Fatal(err)
	}
	h := NewAuthHandler(as, signer)

	root := http.NewServeMux()
	root.HandleFunc("GET /api/auth/setup", h.SetupState)
	root.HandleFunc("POST /api/auth/register", h.Register)
	root.HandleFunc("POST /api/auth/login", h.Login)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/me", h.Me)
	mux.HandleFunc("POST /api/auth/change-password", h.ChangePassword)
	mux.HandleFunc("GET /api/streams", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("POST /api/streams", auth.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})))
	mux.Handle("POST /api/auth/users", auth.RequireAdmin(http.HandlerFunc(h.CreateUser)))
	root.Handle("/api/", signer.Require(mux))
	return root
}

func do(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestProtectedRouteRequiresToken(t *testing.T) {
	h := buildRouter(t)
	if got := do(t, h, "GET", "/api/streams", "", "").Code; got != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /api/streams = %d, want 401", got)
	}
}

func TestRegisterThenAccess(t *testing.T) {
	h := buildRouter(t)

	w := do(t, h, "POST", "/api/auth/register", "", `{"username":"imad","password":"correcthorsebattery"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("register = %d: %s", w.Code, w.Body)
	}
	var reg struct {
		Token string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &reg)
	if reg.Token == "" {
		t.Fatal("no token issued")
	}

	if got := do(t, h, "GET", "/api/streams", reg.Token, "").Code; got != http.StatusOK {
		t.Fatalf("authed /api/streams = %d, want 200", got)
	}

	// Second registration must be refused.
	if got := do(t, h, "POST", "/api/auth/register", "", `{"username":"x","password":"correcthorsebattery"}`).Code; got != http.StatusForbidden {
		t.Fatalf("second register = %d, want 403", got)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	h := buildRouter(t)
	do(t, h, "POST", "/api/auth/register", "", `{"username":"imad","password":"correcthorsebattery"}`)

	if got := do(t, h, "POST", "/api/auth/login", "", `{"username":"imad","password":"wrongwrongwrong"}`).Code; got != http.StatusUnauthorized {
		t.Fatalf("bad password = %d, want 401", got)
	}
	if got := do(t, h, "POST", "/api/auth/login", "", `{"username":"imad","password":"correcthorsebattery"}`).Code; got != http.StatusOK {
		t.Fatalf("good password = %d, want 200", got)
	}
}

func TestViewerCannotCreateUsers(t *testing.T) {
	h := buildRouter(t)
	w := do(t, h, "POST", "/api/auth/register", "", `{"username":"admin1","password":"correcthorsebattery"}`)
	var reg struct {
		Token string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &reg)

	// admin creates a viewer
	if got := do(t, h, "POST", "/api/auth/users", reg.Token, `{"username":"viewer1","password":"correcthorsebattery","role":"viewer"}`).Code; got != http.StatusCreated {
		t.Fatalf("admin create user = %d", got)
	}

	lw := do(t, h, "POST", "/api/auth/login", "", `{"username":"viewer1","password":"correcthorsebattery"}`)
	var vl struct {
		Token string `json:"token"`
	}
	json.Unmarshal(lw.Body.Bytes(), &vl)

	if got := do(t, h, "POST", "/api/auth/users", vl.Token, `{"username":"z","password":"correcthorsebattery"}`).Code; got != http.StatusForbidden {
		t.Fatalf("viewer create user = %d, want 403", got)
	}
}

func TestViewerCannotCreateStream(t *testing.T) {
	h := buildRouter(t)
	w := do(t, h, "POST", "/api/auth/register", "", `{"username":"admin2","password":"correcthorsebattery"}`)
	var reg struct {
		Token string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &reg)

	if got := do(t, h, "POST", "/api/auth/users", reg.Token, `{"username":"viewer2","password":"correcthorsebattery","role":"viewer"}`).Code; got != http.StatusCreated {
		t.Fatalf("admin create user = %d", got)
	}

	lw := do(t, h, "POST", "/api/auth/login", "", `{"username":"viewer2","password":"correcthorsebattery"}`)
	var vl struct {
		Token string `json:"token"`
	}
	json.Unmarshal(lw.Body.Bytes(), &vl)

	if got := do(t, h, "POST", "/api/streams", vl.Token, `{}`).Code; got != http.StatusForbidden {
		t.Fatalf("viewer create stream = %d, want 403", got)
	}
	if got := do(t, h, "POST", "/api/streams", reg.Token, `{}`).Code; got != http.StatusCreated {
		t.Fatalf("admin create stream = %d, want 201", got)
	}
}

func TestChangePassword(t *testing.T) {
	h := buildRouter(t)
	w := do(t, h, "POST", "/api/auth/register", "", `{"username":"pwuser","password":"correcthorsebattery"}`)
	var reg struct {
		Token string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &reg)

	// Wrong current password is rejected.
	if got := do(t, h, "POST", "/api/auth/change-password", reg.Token,
		`{"current_password":"wrongwrongwrong","new_password":"newcorrecthorsebattery"}`).Code; got != http.StatusUnauthorized {
		t.Fatalf("wrong current password = %d, want 401", got)
	}

	// Correct current password succeeds.
	if got := do(t, h, "POST", "/api/auth/change-password", reg.Token,
		`{"current_password":"correcthorsebattery","new_password":"newcorrecthorsebattery"}`).Code; got != http.StatusOK {
		t.Fatalf("change password = %d, want 200", got)
	}

	// Old password no longer works.
	if got := do(t, h, "POST", "/api/auth/login", "", `{"username":"pwuser","password":"correcthorsebattery"}`).Code; got != http.StatusUnauthorized {
		t.Fatalf("login with old password = %d, want 401", got)
	}

	// New password works.
	if got := do(t, h, "POST", "/api/auth/login", "", `{"username":"pwuser","password":"newcorrecthorsebattery"}`).Code; got != http.StatusOK {
		t.Fatalf("login with new password = %d, want 200", got)
	}
}

func TestShortPasswordRejected(t *testing.T) {
	h := buildRouter(t)
	if got := do(t, h, "POST", "/api/auth/register", "", `{"username":"imad","password":"short"}`).Code; got != http.StatusBadRequest {
		t.Fatalf("short password = %d, want 400", got)
	}
}
