package ipam

import (
	"context"
	"iter"

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
	resources resource.Collection[Block, BlockSpec, BlockChanges]
}

// Networks provides typed network creation under an IP block.
type Networks struct {
	resources resource.Collection[Network, NetworkSpec, NetworkChanges]
}

// BlockItems and NetworkItems operate on existing resources by global ID.
type BlockItems struct {
	resources resource.Items[Block, BlockChanges]
}

// NetworkItems provides existing network operations by global ID.
type NetworkItems struct {
	resources resource.Items[Network, NetworkChanges]
}

// Page fetches one page within the selected collection scope.
func (b Blocks) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Block], error) {
	return b.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (b Blocks) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Block, error] {
	return b.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (b Blocks) Get(ctx context.Context, id int64) (Block, error) {
	return b.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (b Blocks) Document(ctx context.Context, id int64) (bam.Document, error) {
	return b.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (b Blocks) Update(ctx context.Context, id int64, changes BlockChanges, options ...bam.WriteOption) (Block, error) {
	return b.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (b Blocks) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return b.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (b Blocks) Create(ctx context.Context, spec BlockSpec, options ...bam.WriteOption) (Block, error) {
	return b.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (n Networks) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Network], error) {
	return n.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (n Networks) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Network, error] {
	return n.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (n Networks) Get(ctx context.Context, id int64) (Network, error) {
	return n.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (n Networks) Document(ctx context.Context, id int64) (bam.Document, error) {
	return n.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (n Networks) Update(ctx context.Context, id int64, changes NetworkChanges, options ...bam.WriteOption) (Network, error) {
	return n.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (n Networks) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return n.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (n Networks) Create(ctx context.Context, spec NetworkSpec, options ...bam.WriteOption) (Network, error) {
	return n.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (b BlockItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Block], error) {
	return b.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (b BlockItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Block, error] {
	return b.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (b BlockItems) Get(ctx context.Context, id int64) (Block, error) {
	return b.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (b BlockItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return b.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (b BlockItems) Update(ctx context.Context, id int64, changes BlockChanges, options ...bam.WriteOption) (Block, error) {
	return b.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (b BlockItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return b.resources.Delete(ctx, id, options...)
}

// Page fetches one page within the selected collection scope.
func (n NetworkItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Network], error) {
	return n.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (n NetworkItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Network, error] {
	return n.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (n NetworkItems) Get(ctx context.Context, id int64) (Network, error) {
	return n.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (n NetworkItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return n.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (n NetworkItems) Update(ctx context.Context, id int64, changes NetworkChanges, options ...bam.WriteOption) (Network, error) {
	return n.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (n NetworkItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return n.resources.Delete(ctx, id, options...)
}
