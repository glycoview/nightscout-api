package deps

import (
	"testing"

	"github.com/glycoview/nightscout-api/config"
	"github.com/glycoview/nightscout-api/internal/testsupport"
)

func TestDependenciesStruct(t *testing.T) {
	dep := Dependencies{
		Config: config.Config{APISecret: "secret"},
		Store:  testsupport.NewMemoryStore(),
		Auth:   testsupport.NewAuthManager("secret", []string{"readable"}),
	}
	if dep.Config.APISecret != "secret" || dep.Store == nil || dep.Auth == nil {
		t.Fatalf("Dependencies struct mismatch: %#v", dep)
	}
}
