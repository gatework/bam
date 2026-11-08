package dhcp

import (
	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
	"github.com/gatework/bam/ipam"
)

// Client manages DHCPv4 ranges, reservations, deployment settings and MAC pools.
// Its operations do not submit deployment jobs; deploy explicitly afterwards.
type Client struct{ client *bam.Client }

func New(client *bam.Client) *Client { return &Client{client: client} }

// Ranges selects DHCPv4 ranges. Existing IPv6 ranges remain accessible through
// RawRanges until the DHCPv6 domain model is implemented.
func (c *Client) Ranges(networkID int64) Ranges {
	col := bam.ChildCollection[Range](c.client, "networks", networkID, "ranges").WithFilter(bam.Match("type", "IPv4DHCPRange"))
	return Ranges{resource.NewCollection[Range, RangeSpec, RangeChanges](col, "IPv4DHCPRange")}
}
func (c *Client) GlobalRanges() RangeItems {
	return RangeItems{resource.NewItems[Range, RangeChanges](bam.NewCollection[Range](c.client, "ranges").WithFilter(bam.Match("type", "IPv4DHCPRange")), "IPv4DHCPRange")}
}

// RawRanges preserves access to version-specific and DHCPv6 range contracts.
func (c *Client) RawRanges(networkID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "networks", networkID, "ranges")
}

func (c *Client) Reservations(networkID int64) Reservations {
	col := bam.ChildCollection[ipam.Address](c.client, "networks", networkID, "addresses").WithFilter(bam.And(bam.Match("type", "IPv4Address"), bam.Match("state", string(ipam.StateDHCPReserved))))
	return Reservations{resource.NewList(col)}
}
func (c *Client) MACPools(configurationID int64) MACPools {
	col := bam.ChildCollection[MACPool](c.client, "configurations", configurationID, "macPools").WithFilter(bam.Match("type", "MACPool"))
	return MACPools{resource.NewCollection[MACPool, MACPoolSpec, MACPoolChanges](col, "MACPool")}
}
func (c *Client) GlobalMACPools() MACPoolItems {
	col := bam.NewCollection[MACPool](c.client, "macPools").WithFilter(bam.Match("type", "MACPool"))
	return MACPoolItems{resource.NewItems[MACPool, MACPoolChanges](col, "MACPool")}
}
func (c *Client) MACMembers(poolID int64) MACMembers {
	return MACMembers{resource.NewList(bam.LinkedCollection[ipam.MACAddress](c.client, "macPools", poolID, "macAddresses"))}
}

func (c *Client) ConfigurationOptions(id int64) Options { return c.options("configurations", id) }
func (c *Client) BlockOptions(id int64) Options         { return c.options("blocks", id) }
func (c *Client) NetworkOptions(id int64) Options       { return c.options("networks", id) }
func (c *Client) RangeOptions(id int64) Options         { return c.options("ranges", id) }
func (c *Client) AddressOptions(id int64) Options       { return c.options("addresses", id) }
func (c *Client) MACPoolOptions(id int64) Options       { return c.options("macPools", id) }
func (c *Client) MACOptions(id int64) Options           { return c.options("macAddresses", id) }
func (c *Client) ServerOptions(id int64) Options        { return c.options("servers", id) }
func (c *Client) ServerGroupOptions(id int64) Options   { return c.options("serverGroups", id) }
func (c *Client) options(parent string, id int64) Options {
	col := bam.ChildCollection[Option](c.client, parent, id, "deploymentOptions").WithFilter(optionFilter())
	return Options{resource.NewCollection[Option, OptionSpec, OptionChanges](col, "DHCPv4ClientOption", "DHCPv4ServiceOption", "DHCPv4RawOption")}
}
func (c *Client) GlobalOptions() OptionItems {
	return OptionItems{resource.NewItems[Option, OptionChanges](bam.NewCollection[Option](c.client, "deploymentOptions").WithFilter(optionFilter()), "DHCPv4ClientOption", "DHCPv4ServiceOption", "DHCPv4RawOption")}
}
func optionFilter() bam.Filter {
	return bam.Or(bam.Match("type", "DHCPv4ClientOption"), bam.Match("type", "DHCPv4ServiceOption"), bam.Match("type", "DHCPv4RawOption"))
}
func (c *Client) OptionDefinitions() Definitions {
	col := bam.NewCollection[OptionDefinition](c.client, "deploymentOptionDefinitions").WithFilter(bam.Or(bam.Match("type", "DHCPv4ClientOptionDefinition"), bam.Match("type", "DHCPv4CustomClientOptionDefinition"), bam.Match("type", "DHCPv4ServiceOptionDefinition")))
	return Definitions{resource.NewReader(col, "DHCPv4ClientOptionDefinition", "DHCPv4CustomClientOptionDefinition", "DHCPv4ServiceOptionDefinition")}
}

func (c *Client) BlockRoles(id int64) Roles   { return c.roles("blocks", id) }
func (c *Client) NetworkRoles(id int64) Roles { return c.roles("networks", id) }
func (c *Client) MACPoolRoles(id int64) Roles { return c.roles("macPools", id) }
func (c *Client) roles(parent string, id int64) Roles {
	col := bam.ChildCollection[Role](c.client, parent, id, "deploymentRoles").WithFilter(bam.Match("type", "DHCPDeploymentRole"))
	return Roles{resource.NewCollection[Role, RoleSpec, RoleChanges](col, "DHCPDeploymentRole")}
}
func (c *Client) GlobalRoles() RoleItems {
	col := bam.NewCollection[Role](c.client, "deploymentRoles").WithFilter(bam.Match("type", "DHCPDeploymentRole"))
	return RoleItems{resource.NewItems[Role, RoleChanges](col, "DHCPDeploymentRole")}
}
func (c *Client) RoleInterfaces(id int64) RoleInterfaces {
	return RoleInterfaces{resource.NewList(bam.ChildCollection[RoleInterface](c.client, "deploymentRoles", id, "interfaces"))}
}
