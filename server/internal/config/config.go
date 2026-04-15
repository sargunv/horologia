package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	DB             string `koanf:"db"`
	Addr           string `koanf:"addr"`
	PublicURL      string `koanf:"public_url"`
	LogFormat      string `koanf:"log_format"`
	LogLevel       string `koanf:"log_level"`
	SecureCookies  bool   `koanf:"secure_cookies"`
	APIDocsEnabled bool   `koanf:"api_docs_enabled"`

	OIDCIssuer       string `koanf:"oidc_issuer"`
	OIDCClientID     string `koanf:"oidc_client_id"`
	OIDCClientSecret string `koanf:"oidc_client_secret"`
	OIDCLabel        string `koanf:"oidc_label"`
	OIDCAutoRedirect bool   `koanf:"oidc_auto_redirect"`
	OIDCLinkConsent  bool   `koanf:"oidc_link_consent"`

	PasswordAuthEnabled bool `koanf:"password_auth_enabled"`
	HIBPEnabled         bool `koanf:"hibp_enabled"`

	InitOwnerEmail    string `koanf:"init_owner_email"`
	InitOwnerName     string `koanf:"init_owner_name"`
	InitOwnerPassword string `koanf:"init_owner_password"`
}

func Load() (Config, error) {
	k := koanf.New(".")

	if err := k.Load(env.Provider("HOROLOGIA_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "HOROLOGIA_"))
	}), nil); err != nil {
		return Config{}, fmt.Errorf("load env: %w", err)
	}

	cfg := Config{
		Addr:                ":8080",
		LogFormat:           "text",
		LogLevel:            "info",
		SecureCookies:       true,
		APIDocsEnabled:      true,
		OIDCLabel:           "OIDC",
		OIDCLinkConsent:     true,
		PasswordAuthEnabled: true,
		HIBPEnabled:         true,
	}

	if err := k.Unmarshal("", &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.DB == "" {
		return Config{}, errors.New("HOROLOGIA_DB is required")
	}
	if cfg.PublicURL != "" {
		if !strings.HasPrefix(cfg.PublicURL, "http://") && !strings.HasPrefix(cfg.PublicURL, "https://") {
			return Config{}, errors.New("HOROLOGIA_PUBLIC_URL must start with http:// or https://")
		}
		cfg.PublicURL = strings.TrimRight(cfg.PublicURL, "/")
	}

	// OIDC requires a public URL to derive the redirect callback.
	if cfg.OIDCIssuer != "" && cfg.PublicURL == "" {
		return Config{}, errors.New("HOROLOGIA_PUBLIC_URL is required when HOROLOGIA_OIDC_ISSUER is set")
	}

	// Disabling password auth without OIDC would lock out all users.
	if !cfg.PasswordAuthEnabled && cfg.OIDCIssuer == "" {
		return Config{}, errors.New("HOROLOGIA_PASSWORD_AUTH_ENABLED=false requires HOROLOGIA_OIDC_ISSUER to be set")
	}

	// Auto-redirect requires OIDC to be configured and password auth to be disabled.
	if cfg.OIDCAutoRedirect && cfg.OIDCIssuer == "" {
		return Config{}, errors.New("HOROLOGIA_OIDC_AUTO_REDIRECT=true requires HOROLOGIA_OIDC_ISSUER to be set")
	}
	if cfg.OIDCAutoRedirect && cfg.PasswordAuthEnabled {
		return Config{}, errors.New("HOROLOGIA_OIDC_AUTO_REDIRECT=true requires HOROLOGIA_PASSWORD_AUTH_ENABLED=false")
	}

	// HOROLOGIA_INIT_OWNER_* validation depends on whether password auth is enabled.
	if cfg.PasswordAuthEnabled {
		initOwnerSet := 0
		for _, f := range []string{cfg.InitOwnerEmail, cfg.InitOwnerName, cfg.InitOwnerPassword} {
			if f != "" {
				initOwnerSet++
			}
		}
		if initOwnerSet != 0 && initOwnerSet != 3 {
			return Config{}, errors.New("HOROLOGIA_INIT_OWNER_EMAIL, HOROLOGIA_INIT_OWNER_NAME, and HOROLOGIA_INIT_OWNER_PASSWORD must all be set together")
		}
	} else {
		if cfg.InitOwnerPassword != "" {
			return Config{}, errors.New("HOROLOGIA_INIT_OWNER_PASSWORD must not be set when HOROLOGIA_PASSWORD_AUTH_ENABLED=false")
		}
		if (cfg.InitOwnerEmail != "") != (cfg.InitOwnerName != "") {
			return Config{}, errors.New("HOROLOGIA_INIT_OWNER_EMAIL and HOROLOGIA_INIT_OWNER_NAME must both be set together")
		}
	}

	return cfg, nil
}
