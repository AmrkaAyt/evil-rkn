package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"evil-rkn/internal/domain"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		// Make sure we don’t end up with "//domains/" in the final URL.
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchRegistry implements the Fetcher interface.
// It calls /api/v3/domains/ and builds a registry using only domain names.
func (c *Client) FetchRegistry(ctx context.Context) (*domain.Registry, error) {
	// Hard timeout for the whole operation, just to be safe.
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	u, err := url.Parse(c.baseURL + "/domains/")
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	dec := json.NewDecoder(resp.Body)

	// Expect a JSON array like: ["example.com", "foo.bar", ...].
	t, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("read opening token: %w", err)
	}
	if d, ok := t.(json.Delim); !ok || d != '[' {
		return nil, fmt.Errorf("expected JSON array from /domains/")
	}

	domainHashes := make([]uint64, 0, 1_000_000)

	var (
		skippedEmpty     int
		skippedNormalize int
		skippedBadDomain int
		samples          []string
	)

	for dec.More() {
		var raw string
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("decode domain: %w", err)
		}

		raw = strings.TrimSpace(strings.ToLower(raw))
		if raw == "" {
			// Completely empty entry — just ignore it.
			skippedEmpty++
			continue
		}
		if strings.Contains(raw, "_") {
			// RKN occasionally returns garbage like "bad_domain".
			skippedBadDomain++
			continue
		}

		// Run through our URL normalizer by faking a scheme.
		host, err := domain.NormalizeHost(raw)
		if err != nil {
			skippedNormalize++
			continue
		}

		if len(samples) < 5 {
			samples = append(samples, host)
		}

		domainHashes = append(domainHashes, domain.HashString64(host))

	}

	// Consume the closing ']' token.
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("read closing token: %w", err)
	}

	// Sort and compact hashes to get rid of duplicates.
	sort.Slice(domainHashes, func(i, j int) bool { return domainHashes[i] < domainHashes[j] })
	domainHashes = compactUint64(domainHashes)

	log.Printf("rknapi: skipped %d domains with '_' in name", skippedBadDomain)
	log.Printf("rknapi: skipped %d domains due to normalize errors", skippedNormalize)
	log.Printf("rknapi: skipped %d empty domains", skippedEmpty)
	log.Printf("rknapi: registry built: %d domains, 0 urls, 0 ips", len(domainHashes))
	for i, s := range samples {
		log.Printf("rknapi: sample domain[%d]=%s", i, s)
	}

	reg := &domain.Registry{
		DomainHashes: domainHashes,
		URLHashes:    nil,
		IPs:          make(map[string]struct{}),
	}
	return reg, nil
}

// compactUint64 removes duplicates from a sorted slice.
func compactUint64(src []uint64) []uint64 {
	if len(src) == 0 {
		return src
	}

	dst := src[:1]
	last := src[0]
	for _, v := range src[1:] {
		if v != last {
			dst = append(dst, v)
			last = v
		}
	}
	return dst
}
