package ipam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// AddressSpec assigns an exact address. State defaults to STATIC. Use
// AllocateNext for server-side selection rather than a query followed by Create.
type AddressSpec struct {
	Name       string        `json:"name,omitempty"`
	Address    netip.Addr    `json:"address"`
	State      State         `json:"state"`
	MACAddress *MACReference `json:"macAddress,omitempty"`
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s AddressSpec) MarshalJSON() ([]byte, error) {
	family, err := addressFamily(s.Address)
	if err != nil {
		return nil, err
	}
	if s.State == "" {
		s.State = StateStatic
	}
	if err := validateAssignment(family+"Address", s.State, s.MACAddress); err != nil {
		return nil, err
	}
	type spec AddressSpec
	return json.Marshal(struct {
		spec
		Type string `json:"type"`
	}{spec: spec(s), Type: family + "Address"})
}

// NextAddressSpec selects the next available IPv4 address atomically on BAM.
// State defaults to STATIC. Exclusions and ping behavior are request options.
type NextAddressSpec struct {
	Name       string        `json:"name,omitempty"`
	State      State         `json:"state"`
	MACAddress *MACReference `json:"macAddress,omitempty"`
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s NextAddressSpec) MarshalJSON() ([]byte, error) {
	if s.State == "" {
		s.State = StateStatic
	}
	if err := validateAssignment("IPv4Address", s.State, s.MACAddress); err != nil {
		return nil, err
	}
	type spec NextAddressSpec
	return json.Marshal(struct {
		spec
		Type string `json:"type"`
	}{spec: spec(s), Type: "IPv4Address"})
}
func validateAssignment(kind string, state State, mac *MACReference) error {
	if err := validateAssignmentState(kind, state); err != nil {
		return err
	}
	if state == StateDHCPReserved && mac == nil {
		return fmt.Errorf("ipam: dhcp reservation requires a mac address")
	}
	if mac != nil {
		if _, err := mac.MarshalJSON(); err != nil {
			return err
		}
	}
	return nil
}

func validateAssignmentState(kind string, state State) error {
	if kind != "IPv4Address" && kind != "IPv6Address" {
		return fmt.Errorf("ipam: unsupported address type %q", kind)
	}
	switch state {
	case StateStatic, StateDHCPReserved:
		return nil
	case StateReserved, StateGateway:
		if kind == "IPv4Address" {
			return nil
		}
	default:
		// Other states are rejected below.
	}
	return fmt.Errorf("ipam: state %q cannot be assigned to %s", state, kind)
}

// AddressChanges updates allocation metadata or transitions an allocation
// between states allowed by its address family. IPv6 supports STATIC and
// DHCP_RESERVED. Release an allocation with Delete, not UNASSIGNED.
type AddressChanges struct {
	Name       bam.Field[string]
	State      bam.Field[State]
	MACAddress bam.Field[MACReference]
}

// Apply validates and applies explicitly set fields to the complete resource document.
func (c AddressChanges) Apply(d bam.Document) error {
	if c.State.IsNull() {
		return fmt.Errorf("ipam: address state cannot be cleared")
	}
	if mac, ok := c.MACAddress.Value(); ok {
		if _, err := mac.MarshalJSON(); err != nil {
			return err
		}
	}
	if err := errors.Join(c.Name.Apply(d, "name"), c.State.Apply(d, "state"), c.MACAddress.Apply(d, "macAddress")); err != nil {
		return err
	}
	if c.State.IsSet() || c.MACAddress.IsSet() {
		var address struct {
			Type  string          `json:"type"`
			State State           `json:"state"`
			MAC   json.RawMessage `json:"macAddress"`
		}
		if err := d.Decode(&address); err != nil {
			return err
		}
		if err := validateAssignmentState(address.Type, address.State); err != nil {
			return err
		}
		if address.State == StateDHCPReserved {
			if len(address.MAC) == 0 || string(address.MAC) == "null" {
				return fmt.Errorf("ipam: dhcp reservation requires a mac address")
			}
		}
	}
	return nil
}

// Addresses manages allocations within a network.
type Addresses struct {
	resource.Collection[Address, AddressSpec, AddressChanges]
}

// AddressItems manages existing allocations across networks.
type AddressItems struct {
	resource.Items[Address, AddressChanges]
}

// AssignStatic creates a static allocation, optionally recording its MAC.
func (a Addresses) AssignStatic(ctx context.Context, spec AddressSpec, options ...bam.WriteOption) (Address, error) {
	if spec.State != "" && spec.State != StateStatic {
		return Address{}, fmt.Errorf("ipam: static assignment cannot specify another state")
	}
	spec.State = StateStatic
	return a.Create(ctx, spec, options...)
}

// Reserve reserves an IPv4 address administratively, without creating a DHCP binding.
func (a Addresses) Reserve(ctx context.Context, spec AddressSpec, options ...bam.WriteOption) (Address, error) {
	if !spec.Address.Is4() {
		return Address{}, fmt.Errorf("ipam: administrative reservation requires an ipv4 address")
	}
	if spec.State != "" && spec.State != StateReserved {
		return Address{}, fmt.Errorf("ipam: reservation cannot specify another state")
	}
	spec.State = StateReserved
	return a.Create(ctx, spec, options...)
}

// AllocateNext submits one allocation request. It does not enumerate free
// addresses, and never replays a failed write. BAM performs address selection.
func (a Addresses) AllocateNext(ctx context.Context, spec NextAddressSpec, options ...bam.WriteOption) (Address, error) {
	result, err := resource.CollectionOf(a.List).Create(ctx, spec, options...)
	if err != nil {
		return result, err
	}
	if result.Type != "IPv4Address" {
		return Address{}, fmt.Errorf("ipam: allocation response is not an ipv4 address")
	}
	return result, nil
}

// Release removes an allocation. Dependent resource conflicts are returned by
// BAM; this operation never deletes the associated MAC resource separately.
func (a AddressItems) Release(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return a.Delete(ctx, id, options...)
}

// Release releases the address allocation without retrying the mutation.
func (a Addresses) Release(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return a.Delete(ctx, id, options...)
}
