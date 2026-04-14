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
	r.Post("/treatments", dep.Auth.Require("api:treatments:create", false, handlers.TreatmentsCreate(dep)))
	r.Post("/treatments/", dep.Auth.Require("api:treatments:create", false, handlers.TreatmentsCreate(dep)))
	r.Delete("/treatments", dep.Auth.Require("api:treatments:delete", false, handlers.TreatmentsDelete(dep)))
	r.Delete("/treatments/", dep.Auth.Require("api:treatments:delete", false, handlers.TreatmentsDelete(dep)))
}

func registerDeviceStatusRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/devicestatus", dep.Auth.Require("api:devicestatus:read", true, handlers.GenericCollectionList(dep, "devicestatus", "created_at")))
	r.Get("/devicestatus.json", dep.Auth.Require("api:devicestatus:read", true, handlers.GenericCollectionList(dep, "devicestatus", "created_at")))
	r.Post("/devicestatus", dep.Auth.Require("api:devicestatus:create", false, handlers.GenericCollectionCreate(dep, "devicestatus")))
	r.Post("/devicestatus/", dep.Auth.Require("api:devicestatus:create", false, handlers.GenericCollectionCreate(dep, "devicestatus")))
	r.Delete("/devicestatus", dep.Auth.Require("api:devicestatus:delete", false, handlers.GenericCollectionDelete(dep, "devicestatus", "created_at")))
	r.Delete("/devicestatus/", dep.Auth.Require("api:devicestatus:delete", false, handlers.GenericCollectionDelete(dep, "devicestatus", "created_at")))
}

func registerProfileRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/profile", dep.Auth.Require("api:profile:read", true, handlers.ProfileList(dep)))
	r.Get("/profile.json", dep.Auth.Require("api:profile:read", true, handlers.ProfileList(dep)))
}

func registerSettingRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/settings", dep.Auth.Require("api:settings:read", true, handlers.GenericCollectionList(dep, "settings", "created_at")))
	r.Get("/settings.json", dep.Auth.Require("api:settings:read", true, handlers.GenericCollectionList(dep, "settings", "created_at")))
}

func registerFoodRoutes(r *chi.Mux, dep deps.Dependencies) {
	r.Get("/food", dep.Auth.Require("api:food:read", true, handlers.GenericCollectionList(dep, "food", "created_at")))
	r.Get("/food.json", dep.Auth.Require("api:food:read", true, handlers.GenericCollectionList(dep, "food", "created_at")))
}
