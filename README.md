# nightscout-api

Go library for serving a Nightscout-compatible HTTP API.

This repository provides:

- Nightscout v1 and v3 routers
- request/query parsing
- payload normalization helpers
- shared record and response utilities
- interfaces for auth and persistence

This repository does not provide:

- a production database implementation
- a production auth implementation
- the full Nightscout web app, plugins, or bridge stack

## Status

The library is focused on the API layer. It is useful when you want to expose a Nightscout-like API from your own Go application and plug it into your own storage and auth model.

## Installation

```bash
go get github.com/glycoview/nightscout-api
```

## Quick Start

You provide your own `store.Store` and `auth.Manager`, then mount the routers in your app.

```go
package main

import (
	"net/http"

	v1 "github.com/glycoview/nightscout-api/api/v1"
	v3 "github.com/glycoview/nightscout-api/api/v3"
	"github.com/glycoview/nightscout-api/config"
	"github.com/glycoview/nightscout-api/deps"
)

func main() {
	cfg := config.Config{
		APISecret:    "replace-me",
		AppVersion:   "1.0.0",
		DefaultRoles: []string{"readable"},
	}.WithDefaults()

	dependencies := deps.Dependencies{
		Config: cfg,
		Store:  newMyStore(),       // your implementation
		Auth:   newMyAuthManager(), // your implementation
	}

	mux := http.NewServeMux()

	// Nightscout v1 endpoints under /api
	mux.Handle("/api/", http.StripPrefix("/api", v1.NewNightscoutV1Router(dependencies)))

	// Nightscout v3 endpoints under /api/v3
	mux.Handle("/api/v3/", http.StripPrefix("/api/v3", v3.NewNightscoutV3Router(dependencies)))

	http.ListenAndServe(":8080", mux)
}
```

## Main Packages

- `api/v1`: Nightscout v1 router
- `api/v3`: Nightscout v3 router
- `auth`: auth interfaces and request identity types
- `store`: persistence interfaces and query helpers
- `model`: normalized record and document helpers
- `query`: Nightscout query parsing and pattern expansion
- `httpx`: JSON and HTTP helper functions
- `util`: shared response and route helpers

## What You Need To Implement

### `store.Store`

Your store implementation is responsible for:

- creating, reading, updating, and deleting records
- list/search behavior
- soft delete vs permanent delete semantics
- history lookups
- last-modified tracking

The handlers operate on normalized `model.Record` values and `store.Query`.

### `auth.Manager`

Your auth implementation is responsible for:

- authenticating API secrets, bearer tokens, or JWTs
- mapping callers to an `auth.Identity`
- checking permissions such as `api:entries:read` or `api:treatments:create`
- wrapping route handlers with `Require` and `RequireV1Write`

## GoDoc / Editor Hover

Exported types and functions in the public packages include GoDoc comments so editor hover text is useful when wiring the library into an application.

The most important entry points are:

- `v1.NewNightscoutV1Router`
- `v3.NewNightscoutV3Router`
- `deps.Dependencies`
- `config.Config`
- `store.Store`
- `auth.Manager`

## Behavior Notes

- v1 and v3 query parsing follow Nightscout-style parameter conventions.
- payload normalization handles fields such as `date`, `created_at`, `dateString`, `utcOffset`, and common numeric values
- API v3 supports create/read/search/update/patch/delete/history flows
- collection names are normalized through `model.NormalizeCollection`

## Testing

Run the full test suite with:

```bash
go test ./...
```

Cross-package coverage can be checked with:

```bash
go test ./... -coverpkg=./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Releases

Releases use semantic versioning.

- tag pushes like `v1.2.3` trigger the release workflow
- the tag workflow validates the tag format, runs `go test ./...`, and creates a GitHub release
- `release-please` keeps a release PR updated from Conventional Commits and updates `CHANGELOG.md`

### Conventional Commits

- `fix:` -> patch
- `feat:` -> minor
- `feat!:` -> major
- any commit with `BREAKING CHANGE:` -> major

## Limitations

This is not a drop-in replacement for the full `nightscout/cgm-remote-monitor` repository.

Missing from this repository:

- the Nightscout frontend/web app
- plugin and notification subsystems
- bridge integrations
- production Mongo-backed persistence
- production auth/session/user management

If you need full Nightscout parity, you still need to implement or integrate those missing subsystems in your own application.
