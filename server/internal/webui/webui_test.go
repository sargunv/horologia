package webui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerRedirectsToDevServerWhenEmbeddedSPAIsMissing(t *testing.T) {
	t.Setenv("WEB_PORT", "5173")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost:8080/login?redirect=%2Foauth%2Fauthorize", nil)
	rr := httptest.NewRecorder()

	Handler().ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusTemporaryRedirect; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got, want := rr.Header().Get("Location"), "http://localhost:5173/login?redirect=%2Foauth%2Fauthorize"; got != want {
		t.Fatalf("location = %q, want %q", got, want)
	}
}
