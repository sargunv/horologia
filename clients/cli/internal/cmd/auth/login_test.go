package authcmd

import (
	"strings"
	"testing"

	"github.com/sargunv/horologia/clients/cli/internal/runtime"
)

func TestPrintAuthorizationURL(t *testing.T) {
	var output strings.Builder
	app := &runtime.App{Stdout: &output}

	printAuthorizationURL(app, "http://localhost:8080/oauth/authorize?foo=bar")

	got := output.String()
	if !strings.Contains(got, "Open this URL in your browser to continue:") {
		t.Fatalf("missing login prompt in output: %q", got)
	}
	if !strings.Contains(got, "http://localhost:8080/oauth/authorize?foo=bar") {
		t.Fatalf("missing auth URL in output: %q", got)
	}
}

func TestEnvTruthy(t *testing.T) {
	trueValues := []string{"1", "true", "TRUE", " yes ", "on"}
	for _, value := range trueValues {
		if !envTruthy(value) {
			t.Fatalf("envTruthy(%q) = false, want true", value)
		}
	}

	falseValues := []string{"", "0", "false", "no", "off", "abc"}
	for _, value := range falseValues {
		if envTruthy(value) {
			t.Fatalf("envTruthy(%q) = true, want false", value)
		}
	}
}
