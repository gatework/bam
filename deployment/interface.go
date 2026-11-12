package deployment

import (
	"context"
	"iter"
	"net/netip"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Interface is the discovery projection for NetworkInterface and
// PublishedInterface resources. Network interfaces expose management and
// service addresses; published interfaces expose PrimaryAddress. Its ID is
// the interface reference used when assigning a DNS or DHCP deployment role.
type Interface struct {
	ID                       int64         `json:"id"`
	Type                     string        `json:"type"`
	Name                     string        `json:"name"`
	Configuration            bam.Reference `json:"configuration"`
	Server                   bam.Reference `json:"server"`
	SecondaryAddress         netip.Addr    `json:"secondaryAddress"`
	PrimaryAddress           netip.Addr    `json:"primaryAddress"`
	ManagementAddress        netip.Addr    `json:"managementAddress"`
	ServicesIPv4Address      netip.Addr    `json:"servicesIpv4Address"`
	ServicesIPv4PrefixLength int           `json:"servicesIpv4PrefixLength"`
	ServicesIPv6Address      netip.Addr    `json:"servicesIpv6Address"`
	ServicesIPv6PrefixLength int           `json:"servicesIpv6PrefixLength"`
}

// Interfaces discovers the server interfaces referenced by deployment roles.
// Interface creation, replacement and deletion are intentionally not exposed.
type Interfaces struct {
	resources resource.Reader[Interface]
}

// ServerInterfaces selects all interfaces belonging to a server.
func (c *Client) ServerInterfaces(serverID int64) Interfaces {
	return Interfaces{resources: resource.NewReader(
		bam.ChildCollection[Interface](c.client, "servers", serverID, "interfaces"), "NetworkInterface", "PublishedInterface")}
}

// GlobalInterfaces selects interfaces across servers.
func (c *Client) GlobalInterfaces() Interfaces {
	return Interfaces{resources: resource.NewReader(
		bam.NewCollection[Interface](c.client, "interfaces"), "NetworkInterface", "PublishedInterface")}
}

// Page fetches one page within the selected collection scope.
func (i Interfaces) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Interface], error) {
	return i.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (i Interfaces) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Interface, error] {
	return i.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (i Interfaces) Get(ctx context.Context, id int64) (Interface, error) {
	return i.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (i Interfaces) Document(ctx context.Context, id int64) (bam.Document, error) {
	return i.resources.Document(ctx, id)
}
