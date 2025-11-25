package registry

import (
	"context"
	"os"
	"testing"
	"time"
)

func Test_RoskomsvobodaReachable(t *testing.T) {
	if os.Getenv("RKN_INTEGRATION") != "1" {
		t.Skip("integration test is disabled, set RKN_INTEGRATION=1 to run")
	}

	baseURL := os.Getenv("RKN_API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://reestr.rublacklist.net/api/v3"
	}

	client := NewClient(baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reg, err := client.FetchRegistry(ctx)
	if err != nil {
		t.Fatalf("failed to fetch registry from %s: %v", baseURL, err)
	}

	if len(reg.DomainHashes) == 0 {
		t.Logf("warning: fetched registry with 0 domains from %s; maybe registry is empty or changed", baseURL)
	}
}
