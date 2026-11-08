package dns_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gatework/bam/dns"
)

func TestRecordDecodesSubtypeDetails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    string
		details dns.RecordDetails
	}{
		{"alias", `{"type":"AliasRecord","linkedRecord":{"id":42,"type":"HostRecord","absoluteName":"www.example.test"}}`, &dns.AliasDetails{LinkedRecord: dns.RecordTarget{ID: 42, Type: dns.Host, AbsoluteName: "www.example.test"}}},
		{"mx", `{"type":"MXRecord","linkedRecord":{"id":42},"priority":0}`, &dns.MXDetails{LinkedRecord: dns.RecordTarget{ID: 42}, Priority: 0}},
		{"txt", `{"type":"TXTRecord","text":"some text"}`, &dns.TXTDetails{Text: "some text"}},
		{"srv", `{"type":"SRVRecord","linkedRecord":{"id":42},"priority":10,"weight":0,"port":443}`, &dns.SRVDetails{LinkedRecord: dns.RecordTarget{ID: 42}, Priority: 10, Port: 443}},
		{"hinfo", `{"type":"HINFORecord","cpu":"amd64","os":"linux"}`, &dns.HINFODetails{CPU: "amd64", OS: "linux"}},
		{"naptr", `{"type":"NAPTRRecord","order":1,"preference":2,"flags":"s","service":"SIP+D2U","replacement":"_sip._udp.example.test"}`, &dns.NAPTRDetails{Order: 1, Preference: 2, Flags: "s", Service: "SIP+D2U", Replacement: "_sip._udp.example.test"}},
		{"generic ptr", `{"type":"GenericRecord","recordType":"PTR","rdata":"host.example.test."}`, &dns.GenericDetails{RecordType: "PTR", RData: "host.example.test."}},
		{"external", `{"type":"ExternalHostRecord"}`, &dns.ExternalHostDetails{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var record dns.Record
			if err := json.Unmarshal([]byte(tt.body), &record); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(record.Details, tt.details) {
				t.Fatalf("details = %#v, want %#v", record.Details, tt.details)
			}
		})
	}
}

func TestRecordRetainsUnmodeledSubtype(t *testing.T) {
	t.Parallel()
	var record dns.Record
	if err := json.Unmarshal([]byte(`{"id":42,"type":"HTTPSRecord","name":"www","vendorField":{"integer":9007199254740993,"nullable":null}}`), &record); err != nil {
		t.Fatal(err)
	}
	details, ok := record.Details.(*dns.UnknownDetails)
	if !ok || record.Type != "HTTPSRecord" || string(details.Fields["vendorField"]) != `{"integer":9007199254740993,"nullable":null}` {
		t.Fatalf("unknown details were not retained: %#v", record)
	}
	if err := json.Unmarshal([]byte(`{"id":43,"type":"TXTRecord","text":"changed"}`), &record); err != nil {
		t.Fatal(err)
	}
	if _, ok := record.Details.(*dns.TXTDetails); !ok {
		t.Fatalf("stale details: %#v", record.Details)
	}
}
