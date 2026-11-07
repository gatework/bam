package ipam

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/gatework/bam"
)

// Block is a read model for IPv4Block and IPv6Block resources. Range retains the
// server's representation; a partial read model must not be used as a full PUT.
type Block struct {
	bam.Resource
	Configuration bam.Reference `json:"configuration"`
	Range         string        `json:"range"`
}

// Network is a read model for IPv4Network and IPv6Network resources. Use a
// Document when an update must retain fields not represented here.
type Network struct {
	bam.Resource
	Configuration bam.Reference `json:"configuration"`
	Range         string        `json:"range"`
}

// Address is a read model for IPv4Address and IPv6Address resources. An address
// omitted by a fields projection has the zero netip.Addr value.
type Address struct {
	bam.Resource
	Configuration bam.Reference `json:"configuration"`
	Address       netip.Addr    `json:"address"`
	State         State         `json:"state"`
}

// State is the server's address state. This is an open set: values not listed
// below are preserved, and the server validates states supported by an operation.
type State string

// States used in the Address Manager 9.6 RESTful v2 filtering examples.
const (
	StateUnassigned   State = "UNASSIGNED"
	StateReserved     State = "RESERVED"
	StateDHCPReserved State = "DHCP_RESERVED"
)

// BlockSpec describes the common fields for creating a block with a CIDR range.
// For size-only or other server-supported ranges use Collection.Create with a
// body based on the target server's OpenAPI schema.
type BlockSpec struct {
	Name  string       `json:"name,omitempty"`
	Range netip.Prefix `json:"range"`
}

// MarshalJSON validates the range and supplies the IPv4Block or IPv6Block type.
// Prefixes containing host bits are rejected instead of silently masked.
func (s BlockSpec) MarshalJSON() ([]byte, error) {
	return marshalPrefix(s.Name, s.Range, "Block")
}

// NetworkSpec describes the common fields for creating a network with a CIDR range.
type NetworkSpec struct {
	Name  string       `json:"name,omitempty"`
	Range netip.Prefix `json:"range"`
}

// MarshalJSON validates the range and supplies the IPv4Network or IPv6Network type.
// Prefixes containing host bits are rejected instead of silently masked.
func (s NetworkSpec) MarshalJSON() ([]byte, error) {
	return marshalPrefix(s.Name, s.Range, "Network")
}

// AddressSpec describes an exact address assignment. The resource type is
// inferred from Address. An empty State omits the field; the server validates
// which states an assignment supports.
type AddressSpec struct {
	Name    string     `json:"name,omitempty"`
	Address netip.Addr `json:"address"`
	State   State      `json:"state,omitempty"`
}

func marshalPrefix(name string, prefix netip.Prefix, kind string) ([]byte, error) {
	if !prefix.IsValid() {
		return nil, fmt.Errorf("ipam: range must be a valid prefix")
	}
	if prefix != prefix.Masked() {
		return nil, fmt.Errorf("ipam: range %s contains host bits", prefix)
	}
	family, err := addressFamily(prefix.Addr())
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Type  string `json:"type"`
		Name  string `json:"name,omitempty"`
		Range string `json:"range"`
	}{
		Type:  family + kind,
		Name:  name,
		Range: prefix.String(),
	})
}

// MarshalJSON validates the address and supplies the IPv4Address or IPv6Address type.
func (s AddressSpec) MarshalJSON() ([]byte, error) {
	family, err := addressFamily(s.Address)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Type    string `json:"type"`
		Name    string `json:"name,omitempty"`
		Address string `json:"address"`
		State   State  `json:"state,omitempty"`
	}{
		Type:    family + "Address",
		Name:    s.Name,
		Address: s.Address.String(),
		State:   s.State,
	})
}

func addressFamily(address netip.Addr) (string, error) {
	if !address.IsValid() {
		return "", fmt.Errorf("ipam: a valid IP address is required")
	}
	if address.Zone() != "" {
		return "", fmt.Errorf("ipam: scoped IPv6 addresses are not supported")
	}
	if address.Is4() {
		return "IPv4", nil
	}
	return "IPv6", nil
}
