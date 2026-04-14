package handlers

import (
	"net/http"

	"github.com/glycoview/nightscout-api/auth"
	"github.com/glycoview/nightscout-api/deps"
	"github.com/glycoview/nightscout-api/httpx"
	"github.com/glycoview/nightscout-api/store"
	"github.com/glycoview/nightscout-api/util"
)

func ProfileList(dep deps.Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		records, err := dep.Store.Search(r.Context(), "profile", store.Query{Limit: 100, SortField: "created_at", SortDesc: true})
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		util.WriteRecords(w, records, nil)
	}
}
