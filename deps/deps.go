package deps

import (
	"github.com/glycoview/nightscout-api/auth"
	"github.com/glycoview/nightscout-api/config"
	"github.com/glycoview/nightscout-api/store"
)

type Dependencies struct {
	Config config.Config
	Store  store.Store
	Auth   auth.Manager
}
