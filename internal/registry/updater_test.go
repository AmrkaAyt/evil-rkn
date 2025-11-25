package registry

import (
	"context"
	"testing"

	"evil-rkn/internal/domain"
)

type fakeSource struct {
	reg *domain.Registry
	err error
}

func (f *fakeSource) FetchRegistry(ctx context.Context) (*domain.Registry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.reg, nil
}

func TestUpdateOnce_Success(t *testing.T) {
	holder := NewHolder()

	h := domain.HashString64("example.com")

	src := &fakeSource{
		reg: &domain.Registry{
			DomainHashes: []uint64{h},
			URLHashes:    nil,
			IPs:          make(map[string]struct{}),
		},
	}

	ctx := context.Background()
	if err := updateOnce(ctx, src, holder); err != nil {
		t.Fatalf("updateOnce error: %v", err)
	}

	got := holder.Get()
	if len(got.DomainHashes) != 1 || got.DomainHashes[0] != h {
		t.Fatalf("DomainHashes = %v, want [%d]", got.DomainHashes, h)
	}
}
