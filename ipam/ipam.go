package ipam

import (
	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Client shares the authenticated transport used by other BAM domains.
type Client struct{ client *bam.Client }

// New creates a domain client that shares the supplied BAM transport.
func New(client *bam.Client) *Client { return &Client{client: client} }

// Blocks manages blocks directly beneath a configuration.
func (c *Client) Blocks(configurationID int64) Blocks {
	return Blocks{resource.NewCollection[Block, BlockSpec, BlockChanges](bam.ChildCollection[Block](c.client, "configurations", configurationID, "blocks"), "IPv4Block", "IPv6Block")}
}

// Subblocks selects blocks directly beneath the supplied block.
func (c *Client) Subblocks(blockID int64) Blocks {
	return Blocks{resource.NewCollection[Block, BlockSpec, BlockChanges](bam.ChildCollection[Block](c.client, "blocks", blockID, "blocks"), "IPv4Block", "IPv6Block")}
}

// Networks selects networks directly beneath the supplied block.
func (c *Client) Networks(blockID int64) Networks {
	return Networks{resource.NewCollection[Network, NetworkSpec, NetworkChanges](bam.ChildCollection[Network](c.client, "blocks", blockID, "networks"), "IPv4Network", "IPv6Network")}
}

// Addresses selects addresses directly beneath the supplied network.
func (c *Client) Addresses(networkID int64) Addresses {
	return Addresses{resource.NewCollection[Address, AddressSpec, AddressChanges](bam.ChildCollection[Address](c.client, "networks", networkID, "addresses"), "IPv4Address", "IPv6Address")}
}

// GlobalBlocks selects existing blocks by global resource ID.
func (c *Client) GlobalBlocks() BlockItems {
	return BlockItems{resource.NewItems[Block, BlockChanges](bam.NewCollection[Block](c.client, "blocks"), "IPv4Block", "IPv6Block")}
}

// GlobalNetworks selects existing networks by global resource ID.
func (c *Client) GlobalNetworks() NetworkItems {
	return NetworkItems{resource.NewItems[Network, NetworkChanges](bam.NewCollection[Network](c.client, "networks"), "IPv4Network", "IPv6Network")}
}

// GlobalAddresses selects existing addresses by global resource ID.
func (c *Client) GlobalAddresses() AddressItems {
	return AddressItems{resource.NewItems[Address, AddressChanges](bam.NewCollection[Address](c.client, "addresses"), "IPv4Address", "IPv6Address")}
}

// MACAddresses selects MAC address resources under the configuration.
func (c *Client) MACAddresses(configurationID int64) MACAddresses {
	return MACAddresses{resource.NewCollection[MACAddress, MACSpec, MACChanges](bam.ChildCollection[MACAddress](c.client, "configurations", configurationID, "macAddresses"), "MACAddress")}
}

// GlobalMACAddresses selects existing MAC addresses by global resource ID.
func (c *Client) GlobalMACAddresses() MACItems {
	return MACItems{resource.NewItems[MACAddress, MACChanges](bam.NewCollection[MACAddress](c.client, "macAddresses"), "MACAddress")}
}

// AddressesForMAC selects IP address associations for a MAC address.
func (c *Client) AddressesForMAC(macID int64) AddressLinks {
	return AddressLinks{resource.NewList(bam.ChildCollection[Address](c.client, "macAddresses", macID, "addresses"))}
}

// Blocks and Networks expose typed creation only at a parent scope.
type Blocks struct {
	resource.Collection[Block, BlockSpec, BlockChanges]
}

// Networks provides typed network creation under an IP block.
type Networks struct {
	resource.Collection[Network, NetworkSpec, NetworkChanges]
}

// BlockItems and NetworkItems operate on existing resources by global ID.
type BlockItems struct {
	resource.Items[Block, BlockChanges]
}

// NetworkItems provides existing network operations by global ID.
type NetworkItems struct {
	resource.Items[Network, NetworkChanges]
}
