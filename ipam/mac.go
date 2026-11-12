package ipam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// MAC is an immutable, comparable 48-bit hardware address. Its zero value is
// unset; ParseMAC accepts colon, hyphen, and Cisco dotted notation.
type MAC struct {
	octets  [6]byte
	isValid bool
}

// ParseMAC parses an EUI-48 address, rejecting EUI-64 and other hardware sizes.
func ParseMAC(value string) (MAC, error) {
	address, err := net.ParseMAC(value)
	if err != nil || len(address) != 6 {
		return MAC{}, fmt.Errorf("ipam: expected a 48-bit mac address")
	}
	var result MAC
	copy(result.octets[:], address)
	result.isValid = true
	return result, nil
}

// IsValid reports whether this value was parsed successfully.
func (m MAC) IsValid() bool { return m.isValid }

// String returns BAM's canonical uppercase, hyphen-separated representation.
func (m MAC) String() string {
	if !m.isValid {
		return ""
	}
	return strings.ToUpper(strings.ReplaceAll(net.HardwareAddr(m.octets[:]).String(), ":", "-"))
}

// MarshalText encodes the canonical MAC address text.
func (m MAC) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText parses and validates the MAC address text.
func (m *MAC) UnmarshalText(text []byte) error {
	address, err := ParseMAC(string(text))
	if err != nil {
		return err
	}
	*m = address
	return nil
}

// MACAddress is a managed MAC resource. Pool membership is separate from the
// lifetime of this resource; DHCP exposes explicit pool linking operations.
type MACAddress struct {
	ID            int64          `json:"id"`
	Type          string         `json:"type"`
	Name          string         `json:"name"`
	Configuration bam.Reference  `json:"configuration"`
	Address       MAC            `json:"address"`
	MACPool       *bam.Reference `json:"macPool"`
}

// MACReference associates an IP with an existing MAC ID or a hardware address.
// Supply exactly one of ID and Address. PoolID optionally assigns a new MAC to
// a MAC pool when the address allocation creates the MAC resource.
type MACReference struct {
	ID      int64
	Address MAC
	PoolID  int64
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (m MACReference) MarshalJSON() ([]byte, error) {
	if (m.ID > 0) == m.Address.IsValid() || m.ID < 0 || m.PoolID < 0 {
		return nil, fmt.Errorf("ipam: mac reference requires either a positive id or an address")
	}
	if m.ID > 0 {
		if m.PoolID != 0 {
			return nil, fmt.Errorf("ipam: pool assignment requires a mac address, not an existing id")
		}
		return json.Marshal(struct {
			Type string `json:"type"`
			ID   int64  `json:"id"`
		}{"MACAddress", m.ID})
	}
	var pool *bam.Reference
	if m.PoolID > 0 {
		pool = &bam.Reference{ID: m.PoolID}
	}
	return json.Marshal(struct {
		Type    string         `json:"type"`
		Address MAC            `json:"address"`
		Pool    *bam.Reference `json:"macPool,omitempty"`
	}{"MACAddress", m.Address, pool})
}

// MACSpec creates a managed hardware address within a configuration.
type MACSpec struct {
	Name    string         `json:"name,omitempty"`
	Address MAC            `json:"address"`
	MACPool *bam.Reference `json:"macPool,omitempty"`
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s MACSpec) MarshalJSON() ([]byte, error) {
	if !s.Address.IsValid() {
		return nil, fmt.Errorf("ipam: mac address is required")
	}
	if err := validReference(s.MACPool); err != nil {
		return nil, err
	}
	type spec MACSpec
	return json.Marshal(struct {
		spec
		Type string `json:"type"`
	}{spec: spec(s), Type: "MACAddress"})
}

// MACChanges modifies a MAC resource, preserving any unmodeled fields.
type MACChanges struct {
	Name    bam.Field[string]
	Address bam.Field[MAC]
}

// Apply validates and applies explicitly set fields to the complete resource document.
func (c MACChanges) Apply(d bam.Document) error {
	if c.Address.IsNull() {
		return fmt.Errorf("ipam: mac address cannot be cleared")
	}
	if address, ok := c.Address.Value(); ok && !address.IsValid() {
		return fmt.Errorf("ipam: mac address is required")
	}
	return errors.Join(c.Name.Apply(d, "name"), c.Address.Apply(d, "address"))
}

// MACAddresses manages hardware addresses in a configuration.
type MACAddresses struct {
	resources resource.Collection[MACAddress, MACSpec, MACChanges]
}

// MACItems manages existing MAC addresses across configurations.
type MACItems struct {
	resources resource.Items[MACAddress, MACChanges]
}

// AddressLinks lists addresses associated with a MAC; it cannot assign or delete them.
type AddressLinks struct{ resources resource.List[Address] }

// Page fetches one page within the selected collection scope.
func (m MACAddresses) Page(ctx context.Context, options bam.ListOptions) (bam.Page[MACAddress], error) {
	return m.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (m MACAddresses) All(ctx context.Context, options bam.ListOptions) iter.Seq2[MACAddress, error] {
	return m.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (m MACAddresses) Get(ctx context.Context, id int64) (MACAddress, error) {
	return m.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (m MACAddresses) Document(ctx context.Context, id int64) (bam.Document, error) {
	return m.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (m MACAddresses) Update(ctx context.Context, id int64, changes MACChanges, options ...bam.WriteOption) (MACAddress, error) {
	return m.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (m MACAddresses) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return m.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (m MACAddresses) Create(ctx context.Context, spec MACSpec, options ...bam.WriteOption) (MACAddress, error) {
	return m.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (m MACItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[MACAddress], error) {
	return m.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (m MACItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[MACAddress, error] {
	return m.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (m MACItems) Get(ctx context.Context, id int64) (MACAddress, error) {
	return m.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (m MACItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return m.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (m MACItems) Update(ctx context.Context, id int64, changes MACChanges, options ...bam.WriteOption) (MACAddress, error) {
	return m.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (m MACItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return m.resources.Delete(ctx, id, options...)
}

// Page fetches one page within the selected collection scope.
func (a AddressLinks) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Address], error) {
	return a.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (a AddressLinks) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Address, error] {
	return a.resources.All(ctx, options)
}
