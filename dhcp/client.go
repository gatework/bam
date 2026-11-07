package dhcp

import "github.com/gatework/bam"

// Client selects DHCP collections using a shared RESTful v2 client.
type Client struct {
	client *bam.Client
}

// New creates DHCP selectors. It does not authenticate or send requests.
func New(client *bam.Client) *Client {
	return &Client{client: client}
}

// Ranges selects DHCP ranges belonging to an IPv4 or IPv6 network.
func (c *Client) Ranges(networkID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "networks", networkID, "ranges")
}

// MACPools selects MAC pools belonging to a configuration.
func (c *Client) MACPools(configurationID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "configurations", configurationID, "macPools")
}

// MACAddresses selects MAC address resources belonging to a configuration.
func (c *Client) MACAddresses(configurationID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "configurations", configurationID, "macAddresses")
}

// ConfigurationOptions selects deployment options at configuration scope.
func (c *Client) ConfigurationOptions(configurationID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "configurations", configurationID, "deploymentOptions")
}

// BlockOptions selects deployment options at IP block scope.
func (c *Client) BlockOptions(blockID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "blocks", blockID, "deploymentOptions")
}

// NetworkOptions selects deployment options at IP network scope.
func (c *Client) NetworkOptions(networkID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "networks", networkID, "deploymentOptions")
}

// RangeOptions selects deployment options at DHCP range scope.
func (c *Client) RangeOptions(rangeID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "ranges", rangeID, "deploymentOptions")
}

// AddressOptions selects deployment options at DHCP reserved address scope.
func (c *Client) AddressOptions(addressID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "addresses", addressID, "deploymentOptions")
}

// BlockRoles selects deployment roles at IP block scope.
func (c *Client) BlockRoles(blockID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "blocks", blockID, "deploymentRoles")
}

// NetworkRoles selects deployment roles at IP network scope.
func (c *Client) NetworkRoles(networkID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "networks", networkID, "deploymentRoles")
}
