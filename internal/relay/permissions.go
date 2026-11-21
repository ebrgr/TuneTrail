package relay

import "net/netip"

// Permission separates cleartext from access to a specific reserved network.
type Permission struct {
	HostPort      string
	Prefix        netip.Prefix
	HTTP, Private bool
}

// Permissions is an immutable, profile-scoped snapshot for a single claim.
type Permissions []Permission

// Allows requires both permissions when a cleartext request reaches a reserved IP.
func (permissions Permissions) Allows(scheme, hostPort string, ip netip.Addr) bool {
	cleartext, reserved := scheme != "http", !IsReservedAddress(ip)
	for _, p := range permissions {
		if p.HostPort != hostPort {
			continue
		}
		if p.HTTP && (!p.Prefix.IsValid() || p.Prefix.Contains(ip)) {
			cleartext = true
		}
		if p.Private && p.Prefix.Contains(ip) {
			reserved = true
		}
	}
	return cleartext && reserved
}
