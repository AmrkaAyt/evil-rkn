package domain

import (
	"net"
	"sort"
	"strings"
)

func IsBlocked(reg *Registry, n NormalizedURL) bool {
	if reg == nil {
		return false
	}

	// 1) IP
	if ip := net.ParseIP(n.Host); ip != nil {
		if _, ok := reg.IPs[ip.String()]; ok {
			return true
		}
	}

	// 2) domains / subdomains
	host := n.Host
	hs := reg.DomainHashes
	for {
		h := HashString64(host)
		i := sort.Search(len(hs), func(i int) bool { return hs[i] >= h })
		if i < len(hs) && hs[i] == h {
			return true
		}

		j := strings.IndexByte(host, '.')
		if j == -1 {
			break
		}
		host = host[j+1:]
	}

	return false
}
