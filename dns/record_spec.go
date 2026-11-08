package dns

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"

	"github.com/gatework/bam"
)

// RecordSpec is a validated create specification for a known record subtype.
// Use the concrete HostRecordSpec, AliasRecordSpec and other record specs.
type RecordSpec interface {
	json.Marshaler
	recordKind() RecordKind
}

func validateRecord(name string, ttl *int64) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("dns: record name is required")
	}
	if ttl != nil && (*ttl < 0 || *ttl > 2147483647) {
		return fmt.Errorf("dns: ttl must be between 0 and 2147483647 seconds")
	}
	return nil
}

func (t RecordTarget) validate(hostOnly bool) error {
	if t.ID < 0 || t.ID == 0 && strings.TrimSpace(t.AbsoluteName) == "" {
		return fmt.Errorf("dns: target requires a positive id or absolute name")
	}
	switch t.Type {
	case "", Host, External:
		return nil
	case Alias:
		if !hostOnly {
			return nil
		}
	}
	return fmt.Errorf("dns: unsupported target record type")
}

type linkedAddress struct {
	ID      int64  `json:"id,omitempty"`
	Type    string `json:"type"`
	Address string `json:"address,omitempty"`
}

func addressValues(addresses []netip.Addr) ([]linkedAddress, error) {
	if len(addresses) == 0 {
		return nil, fmt.Errorf("dns: host requires at least one address")
	}
	values := make([]linkedAddress, len(addresses))
	for i, address := range addresses {
		if !address.IsValid() || address.Zone() != "" {
			return nil, fmt.Errorf("dns: address %d is invalid or contains a scope identifier", i)
		}
		kind := "IPv6Address"
		if address.Is4() {
			kind = "IPv4Address"
		}
		values[i] = linkedAddress{Type: kind, Address: address.String()}
	}
	return values, nil
}

// HostRecordSpec creates a host with A/AAAA associations and an explicit PTR
// setting. IPv4-mapped IPv6 addresses retain their IPv6 representation.
type HostRecordSpec struct {
	Name              string            `json:"name"`
	TTL               *int64            `json:"ttl,omitempty"`
	Comment           string            `json:"comment,omitempty"`
	Addresses         []netip.Addr      `json:"-"`
	ReverseRecord     bool              `json:"reverseRecord"`
	UserDefinedFields map[string]string `json:"userDefinedFields,omitempty"`
}

func (HostRecordSpec) recordKind() RecordKind { return Host }

// MarshalJSON supplies the record and address resource types.
func (s HostRecordSpec) MarshalJSON() ([]byte, error) {
	if err := validateRecord(s.Name, s.TTL); err != nil {
		return nil, err
	}
	addresses, err := addressValues(s.Addresses)
	if err != nil {
		return nil, err
	}
	type fields HostRecordSpec
	data, err := marshalSpec(string(Host), fields(s))
	if err != nil {
		return nil, err
	}
	var document bam.Document
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if err := document.Set("addresses", addresses); err != nil {
		return nil, err
	}
	return json.Marshal(document)
}

// ExternalHostRecordSpec creates an external hostname in a view's
// ExternalHostsZone. Name is the external host's fully qualified name.
type ExternalHostRecordSpec struct {
	Name              string            `json:"name"`
	TTL               *int64            `json:"ttl,omitempty"`
	Comment           string            `json:"comment,omitempty"`
	UserDefinedFields map[string]string `json:"userDefinedFields,omitempty"`
}

func (ExternalHostRecordSpec) recordKind() RecordKind { return External }

// MarshalJSON supplies the external host discriminator.
func (s ExternalHostRecordSpec) MarshalJSON() ([]byte, error) {
	if err := validateRecord(s.Name, s.TTL); err != nil {
		return nil, err
	}
	type fields ExternalHostRecordSpec
	return marshalSpec(string(External), fields(s))
}
