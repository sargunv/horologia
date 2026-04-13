package pwdcheck_test

import (
	"crypto/sha1" //nolint:gosec // SHA-1 is required by the HIBP API
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sargunv/horologia/server/internal/pwdcheck"
	"github.com/sargunv/horologia/server/internal/types"
)

func sha1Hex(s string) string {
	sum := sha1.Sum([]byte(s)) //nolint:gosec // SHA-1 is required by the HIBP API
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func newFakeHIBP(t *testing.T, responses map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix := strings.TrimPrefix(r.URL.Path, "/range/")
		body, ok := responses[strings.ToUpper(prefix)]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newHIBPCheckerWithBase(t *testing.T, baseURL string) *pwdcheck.HIBPChecker {
	t.Helper()
	return pwdcheck.NewHIBPCheckerWithBaseURL(&http.Client{}, baseURL)
}

func TestHIBPChecker_PasswordFound(t *testing.T) {
	password := "password123"
	hash := sha1Hex(password)
	prefix := hash[:5]
	suffix := hash[5:]

	srv := newFakeHIBP(t, map[string]string{
		prefix: fmt.Sprintf("0000000000000000000000000000000AAAA:3\r\n%s:42\r\n0000000000000000000000000000000BBBB:1\r\n", suffix),
	})

	checker := newHIBPCheckerWithBase(t, srv.URL)
	err := checker.Check(t.Context(), password)
	if err == nil {
		t.Fatal("expected error for pwned password, got nil")
	}
	if !types.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "data breach") {
		t.Fatalf("error %q does not mention data breach", err.Error())
	}
}

func TestHIBPChecker_PasswordNotFound(t *testing.T) {
	password := "this-is-a-very-unique-test-password-xyz"
	hash := sha1Hex(password)
	prefix := hash[:5]

	// Response contains suffixes that do NOT match.
	srv := newFakeHIBP(t, map[string]string{
		prefix: "0000000000000000000000000000000AAAA:3\r\n0000000000000000000000000000000BBBB:1\r\n",
	})

	checker := newHIBPCheckerWithBase(t, srv.URL)
	err := checker.Check(t.Context(), password)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHIBPChecker_ServerError_FailOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	checker := newHIBPCheckerWithBase(t, srv.URL)
	err := checker.Check(t.Context(), "somepassword")
	if err != nil {
		t.Fatalf("expected nil (fail-open), got: %v", err)
	}
}

func TestHIBPChecker_NetworkError_FailOpen(t *testing.T) {
	// Point at a closed server to trigger a network error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()

	checker := newHIBPCheckerWithBase(t, srv.URL)
	err := checker.Check(t.Context(), "somepassword")
	if err != nil {
		t.Fatalf("expected nil (fail-open), got: %v", err)
	}
}

func TestHIBPChecker_UsesKAnonymity(t *testing.T) {
	password := "test-k-anonymity"
	hash := sha1Hex(password)
	prefix := hash[:5]

	var requestedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		_, _ = fmt.Fprint(w, "0000000000000000000000000000000AAAA:1\r\n")
	}))
	t.Cleanup(srv.Close)

	checker := newHIBPCheckerWithBase(t, srv.URL)
	_ = checker.Check(t.Context(), password)

	wantPath := "/range/" + prefix
	if requestedPath != wantPath {
		t.Fatalf("expected request path %q, got %q", wantPath, requestedPath)
	}
	// Ensure full hash was NOT sent.
	if strings.Contains(requestedPath, hash) {
		t.Fatal("full SHA-1 hash was sent in request path — k-anonymity violated")
	}
}
