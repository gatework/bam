package dns

import (
	"encoding/json"

	"github.com/gatework/bam"
)

// HostDetails contains a host's PTR setting and embedded addresses.
// Addresses is populated when the response includes embed(addresses). Use
// Client.Addresses for a complete independently paginated association query.
type HostDetails struct {
	ReverseRecord bool      `json:"reverseRecord"`
	Addresses     []Address `json:"-"`
}

// UnmarshalJSON reads addresses from the documented HAL embedding.
func (h *HostDetails) UnmarshalJSON(data []byte) error {
	var wire struct {
		ReverseRecord bool `json:"reverseRecord"`
		Embedded      struct {
			Addresses []Address `json:"addresses"`
		} `json:"_embedded"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*h = HostDetails{ReverseRecord: wire.ReverseRecord, Addresses: wire.Embedded.Addresses}
	return nil
}

// AliasDetails contains the CNAME target.
type AliasDetails struct {
	LinkedRecord RecordTarget `json:"linkedRecord"`
}

func (*AliasDetails) recordDetails() {}

// MXDetails describes a mail exchanger.
type MXDetails struct {
	LinkedRecord RecordTarget `json:"linkedRecord"`
	Priority     int32        `json:"priority"`
}

func (*MXDetails) recordDetails() {}

// TXTDetails contains the record's text.
type TXTDetails struct {
	Text string `json:"text"`
}

func (*TXTDetails) recordDetails() {}

// SRVDetails describes a service endpoint.
type SRVDetails struct {
	LinkedRecord RecordTarget `json:"linkedRecord"`
	Priority     int32        `json:"priority"`
	Weight       int32        `json:"weight"`
	Port         uint16       `json:"port"`
}

func (*SRVDetails) recordDetails() {}

// HINFODetails describes a host's CPU and operating system.
type HINFODetails struct {
	CPU string `json:"cpu"`
	OS  string `json:"os"`
}

func (*HINFODetails) recordDetails() {}

// NAPTRDetails describes a naming authority pointer.
type NAPTRDetails struct {
	Order             uint16 `json:"order"`
	Preference        uint16 `json:"preference"`
	Service           string `json:"service"`
	RegularExpression string `json:"regularExpression"`
	Replacement       string `json:"replacement"`
	Flags             string `json:"flags"`
}

func (*NAPTRDetails) recordDetails() {}

// GenericDetails retains the DNS RR type and its presentation-format RDATA.
type GenericDetails struct {
	RecordType string `json:"recordType"`
	RData      string `json:"rdata"`
}

func (*GenericDetails) recordDetails() {}

// ExternalHostDetails is the payload of an external host reference.
type ExternalHostDetails struct{}

func (*ExternalHostDetails) recordDetails() {}

// UnknownDetails preserves fields of a server subtype without guessing its
// request schema. This includes HTTPS/SVCB/URI until a 9.6 schema is available.
type UnknownDetails struct{ Fields bam.Document }

func (*UnknownDetails) recordDetails() {}

func (*HostDetails) recordDetails() {}
