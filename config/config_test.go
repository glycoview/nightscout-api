package config

import "testing"

func TestConfigWithDefaults(t *testing.T) {
	cfg := Config{}.WithDefaults()
	if cfg.APISecret != "change-me" {
		t.Fatalf("unexpected default api secret: %q", cfg.APISecret)
	}
	if cfg.JWTSecret != cfg.APISecret {
		t.Fatalf("expected jwt secret to default to api secret")
	}
	if cfg.AppVersion == "" || len(cfg.Enable) == 0 || len(cfg.DefaultRoles) == 0 || cfg.API3MaxLimit != 1000 {
		t.Fatalf("defaults not applied correctly: %#v", cfg)
	}
}

func TestConfigValidate(t *testing.T) {
	if err := (Config{}).Validate(); err == nil {
		t.Fatalf("expected missing api secret validation error")
	}
	if err := (Config{APISecret: "secret"}).Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
