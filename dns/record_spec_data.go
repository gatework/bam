package dns

import (
	"fmt"
	"strings"
)

// TXTRecordSpec creates a text record.
type TXTRecordSpec struct {
	Name              string            `json:"name"`
	TTL               *int64            `json:"ttl,omitempty"`
	Comment           string            `json:"comment,omitempty"`
	Text              string            `json:"text"`
	UserDefinedFields map[string]string `json:"userDefinedFields,omitempty"`
}

// MarshalJSON supplies the TXT discriminator.
func (s TXTRecordSpec) MarshalJSON() ([]byte, error) {
	if err := validateRecord(s.Name, s.TTL); err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.Text) == "" {
		return nil, fmt.Errorf("dns: txt text is required")
	}
	type fields TXTRecordSpec
	return marshalSpec(string(TXT), fields(s))
}

// HINFORecordSpec creates host information.
type HINFORecordSpec struct {
	Name              string            `json:"name"`
	TTL               *int64            `json:"ttl,omitempty"`
	Comment           string            `json:"comment,omitempty"`
	CPU               string            `json:"cpu,omitempty"`
	OS                string            `json:"os,omitempty"`
	UserDefinedFields map[string]string `json:"userDefinedFields,omitempty"`
}

// MarshalJSON supplies the HINFO discriminator.
func (s HINFORecordSpec) MarshalJSON() ([]byte, error) {
	if err := validateRecord(s.Name, s.TTL); err != nil {
		return nil, err
	}
	type fields HINFORecordSpec
	return marshalSpec(string(HINFO), fields(s))
}

// NAPTRRecordSpec creates a naming authority pointer.
type NAPTRRecordSpec struct {
	Name              string            `json:"name"`
	TTL               *int64            `json:"ttl,omitempty"`
	Comment           string            `json:"comment,omitempty"`
	Order             uint16            `json:"order"`
	Preference        uint16            `json:"preference"`
	Service           string            `json:"service,omitempty"`
	RegularExpression string            `json:"regularExpression,omitempty"`
	Replacement       string            `json:"replacement,omitempty"`
	Flags             string            `json:"flags,omitempty"`
	UserDefinedFields map[string]string `json:"userDefinedFields,omitempty"`
}

// MarshalJSON supplies the NAPTR discriminator.
func (s NAPTRRecordSpec) MarshalJSON() ([]byte, error) {
	if err := validateRecord(s.Name, s.TTL); err != nil {
		return nil, err
	}
	type fields NAPTRRecordSpec
	return marshalSpec(string(NAPTR), fields(s))
}

// GenericRecordSpec creates a DNS RR using BAM's generic-record resource.
// RecordType is the DNS mnemonic, such as PTR, CAA, NS or TLSA; RData is the DNS
// presentation format for that type, validated by BAM.
type GenericRecordSpec struct {
	Name              string            `json:"name"`
	TTL               *int64            `json:"ttl,omitempty"`
	Comment           string            `json:"comment,omitempty"`
	RecordType        string            `json:"recordType"`
	RData             string            `json:"rdata"`
	UserDefinedFields map[string]string `json:"userDefinedFields,omitempty"`
}

// MarshalJSON supplies the generic discriminator without inferring RDATA.
func (s GenericRecordSpec) MarshalJSON() ([]byte, error) {
	if err := validateRecord(s.Name, s.TTL); err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.RecordType) == "" || strings.TrimSpace(s.RData) == "" {
		return nil, fmt.Errorf("dns: generic record type and rdata are required")
	}
	type fields GenericRecordSpec
	return marshalSpec(string(Generic), fields(s))
}

func (TXTRecordSpec) recordKind() RecordKind { return TXT }

func (HINFORecordSpec) recordKind() RecordKind { return HINFO }

func (NAPTRRecordSpec) recordKind() RecordKind { return NAPTR }

func (GenericRecordSpec) recordKind() RecordKind { return Generic }
