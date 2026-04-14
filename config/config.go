package config

import "errors"

type Config struct {
	APISecret    string
	JWTSecret    string
	Enable       []string
	DefaultRoles []string
	API3MaxLimit int
	AppVersion   string
}

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

func (c Config) Validate() error {
	if c.APISecret == "" {
		return errors.New("api secret is required")
	}
	return nil
}
