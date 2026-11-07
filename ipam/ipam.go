package ipam

import "github.com/gatework/bam"

// Client uses a shared authenticated client for IP address management.
type Client struct {
	client *bam.Client
}

// New creates an IPAM client. It does not make a network request.
func New(client *bam.Client) *Client {
	return &Client{client: client}
}

// Blocks returns the block collection belonging to a configuration.
func (c *Client) Blocks(configurationID int64) bam.Collection[Block] {
	return bam.ChildCollection[Block](c.client, "configurations", configurationID, "blocks")
}

// Subblocks returns the block collection belonging to a parent block.
func (c *Client) Subblocks(blockID int64) bam.Collection[Block] {
	return bam.ChildCollection[Block](c.client, "blocks", blockID, "blocks")
}

// Networks returns the network collection belonging to a block.
func (c *Client) Networks(blockID int64) bam.Collection[Network] {
	return bam.ChildCollection[Network](c.client, "blocks", blockID, "networks")
}

// Addresses returns the address collection belonging to a network.
func (c *Client) Addresses(networkID int64) bam.Collection[Address] {
	return bam.ChildCollection[Address](c.client, "networks", networkID, "addresses")
}

// GlobalBlocks accesses blocks across configurations for filtering and lookup.
// Create blocks through Blocks or Subblocks to supply the parent scope.
func (c *Client) GlobalBlocks() bam.Collection[Block] {
	return bam.NewCollection[Block](c.client, "blocks")
}

// GlobalNetworks accesses networks across configurations for filtering and lookup.
// Create networks through Networks to supply the parent block.
func (c *Client) GlobalNetworks() bam.Collection[Network] {
	return bam.NewCollection[Network](c.client, "networks")
}

// GlobalAddresses accesses addresses across configurations for filtering and lookup.
// Create addresses through Addresses to supply the parent network.
func (c *Client) GlobalAddresses() bam.Collection[Address] {
	return bam.NewCollection[Address](c.client, "addresses")
}
