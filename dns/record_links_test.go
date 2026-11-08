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

func TestAddressRecordLinkAndUnlinkUseRelationshipURI(t *testing.T) {
	t.Parallel()
	requests := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			var target dns.RecordTarget
			if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
				t.Error(err)
				return
			}
			if target.ID != 77 || target.Type != dns.Host {
				t.Errorf("target = %#v", target)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":77,"type":"HostRecord","name":"host","reverseRecord":false}`)
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"id":77,"type":"HostRecord","name":"host","reverseRecord":false}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(server.Close)
	links := dns.New(newClient(t, server)).RecordsForAddress(88)
	ctx := context.Background()
	if _, err := links.Link(ctx, dns.RecordTarget{ID: 77, Type: dns.Host}); err != nil {
		t.Fatal(err)
	}
	if _, err := links.Get(ctx, 77); err != nil {
		t.Fatal(err)
	}
	if err := links.Unlink(ctx, 77); err != nil {
		t.Fatal(err)
	}
	var got []string
	for range 3 {
		got = append(got, <-requests)
	}
	want := []string{
		"POST /api/v2/addresses/88/resourceRecords",
		"GET /api/v2/addresses/88/resourceRecords/77",
		"DELETE /api/v2/addresses/88/resourceRecords/77",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requests = %v, want %v", got, want)
	}
}

func TestDependentRecordLookupKeepsParentScope(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/resourceRecords/77/dependentRecords/99" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":99,"type":"AliasRecord","name":"www","linkedRecord":{"id":77}}`)
	}))
	t.Cleanup(server.Close)
	record, err := dns.New(newClient(t, server)).DependentRecords(77).Get(context.Background(), 99)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != 99 {
		t.Fatalf("record = %#v", record)
	}
}
