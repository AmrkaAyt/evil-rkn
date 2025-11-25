package registry

import (
	"sync/atomic"

	"evil-rkn/internal/domain"
)

type Holder struct {
	value atomic.Pointer[domain.Registry]
}

func NewHolder() *Holder {
	h := &Holder{}
	empty := &domain.Registry{
		DomainHashes: nil,
		URLHashes:    nil,
		IPs:          make(map[string]struct{}),
	}
	h.value.Store(empty)
	return h
}

func (h *Holder) Get() *domain.Registry {
	return h.value.Load()
}

func (h *Holder) Set(reg *domain.Registry) {
	h.value.Store(reg)
}
