package dns

import "fmt"

// AliasRecordSpec creates a CNAME linked to a host, alias or external host.
type AliasRecordSpec struct {
	Name              string            `json:"name"`
	TTL               *int64            `json:"ttl,omitempty"`
	Comment           string            `json:"comment,omitempty"`
	LinkedRecord      RecordTarget      `json:"linkedRecord"`
	UserDefinedFields map[string]string `json:"userDefinedFields,omitempty"`
}

func (AliasRecordSpec) recordKind() RecordKind { return Alias }

// MarshalJSON validates the target and supplies the discriminator.
func (s AliasRecordSpec) MarshalJSON() ([]byte, error) {
	if err := validateRecord(s.Name, s.TTL); err != nil {
		return nil, err
	}
	if err := s.LinkedRecord.validate(false); err != nil {
		return nil, err
	}
	type fields AliasRecordSpec
	return marshalSpec(string(Alias), fields(s))
}

// MXRecordSpec creates a mail exchanger. Priority zero is sent explicitly.
type MXRecordSpec struct {
	Name              string            `json:"name"`
	TTL               *int64            `json:"ttl,omitempty"`
	Comment           string            `json:"comment,omitempty"`
	LinkedRecord      RecordTarget      `json:"linkedRecord"`
	Priority          int32             `json:"priority"`
	UserDefinedFields map[string]string `json:"userDefinedFields,omitempty"`
}

func (MXRecordSpec) recordKind() RecordKind { return MX }

// MarshalJSON validates a mail exchanger and supplies the discriminator.
func (s MXRecordSpec) MarshalJSON() ([]byte, error) {
	if err := validateRecord(s.Name, s.TTL); err != nil {
		return nil, err
	}
	if err := s.LinkedRecord.validate(true); err != nil {
		return nil, err
	}
	if s.Priority < 0 {
		return nil, fmt.Errorf("dns: mx priority cannot be negative")
	}
	type fields MXRecordSpec
	return marshalSpec(string(MX), fields(s))
}

// SRVRecordSpec creates a service endpoint.
type SRVRecordSpec struct {
	Name              string            `json:"name"`
	TTL               *int64            `json:"ttl,omitempty"`
	Comment           string            `json:"comment,omitempty"`
	LinkedRecord      RecordTarget      `json:"linkedRecord"`
	Priority          int32             `json:"priority"`
	Weight            int32             `json:"weight"`
	Port              uint16            `json:"port"`
	UserDefinedFields map[string]string `json:"userDefinedFields,omitempty"`
}

func (SRVRecordSpec) recordKind() RecordKind { return SRV }

// MarshalJSON validates a service endpoint and supplies the discriminator.
func (s SRVRecordSpec) MarshalJSON() ([]byte, error) {
	if err := validateRecord(s.Name, s.TTL); err != nil {
		return nil, err
	}
	if err := s.LinkedRecord.validate(true); err != nil {
		return nil, err
	}
	if s.Priority < 0 || s.Weight < 0 {
		return nil, fmt.Errorf("dns: srv priority and weight cannot be negative")
	}
	type fields SRVRecordSpec
	return marshalSpec(string(SRV), fields(s))
}
