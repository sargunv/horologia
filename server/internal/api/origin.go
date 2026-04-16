package api

import (
	"net/http"
	"net/url"
	"strings"
)

func sameOriginRequest(r *http.Request, publicURL string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		return false
	}

	requestURL, err := requestOriginURL(r, publicURL)
	if err != nil {
		return false
	}

	return originURL.Scheme == requestURL.Scheme && originURL.Host == requestURL.Host
}

func requestOriginURL(r *http.Request, publicURL string) (*url.URL, error) {
	if publicURL != "" {
		return url.Parse(publicURL)
	}

	scheme := "http"
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		if first, _, _ := strings.Cut(forwarded, ","); first != "" {
			scheme = strings.TrimSpace(first)
		}
	} else if r.TLS != nil {
		scheme = "https"
	}

	return url.Parse(scheme + "://" + r.Host)
}
