package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	envprovider "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/v2"
)

const (
	envServer = "TEND_SERVER"
	envToken  = "TEND_TOKEN"

	configDirName  = "tend"
	configFileName = "config.json"
)

// ValueSource identifies where a config value came from.
type ValueSource string

const (
	ValueSourceUnset    ValueSource = "unset"
	ValueSourceFile     ValueSource = "file"
	ValueSourceEnv      ValueSource = "env"
	ValueSourceKeychain ValueSource = "keychain"
)

// ResolveInput contains raw env/flag inputs before normalization.
type ResolveInput struct {
	JSON bool
}

// Config is the effective CLI configuration.
type Config struct {
	ServerRaw    string
	Server       *url.URL
	ServerSource ValueSource

	Token       string
	TokenSource ValueSource
	OAuth       *OAuthCredentials

	JSON bool
}

// ResolveConfig resolves the effective CLI config from flags and environment.
func ResolveConfig(input ResolveInput) (Config, error) {
	cfg := Config{
		JSON: input.JSON,
	}

	fileValues, err := loadFile()
	if err != nil {
		return Config{}, err
	}

	envValues, err := loadEnv()
	if err != nil {
		return Config{}, err
	}

	serverRaw, serverSource := fileValues.Server, ValueSourceUnset
	if serverRaw != "" {
		serverSource = ValueSourceFile
	}
	if envValues.Server != "" {
		serverRaw = envValues.Server
		serverSource = ValueSourceEnv
	}
	serverRaw = strings.TrimSpace(serverRaw)
	if serverRaw != "" {
		serverURL, err := normalizeServerURL(serverRaw)
		if err != nil {
			return Config{}, err
		}
		cfg.ServerRaw = serverRaw
		cfg.Server = serverURL
		cfg.ServerSource = serverSource
	} else {
		cfg.ServerSource = serverSource
	}

	token, tokenSource := envValues.Token, ValueSourceUnset
	if token != "" {
		tokenSource = ValueSourceEnv
	}
	cfg.Token = strings.TrimSpace(token)
	cfg.TokenSource = tokenSource
	if cfg.Token == "" && cfg.Server != nil {
		creds, err := LoadOAuthCredentials(cfg.ServerString())
		if err != nil {
			return Config{}, err
		}
		if creds != nil {
			cfg.OAuth = creds
			cfg.Token = strings.TrimSpace(creds.AccessToken)
			if cfg.Token != "" {
				cfg.TokenSource = ValueSourceKeychain
			}
		}
	}

	return cfg, nil
}

type envConfig struct {
	Server string `koanf:"server"`
	Token  string `koanf:"token"`
}

type fileConfig struct {
	Server string `json:"server"`
}

// ConfigPath returns the persisted CLI config path.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}
	return filepath.Join(dir, configDirName, configFileName), nil
}

func loadFile() (fileConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return fileConfig{}, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fileConfig{}, nil
	}
	if err != nil {
		return fileConfig{}, fmt.Errorf("read persisted config: %w", err)
	}

	var out fileConfig
	if err := json.Unmarshal(data, &out); err != nil {
		return fileConfig{}, fmt.Errorf("decode persisted config: %w", err)
	}
	out.Server = strings.TrimSpace(out.Server)
	return out, nil
}

// SaveServer persists the default server URL and returns the config path and normalized value.
func SaveServer(raw string) (path string, normalized string, err error) {
	serverURL, err := normalizeServerURL(strings.TrimSpace(raw))
	if err != nil {
		return "", "", err
	}

	path, err = ConfigPath()
	if err != nil {
		return "", "", err
	}

	cfg := fileConfig{
		Server: strings.TrimRight(serverURL.String(), "/"),
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("encode persisted config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", "", fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", "", fmt.Errorf("write persisted config: %w", err)
	}

	return path, cfg.Server, nil
}

// UnsetServer removes the persisted CLI config file and returns its path.
func UnsetServer() (string, error) {
	path, err := ConfigPath()
	if err != nil {
		return "", err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("remove persisted config: %w", err)
	}

	return path, nil
}

func loadEnv() (envConfig, error) {
	k := koanf.New(".")
	err := k.Load(envprovider.Provider(".", envprovider.Opt{
		Prefix: "TEND_",
		TransformFunc: func(key string, value string) (string, any) {
			switch key {
			case envServer:
				return "server", strings.TrimSpace(value)
			case envToken:
				return "token", strings.TrimSpace(value)
			default:
				return "", nil
			}
		},
	}), nil)
	if err != nil {
		return envConfig{}, fmt.Errorf("load environment config: %w", err)
	}

	var out envConfig
	if err := k.Unmarshal("", &out); err != nil {
		return envConfig{}, fmt.Errorf("decode environment config: %w", err)
	}
	return out, nil
}

func normalizeServerURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL %q: %w", raw, err)
	}
	if u.Scheme == "" {
		return nil, fmt.Errorf("invalid server URL %q: missing scheme (expected http:// or https://)", raw)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("invalid server URL %q: missing host", raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid server URL %q: unsupported scheme %q", raw, u.Scheme)
	}

	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")
	u.Path = strings.TrimSuffix(u.Path, "/api")
	u.RawPath = strings.TrimSuffix(u.RawPath, "/api")
	if u.Path == "" {
		u.Path = "/"
	}
	if u.RawPath == "" && u.Path == "/" {
		u.RawPath = ""
	}

	return u, nil
}

// ServerString returns the normalized server URL or an empty string.
func (c Config) ServerString() string {
	if c.Server == nil {
		return ""
	}
	return strings.TrimRight(c.Server.String(), "/")
}

// APIBaseString returns the normalized API base URL or an empty string.
func (c Config) APIBaseString() string {
	if c.Server == nil {
		return ""
	}
	return strings.TrimRight(resolveURL(c.Server, "/api").String(), "/")
}

// HasServer reports whether a server is configured.
func (c Config) HasServer() bool {
	return c.Server != nil
}

// HasToken reports whether a token is configured.
func (c Config) HasToken() bool {
	return c.Token != ""
}

// RedactedToken returns a safe token preview for display.
func (c Config) RedactedToken() string {
	if c.Token == "" {
		return ""
	}
	if len(c.Token) <= 8 {
		return strings.Repeat("*", len(c.Token))
	}
	return c.Token[:4] + "..." + c.Token[len(c.Token)-4:]
}
