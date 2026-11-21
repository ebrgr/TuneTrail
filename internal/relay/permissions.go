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

// NetworkPolicy combines the profile exceptions with an instance configuration snapshot.
type NetworkPolicy struct {
	Permissions             Permissions
	AllowHTTP, AllowPrivate bool
}

// Allows checks exceptions without granting instance-wide permissions.
func (permissions Permissions) Allows(scheme, hostPort string, ip netip.Addr) bool {
	return (NetworkPolicy{Permissions: permissions}).Allows(scheme, hostPort, ip)
}

// Allows requires both permissions when cleartext reaches a reserved address.
func (policy NetworkPolicy) Allows(scheme, hostPort string, ip netip.Addr) bool {
	ip = ip.Unmap().WithZone("")
	if IsForbiddenAddress(ip) {
		return false
	}
	globalPrivate := policy.AllowPrivate && !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
	cleartext, reserved := scheme != "http" || policy.AllowHTTP, !IsReservedAddress(ip) || globalPrivate
	for _, p := range policy.Permissions {
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

// BroadCIDR warns about broad exceptions without changing their validation.
func (e Exception) BroadCIDR() bool {
	prefix, err := netip.ParsePrefix(e.CIDR)
	if err != nil {
		return false
	}
	if prefix.Addr().Is4() {
		return prefix.Bits() < 24
	}
	return prefix.Bits() < 56
}

// ProfileView contains only root-visible policy configuration and diagnostics.
type ProfileView struct {
	ID                 int32
	Name               string
	Configured, Paused bool
	Diagnostic         Diagnostic
	Exceptions         []Exception
}
