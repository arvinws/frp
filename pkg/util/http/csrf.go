package http

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// ProtectWrites rejects cross-site state changes. Browser clients send
// Origin, Referer, or Sec-Fetch-Site on POST/PUT/DELETE. Non-browser API
// clients that send none of those headers are still accepted, so scripts and
// the existing dashboard keep working. A browser request from another site
// cannot omit those headers.
func ProtectWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if err := checkWriteRequest(r); err != nil {
			http.Error(w, "cross-site request blocked", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func checkWriteRequest(r *http.Request) error {
	if site := strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")); site != "" && site != "same-origin" && site != "none" {
		return errCrossSite
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		if origin == "null" || !sameRequestHost(origin, r.Host) {
			return errCrossSite
		}
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" && !sameRequestHost(referer, r.Host) {
		return errCrossSite
	}
	return nil
}

func sameRequestHost(rawURL, reqHost string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return false
	}
	return hostsMatch(parsed.Host, reqHost)
}

func hostsMatch(left, right string) bool {
	leftHost, leftPort, leftErr := net.SplitHostPort(left)
	rightHost, rightPort, rightErr := net.SplitHostPort(right)
	if leftErr != nil {
		leftHost, leftPort = left, ""
	}
	if rightErr != nil {
		rightHost, rightPort = right, ""
	}
	if !strings.EqualFold(leftHost, rightHost) {
		return false
	}
	if leftPort == "" || rightPort == "" {
		return true
	}
	return leftPort == rightPort
}

type crossSiteError struct{}

func (crossSiteError) Error() string { return "cross-site request blocked" }

var errCrossSite = crossSiteError{}
