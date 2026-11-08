package ipam

import (
	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Client shares the authenticated transport used by other BAM domains.
type Client struct{ client *bam.Client }

func New(client *bam.Client) *Client { return &Client{client: client} }

// Blocks manages blocks directly beneath a configuration.
func (c *Client) Blocks(configurationID int64) Blocks {
	return Blocks{resource.NewCollection[Block, BlockSpec, BlockChanges](bam.ChildCollection[Block](c.client, "configurations", configurationID, "blocks"), "IPv4Block", "IPv6Block")}
}
func (c *Client) Subblocks(blockID int64) Blocks {
	return Blocks{resource.NewCollection[Block, BlockSpec, BlockChanges](bam.ChildCollection[Block](c.client, "blocks", blockID, "blocks"), "IPv4Block", "IPv6Block")}
}
func (c *Client) Networks(blockID int64) Networks {
	return Networks{resource.NewCollection[Network, NetworkSpec, NetworkChanges](bam.ChildCollection[Network](c.client, "blocks", blockID, "networks"), "IPv4Network", "IPv6Network")}
}
func (c *Client) Addresses(networkID int64) Addresses {
	return Addresses{resource.NewCollection[Address, AddressSpec, AddressChanges](bam.ChildCollection[Address](c.client, "networks", networkID, "addresses"), "IPv4Address", "IPv6Address")}
}
func (c *Client) GlobalBlocks() BlockItems {
	return BlockItems{resource.NewItems[Block, BlockChanges](bam.NewCollection[Block](c.client, "blocks"), "IPv4Block", "IPv6Block")}
}
func (c *Client) GlobalNetworks() NetworkItems {
	return NetworkItems{resource.NewItems[Network, NetworkChanges](bam.NewCollection[Network](c.client, "networks"), "IPv4Network", "IPv6Network")}
}
func (c *Client) GlobalAddresses() AddressItems {
	return AddressItems{resource.NewItems[Address, AddressChanges](bam.NewCollection[Address](c.client, "addresses"), "IPv4Address", "IPv6Address")}
}
func (c *Client) MACAddresses(configurationID int64) MACAddresses {
	return MACAddresses{resource.NewCollection[MACAddress, MACSpec, MACChanges](bam.ChildCollection[MACAddress](c.client, "configurations", configurationID, "macAddresses"), "MACAddress")}
}
func (c *Client) GlobalMACAddresses() MACItems {
	return MACItems{resource.NewItems[MACAddress, MACChanges](bam.NewCollection[MACAddress](c.client, "macAddresses"), "MACAddress")}
}
func (c *Client) AddressesForMAC(macID int64) AddressLinks {
	return AddressLinks{resource.NewList(bam.ChildCollection[Address](c.client, "macAddresses", macID, "addresses"))}
}

// Blocks and Networks expose typed creation only at a parent scope.
type Blocks struct {
	resource.Collection[Block, BlockSpec, BlockChanges]
}
type Networks struct {
	resource.Collection[Network, NetworkSpec, NetworkChanges]
}

// BlockItems and NetworkItems operate on existing resources by global ID.
type BlockItems struct {
	resource.Items[Block, BlockChanges]
}
type NetworkItems struct {
	resource.Items[Network, NetworkChanges]
}
