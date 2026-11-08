package ipam

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"

	"github.com/gatework/bam"
)

// BlockSpec creates a block with an exact CIDR. Host bits are rejected.
type BlockSpec struct {
	Name                        string         `json:"name,omitempty"`
	Range                       netip.Prefix   `json:"range"`
	PingBeforeAssignmentEnabled *bool          `json:"pingBeforeAssignmentEnabled,omitempty"`
	DefaultView                 *bam.Reference `json:"defaultView,omitempty"`
}

func (s BlockSpec) MarshalJSON() ([]byte, error) {
	family, err := prefixFamily(s.Range)
	if err != nil {
		return nil, err
	}
	if family == "IPv6" && (s.PingBeforeAssignmentEnabled != nil || s.DefaultView != nil) {
		return nil, fmt.Errorf("ipam: ipv4 settings cannot be applied to an ipv6 block")
	}
	if err := validReference(s.DefaultView); err != nil {
		return nil, err
	}
	type spec BlockSpec
	return json.Marshal(struct {
		Type string `json:"type"`
		spec
	}{Type: family + "Block", spec: spec(s)})
}

// NetworkSpec creates a network with an exact CIDR. Nil settings use server
// defaults; a nonnil false value explicitly disables a boolean setting.
type NetworkSpec struct {
	Name                        string         `json:"name,omitempty"`
	Range                       netip.Prefix   `json:"range"`
	Gateway                     *netip.Addr    `json:"gateway,omitempty"`
	PingBeforeAssignmentEnabled *bool          `json:"pingBeforeAssignmentEnabled,omitempty"`
	DynamicUpdateEnabled        *bool          `json:"dynamicUpdateEnabled,omitempty"`
	DefaultView                 *bam.Reference `json:"defaultView,omitempty"`
}

func (s NetworkSpec) MarshalJSON() ([]byte, error) {
	family, err := prefixFamily(s.Range)
	if err != nil {
		return nil, err
	}
	if family == "IPv6" && (s.Gateway != nil || s.PingBeforeAssignmentEnabled != nil || s.DynamicUpdateEnabled != nil || s.DefaultView != nil) {
		return nil, fmt.Errorf("ipam: ipv4 settings cannot be applied to an ipv6 network")
	}
	if s.Gateway != nil && (!s.Gateway.Is4() || !s.Range.Contains(*s.Gateway)) {
		return nil, fmt.Errorf("ipam: gateway must be an ipv4 address in the network")
	}
	if err := validReference(s.DefaultView); err != nil {
		return nil, err
	}
	type spec NetworkSpec
	return json.Marshal(struct {
		Type string `json:"type"`
		spec
	}{Type: family + "Network", spec: spec(s)})
}

// BlockChanges updates selected block fields while preserving the full resource.
type BlockChanges struct {
	Name                        bam.Field[string]
	Range                       bam.Field[netip.Prefix]
	PingBeforeAssignmentEnabled bam.Field[bool]
	DefaultView                 bam.Field[bam.Reference]
}

func (c BlockChanges) Apply(d bam.Document) error {
	if err := validatePrefixChange(d, c.Range); err != nil {
		return err
	}
	if err := requireIPv4Settings(d, c.PingBeforeAssignmentEnabled.IsSet() || c.DefaultView.IsSet()); err != nil {
		return err
	}
	if ref, ok := c.DefaultView.Value(); ok && ref.ID <= 0 {
		return fmt.Errorf("ipam: default view requires a positive id")
	}
	return errors.Join(c.Name.Apply(d, "name"), c.Range.Apply(d, "range"), c.PingBeforeAssignmentEnabled.Apply(d, "pingBeforeAssignmentEnabled"), c.DefaultView.Apply(d, "defaultView"))
}

// NetworkChanges updates selected network settings. WithNoGateway removes the
// gateway using the protocol's dedicated request option.
type NetworkChanges struct {
	Name                        bam.Field[string]
	Range                       bam.Field[netip.Prefix]
	Gateway                     bam.Field[netip.Addr]
	PingBeforeAssignmentEnabled bam.Field[bool]
	DynamicUpdateEnabled        bam.Field[bool]
	DefaultView                 bam.Field[bam.Reference]
}

func (c NetworkChanges) Apply(d bam.Document) error {
	if err := validatePrefixChange(d, c.Range); err != nil {
		return err
	}
	if err := requireIPv4Settings(d, c.Gateway.IsSet() || c.PingBeforeAssignmentEnabled.IsSet() || c.DynamicUpdateEnabled.IsSet() || c.DefaultView.IsSet()); err != nil {
		return err
	}
	if c.Gateway.IsNull() {
		return fmt.Errorf("ipam: remove a gateway using WithNoGateway")
	}
	if gateway, ok := c.Gateway.Value(); ok && !gateway.Is4() {
		return fmt.Errorf("ipam: gateway must be an ipv4 address")
	}
	if ref, ok := c.DefaultView.Value(); ok && ref.ID <= 0 {
		return fmt.Errorf("ipam: default view requires a positive id")
	}
	return errors.Join(c.Name.Apply(d, "name"), c.Range.Apply(d, "range"), c.Gateway.Apply(d, "gateway"), c.PingBeforeAssignmentEnabled.Apply(d, "pingBeforeAssignmentEnabled"), c.DynamicUpdateEnabled.Apply(d, "dynamicUpdateEnabled"), c.DefaultView.Apply(d, "defaultView"))
}

func prefixFamily(prefix netip.Prefix) (string, error) {
	if !prefix.IsValid() || prefix != prefix.Masked() {
		return "", fmt.Errorf("ipam: range must be a valid prefix without host bits")
	}
	return addressFamily(prefix.Addr())
}
func addressFamily(address netip.Addr) (string, error) {
	if !address.IsValid() || address.Zone() != "" {
		return "", fmt.Errorf("ipam: a valid unscoped ip address is required")
	}
	if address.Is4() {
		return "IPv4", nil
	}
	return "IPv6", nil
}
func validatePrefixChange(d bam.Document, f bam.Field[netip.Prefix]) error {
	if f.IsNull() {
		return fmt.Errorf("ipam: range cannot be cleared")
	}
	if prefix, ok := f.Value(); ok {
		family, err := prefixFamily(prefix)
		if err != nil {
			return err
		}
		var r bam.Resource
		if err := d.Decode(&r); err != nil {
			return err
		}
		if r.Type != family+"Block" && r.Type != family+"Network" {
			return fmt.Errorf("ipam: range cannot change address family")
		}
	}
	return nil
}
func requireIPv4Settings(d bam.Document, hasSettings bool) error {
	if !hasSettings {
		return nil
	}
	var r bam.Resource
	if err := d.Decode(&r); err != nil {
		return err
	}
	if r.Type != "IPv4Block" && r.Type != "IPv4Network" {
		return fmt.Errorf("ipam: settings require an ipv4 resource")
	}
	return nil
}
func validReference(ref *bam.Reference) error {
	if ref != nil && ref.ID <= 0 {
		return fmt.Errorf("ipam: reference requires a positive id")
	}
	return nil
}
