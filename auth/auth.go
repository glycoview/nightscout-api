package auth

import (
	"errors"
	"net/http"
)

var (
	// ErrMissingCredentials indicates that a request did not include any usable
	// authentication material.
	ErrMissingCredentials = errors.New("missing credentials")
	// ErrBadToken indicates that an access token or API secret was provided but
	// did not match a known credential.
	ErrBadToken = errors.New("bad token")
	// ErrBadJWT indicates that a bearer token looked like a JWT but could not be
	// parsed or verified by the authentication backend.
	ErrBadJWT = errors.New("bad jwt")
)

// Role groups permissions that can later be assigned to subjects.
type Role struct {
	Name        string
	Permissions []string
}

// Subject represents an actor known to the auth backend.
//
// Subjects usually map to users, service accounts, or API clients and are
// resolved into an Identity during request authentication.
type Subject struct {
	Name        string
	Roles       []string
	AccessToken string
}

// Identity is the resolved authorization context for a request.
type Identity struct {
	Name        string
	Roles       []string
	Permissions []string
	AccessToken string
	FromDefault bool
}

// Manager defines the authentication and authorization hooks required by the
// API routers in this module.
//
// The library intentionally depends on this interface instead of shipping a
// production auth implementation, so applications can map it to their own
// user, token, and permission model.
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
}
