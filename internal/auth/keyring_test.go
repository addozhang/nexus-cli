package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

// useMockKeyring swaps the keyring seams for an in-memory map; fail makes
// every operation error as if the OS keyring were unavailable. The returned
// func restores the real seams.
func useMockKeyring(store map[string]string, fail bool) func() {
	origGet, origSet, origDelete := keyringGet, keyringSet, keyringDelete
	keyringGet = func(service, user string) (string, error) {
		if fail {
			return "", errors.New("keyring unavailable")
		}
		value, ok := store[user]
		if !ok {
			return "", keyring.ErrNotFound
		}
		return value, nil
	}
	keyringSet = func(service, user, password string) error {
		if fail {
			return errors.New("keyring unavailable")
		}
		store[user] = password
		return nil
	}
	keyringDelete = func(service, user string) error {
		delete(store, user)
		return nil
	}
	return func() { keyringGet, keyringSet, keyringDelete = origGet, origSet, origDelete }
}

func Test_Keyring_SecureRoundTrip(t *testing.T) {
	mock := map[string]string{}
	restore := useMockKeyring(mock, false)
	defer restore()

	path := filepath.Join(t.TempDir(), "nx", "credentials")
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.Add("prod", "https://nexus.example.com", "deployer", "secret", true, true); err != nil {
		t.Fatalf("Add secure: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("credentials file missing: %v", err)
	}
	if strings.Contains(string(raw), "secret") {
		t.Fatalf("token leaked to credentials file: %s", raw)
	}
	if !strings.Contains(string(raw), "secure = true") {
		t.Errorf("secure marker not persisted:\n%s", raw)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	inst, ok := reloaded.Get("prod")
	if !ok || inst.Token != "" || !inst.Secure {
		t.Fatalf("stored instance should have empty token and Secure=true, got %+v (ok=%v)", inst, ok)
	}
	if got := reloaded.SecureInstances(); len(got) != 1 || got[0] != "prod" {
		t.Errorf("SecureInstances() = %v, want [prod]", got)
	}

	token, err := reloaded.Token("prod")
	if err != nil || token != "secret" {
		t.Fatalf("Token() = %q, %v; want secret", token, err)
	}
	resolved, err := Resolve(reloaded, "prod")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Token != "secret" {
		t.Errorf("resolved token = %q, want secret", resolved.Token)
	}

	if err := reloaded.Remove("prod"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, ok := mock["prod"]; ok {
		t.Error("keyring entry was not deleted on Remove")
	}
}

func Test_Keyring_AddFailureKeepsFileClean(t *testing.T) {
	restore := useMockKeyring(map[string]string{}, true)
	defer restore()

	path := filepath.Join(t.TempDir(), "nx", "credentials")
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	err = s.Add("prod", "https://nexus.example.com", "deployer", "secret", true, true)
	var nxErr *nxerrors.Error
	if !errors.As(err, &nxErr) || nxErr.Class() != nxerrors.ClassAuth {
		t.Fatalf("want ClassAuth error, got %v", err)
	}
	if !strings.Contains(err.Error(), "without --secure-storage") {
		t.Errorf("error lacks plain-storage fallback hint: %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("credentials file should not exist, got %v", statErr)
	}
}

func Test_Keyring_MissingTokenOnLookupIsActionable(t *testing.T) {
	restore := useMockKeyring(map[string]string{}, false)
	defer restore()

	s := &Store{instances: map[string]Instance{
		"prod": {URL: "https://nexus.example.com", Secure: true},
	}}
	_, err := s.Token("prod")
	var nxErr *nxerrors.Error
	if !errors.As(err, &nxErr) || nxErr.Class() != nxerrors.ClassAuth {
		t.Fatalf("want ClassAuth error, got %v", err)
	}
	if !strings.Contains(err.Error(), "nx auth add prod --secure-storage") {
		t.Errorf("error lacks re-add hint: %v", err)
	}
}

func Test_Keyring_PlainReAddReplacesSecureEntry(t *testing.T) {
	mock := map[string]string{}
	restore := useMockKeyring(mock, false)
	defer restore()

	path := filepath.Join(t.TempDir(), "nx", "credentials")
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.Add("prod", "https://nexus.example.com", "deployer", "secret", true, true); err != nil {
		t.Fatalf("Add secure: %v", err)
	}
	if err := s.Add("prod", "https://nexus.example.com", "deployer", "plain", true, false); err != nil {
		t.Fatalf("Add plain: %v", err)
	}
	if len(mock) != 0 {
		t.Fatalf("keyring entry was not cleaned up: %v", mock)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	token, err := reloaded.Token("prod")
	if err != nil || token != "plain" {
		t.Fatalf("Token() = %q, %v; want plain", token, err)
	}
	if got := reloaded.SecureInstances(); len(got) != 0 {
		t.Errorf("SecureInstances() = %v, want empty", got)
	}
}

func Test_Keyring_SecureInstanceRendersKeyringMarker(t *testing.T) {
	s := &Store{instances: map[string]Instance{
		"prod": {URL: "https://nexus.example.com", Username: "deployer", Default: true, Secure: true},
	}}
	inst, _ := s.Get("prod")
	if got, want := inst.String("prod"), "prod: https://nexus.example.com as deployer (default) (keyring)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
