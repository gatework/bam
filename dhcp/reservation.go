package dhcp

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/netip"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
	"github.com/gatework/bam/ipam"
)

// ReservationSpec binds an exact IPv4 address to a MAC for DHCP. This is distinct
// from an IPAM administrative reservation, which does not create a DHCP binding.
type ReservationSpec struct {
	Name    string
	Address netip.Addr
	MAC     ipam.MACReference
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s ReservationSpec) MarshalJSON() ([]byte, error) {
	if !s.Address.Is4() {
		return nil, fmt.Errorf("dhcp: reservation requires an ipv4 address")
	}
	return json.Marshal(ipam.AddressSpec{Name: s.Name, Address: s.Address, State: ipam.StateDHCPReserved, MACAddress: &s.MAC})
}

// ReservationChanges updates a reservation's name or MAC while retaining its
// address and DHCP_RESERVED state. Reassign an address through IPAM explicitly.
type ReservationChanges struct {
	Name bam.Field[string]
	MAC  bam.Field[ipam.MACReference]
}

// Apply validates and applies explicitly set fields to the complete resource document.
func (c ReservationChanges) Apply(d bam.Document) error {
	if err := requireReservation(d); err != nil {
		return err
	}
	if c.MAC.IsNull() {
		return fmt.Errorf("dhcp: reservation cannot clear its mac binding")
	}
	return (ipam.AddressChanges{Name: c.Name, MACAddress: c.MAC}).Apply(d)
}

// Reservations manages DHCPv4 bindings in a network. Global resource IDs are
// checked for DHCP_RESERVED before reads, updates, or deletion.
type Reservations struct {
	resources resource.List[ipam.Address]
}

// Create creates one typed resource without retrying the mutation.
func (r Reservations) Create(ctx context.Context, spec ReservationSpec, options ...bam.WriteOption) (ipam.Address, error) {
	result, err := resource.CollectionOf(r.resources).Create(ctx, spec, options...)
	if err != nil {
		return result, err
	}
	if result.Type != "IPv4Address" || result.State != ipam.StateDHCPReserved {
		return ipam.Address{}, fmt.Errorf("dhcp: response is not a dhcpv4 reservation")
	}
	return result, nil
}

// Get reads and validates the resource with the supplied ID.
func (r Reservations) Get(ctx context.Context, id int64) (ipam.Address, error) {
	document, err := resource.CollectionOf(r.resources).Document(ctx, id)
	if err != nil {
		return ipam.Address{}, err
	}
	if err := requireReservation(document); err != nil {
		return ipam.Address{}, err
	}
	var result ipam.Address
	err = document.Decode(&result)
	return result, err
}

// Update reads the complete resource and applies typed changes without retrying the mutation.
func (r Reservations) Update(ctx context.Context, id int64, changes ReservationChanges, options ...bam.WriteOption) (ipam.Address, error) {
	return resource.CollectionOf(r.resources).Update(ctx, id, changes.Apply, options...)
}

// Delete deletes the resource without retrying the mutation.
func (r Reservations) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	if _, err := r.Get(ctx, id); err != nil {
		return err
	}
	return resource.CollectionOf(r.resources).Delete(ctx, id, options...)
}

// AllocateNext creates a binding using one server-side allocation request.
func (r Reservations) AllocateNext(ctx context.Context, name string, mac ipam.MACReference, options ...bam.WriteOption) (ipam.Address, error) {
	result, err := resource.CollectionOf(r.resources).Create(ctx, ipam.NextAddressSpec{Name: name, State: ipam.StateDHCPReserved, MACAddress: &mac}, options...)
	if err != nil {
		return result, err
	}
	if result.Type != "IPv4Address" || result.State != ipam.StateDHCPReserved {
		return ipam.Address{}, fmt.Errorf("dhcp: response is not a dhcpv4 reservation")
	}
	return result, nil
}
func requireReservation(d bam.Document) error {
	var address ipam.Address
	if err := d.Decode(&address); err != nil {
		return err
	}
	if address.Type != "IPv4Address" || address.State != ipam.StateDHCPReserved {
		return fmt.Errorf("dhcp: operation requires a dhcpv4 reservation")
	}
	return nil
}

// Page fetches one page within the selected collection scope.
func (r Reservations) Page(ctx context.Context, options bam.ListOptions) (bam.Page[ipam.Address], error) {
	return r.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r Reservations) All(ctx context.Context, options bam.ListOptions) iter.Seq2[ipam.Address, error] {
	return r.resources.All(ctx, options)
}
