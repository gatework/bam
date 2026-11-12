package dhcp

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
	"github.com/gatework/bam/ipam"
)

// FindByIP requires exactly one DHCPv4 reservation at address in this network.
// Static allocations and administrative reservations never match.
func (r Reservations) FindByIP(ctx context.Context, address netip.Addr) (ipam.Address, error) {
	if !address.Is4() {
		return ipam.Address{}, fmt.Errorf("dhcp: reservation lookup requires an ipv4 address")
	}
	options := bam.ListOptions{Where: bam.Match("address", address.String()), Limit: 2}
	return resource.OneMatching(r.All(ctx, options), func(item ipam.Address) bool {
		return isReservation(item) && item.Address == address
	})
}

// FindByMAC requires one reservation for the hardware address in this network.
// It compares complete paginated resources rather than assuming a nested MAC
// filter is supported. Multiple bindings return bam.ErrAmbiguous.
func (r Reservations) FindByMAC(ctx context.Context, mac ipam.MAC) (ipam.Address, error) {
	if !mac.IsValid() {
		return ipam.Address{}, fmt.Errorf("dhcp: reservation lookup requires a mac address")
	}
	return resource.OneMatching(r.All(ctx, bam.ListOptions{}), func(item ipam.Address) bool {
		return isReservation(item) && item.MACAddress != nil && item.MACAddress.Address == mac
	})
}

func isReservation(address ipam.Address) bool {
	return address.Type == "IPv4Address" && address.State == ipam.StateDHCPReserved
}
