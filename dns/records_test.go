package dns_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dns"
)

func TestHostUpdatePreservesUnknownFieldsAndAllAddressLinks(t *testing.T) {
	t.Parallel()
	var reads, writes atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/resourceRecords/77":
			_, _ = io.WriteString(w, `{"id":77,"type":"HostRecord","name":"old","reverseRecord":true,"ttl":600,"opaque":{"number":9007199254740993,"enabled":false,"nullable":null},"_links":{"self":{"href":"/api/v2/resourceRecords/77"}}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/resourceRecords/77/addresses":
			reads.Add(1)
			if r.URL.Query().Get("offset") == "1" {
				_, _ = io.WriteString(w, `{"count":1,"data":[{"id":89,"type":"IPv6Address","address":"2001:db8::8"}]}`)
			} else {
				_, _ = io.WriteString(w, `{"count":1,"data":[{"id":88,"type":"IPv4Address","address":"192.0.2.8"}],"_links":{"next":{"href":"?offset=1"}}}`)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/resourceRecords/77":
			writes.Add(1)
			var body bam.Document
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			if string(body["opaque"]) != `{"enabled":false,"nullable":null,"number":9007199254740993}` &&
				string(body["opaque"]) != `{"number":9007199254740993,"enabled":false,"nullable":null}` {
				t.Errorf("unknown fields changed: %s", body["opaque"])
			}
			if string(body["name"]) != `"new"` || string(body["ttl"]) != "0" || string(body["reverseRecord"]) != "true" {
				t.Errorf("updated fields = %v", body)
			}
			if _, ok := body["_links"]; ok {
				t.Error("HAL metadata sent in PUT")
			}
			if string(body["addresses"]) != `[{"id":88,"type":"IPv4Address"},{"id":89,"type":"IPv6Address"}]` {
				t.Errorf("address links = %s", body["addresses"])
			}
			if r.Header.Get("X-Bcn-Update-Servers") != "false" || r.Header.Get("X-Bcn-Change-Control-Comment") != "rename" {
				t.Errorf("write options lost: %v", r.Header)
			}
			_ = json.NewEncoder(w).Encode(body)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	record, err := dns.New(newClient(t, server)).GlobalRecords().Update(context.Background(), 77,
		dns.MetadataChanges{Name: bam.Set("new"), TTL: bam.Set(int64(0))},
		dns.WithServerUpdates(false), bam.WithChangeControlComment("rename"))
	if err != nil {
		t.Fatal(err)
	}
	if record.Name != "new" || reads.Load() != 2 || writes.Load() != 1 {
		t.Fatalf("record=%#v address reads=%d writes=%d", record, reads.Load(), writes.Load())
	}
}

func TestHostUpdateUsesExplicitReplacementAddresses(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v2/resourceRecords/77" {
			t.Errorf("unexpected association read: %s", r.URL)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `{"id":77,"type":"HostRecord","name":"host","reverseRecord":true,"ttl":600}`)
			return
		}
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body bam.Document
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if string(body["addresses"]) != `[{"type":"IPv4Address","address":"192.0.2.9"}]` || string(body["reverseRecord"]) != "false" || string(body["ttl"]) != "600" {
			t.Errorf("host changes = %v", body)
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)
	_, err := dns.New(newClient(t, server)).GlobalRecords().Update(context.Background(), 77, dns.HostChanges{
		Addresses:     bam.Set([]netip.Addr{netip.MustParseAddr("192.0.2.9")}),
		ReverseRecord: bam.Set(false),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRecordUpdateRejectsWrongSubtypeAndUnknownSchema(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		kind    string
		changes dns.RecordChanges
	}{
		{"wrong subtype", "TXTRecord", dns.MXChanges{Priority: bam.Set(int32(1))}},
		{"unverified schema", "HTTPSRecord", dns.MetadataChanges{Name: bam.Set("new")}},
		{"null ttl", "TXTRecord", dns.MetadataChanges{TTL: bam.Clear[int64]()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var writes atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					writes.Add(1)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "type": tt.kind, "name": "old"})
			}))
			t.Cleanup(server.Close)
			if _, err := dns.New(newClient(t, server)).GlobalRecords().Update(context.Background(), 77, tt.changes); err == nil {
				t.Fatal("invalid changes succeeded")
			}
			if writes.Load() != 0 {
				t.Fatal("invalid change reached a write endpoint")
			}
		})
	}
}

func TestRecordCreateReadUpdateDelete(t *testing.T) {
	t.Parallel()
	var stored bam.Document
	var sequence []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			if r.URL.Path != "/api/v2/zones/33/resourceRecords" {
				t.Errorf("create path = %s", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&stored); err != nil {
				t.Error(err)
				return
			}
			stored["id"] = json.RawMessage("77")
			stored["vendorField"] = json.RawMessage(`{"large":9007199254740993}`)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(stored)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(stored)
		case http.MethodPut:
			if err := json.NewDecoder(r.Body).Decode(&stored); err != nil {
				t.Error(err)
				return
			}
			if string(stored["vendorField"]) != `{"large":9007199254740993}` {
				t.Error("opaque field lost")
			}
			if string(stored["linkedRecord"]) != `{"type":"ExternalHostRecord","absoluteName":"new.example.net"}` {
				t.Errorf("target = %s", stored["linkedRecord"])
			}
			_ = json.NewEncoder(w).Encode(stored)
		case http.MethodDelete:
			if r.URL.Path != "/api/v2/resourceRecords/77" {
				t.Errorf("delete path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(server.Close)
	records := dns.New(newClient(t, server)).Records(33)
	ctx := context.Background()
	created, err := records.Create(ctx, dns.AliasRecordSpec{Name: "www", LinkedRecord: dns.RecordTarget{ID: 42}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := records.Get(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	updated, err := records.Update(ctx, created.ID, dns.AliasChanges{LinkedRecord: bam.Set(dns.ExternalTarget("new.example.net"))})
	if err != nil {
		t.Fatal(err)
	}
	if details, ok := updated.Details.(*dns.AliasDetails); !ok || details.LinkedRecord.AbsoluteName != "new.example.net" {
		t.Fatalf("updated alias = %#v", updated)
	}
	if err := records.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	want := []string{"POST /api/v2/zones/33/resourceRecords", "GET /api/v2/resourceRecords/77", "GET /api/v2/resourceRecords/77", "PUT /api/v2/resourceRecords/77", "DELETE /api/v2/resourceRecords/77"}
	if !reflect.DeepEqual(sequence, want) {
		t.Fatalf("requests = %v, want %v", sequence, want)
	}
}
