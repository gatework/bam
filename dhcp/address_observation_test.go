package dhcp_test

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/ipam"
)

func TestClient_LeasedAddresses(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/networks/7/addresses" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		for _, clause := range []string{`type:eq("IPv4Address")`, `state:eq("DHCP_LEASED")`, `name:eq("laptop")`} {
			if !strings.Contains(r.URL.Query().Get("filter"), clause) {
				t.Errorf("missing %s", clause)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"count":1,"data":[{"id":10,"type":"IPv4Address","address":"192.0.2.10","state":"DHCP_LEASED","leaseDateTime":"2026-11-11 00:00:00 UTC","leaseExpirationDateTime":"2026-11-12 00:00:00 UTC","clientIdentifier":{"uid":"example-client"}}],"_links":{"next":{"href":"?offset=1"}}}`)
	})
	items := client.LeasedAddresses(t.Context(), 7, bam.ListOptions{Where: bam.Match("name", "laptop")})
	if calls.Load() != 0 {
		t.Fatal("observation was not lazy")
	}
	found := false
	for address, err := range items {
		if err != nil {
			t.Fatal(err)
		}
		if address.State != ipam.StateDHCPLeased || address.LeaseExpirationDateTime != "2026-11-12 00:00:00 UTC" || address.ClientIdentifier == nil || address.ClientIdentifier.UID != "example-client" {
			t.Fatalf("lost observed fields: %+v", address)
		}
		found = true
		break
	}
	if !found || calls.Load() != 1 {
		t.Fatalf("found=%v requests=%d", found, calls.Load())
	}
}
