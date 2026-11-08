package ipam

import (
	"context"
	"fmt"
	"iter"
	"net/netip"
	"slices"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// AddressQuery selects allocations. Zero-valued fields add no restriction.
// ConfigurationID is useful when an address occurs in several configurations.
type AddressQuery struct {
	ConfigurationID int64
	Address         netip.Addr
	Name            string
	State           State
}

func (q AddressQuery) Filter() (bam.Filter, error) {
	var predicates []bam.Filter
	if q.ConfigurationID < 0 {
		return bam.Filter{}, fmt.Errorf("ipam: configuration id cannot be negative")
	}
	if q.ConfigurationID > 0 {
		predicates = append(predicates, bam.MatchID("configuration.id", q.ConfigurationID))
	}
	if q.Address.IsValid() {
		if _, err := addressFamily(q.Address); err != nil {
			return bam.Filter{}, err
		}
		predicates = append(predicates, bam.Match("address", q.Address.String()))
	}
	if q.Name != "" {
		predicates = append(predicates, bam.Match("name", q.Name))
	}
	if q.State != "" {
		predicates = append(predicates, bam.Match("state", string(q.State)))
	}
	return bam.And(predicates...), nil
}

func (a Addresses) Search(ctx context.Context, query AddressQuery, options bam.ListOptions) iter.Seq2[Address, error] {
	return searchAddresses(ctx, resource.CollectionOf(a.List), query, options)
}
func (a AddressItems) Search(ctx context.Context, query AddressQuery, options bam.ListOptions) iter.Seq2[Address, error] {
	return searchAddresses(ctx, resource.CollectionOf(a.List), query, options)
}
func searchAddresses(ctx context.Context, collection bam.Collection[Address], query AddressQuery, options bam.ListOptions) iter.Seq2[Address, error] {
	filter, filterErr := query.Filter()
	options.Where = bam.And(options.Where, filter)
	addresses := collection.All(ctx, options)
	return func(yield func(Address, error) bool) {
		if filterErr != nil {
			yield(Address{}, filterErr)
			return
		}
		for address, err := range addresses {
			if !yield(address, err) {
				return
			}
		}
	}
}

// FindByIP returns exactly one matching allocation in this network.
func (a Addresses) FindByIP(ctx context.Context, address netip.Addr) (Address, error) {
	if _, err := addressFamily(address); err != nil {
		return Address{}, err
	}
	return resource.One(a.All(ctx, bam.ListOptions{Where: bam.Match("address", address.String()), Limit: 2}))
}

// LookupIP scopes an exact address lookup to one configuration. It never picks
// an arbitrary first result when overlapping address spaces produce ambiguity.
func (c *Client) LookupIP(ctx context.Context, configurationID int64, address netip.Addr) (Address, error) {
	if configurationID <= 0 {
		return Address{}, fmt.Errorf("ipam: configuration id must be positive")
	}
	if _, err := addressFamily(address); err != nil {
		return Address{}, err
	}
	return resource.One(c.GlobalAddresses().Search(ctx, AddressQuery{ConfigurationID: configurationID, Address: address}, bam.ListOptions{Limit: 2}))
}

func (m MACAddresses) FindByAddress(ctx context.Context, address MAC) (MACAddress, error) {
	if !address.IsValid() {
		return MACAddress{}, fmt.Errorf("ipam: mac address is required")
	}
	return resource.One(m.All(ctx, bam.ListOptions{Where: bam.Match("address", address.String()), Limit: 2}))
}
func (c *Client) LookupMAC(ctx context.Context, configurationID int64, address MAC) (MACAddress, error) {
	if configurationID <= 0 {
		return MACAddress{}, fmt.Errorf("ipam: configuration id must be positive")
	}
	return c.MACAddresses(configurationID).FindByAddress(ctx, address)
}

// AddressesByMAC resolves a hardware address in a configuration, then traverses
// its address association. Results include the current IP state and lease data.
func (c *Client) AddressesByMAC(ctx context.Context, configurationID int64, address MAC, options bam.ListOptions) iter.Seq2[Address, error] {
	options.Fields = slices.Clone(options.Fields)
	options.OrderBy = slices.Clone(options.OrderBy)
	return func(yield func(Address, error) bool) {
		mac, err := c.LookupMAC(ctx, configurationID, address)
		if err != nil {
			yield(Address{}, err)
			return
		}
		for address, err := range c.AddressesForMAC(mac.ID).All(ctx, options) {
			if !yield(address, err) {
				return
			}
		}
	}
}

func (n Networks) FindByPrefix(ctx context.Context, prefix netip.Prefix) (Network, error) {
	if _, err := prefixFamily(prefix); err != nil {
		return Network{}, err
	}
	return resource.One(n.All(ctx, bam.ListOptions{Where: bam.Match("range", prefix.String()), Limit: 2}))
}
func (b Blocks) FindByPrefix(ctx context.Context, prefix netip.Prefix) (Block, error) {
	if _, err := prefixFamily(prefix); err != nil {
		return Block{}, err
	}
	return resource.One(b.All(ctx, bam.ListOptions{Where: bam.Match("range", prefix.String()), Limit: 2}))
}

// NetworksContaining returns every containing network in the configuration;
// overlapping or nested results remain explicit to the caller.
func (c *Client) NetworksContaining(ctx context.Context, configurationID int64, address netip.Addr, options bam.ListOptions) iter.Seq2[Network, error] {
	options.Where = bam.And(options.Where, bam.MatchID("configuration.id", configurationID), bam.Contains("range", address.String()))
	networks := c.GlobalNetworks().All(ctx, options)
	return func(yield func(Network, error) bool) {
		if configurationID <= 0 {
			yield(Network{}, fmt.Errorf("ipam: configuration id must be positive"))
			return
		}
		if _, err := addressFamily(address); err != nil {
			yield(Network{}, err)
			return
		}
		for network, err := range networks {
			if !yield(network, err) {
				return
			}
		}
	}
}

// Available lists UNASSIGNED candidates from the network's assigned and
// unassigned address collection. It does not list DHCP_FREE pool addresses or
// reserve candidates. AllocateNext selects and allocates in one server mutation.
func (a Addresses) Available(ctx context.Context, options bam.ListOptions) iter.Seq2[Address, error] {
	options.Where = bam.And(options.Where, bam.Match("state", string(StateUnassigned)))
	return a.All(ctx, options)
}
