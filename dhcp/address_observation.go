package dhcp

import (
	"context"
	"iter"

	"github.com/gatework/bam"
	"github.com/gatework/bam/ipam"
)

// LeasedAddresses observes IPv4 addresses recorded as DHCP_LEASED in a network.
// Address includes the reported MAC, client identifier and lease timestamps.
// These are BAM address observations, not a live DHCP server query or a lease
// history. This read never renews or releases a lease, or changes a reservation.
func (c *Client) LeasedAddresses(ctx context.Context, networkID int64, options bam.ListOptions) iter.Seq2[ipam.Address, error] {
	collection := bam.ChildCollection[ipam.Address](c.client, "networks", networkID, "addresses").WithFilter(bam.And(
		bam.Match("type", "IPv4Address"), bam.Match("state", string(ipam.StateDHCPLeased)),
	))
	return collection.All(ctx, options)
}
