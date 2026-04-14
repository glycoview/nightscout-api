package deps

import (
	"github.com/glycoview/nightscout-api/auth"
	"github.com/glycoview/nightscout-api/config"
	"github.com/glycoview/nightscout-api/store"
)

// Dependencies bundles the collaborators required to construct the Nightscout
// API routers.
type Dependencies struct {
	Config config.Config
	Store  store.Store
	Auth   auth.Manager
}
