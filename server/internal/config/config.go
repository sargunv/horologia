package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	DB            string `koanf:"db"`
	Addr          string `koanf:"addr"`
	LogFormat     string `koanf:"log_format"`
	LogLevel      string `koanf:"log_level"`
	SecureCookies bool   `koanf:"secure_cookies"`

	OIDCIssuer       string `koanf:"oidc_issuer"`
	OIDCClientID     string `koanf:"oidc_client_id"`
	OIDCClientSecret string `koanf:"oidc_client_secret"`
	OIDCRedirectURL  string `koanf:"oidc_redirect_url"`
}

func Load() (Config, error) {
	k := koanf.New(".")

	if err := k.Load(env.Provider("TEND_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "TEND_"))
	}), nil); err != nil {
		return Config{}, fmt.Errorf("load env: %w", err)
	}

	cfg := Config{
		Addr:          ":8080",
		LogFormat:     "text",
		LogLevel:      "info",
		SecureCookies: true,
	}

	if err := k.Unmarshal("", &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.DB == "" {
		return Config{}, errors.New("TEND_DB is required")
	}

	return cfg, nil
}
