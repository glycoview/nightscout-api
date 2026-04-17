package v1

import (
	"net/http"

	"github.com/glycoview/nightscout-api/api/v1/handlers"
	"github.com/glycoview/nightscout-api/deps"
	"github.com/go-chi/chi/v5"
)

// NewNightscoutV1Router constructs a router that serves the supported
// Nightscout API v1 endpoints.
func NewNightscoutV1Router(dep deps.Dependencies) http.Handler {
	r := chi.NewRouter()

	registerStatusRoutes(r, dep)
	registerSearchRoutes(r, dep)
	registerEntryRoutes(r, dep)
	registerTreatmentRoutes(r, dep)
	registerDeviceStatusRoutes(r, dep)
	registerProfileRoutes(r, dep)
	registerSettingRoutes(r, dep)
	registerFoodRoutes(r, dep)

	return r
}

func registerStatusRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/status.json", handlers.StatusJSON(dep))
	r.Get("/status.txt", handlers.StatusTxt(dep))
	r.Get("/status.html", handlers.StatusHtml(dep))
	r.Get("/status.js", handlers.StatusJs(dep))
	r.Get("/status.svg", handlers.StatusSvg(dep))
	r.Get("/status.png", handlers.StatusPng(dep))
}

func registerSearchRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/verifyauth", handlers.VerifyAuth(dep))
	r.Get("/versions", handlers.Versions())
	r.Get("/echo/{collection}/{spec}.json", dep.Auth.Require("api:entries:read", true, handlers.EchoRoute(dep)))
	r.Get("/slice/{collection}/{field}/{type}/{prefix}.json", dep.Auth.Require("api:entries:read", true, handlers.SliceRoute(dep)))
	r.Get("/times/echo/*", dep.Auth.Require("api:entries:read", true, handlers.TimesEchoRoute(dep)))
	r.Get("/times/*", dep.Auth.Require("api:entries:read", true, handlers.TimesRoute(dep)))
}

func registerEntryRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/entries/current.json", dep.Auth.Require("api:entries:read", true, handlers.EntriesCurrent(dep)))
	r.Get("/entries.json", dep.Auth.Require("api:entries:read", true, handlers.EntriesList(dep)))
	r.Get("/entries/{spec}.json", dep.Auth.Require("api:entries:read", true, handlers.EntriesSpec(dep)))
	r.Post("/entries", dep.Auth.RequireV1Write("api:entries:create", handlers.EntriesCreate(dep, true)))
	r.Post("/entries/", dep.Auth.RequireV1Write("api:entries:create", handlers.EntriesCreate(dep, true)))
	r.Post("/entries.json", dep.Auth.RequireV1Write("api:entries:create", handlers.EntriesCreate(dep, true)))
	r.Post("/entries/preview.json", dep.Auth.RequireV1Write("api:entries:create", handlers.EntriesCreate(dep, false)))
	r.Delete("/entries.json", dep.Auth.RequireV1Write("api:entries:delete", handlers.EntriesDelete(dep)))
	r.Delete("/entries", dep.Auth.RequireV1Write("api:entries:delete", handlers.EntriesDelete(dep)))
	r.Delete("/entries/{spec}", dep.Auth.RequireV1Write("api:entries:delete", handlers.EntriesDelete(dep)))
}

func registerTreatmentRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/treatments", dep.Auth.Require("api:treatments:read", true, handlers.TreatmentsList(dep)))
	r.Get("/treatments.json", dep.Auth.Require("api:treatments:read", true, handlers.TreatmentsList(dep)))
	create := dep.Auth.Require("api:treatments:create", false, handlers.TreatmentsCreate(dep))
	for _, path := range []string{"/treatments", "/treatments/", "/treatments.json"} {
		r.Post(path, create)
		r.Put(path, create)
	}
	r.Delete("/treatments", dep.Auth.Require("api:treatments:delete", false, handlers.TreatmentsDelete(dep)))
	r.Delete("/treatments/", dep.Auth.Require("api:treatments:delete", false, handlers.TreatmentsDelete(dep)))
	r.Delete("/treatments.json", dep.Auth.Require("api:treatments:delete", false, handlers.TreatmentsDelete(dep)))
}

func registerDeviceStatusRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/devicestatus", dep.Auth.Require("api:devicestatus:read", true, handlers.GenericCollectionList(dep, "devicestatus", "created_at")))
	r.Get("/devicestatus.json", dep.Auth.Require("api:devicestatus:read", true, handlers.GenericCollectionList(dep, "devicestatus", "created_at")))
	create := dep.Auth.Require("api:devicestatus:create", false, handlers.GenericCollectionCreate(dep, "devicestatus"))
	for _, path := range []string{"/devicestatus", "/devicestatus/", "/devicestatus.json"} {
		r.Post(path, create)
		r.Put(path, create)
	}
	r.Delete("/devicestatus", dep.Auth.Require("api:devicestatus:delete", false, handlers.GenericCollectionDelete(dep, "devicestatus", "created_at")))
	r.Delete("/devicestatus/", dep.Auth.Require("api:devicestatus:delete", false, handlers.GenericCollectionDelete(dep, "devicestatus", "created_at")))
	r.Delete("/devicestatus.json", dep.Auth.Require("api:devicestatus:delete", false, handlers.GenericCollectionDelete(dep, "devicestatus", "created_at")))
}

func registerProfileRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/profile", dep.Auth.Require("api:profile:read", true, handlers.ProfileList(dep)))
	r.Get("/profile.json", dep.Auth.Require("api:profile:read", true, handlers.ProfileList(dep)))
	r.Get("/profile/current.json", dep.Auth.Require("api:profile:read", true, handlers.ProfileCurrent(dep)))

	create := dep.Auth.Require("api:profile:create", false, handlers.GenericCollectionCreate(dep, "profile"))
	for _, path := range []string{"/profile", "/profile/", "/profile.json"} {
		r.Post(path, create)
		r.Put(path, create)
	}

	del := dep.Auth.Require("api:profile:delete", false, handlers.GenericCollectionDelete(dep, "profile", "created_at"))
	for _, path := range []string{"/profile", "/profile/", "/profile.json"} {
		r.Delete(path, del)
	}
}

func registerSettingRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/settings", dep.Auth.Require("api:settings:read", true, handlers.GenericCollectionList(dep, "settings", "created_at")))
	r.Get("/settings.json", dep.Auth.Require("api:settings:read", true, handlers.GenericCollectionList(dep, "settings", "created_at")))

	create := dep.Auth.Require("api:settings:create", false, handlers.GenericCollectionCreate(dep, "settings"))
	for _, path := range []string{"/settings", "/settings/", "/settings.json"} {
		r.Post(path, create)
		r.Put(path, create)
	}

	del := dep.Auth.Require("api:settings:delete", false, handlers.GenericCollectionDelete(dep, "settings", "created_at"))
	for _, path := range []string{"/settings", "/settings/", "/settings.json"} {
		r.Delete(path, del)
	}
}

func registerFoodRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/food", dep.Auth.Require("api:food:read", true, handlers.GenericCollectionList(dep, "food", "created_at")))
	r.Get("/food.json", dep.Auth.Require("api:food:read", true, handlers.GenericCollectionList(dep, "food", "created_at")))

	create := dep.Auth.Require("api:food:create", false, handlers.GenericCollectionCreate(dep, "food"))
	for _, path := range []string{"/food", "/food/", "/food.json"} {
		r.Post(path, create)
		r.Put(path, create)
	}

	del := dep.Auth.Require("api:food:delete", false, handlers.GenericCollectionDelete(dep, "food", "created_at"))
	for _, path := range []string{"/food", "/food/", "/food.json"} {
		r.Delete(path, del)
	}
}
