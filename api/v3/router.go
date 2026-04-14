package v3

import (
	"net/http"

	"github.com/glycoview/nightscout-api/deps"
	"github.com/go-chi/chi/v5"
)

func NewNightscoutV3Router(dep deps.Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Get("/version", Version(dep))
	r.Get("/status", dep.Auth.Require("api:entries:read", false, Status(dep)))
	r.Get("/test", dep.Auth.Require("api:entries:read", false, Test(dep)))
	r.Get("/lastModified", dep.Auth.Require("api:entries:read", false, LastModified(dep)))

	r.Get("/{collection}/history/{since}", History(dep))
	r.Get("/{collection}/{identifier}", Read(dep))
	r.Put("/{collection}/{identifier}", Update(dep))
	r.Patch("/{collection}/{identifier}", Patch(dep))
	r.Delete("/{collection}/{identifier}", Remove(dep))
	r.Get("/{collection}", Search(dep))
	r.Post("/{collection}", Create(dep))

	return r
}
