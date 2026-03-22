package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/zalando/go-keyring"

	"github.com/sargunv/tend/server/api/gen"
)

const (
	keyringService = "tend"
	keyringUser    = "token"
)

// Config holds the CLI's persisted configuration (server URL only).
type Config struct {
	ServerURL string `json:"serverUrl"`
}

func configPath() string {
	return filepath.Join(xdg.ConfigHome, "tend", "config.json")
}

// LoadFile reads the config file from the XDG config directory without
// applying environment variable overrides. If the file doesn't exist,
// returns a zero Config (not an error).
func LoadFile() (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(configPath())
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}
	return cfg, nil
}

// Load reads the config file and applies the TEND_URL environment variable override.
func Load() (*Config, error) {
	cfg, err := LoadFile()
	if err != nil {
		return nil, err
	}
	if v := os.Getenv("TEND_URL"); v != "" {
		cfg.ServerURL = v
	}
	return cfg, nil
}

// Save writes the config file atomically to the XDG config directory
// with restricted permissions (0600).
func Save(cfg *Config) error {
	p := configPath()
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpName, p); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// GetToken retrieves the API token from the OS keychain,
// falling back to the TEND_TOKEN environment variable.
func GetToken() (string, error) {
	if v := os.Getenv("TEND_TOKEN"); v != "" {
		return v, nil
	}
	token, err := keyring.Get(keyringService, keyringUser)
	if err == keyring.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read keychain: %w", err)
	}
	return token, nil
}

// SetToken stores the API token in the OS keychain.
func SetToken(token string) error {
	if err := keyring.Set(keyringService, keyringUser, token); err != nil {
		return fmt.Errorf("save to keychain: %w", err)
	}
	return nil
}

// ClearToken removes the API token from the OS keychain.
func ClearToken() error {
	if err := keyring.Delete(keyringService, keyringUser); err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("clear keychain: %w", err)
	}
	return nil
}

// NewClient constructs an ogen API client from the server URL and token.
func NewClient(serverURL, token string) (*gen.Client, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("server URL not configured; set TEND_URL or use --server-url with tend login")
	}
	if token == "" {
		return nil, fmt.Errorf("no token configured; set TEND_TOKEN or run 'tend login'")
	}

	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	if u.Scheme == "http" && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" {
		fmt.Fprintf(os.Stderr, "warning: server URL uses HTTP; token will be sent in cleartext\n")
	}

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return gen.NewClient(serverURL, tokenSource{token: token}, gen.WithClient(httpClient))
}

// tokenSource implements gen.SecuritySource.
type tokenSource struct {
	token string
}

func (s tokenSource) BearerAuth(_ context.Context, _ gen.OperationName) (gen.BearerAuth, error) {
	return gen.BearerAuth{Token: s.token}, nil
}
