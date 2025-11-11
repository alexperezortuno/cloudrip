package provider

import (
	"net/netip"

	"github.com/alexperezortuno/cloudrip/internal/domain"
)

func IsCloudflareIP(ipStr string, cf domain.CfRanges) bool {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false
	}
	if addr.Is4() {
		for _, p := range cf.IPv4 {
			if p.Contains(addr) {
				return true
			}
		}
		return false
	}
	// IPv6
	for _, p := range cf.IPv6 {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
