package relay

import (
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"net/netip"
)

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
	if IsForbiddenAddress(ip) {
		return false
	}
	globalHTTP, globalPrivate := cfg.RelayNetworkPolicy()
	cleartext, reserved := scheme != "http" || globalHTTP, !IsReservedAddress(ip) || globalPrivate
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

// Exception is the stored key and display state of a destination permission.
type Exception struct {
	HostPort, CIDR, Origin string
	HTTP, Private          bool
}

// ProfileView contains only root-visible policy configuration and diagnostics.
type ProfileView struct {
	ID                 int32
	Name               string
	Configured, Paused bool
	Diagnostic         Diagnostic
	Exceptions         []Exception
}
