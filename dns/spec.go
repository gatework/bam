package dns

import (
	"encoding/json"
	"fmt"
	"net/netip"
)

// ViewSpec creates a view within a configuration. Additional fields supported
// by a particular server can be sent using a bam.Document.
type ViewSpec struct {
	Name string `json:"name"`
}

// MarshalJSON supplies the RESTful v2 resource type.
func (s ViewSpec) MarshalJSON() ([]byte, error) {
	type spec ViewSpec
	return json.Marshal(struct {
		Type string `json:"type"`
		spec
	}{Type: "View", spec: spec(s)})
}

// ZoneSpec creates a zone within a view or another zone. Name is relative to
// that parent; AbsoluteName requests creation by absolute name. The server
// applies its zone naming policy and validates the supplied fields.
type ZoneSpec struct {
	Name         string `json:"name,omitempty"`
	AbsoluteName string `json:"absoluteName,omitempty"`
}

// MarshalJSON supplies the RESTful v2 resource type.
func (s ZoneSpec) MarshalJSON() ([]byte, error) {
	type spec ZoneSpec
	return json.Marshal(struct {
		Type string `json:"type"`
		spec
	}{Type: "Zone", spec: spec(s)})
}

// HostRecordSpec creates a host record within a zone. Addresses are encoded as
// IPv4Address or IPv6Address resources. ReverseRecord is always sent, including
// false, so disabling reverse-record creation is explicit.
//
// Address Manager validates the record name and address-allocation rules.
// IPv4-mapped IPv6 addresses retain their IPv6 representation.
type HostRecordSpec struct {
	Name          string       `json:"name"`
	Addresses     []netip.Addr `json:"addresses"`
	ReverseRecord bool         `json:"reverseRecord"`
}

// MarshalJSON validates address syntax and supplies the record and address
// resource types. IPv6 scope identifiers are local to a host and cannot be
// represented as DNS AAAA address data.
func (s HostRecordSpec) MarshalJSON() ([]byte, error) {
	type address struct {
		Type    string `json:"type"`
		Address string `json:"address"`
	}
	addresses := make([]address, len(s.Addresses))
	for i, addr := range s.Addresses {
		if !addr.IsValid() {
			return nil, fmt.Errorf("dns: address %d is invalid", i)
		}
		if addr.Zone() != "" {
			return nil, fmt.Errorf("dns: address %d contains an ipv6 scope identifier", i)
		}
		kind := "IPv6Address"
		if addr.Is4() {
			kind = "IPv4Address"
		}
		addresses[i] = address{Type: kind, Address: addr.String()}
	}

	return json.Marshal(struct {
		Type          string    `json:"type"`
		Name          string    `json:"name"`
		Addresses     []address `json:"addresses"`
		ReverseRecord bool      `json:"reverseRecord"`
	}{
		Type:          "HostRecord",
		Name:          s.Name,
		Addresses:     addresses,
		ReverseRecord: s.ReverseRecord,
	})
}
