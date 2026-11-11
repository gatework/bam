package ipam

import (
	"encoding/json"
	"fmt"
	"net/netip"
)

// AddressRange is an inclusive range of addresses in one family.
type AddressRange struct {
	Start netip.Addr
	End   netip.Addr
}

// Validate checks ordering, family, and scope without normalizing the endpoints.
func (r AddressRange) Validate() error {
	if !r.Start.IsValid() || !r.End.IsValid() || r.Start.Zone() != "" || r.End.Zone() != "" || r.Start.Is4() != r.End.Is4() || r.Start.Compare(r.End) > 0 {
		return fmt.Errorf("ipam: address range requires ordered endpoints in one family")
	}
	return nil
}
func (r AddressRange) String() string { return r.Start.String() + "-" + r.End.String() }

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (r AddressRange) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r.String())
}

// Contains reports whether address belongs to this inclusive range.
func (r AddressRange) Contains(address netip.Addr) bool {
	return r.Validate() == nil && address.IsValid() && address.Zone() == "" && address.Is4() == r.Start.Is4() && r.Start.Compare(address) <= 0 && r.End.Compare(address) >= 0
}
