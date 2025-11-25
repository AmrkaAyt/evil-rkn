package domain

// Registry — in memory representation of blocking list.
// NOTE: domain and URL hashes are 64-bit and collisions are theoretically possible,
// but considered acceptable for this task. If false positives become critical,
// consider storing original strings or using a stronger scheme (e.g. hash + length).
type Registry struct {
	DomainHashes []uint64 // Sorted hash domains
	URLHashes    []uint64
	IPs          map[string]struct{}
}

// NormalizedURL — result of normalize
type NormalizedURL struct {
	Scheme string // "http" or "https"
	Host   string // example.com
	Path   string // normalize path
}
