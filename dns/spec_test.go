package dns_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam/dns"
)

func TestHostRecordSpecCreatesDualStackRecord(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/zones/33/resourceRecords" {
			t.Errorf("request = %s %s, want POST /api/v2/zones/33/resourceRecords", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct {
			Type          string `json:"type"`
			Name          string `json:"name"`
			ReverseRecord *bool  `json:"reverseRecord"`
			Addresses     []struct {
				Type    string `json:"type"`
				Address string `json:"address"`
			} `json:"addresses"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body.Type != "HostRecord" || body.Name != "www" {
			t.Errorf("unexpected record body: %+v", body)
		}
		if body.ReverseRecord == nil || *body.ReverseRecord {
			t.Error("reverseRecord must be present and false")
		}
		if len(body.Addresses) != 3 {
			t.Errorf("address count = %d, want 3", len(body.Addresses))
		} else {
			if body.Addresses[0].Type != "IPv4Address" || body.Addresses[0].Address != "192.0.2.8" {
				t.Errorf("unexpected ipv4 address: %+v", body.Addresses[0])
			}
			if body.Addresses[1].Type != "IPv6Address" || body.Addresses[1].Address != "2001:db8::8" {
				t.Errorf("unexpected ipv6 address: %+v", body.Addresses[1])
			}
			if body.Addresses[2].Type != "IPv6Address" || body.Addresses[2].Address != "::ffff:192.0.2.9" {
				t.Errorf("unexpected mapped ipv6 address: %+v", body.Addresses[2])
			}
		}
		w.Header().Set("Content-Type", "application/hal+json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":44,"type":"HostRecord","name":"www","absoluteName":"www.example.test","reverseRecord":false}`))
	}))
	t.Cleanup(server.Close)

	record, err := dns.New(newClient(t, server)).Records(33).Create(context.Background(), dns.HostRecordSpec{
		Name: "www",
		Addresses: []netip.Addr{
			netip.MustParseAddr("192.0.2.8"),
			netip.MustParseAddr("2001:db8::8"),
			netip.MustParseAddr("::ffff:192.0.2.9"),
		},
		ReverseRecord: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != 44 || record.Type != "HostRecord" || record.AbsoluteName != "www.example.test" {
		t.Fatalf("unexpected created record: %+v", record)
	}
}

func TestHostRecordSpecRejectsInvalidAddressBeforeSending(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		address netip.Addr
	}{
		{name: "zero address"},
		{name: "ipv6 scope", address: netip.MustParseAddr("fe80::1%en0")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "application/hal+json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":44,"type":"HostRecord","name":"www"}`))
			}))
			t.Cleanup(server.Close)
			_, err := dns.New(newClient(t, server)).Records(33).Create(context.Background(), dns.HostRecordSpec{
				Name:      "www",
				Addresses: []netip.Addr{tt.address},
			})
			if err == nil {
				t.Error("expected an address validation error")
			}
			if got := requests.Load(); got != 0 {
				t.Errorf("made %d network requests for an invalid address", got)
			}
		})
	}
}
