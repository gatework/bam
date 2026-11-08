package dns_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dns"
)

func TestRecordQueryKeepsScopeAndRejectsAmbiguousLookup(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/resourceRecords" {
			t.Errorf("path = %s", r.URL.Path)
		}
		filter := r.URL.Query().Get("filter")
		for _, clause := range []string{`configuration.id:eq(11)`, `absoluteName:eq("www.example.test")`, `type:eq("AliasRecord")`, `linkedRecord.id:eq(42)`} {
			if !strings.Contains(filter, clause) {
				t.Errorf("filter %q missing %q", filter, clause)
			}
		}
		if r.URL.Query().Get("limit") != "2" {
			t.Errorf("lookup limit = %s", r.URL.Query().Get("limit"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"count":2,"data":[{"id":77,"type":"AliasRecord"},{"id":78,"type":"AliasRecord"}]}`)
	}))
	t.Cleanup(server.Close)
	_, err := dns.New(newClient(t, server)).GlobalRecords().Find(context.Background(), dns.RecordQuery{
		ConfigurationID: 11, AbsoluteName: "www.example.test", Kind: dns.Alias, LinkedRecordID: 42,
	})
	if !errors.Is(err, bam.ErrAmbiguous) {
		t.Fatalf("error = %v, want ambiguous lookup", err)
	}
}

func TestRecordsByIPFollowsAddressLinksAndPreservesQueryInput(t *testing.T) {
	t.Parallel()
	requests := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v2/addresses" {
			filter := r.URL.Query().Get("filter")
			if !strings.Contains(filter, `configuration.id:eq(11)`) || !strings.Contains(filter, `address:eq("192.0.2.8")`) {
				t.Errorf("address filter = %s", filter)
			}
			_, _ = io.WriteString(w, `{"count":2,"data":[{"id":88,"type":"IPv4Address","address":"192.0.2.8"},{"id":89,"type":"IPv4Address","address":"192.0.2.8"}]}`)
			return
		}
		if r.URL.Path != "/api/v2/addresses/88/resourceRecords" && r.URL.Path != "/api/v2/addresses/89/resourceRecords" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("fields") != "name,id" {
			t.Errorf("fields = %s", r.URL.Query().Get("fields"))
		}
		_, _ = io.WriteString(w, `{"count":1,"data":[{"id":77,"name":"host"}]}`)
	}))
	t.Cleanup(server.Close)
	fields := make([]string, 1, 3)
	fields[0] = "name"
	var ids []int64
	for record, err := range dns.New(newClient(t, server)).RecordsByIP(
		context.Background(), 11, netip.MustParseAddr("192.0.2.8"), bam.ListOptions{Fields: fields}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, record.ID)
	}
	if !reflect.DeepEqual(ids, []int64{77}) {
		t.Fatalf("ids = %v", ids)
	}
	if fields[:cap(fields)][1] != "" {
		t.Fatal("caller's fields backing array was mutated")
	}
	for _, want := range []string{"GET /api/v2/addresses", "GET /api/v2/addresses/88/resourceRecords", "GET /api/v2/addresses/89/resourceRecords"} {
		if got := <-requests; got != want {
			t.Errorf("request = %s, want %s", got, want)
		}
	}
}

func TestRecordsByIPStopsWhenCallerStops(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v2/addresses" {
			_, _ = io.WriteString(w, `{"count":1,"data":[{"id":88,"type":"IPv4Address","address":"192.0.2.8"}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"count":1,"data":[{"id":77,"type":"HostRecord"}],"_links":{"next":{"href":"?offset=1"}}}`)
	}))
	t.Cleanup(server.Close)
	for _, err := range dns.New(newClient(t, server)).RecordsByIP(context.Background(), 11, netip.MustParseAddr("192.0.2.8"), bam.ListOptions{}) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if calls.Load() != 2 {
		t.Fatalf("requests = %d, want 2", calls.Load())
	}
}

func TestRecordsByIPRejectsMissingAssociationID(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v2/addresses" {
			_, _ = io.WriteString(w, `{"count":1,"data":[{"id":88,"type":"IPv4Address","address":"192.0.2.8"}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"count":1,"data":[{"type":"HostRecord","name":"host"}]}`)
	}))
	t.Cleanup(server.Close)
	var foundError bool
	for record, err := range dns.New(newClient(t, server)).RecordsByIP(context.Background(), 11, netip.MustParseAddr("192.0.2.8"), bam.ListOptions{}) {
		if err == nil {
			t.Fatalf("invalid record association accepted: %#v", record)
		}
		foundError = true
	}
	if !foundError {
		t.Fatal("missing id was silently discarded")
	}
}
