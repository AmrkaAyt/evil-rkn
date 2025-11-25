package domain

import (
	"fmt"
	"net"
	"path"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

// Normalize takes a raw URL as a user would type it in the browser
// and converts it to a deterministic, canonical form.
func Normalize(raw string) (NormalizedURL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return NormalizedURL{}, fmt.Errorf("empty url")
	}

	// Find and validate scheme.
	i := strings.Index(raw, "://")
	if i <= 0 {
		return NormalizedURL{}, fmt.Errorf("url must contain scheme")
	}

	schemePart := raw[:i]
	rest := raw[i+3:]
	if rest == "" {
		return NormalizedURL{}, fmt.Errorf("empty host")
	}

	var scheme string
	switch {
	case strings.EqualFold(schemePart, "http"):
		scheme = "http"
	case strings.EqualFold(schemePart, "https"):
		scheme = "https"
	default:
		return NormalizedURL{}, fmt.Errorf("unsupported scheme: %s", schemePart)
	}

	// Split host[:port] and path.
	hostport := rest
	p := ""
	if slash := strings.IndexByte(rest, '/'); slash != -1 {
		hostport = rest[:slash]
		p = rest[slash:]
	}

	host, err := normalizeHost(hostport)
	if err != nil {
		return NormalizedURL{}, err
	}

	p = cleanPathFast(p)

	return NormalizedURL{
		Scheme: scheme,
		Host:   host,
		Path:   p,
	}, nil
}

// NormalizeHost normalizes a raw host/domain string (no scheme, no path).
// Used in places where only the hostname matters (e.g. registry loading).
func NormalizeHost(raw string) (string, error) {
	return normalizeHost(raw)
}

func normalizeHost(hostport string) (string, error) {
	hostport = strings.TrimSpace(hostport)
	if hostport == "" {
		return "", fmt.Errorf("empty host")
	}

	// Strip userinfo if present: user:pass@host
	if at := strings.LastIndexByte(hostport, '@'); at != -1 {
		hostport = hostport[at+1:]
	}

	host := hostport

	// Best-effort host:port split. Works for both IPv4 and IPv6 with brackets.
	if strings.Contains(hostport, ":") {
		if h, _, err := net.SplitHostPort(hostport); err == nil {
			host = h
		}
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("empty host")
	}

	// Drop trailing dot: "example.com." → "example.com".
	host = strings.TrimSuffix(host, ".")

	// IPv6 literals come wrapped in brackets: "[2001:db8::1]".
	if len(host) > 2 && host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}

	if host == "" {
		return "", fmt.Errorf("empty host")
	}

	// If it's an IP, let the stdlib normalize it.
	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), nil
	}

	// ASCII-only host: lowercase in place and skip IDNA.
	if isASCII(host) {
		b := []byte(host)
		for i := 0; i < len(b); i++ {
			c := b[i]
			if c >= 'A' && c <= 'Z' {
				b[i] = c + 32
			}
		}
		return string(b), nil
	}

	// Non-ASCII: delegate to IDNA and then lowercase.
	asciiHost, err := idna.Lookup.ToASCII(host)
	if err != nil {
		return "", fmt.Errorf("idna: %w", err)
	}
	return strings.ToLower(asciiHost), nil
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func cleanPathFast(p string) string {
	if p == "" {
		return "/"
	}

	// Cheap path handling for common cases:
	// if there are no "." segments and no "//", we can skip path.Clean.
	if !strings.Contains(p, ".") && !strings.Contains(p, "//") {
		if p[0] != '/' {
			return "/" + p
		}
		return p
	}

	// Fallback to full normalization.
	c := path.Clean(p)
	if c == "" {
		return "/"
	}
	if c[0] != '/' {
		return "/" + c
	}
	return c
}
