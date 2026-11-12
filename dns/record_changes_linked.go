package dns

import (
	"fmt"

	"github.com/gatework/bam"
)

// AliasChanges changes a CNAME target or common record metadata.
type AliasChanges struct {
	MetadataChanges MetadataChanges
	LinkedRecord    bam.Field[RecordTarget]
}

// Apply verifies the subtype before changing its fields.
func (c AliasChanges) Apply(d bam.Document) error {
	if err := expectRecord(d, Alias); err != nil {
		return err
	}
	if err := applyTarget(d, c.LinkedRecord, false); err != nil {
		return err
	}
	return c.MetadataChanges.Apply(d)
}

// MXChanges changes a mail exchanger.
type MXChanges struct {
	MetadataChanges MetadataChanges
	LinkedRecord    bam.Field[RecordTarget]
	Priority        bam.Field[int32]
}

// Apply verifies the subtype before changing its fields.
func (c MXChanges) Apply(d bam.Document) error {
	if err := expectRecord(d, MX); err != nil {
		return err
	}
	if err := nonnegative(c.Priority, "priority"); err != nil {
		return err
	}
	if err := applyTarget(d, c.LinkedRecord, true); err != nil {
		return err
	}
	if err := c.MetadataChanges.Apply(d); err != nil {
		return err
	}
	return c.Priority.Apply(d, "priority")
}

// SRVChanges changes a service endpoint.
type SRVChanges struct {
	MetadataChanges MetadataChanges
	LinkedRecord    bam.Field[RecordTarget]
	Priority        bam.Field[int32]
	Weight          bam.Field[int32]
	Port            bam.Field[uint16]
}

// Apply verifies the subtype before changing its fields.
func (c SRVChanges) Apply(d bam.Document) error {
	if err := expectRecord(d, SRV); err != nil {
		return err
	}
	if err := nonnegative(c.Priority, "priority"); err != nil {
		return err
	}
	if err := nonnegative(c.Weight, "weight"); err != nil {
		return err
	}
	if c.Port.IsNull() {
		return fmt.Errorf("dns: port cannot be null")
	}
	if err := applyTarget(d, c.LinkedRecord, true); err != nil {
		return err
	}
	if err := c.MetadataChanges.Apply(d); err != nil {
		return err
	}
	return applyFields(c.Priority.Apply(d, "priority"), c.Weight.Apply(d, "weight"), c.Port.Apply(d, "port"))
}

func (AliasChanges) recordChanges() {}

func (MXChanges) recordChanges() {}

func (SRVChanges) recordChanges() {}
