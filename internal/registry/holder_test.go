package registry

import (
	"sync"
	"testing"

	"evil-rkn/internal/domain"
)

func TestHolder_GetSet(t *testing.T) {
	h := NewHolder()

	initial := h.Get()
	if initial == nil {
		t.Fatal("expected non-nil Registry from NewHolder")
	}

	hash := domain.HashString64("example.com")

	reg := &domain.Registry{
		DomainHashes: []uint64{hash},
		URLHashes:    nil,
		IPs:          make(map[string]struct{}),
	}

	h.Set(reg)

	got := h.Get()
	if len(got.DomainHashes) != 1 || got.DomainHashes[0] != hash {
		t.Fatalf("expected DomainHashes to contain %d, got %v", hash, got.DomainHashes)
	}
}

func TestHolder_ConcurrentAccess(t *testing.T) {
	h := NewHolder()
	var wg sync.WaitGroup

	// writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			reg := &domain.Registry{
				DomainHashes: []uint64{domain.HashString64("example.com")},
				URLHashes:    nil,
				IPs:          make(map[string]struct{}),
			}
			h.Set(reg)
		}
	}()

	// readers
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				_ = h.Get()
			}
		}()
	}

	wg.Wait()
}
