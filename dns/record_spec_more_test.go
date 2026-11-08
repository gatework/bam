package dns_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gatework/bam/dns"
)

func TestRecordSpecsSendV2SubtypeFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		spec dns.RecordSpec
		want string
	}{
		{"alias external", dns.AliasRecordSpec{Name: "www", LinkedRecord: dns.ExternalTarget("origin.example.net")}, `{"type":"AliasRecord","name":"www","linkedRecord":{"type":"ExternalHostRecord","absoluteName":"origin.example.net"}}`},
		{"mx zero priority", dns.MXRecordSpec{Name: "mail", LinkedRecord: dns.RecordTarget{ID: 42}}, `{"type":"MXRecord","name":"mail","linkedRecord":{"id":42},"priority":0}`},
		{"srv zero weight", dns.SRVRecordSpec{Name: "_https._tcp", LinkedRecord: dns.RecordTarget{ID: 42}, Port: 443}, `{"type":"SRVRecord","name":"_https._tcp","linkedRecord":{"id":42},"priority":0,"weight":0,"port":443}`},
		{"txt", dns.TXTRecordSpec{Name: "_acme-challenge", Text: "token"}, `{"type":"TXTRecord","name":"_acme-challenge","text":"token"}`},
		{"hinfo", dns.HINFORecordSpec{Name: "host", CPU: "amd64", OS: "linux"}, `{"type":"HINFORecord","name":"host","cpu":"amd64","os":"linux"}`},
		{"naptr", dns.NAPTRRecordSpec{Name: "host", Order: 1, Preference: 2, Flags: "s", Service: "SIP+D2U", Replacement: "_sip._udp.example.test"}, `{"type":"NAPTRRecord","name":"host","order":1,"preference":2,"flags":"s","service":"SIP+D2U","replacement":"_sip._udp.example.test"}`},
		{"generic ptr", dns.GenericRecordSpec{Name: "8", RecordType: "PTR", RData: "host.example.test."}, `{"type":"GenericRecord","name":"8","recordType":"PTR","rdata":"host.example.test."}`},
		{"external", dns.ExternalHostRecordSpec{Name: "origin.example.net"}, `{"type":"ExternalHostRecord","name":"origin.example.net"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/v2/zones/33/resourceRecords" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var body, expected map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				if err := json.Unmarshal([]byte(tt.want), &expected); err != nil {
					t.Error(err)
					return
				}
				assertJSON(t, body, expected)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = io.WriteString(w, `{"id":77,"type":`+string(body["type"])+`,"name":`+string(body["name"])+`}`)
			}))
			t.Cleanup(server.Close)
			record, err := dns.New(newClient(t, server)).Records(33).Create(context.Background(), tt.spec)
			if err != nil {
				t.Fatal(err)
			}
			if record.ID != 77 {
				t.Fatalf("created record = %#v", record)
			}
		})
	}
}

func TestRecordSpecRejectsInvalidTargetsAndTTL(t *testing.T) {
	t.Parallel()
	negativeTTL := int64(-1)
	tests := []struct {
		name string
		spec dns.RecordSpec
	}{
		{"missing target", dns.AliasRecordSpec{Name: "www"}},
		{"mx alias target", dns.MXRecordSpec{Name: "mail", LinkedRecord: dns.RecordTarget{ID: 42, Type: dns.Alias}}},
		{"srv negative priority", dns.SRVRecordSpec{Name: "service", LinkedRecord: dns.RecordTarget{ID: 42}, Priority: -1}},
		{"negative ttl", dns.TXTRecordSpec{Name: "txt", Text: "text", TTL: &negativeTTL}},
		{"empty name", dns.TXTRecordSpec{Text: "text"}},
		{"missing rdata", dns.GenericRecordSpec{Name: "host", RecordType: "PTR"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := json.Marshal(tt.spec); err == nil {
				t.Fatal("invalid specification succeeded")
			}
		})
	}
}

func assertJSON(t *testing.T, actual, expected map[string]json.RawMessage) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("fields = %v, want %v", actual, expected)
	}
	for key, value := range expected {
		var got, want any
		if err := json.Unmarshal(actual[key], &got); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(value, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %#v, want %#v", key, got, want)
		}
	}
}
