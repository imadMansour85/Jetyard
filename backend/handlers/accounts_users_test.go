package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imadMansour85/Jetyard/backend/models"
	"github.com/imadMansour85/Jetyard/backend/nkeys"
	"github.com/imadMansour85/Jetyard/backend/store"
	natsjwt "github.com/nats-io/jwt/v2"
	nkeyspkg "github.com/nats-io/nkeys"
)

func accountsRouter(t *testing.T, st *store.Store) http.Handler {
	t.Helper()
	op, err := nkeyspkg.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	seed, _ := op.Seed()
	t.Setenv("OPERATOR_NKEY_SEED", string(seed))
	om, err := nkeys.NewOperatorManager()
	if err != nil {
		t.Fatal(err)
	}

	ah := NewAccountHandler(st, om)
	uh := NewUserHandler(st)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/accounts", ah.ListAccounts)
	mux.HandleFunc("POST /api/accounts", ah.CreateAccount)
	mux.HandleFunc("GET /api/accounts/{name}", ah.GetAccount)
	mux.HandleFunc("POST /api/users", uh.CreateUser)
	return mux
}

func call(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

// Reproduces the 2026-11-21 bug: after a restart, creating a user under an
// existing account failed with "nkeys: invalid encoded key".
func TestCreateUserAfterStoreReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nats-store.json")
	st, err := store.NewStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if w := call(t, accountsRouter(t, st), "POST", "/api/accounts", `{"name":"orders"}`); w.Code != http.StatusCreated {
		t.Fatalf("create account: %d %s", w.Code, w.Body)
	}

	// Simulate a backend restart.
	reloaded, err := store.NewStoreAt(path)
	if err != nil {
		t.Fatal(err)
	}
	h := accountsRouter(t, reloaded)

	w := call(t, h, "POST", "/api/users", `{"name":"svc","account_name":"orders"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create user after reload: %d %s", w.Code, w.Body)
	}

	var u models.UserWithSeed
	if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil {
		t.Fatal(err)
	}
	claims, err := natsjwt.DecodeUserClaims(u.JWT)
	if err != nil {
		t.Fatalf("user JWT does not decode: %v", err)
	}
	acct, _ := reloaded.GetAccount("orders")
	if claims.Issuer != acct.PublicKey {
		t.Fatalf("user JWT issuer = %s, want account %s", claims.Issuer, acct.PublicKey)
	}
}

func TestAccountSeedNeverReturnedByListOrGet(t *testing.T) {
	st, err := store.NewStoreAt(filepath.Join(t.TempDir(), "nats-store.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := accountsRouter(t, st)
	if w := call(t, h, "POST", "/api/accounts", `{"name":"orders"}`); w.Code != http.StatusCreated {
		t.Fatalf("create account: %d", w.Code)
	}
	acct, _ := st.GetAccount("orders")

	for _, path := range []string{"/api/accounts", "/api/accounts/orders"} {
		body := call(t, h, "GET", path, "").Body.String()
		if strings.Contains(body, acct.Seed) || strings.Contains(body, `"seed"`) {
			t.Fatalf("%s leaks the account seed: %s", path, body)
		}
		if strings.Contains(body, "signing_key_missing") {
			t.Fatalf("%s flags a healthy account as missing its key: %s", path, body)
		}
	}
}

func TestCreateUserUnderLegacyAccountGivesClearError(t *testing.T) {
	st, err := store.NewStoreAt(filepath.Join(t.TempDir(), "nats-store.json"))
	if err != nil {
		t.Fatal(err)
	}
	// An account loaded from a pre-fix store: no seed.
	if _, err := st.CreateAccount("old", "APUB", "jwt", ""); err != nil {
		t.Fatal(err)
	}
	h := accountsRouter(t, st)

	w := call(t, h, "POST", "/api/users", `{"name":"svc","account_name":"old"}`)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "re-create the account") {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if body := call(t, h, "GET", "/api/accounts", "").Body.String(); !strings.Contains(body, `"signing_key_missing":true`) {
		t.Fatalf("list does not flag the legacy account: %s", body)
	}
}
