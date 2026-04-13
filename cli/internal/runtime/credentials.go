package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	dbus "github.com/godbus/dbus/v5"
	"github.com/zalando/go-keyring"
)

type OAuthCredentials struct {
	ClientID     string    `json:"clientId"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	TokenType    string    `json:"tokenType"`
	Scope        string    `json:"scope"`
	Expiry       time.Time `json:"expiry"`
}

type credentialStore interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type keyringStore struct{}

func (keyringStore) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (keyringStore) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (keyringStore) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

var oauthCredentialStore credentialStore = keyringStore{}

func credentialServiceName() (string, error) {
	path, err := ConfigPath()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(filepath.Dir(path)))
	return "tend-" + hex.EncodeToString(sum[:8]), nil
}

func LoadOAuthCredentials(server string) (*OAuthCredentials, error) {
	service, err := credentialServiceName()
	if err != nil {
		return nil, err
	}
	payload, err := oauthCredentialStore.Get(service, server)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, nil
	}
	if isCredentialStoreUnavailable(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load keychain credentials: %w", err)
	}

	var creds OAuthCredentials
	if err := json.Unmarshal([]byte(payload), &creds); err != nil {
		return nil, fmt.Errorf("decode keychain credentials: %w", err)
	}
	return &creds, nil
}

func SaveOAuthCredentials(server string, creds OAuthCredentials) error {
	service, err := credentialServiceName()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("encode keychain credentials: %w", err)
	}
	if err := oauthCredentialStore.Set(service, server, string(payload)); err != nil {
		return fmt.Errorf("save keychain credentials: %w", err)
	}
	return nil
}

func DeleteOAuthCredentials(server string) error {
	service, err := credentialServiceName()
	if err != nil {
		return err
	}
	err = oauthCredentialStore.Delete(service, server)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete keychain credentials: %w", err)
	}
	return nil
}

func isCredentialStoreUnavailable(err error) bool {
	if err == nil {
		return false
	}

	var dbusErr *dbus.Error
	if errors.As(err, &dbusErr) {
		switch dbusErr.Name {
		case "org.freedesktop.DBus.Error.ServiceUnknown", "org.freedesktop.DBus.Error.NameHasNoOwner":
			return true
		}
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "org.freedesktop.secrets") &&
		(strings.Contains(msg, "service files") || strings.Contains(msg, "name has no owner")) {
		return true
	}

	// dbus-launch binary missing (e.g. minimal CI/sandbox environments without D-Bus)
	if strings.Contains(msg, "dbus-launch") && strings.Contains(msg, "executable file not found") {
		return true
	}

	return false
}
