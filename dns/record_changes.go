package dns

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/gatework/bam"
)

// RecordChanges modifies one known record subtype without constructing a
// partial PUT body. MetadataChanges can update common fields on any modeled
// subtype; subtype-specific changes also verify the existing resource type.
type RecordChanges interface {
	Apply(bam.Document) error
	recordChanges()
}

// MetadataChanges modifies common record fields. TTL values are seconds.
// Set(0) is an explicit zero TTL; null TTL is not in the verified schema.
type MetadataChanges struct {
	Name              bam.Field[string]
	TTL               bam.Field[int64]
	Comment           bam.Field[string]
	UserDefinedFields bam.Field[map[string]string]
}

func (MetadataChanges) recordChanges() {}

// Apply changes common fields while preserving the rest of the document.
func (c MetadataChanges) Apply(d bam.Document) error {
	if err := requiredName(c.Name); err != nil {
		return err
	}
	if c.TTL.IsNull() {
		return fmt.Errorf("dns: ttl cannot be null in the verified schema")
	}
	if ttl, ok := c.TTL.Value(); ok && (ttl < 0 || ttl > 2147483647) {
		return fmt.Errorf("dns: ttl must be between 0 and 2147483647 seconds")
	}
	return applyFields(
		c.Name.Apply(d, "name"), c.TTL.Apply(d, "ttl"),
		c.Comment.Apply(d, "comment"), c.UserDefinedFields.Apply(d, "userDefinedFields"),
	)
}

// HostChanges replaces a host's address associations and/or PTR setting.
// An unset Addresses field preserves the complete existing association set.
// A host cannot have an empty address set.
type HostChanges struct {
	MetadataChanges
	Addresses     bam.Field[[]netip.Addr]
	ReverseRecord bam.Field[bool]
}

func (c HostChanges) changesAddresses() bool { return c.Addresses.IsSet() }

// Apply encodes address changes using BAM 9.6's top-level addresses field.
func (c HostChanges) Apply(d bam.Document) error {
	if err := expectRecord(d, Host); err != nil {
		return err
	}
	if c.Addresses.IsNull() || c.ReverseRecord.IsNull() {
		return fmt.Errorf("dns: host addresses and reverse record cannot be null")
	}
	if addresses, ok := c.Addresses.Value(); ok {
		values, err := addressValues(addresses)
		if err != nil {
			return err
		}
		if err := d.Set("addresses", values); err != nil {
			return err
		}
	}
	if err := c.MetadataChanges.Apply(d); err != nil {
		return err
	}
	return c.ReverseRecord.Apply(d, "reverseRecord")
}

func expectRecord(d bam.Document, expected RecordKind) error {
	var actual RecordKind
	if err := json.Unmarshal(d["type"], &actual); err != nil {
		return fmt.Errorf("dns: decode record type: %w", err)
	}
	if actual != expected {
		return fmt.Errorf("dns: changes require %s, got %s", expected, actual)
	}
	return nil
}

func applyTarget(d bam.Document, field bam.Field[RecordTarget], hostOnly bool) error {
	if field.IsNull() {
		return fmt.Errorf("dns: linked record cannot be null")
	}
	if target, ok := field.Value(); ok {
		if err := target.validate(hostOnly); err != nil {
			return err
		}
	}
	return field.Apply(d, "linkedRecord")
}

func nonnegative(field bam.Field[int32], name string) error {
	if field.IsNull() {
		return fmt.Errorf("dns: %s cannot be null", name)
	}
	if value, ok := field.Value(); ok && value < 0 {
		return fmt.Errorf("dns: %s cannot be negative", name)
	}
	return nil
}
