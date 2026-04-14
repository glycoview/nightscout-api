package auth

import (
	"errors"
	"net/http"
)

var (
	ErrMissingCredentials = errors.New("missing credentials")
	ErrBadToken           = errors.New("bad token")
	ErrBadJWT             = errors.New("bad jwt")
)

type Role struct {
	Name        string
	Permissions []string
}

type Subject struct {
	Name        string
	Roles       []string
	AccessToken string
}

type Identity struct {
	Name        string
	Roles       []string
	Permissions []string
	AccessToken string
	FromDefault bool
}

type Manager interface {
	CreateRole(name string, permissions ...string) error
	CreateSubject(name string, roles []string) Subject
	UpdateAPISecret(apiSecret string)
	IssueJWT(accessToken string) (string, error)
	AuthenticateRequest(r *http.Request) (*Identity, error)
	Require(permission string, allowDefault bool, next func(http.ResponseWriter, *http.Request, *Identity)) http.HandlerFunc
	RequireV1Write(permission string, next func(http.ResponseWriter, *http.Request, *Identity)) http.HandlerFunc
	HasPermission(identity Identity, permission string) bool
	AuthenticateExplicit(r *http.Request) (*Identity, error)
	authenticateToken(token string) (*Identity, error)
	authenticateJWT(tokenString string) (*Identity, error)
	identityForSubject(subject Subject, fromDefault bool) Identity
	identityForRoles(name string, roles []string, fromDefault bool) Identity
}
