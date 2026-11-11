package dns

import (
	"encoding/json"
	"net/netip"

	"github.com/gatework/bam"
)

// RecordKind is BAM's resource discriminator, rather than a DNS wire RR type.
// A HostRecord can produce A, AAAA and reverse PTR records.
type RecordKind string

// Supported DNS record resource kinds.
const (
	Host     RecordKind = "HostRecord"
	Alias    RecordKind = "AliasRecord"
	MX       RecordKind = "MXRecord"
	TXT      RecordKind = "TXTRecord"
	SRV      RecordKind = "SRVRecord"
	HINFO    RecordKind = "HINFORecord"
	NAPTR    RecordKind = "NAPTRRecord"
	Generic  RecordKind = "GenericRecord"
	External RecordKind = "ExternalHostRecord"
	HTTPS    RecordKind = "HTTPSRecord"
	SVCB     RecordKind = "SVCBRecord"
	URI      RecordKind = "URIRecord"
)

// Record contains common record metadata and a subtype-specific Details value.
// Use a type switch on Details when processing mixed record collections.
// Read projections are not full replacement bodies.
type Record struct {
	bam.Resource
	Configuration     bam.Reference     `json:"configuration"`
	AbsoluteName      string            `json:"absoluteName"`
	TTL               *int64            `json:"ttl"`
	Comment           string            `json:"comment"`
	Dynamic           bool              `json:"dynamic"`
	UserDefinedFields map[string]string `json:"userDefinedFields"`
	Details           RecordDetails     `json:"-"`
}

// RecordDetails is the payload of one record subtype. New server subtypes use
// UnknownDetails until their write schema has been verified.
type RecordDetails interface{ recordDetails() }

// UnmarshalJSON decodes the discriminator before selecting its payload.
func (r *Record) UnmarshalJSON(data []byte) error {
	type metadata Record
	var result metadata
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	var detail RecordDetails
	switch RecordKind(result.Type) {
	case Host:
		detail = new(HostDetails)
	case Alias:
		detail = new(AliasDetails)
	case MX:
		detail = new(MXDetails)
	case TXT:
		detail = new(TXTDetails)
	case SRV:
		detail = new(SRVDetails)
	case HINFO:
		detail = new(HINFODetails)
	case NAPTR:
		detail = new(NAPTRDetails)
	case Generic:
		detail = new(GenericDetails)
	case External:
		detail = new(ExternalHostDetails)
	default:
		var fields bam.Document
		if err := json.Unmarshal(data, &fields); err != nil {
			return err
		}
		detail = &UnknownDetails{Fields: fields}
	}
	if _, unknown := detail.(*UnknownDetails); !unknown {
		if err := json.Unmarshal(data, detail); err != nil {
			return err
		}
	}
	result.Details = detail
	*r = Record(result)
	return nil
}

// Address is an IP address associated with a host record.
type Address struct {
	bam.Resource
	Address netip.Addr `json:"address"`
}

// RecordTarget identifies the target of an alias, MX or SRV record by ID or
// absolute name. Set Type to External for an external host in BAM 9.6.
type RecordTarget struct {
	ID           int64      `json:"id,omitempty"`
	Type         RecordKind `json:"type,omitempty"`
	AbsoluteName string     `json:"absoluteName,omitempty"`
}

// ExternalTarget names a target outside the managed DNS namespace.
func ExternalTarget(name string) RecordTarget {
	return RecordTarget{Type: External, AbsoluteName: name}
}
