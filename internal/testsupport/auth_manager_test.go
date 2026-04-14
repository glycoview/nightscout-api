package testsupport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glycoview/nightscout-api/auth"
)

func TestAuthManagerAuthenticationAndPermissions(t *testing.T) {
	m := NewAuthManager("secret", []string{"readable"})
	subject := m.CreateSubject("alice", []string{"readable"})
	jwt, err := m.IssueJWT(subject.AccessToken)
	if err != nil {
		t.Fatalf("IssueJWT failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	identity, err := m.AuthenticateExplicit(req)
	if err != nil || identity.Name != "alice" {
		t.Fatalf("AuthenticateExplicit failed: %#v %v", identity, err)
	}
	if !m.HasPermission(*identity, "api:entries:read") {
		t.Fatalf("expected readable permission")
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("api-secret", "secret")
	identity, err = m.AuthenticateExplicit(req)
	if err != nil || !m.HasPermission(*identity, "api:entries:create") {
		t.Fatalf("api-secret auth failed: %#v %v", identity, err)
	}

	m.UpdateAPISecret("new-secret")
	if m.matchesAPISecret("secret") || !m.matchesAPISecret("new-secret") {
		t.Fatalf("UpdateAPISecret mismatch")
	}
}

func TestAuthManagerRequireHelpers(t *testing.T) {
	m := NewAuthManager("secret", []string{"readable"})
	if err := m.CreateRole("writer", "api:entries:create"); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	subject := m.CreateSubject("writer-user", []string{"writer"})
	token, err := m.IssueJWT(subject.AccessToken)
	if err != nil {
		t.Fatalf("IssueJWT failed: %v", err)
	}

	called := false
	handler := m.Require("api:entries:read", true, func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		called = true
	})
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if !called || rec.Code != http.StatusOK {
		t.Fatalf("Require should allow default readable role: called=%v status=%d", called, rec.Code)
	}

	called = false
	handler = m.RequireV1Write("api:entries:create", func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		called = true
	})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler(rec, req)
	if !called {
		t.Fatalf("RequireV1Write should call next")
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	rec = httptest.NewRecorder()
	m.Require("api:entries:read", false, func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {})(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized for invalid token, got %d", rec.Code)
	}
}

func TestAuthManagerHelpers(t *testing.T) {
	m := NewAuthManager("secret", nil)
	subject := m.CreateSubject("reader", []string{"readable"})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+subject.AccessToken)
	if identity, err := m.AuthenticateRequest(req); err != nil || identity.Name != "reader" {
		t.Fatalf("AuthenticateRequest explicit mismatch: %#v %v", identity, err)
	}
	m = NewAuthManager("secret", []string{"readable"})
	if identity, err := m.AuthenticateRequest(httptest.NewRequest(http.MethodGet, "/", nil)); err != nil || !identity.FromDefault {
		t.Fatalf("AuthenticateRequest default mismatch: %#v %v", identity, err)
	}
	if _, err := m.authenticateJWT("not-jwt"); err != auth.ErrBadJWT {
		t.Fatalf("expected ErrBadJWT, got %v", err)
	}
	if bearerToken(httptest.NewRequest(http.MethodGet, "/", nil)) != "" {
		t.Fatalf("unexpected bearer token")
	}
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer abc")
	if bearerToken(req) != "abc" {
		t.Fatalf("bearerToken mismatch")
	}
	if !permissionMatch("api:*:*", "api:entries:read") || !permissionMatch("api:entries:*", "api:entries:read") {
		t.Fatalf("permissionMatch mismatch")
	}
	rec := httptest.NewRecorder()
	writeUnauthorized(rec, auth.ErrBadToken)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("writeUnauthorized mismatch")
	}
	rec = httptest.NewRecorder()
	writeForbidden(rec, "api:entries:read")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("writeForbidden mismatch")
	}
}
