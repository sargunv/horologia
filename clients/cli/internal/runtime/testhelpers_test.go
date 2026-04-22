package runtime

import (
	"errors"
	"os"
	"testing"

	dbus "github.com/godbus/dbus/v5"
	"github.com/zalando/go-keyring"
)

type memoryCredentialStore struct {
	entries map[string]string
}

type errorCredentialStore struct {
	getErr    error
	setErr    error
	deleteErr error
}

func newMemoryCredentialStore() *memoryCredentialStore {
	return &memoryCredentialStore{entries: make(map[string]string)}
}

func newUnavailableCredentialStore() *errorCredentialStore {
	return &errorCredentialStore{
		getErr: &dbus.Error{
			Name: "org.freedesktop.DBus.Error.ServiceUnknown",
			Body: []interface{}{"The name org.freedesktop.secrets was not provided by any .service files"},
		},
	}
}

func (s *memoryCredentialStore) key(service, user string) string {
	return service + "\x00" + user
}

func (s *memoryCredentialStore) Get(service, user string) (string, error) {
	value, ok := s.entries[s.key(service, user)]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return value, nil
}

func (s *memoryCredentialStore) Set(service, user, password string) error {
	s.entries[s.key(service, user)] = password
	return nil
}

func (s *memoryCredentialStore) Delete(service, user string) error {
	key := s.key(service, user)
	if _, ok := s.entries[key]; !ok {
		return keyring.ErrNotFound
	}
	delete(s.entries, key)
	return nil
}

func (s *errorCredentialStore) Get(service, user string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}
	return "", keyring.ErrNotFound
}

func (s *errorCredentialStore) Set(service, user, password string) error {
	if s.setErr != nil {
		return s.setErr
	}
	return nil
}

func (s *errorCredentialStore) Delete(service, user string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return keyring.ErrNotFound
}

func newErrorCredentialStore(err error) *errorCredentialStore {
	if err == nil {
		err = errors.New("credential store error")
	}
	return &errorCredentialStore{getErr: err, setErr: err, deleteErr: err}
}

func withCredentialStore(t *testing.T, store credentialStore) {
	t.Helper()
	previous := oauthCredentialStore
	oauthCredentialStore = store
	t.Cleanup(func() {
		oauthCredentialStore = previous
	})
}

func setEnvValue(t *testing.T, key string, value *string) {
	t.Helper()

	oldValue, hadValue := os.LookupEnv(key)
	t.Cleanup(func() {
		if hadValue {
			if err := os.Setenv(key, oldValue); err != nil {
				t.Fatalf("restore env %s: %v", key, err)
			}
			return
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset env %s: %v", key, err)
		}
	})

	if value == nil {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset env %s: %v", key, err)
		}
		return
	}
	if err := os.Setenv(key, *value); err != nil {
		t.Fatalf("set env %s: %v", key, err)
	}
}

func setConfigHome(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	setEnvValue(t, "HOME", stringPtr(home))
	setEnvValue(t, "XDG_CONFIG_HOME", stringPtr(home))
}

func stringPtr(v string) *string {
	return &v
}
