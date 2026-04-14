package config

import "errors"

// Config collects runtime settings used by the routers and handlers.
type Config struct {
	APISecret    string
	JWTSecret    string
	Enable       []string
	DefaultRoles []string
	API3MaxLimit int
	AppVersion   string
}

// WithDefaults applies library defaults to empty fields and returns the
// resulting configuration.
func (c Config) WithDefaults() Config {
	if c.APISecret == "" {
		c.APISecret = "change-me"
	}
	if c.JWTSecret == "" {
		c.JWTSecret = c.APISecret
	}
	if c.AppVersion == "" {
		c.AppVersion = "0.0.1"
	}

	if len(c.Enable) == 0 {
		c.Enable = []string{"careportal", "api", "rawbg"}
	}
	if len(c.DefaultRoles) == 0 {
		c.DefaultRoles = []string{"readable"}
	}
	if c.API3MaxLimit <= 0 {
		c.API3MaxLimit = 1000
	}
	return c
}

// Validate returns an error when required configuration is missing.
func (c Config) Validate() error {
	if c.APISecret == "" {
		return errors.New("api secret is required")
	}
	return nil
}
