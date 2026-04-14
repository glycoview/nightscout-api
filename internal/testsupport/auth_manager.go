package testsupport

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/glycoview/nightscout-api/auth"
)

type AuthManager struct {
	mu           sync.RWMutex
	apiSecret    string
	defaultRoles []string
	roles        map[string]auth.Role
	subjects     map[string]auth.Subject
	tokenIndex   map[string]string
}

func NewAuthManager(apiSecret string, defaultRoles []string) *AuthManager {
	manager := &AuthManager{
		apiSecret:    apiSecret,
		defaultRoles: append([]string(nil), defaultRoles...),
		roles:        map[string]auth.Role{},
		subjects:     map[string]auth.Subject{},
		tokenIndex:   map[string]string{},
	}
	manager.roles["denied"] = auth.Role{Name: "denied"}
	manager.roles["readable"] = auth.Role{
		Name: "readable",
		Permissions: []string{
			"api:entries:read",
			"api:treatments:read",
			"api:devicestatus:read",
			"api:profile:read",
			"api:settings:read",
			"api:food:read",
		},
	}
	return manager
}

func (m *AuthManager) CreateRole(name string, permissions ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.roles[name] = auth.Role{Name: name, Permissions: append([]string(nil), permissions...)}
	return nil
}

func (m *AuthManager) CreateSubject(name string, roles []string) auth.Subject {
	m.mu.Lock()
	defer m.mu.Unlock()
	subject := auth.Subject{
		Name:        name,
		Roles:       append([]string(nil), roles...),
		AccessToken: "token:" + name,
	}
	m.subjects[name] = subject
	m.tokenIndex[subject.AccessToken] = name
	return subject
}

func (m *AuthManager) UpdateAPISecret(apiSecret string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.apiSecret = apiSecret
}

func (m *AuthManager) IssueJWT(accessToken string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.tokenIndex[accessToken]; !ok {
		return "", auth.ErrBadToken
	}
	return "jwt:" + accessToken, nil
}

func (m *AuthManager) AuthenticateRequest(r *http.Request) (*auth.Identity, error) {
	identity, err := m.AuthenticateExplicit(r)
	if err == nil {
		return identity, nil
	}
	if len(m.defaultRoles) == 0 {
		return nil, err
	}
	fallback := m.identityForRoles("default", m.defaultRoles, true)
	return &fallback, nil
}

func (m *AuthManager) Require(permission string, allowDefault bool, next func(http.ResponseWriter, *http.Request, *auth.Identity)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := m.AuthenticateExplicit(r)
		switch {
		case err == nil:
		case err == auth.ErrMissingCredentials && allowDefault:
			defaultIdentity := m.identityForRoles("default", m.defaultRoles, true)
			if !m.HasPermission(defaultIdentity, permission) {
				writeForbidden(w, permission)
				return
			}
			next(w, r, &defaultIdentity)
			return
		default:
			writeUnauthorized(w, err)
			return
		}
		if !m.HasPermission(*identity, permission) {
			writeForbidden(w, permission)
			return
		}
		next(w, r, identity)
	}
}

func (m *AuthManager) RequireV1Write(permission string, next func(http.ResponseWriter, *http.Request, *auth.Identity)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := m.AuthenticateExplicit(r)
		if err != nil {
			writeUnauthorized(w, err)
			return
		}
		if !m.HasPermission(*identity, permission) {
			writeForbidden(w, permission)
			return
		}
		next(w, r, identity)
	}
}

func (m *AuthManager) HasPermission(identity auth.Identity, permission string) bool {
	for _, granted := range identity.Permissions {
		if permissionMatch(granted, permission) {
			return true
		}
	}
	return false
}

func (m *AuthManager) AuthenticateExplicit(r *http.Request) (*auth.Identity, error) {
	if token := bearerToken(r); token != "" {
		if strings.HasPrefix(token, "jwt:") {
			return m.authenticateJWT(token)
		}
		return m.authenticateToken(token)
	}
	if header := strings.TrimSpace(r.Header.Get("api-secret")); header != "" {
		if m.matchesAPISecret(header) {
			identity := auth.Identity{
				Name:        "api-secret",
				Roles:       []string{"admin"},
				Permissions: []string{"api:*:*"},
			}
			return &identity, nil
		}
		return nil, auth.ErrBadToken
	}
	return nil, auth.ErrMissingCredentials
}

func (m *AuthManager) authenticateToken(token string) (*auth.Identity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	name, ok := m.tokenIndex[token]
	if !ok {
		return nil, auth.ErrBadToken
	}
	subject := m.subjects[name]
	identity := m.identityForSubject(subject, false)
	return &identity, nil
}

func (m *AuthManager) authenticateJWT(token string) (*auth.Identity, error) {
	if !strings.HasPrefix(token, "jwt:") {
		return nil, auth.ErrBadJWT
	}
	return m.authenticateToken(strings.TrimPrefix(token, "jwt:"))
}

func (m *AuthManager) identityForSubject(subject auth.Subject, fromDefault bool) auth.Identity {
	return m.identityForRoles(subject.Name, subject.Roles, fromDefault)
}

func (m *AuthManager) identityForRoles(name string, roles []string, fromDefault bool) auth.Identity {
	m.mu.RLock()
	defer m.mu.RUnlock()
	permissions := make([]string, 0, len(roles)*4)
	for _, roleName := range roles {
		if role, ok := m.roles[roleName]; ok {
			permissions = append(permissions, role.Permissions...)
		}
	}
	subject := m.subjects[name]
	return auth.Identity{
		Name:        name,
		Roles:       append([]string(nil), roles...),
		Permissions: permissions,
		AccessToken: subject.AccessToken,
		FromDefault: fromDefault,
	}
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[len("Bearer "):])
}

func (m *AuthManager) matchesAPISecret(candidate string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if candidate == m.apiSecret {
		return true
	}
	hash := sha1.Sum([]byte(m.apiSecret))
	return strings.EqualFold(candidate, hex.EncodeToString(hash[:]))
}

func permissionMatch(granted, wanted string) bool {
	if granted == "*" || granted == "api:*:*" {
		return true
	}
	gotParts := strings.Split(granted, ":")
	wantParts := strings.Split(wanted, ":")
	if len(gotParts) != len(wantParts) {
		return granted == wanted
	}
	for i := range gotParts {
		if gotParts[i] == "*" || gotParts[i] == "all" {
			continue
		}
		if gotParts[i] != wantParts[i] {
			return false
		}
	}
	return true
}

func writeUnauthorized(w http.ResponseWriter, err error) {
	message := "Unauthorized"
	if err == auth.ErrBadToken || err == auth.ErrBadJWT {
		message = "Bad access token or JWT"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = fmt.Fprintf(w, `{"description":%q,"message":%q,"status":401}`, err.Error(), message)
}

func writeForbidden(w http.ResponseWriter, permission string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = fmt.Fprintf(w, `{"message":%q,"status":403}`, "Missing permission "+permission)
}
