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
	OIDCLabel        string `koanf:"oidc_label"`

	InitOwnerEmail    string `koanf:"init_owner_email"`
	InitOwnerName     string `koanf:"init_owner_name"`
	InitOwnerPassword string `koanf:"init_owner_password"`
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
		OIDCLabel:     "OIDC",
	}

	if err := k.Unmarshal("", &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.DB == "" {
		return Config{}, errors.New("TEND_DB is required")
	}

	// TEND_INIT_OWNER_* fields must all be set together or none set.
	initOwnerSet := 0
	for _, f := range []string{cfg.InitOwnerEmail, cfg.InitOwnerName, cfg.InitOwnerPassword} {
		if f != "" {
			initOwnerSet++
		}
	}
	if initOwnerSet != 0 && initOwnerSet != 3 {
		return Config{}, errors.New("TEND_INIT_OWNER_EMAIL, TEND_INIT_OWNER_NAME, and TEND_INIT_OWNER_PASSWORD must all be set together")
	}

	return cfg, nil
}
