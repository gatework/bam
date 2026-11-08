package dns

import (
	"fmt"
	"strings"

	"github.com/gatework/bam"
)

// TXTChanges changes text and common metadata.
type TXTChanges struct {
	MetadataChanges
	Text bam.Field[string]
}

// Apply verifies the subtype before changing its fields.
func (c TXTChanges) Apply(d bam.Document) error {
	if err := expectRecord(d, TXT); err != nil {
		return err
	}
	if err := nonempty(c.Text, "text"); err != nil {
		return err
	}
	if err := c.MetadataChanges.Apply(d); err != nil {
		return err
	}
	return c.Text.Apply(d, "text")
}

// HINFOChanges changes host CPU and operating system data.
type HINFOChanges struct {
	MetadataChanges
	CPU bam.Field[string]
	OS  bam.Field[string]
}

// Apply verifies the subtype before changing its fields.
func (c HINFOChanges) Apply(d bam.Document) error {
	if err := expectRecord(d, HINFO); err != nil {
		return err
	}
	if err := c.MetadataChanges.Apply(d); err != nil {
		return err
	}
	return applyFields(c.CPU.Apply(d, "cpu"), c.OS.Apply(d, "os"))
}

// NAPTRChanges changes a naming authority pointer.
type NAPTRChanges struct {
	MetadataChanges
	Order             bam.Field[uint16]
	Preference        bam.Field[uint16]
	Service           bam.Field[string]
	RegularExpression bam.Field[string]
	Replacement       bam.Field[string]
	Flags             bam.Field[string]
}

// Apply verifies the subtype before changing its fields.
func (c NAPTRChanges) Apply(d bam.Document) error {
	if err := expectRecord(d, NAPTR); err != nil {
		return err
	}
	if c.Order.IsNull() || c.Preference.IsNull() {
		return fmt.Errorf("dns: naptr order and preference cannot be null")
	}
	if err := c.MetadataChanges.Apply(d); err != nil {
		return err
	}
	return applyFields(
		c.Order.Apply(d, "order"), c.Preference.Apply(d, "preference"),
		c.Service.Apply(d, "service"), c.RegularExpression.Apply(d, "regularExpression"),
		c.Replacement.Apply(d, "replacement"), c.Flags.Apply(d, "flags"),
	)
}

// GenericChanges changes the DNS type or presentation-format RDATA.
type GenericChanges struct {
	MetadataChanges
	RecordType bam.Field[string]
	RData      bam.Field[string]
}

// Apply verifies the subtype before changing its fields.
func (c GenericChanges) Apply(d bam.Document) error {
	if err := expectRecord(d, Generic); err != nil {
		return err
	}
	if err := nonempty(c.RecordType, "record type"); err != nil {
		return err
	}
	if err := nonempty(c.RData, "rdata"); err != nil {
		return err
	}
	if err := c.MetadataChanges.Apply(d); err != nil {
		return err
	}
	return applyFields(c.RecordType.Apply(d, "recordType"), c.RData.Apply(d, "rdata"))
}

func nonempty(field bam.Field[string], name string) error {
	if field.IsNull() {
		return fmt.Errorf("dns: %s cannot be null", name)
	}
	if value, ok := field.Value(); ok && strings.TrimSpace(value) == "" {
		return fmt.Errorf("dns: %s cannot be empty", name)
	}
	return nil
}
