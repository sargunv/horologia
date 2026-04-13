package pwdcheck

import (
	"bufio"
	"context"
	"crypto/sha1" //nolint:gosec // SHA-1 is required by the HIBP k-anonymity API
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sargunv/horologia/server/internal/types"
)

const (
	defaultBaseURL = "https://api.pwnedpasswords.com"
	userAgent      = "horologia/1.0"
	// Guard against pathological responses; a normal range response is ~20KB.
	maxResponseBytes = 64 * 1024
)

// HIBPChecker implements Checker using the HIBP Pwned Passwords range API
// with k-anonymity. Only the first 5 characters of the SHA-1 hash are sent.
type HIBPChecker struct {
	client  *http.Client
	baseURL string
}

// NewHIBPChecker creates a checker that queries the HIBP range API.
func NewHIBPChecker(client *http.Client) *HIBPChecker {
	return NewHIBPCheckerWithBaseURL(client, defaultBaseURL)
}

// NewHIBPCheckerWithBaseURL creates a checker with a custom base URL.
// This is intended for testing with a fake HIBP server.
func NewHIBPCheckerWithBaseURL(client *http.Client, baseURL string) *HIBPChecker {
	if client == nil {
		client = &http.Client{}
	}
	return &HIBPChecker{client: client, baseURL: baseURL}
}

// Check queries the HIBP Pwned Passwords API to determine whether the password
// has appeared in known data breaches. Returns a types.ValidationError if the
// password is found, or nil if it is safe or if the API is unreachable (fail-open).
func (c *HIBPChecker) Check(ctx context.Context, password string) error {
	sum := sha1.Sum([]byte(password)) //nolint:gosec // SHA-1 is required by the HIBP API
	hashHex := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix := hashHex[:5]
	suffix := hashHex[5:]

	url := fmt.Sprintf("%s/range/%s", c.baseURL, prefix)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.WarnContext(ctx, "hibp: failed to build request, skipping check", "error", err)
		return nil
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "hibp: request failed, skipping check", "error", err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "hibp: unexpected status, skipping check", "status", resp.StatusCode)
		return nil
	}

	reader := io.LimitReader(resp.Body, maxResponseBytes)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Each line is "SUFFIX:count"
		hashSuffix, _, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(hashSuffix, suffix) {
			return types.ValidationError(
				"this password has appeared in a data breach and cannot be used; " +
					"for more information, see https://haveibeenpwned.com/Passwords",
			)
		}
	}
	if err := scanner.Err(); err != nil {
		slog.WarnContext(ctx, "hibp: error reading response, skipping check", "error", err)
	}

	return nil
}
