package handlers

import (
	"net/http"

	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
)

// VerifyAuth implements the v1 verifyauth endpoint.
func VerifyAuth(dep deps.Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := dep.Auth.AuthenticateExplicit(r)
		if err != nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{
				"status": "ok",
				"message": map[string]any{
					"message": "UNAUTHORIZED",
					"isAdmin": false,
				},
			})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"message": map[string]any{
				"message": "OK",
				"isAdmin": dep.Auth.HasPermission(*identity, "api:*:admin") || dep.Auth.HasPermission(*identity, "api:entries:create"),
			},
		})
	}
}
