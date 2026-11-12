package dhcp_test

import (
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/ipam"
)

func TestRanges_FindByRangeAndContaining(t *testing.T) {
	t.Parallel()
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		filter := r.URL.Query().Get("filter")
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/networks/7/ranges" || !strings.Contains(filter, `type:eq("IPv4DHCPRange")`) {
			t.Errorf("wrong range scope %s %s", r.Method, r.URL)
		}
		if !strings.Contains(filter, `range:eq("192.0.2.10-192.0.2.50")`) && !strings.Contains(filter, `range:contains("192.0.2.20")`) {
			t.Errorf("wrong range predicate %s", filter)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"count":1,"data":[{"id":10,"type":"IPv4DHCPRange","range":"192.0.2.10-192.0.2.50","exclusionRanges":["192.0.2.20-192.0.2.20"],"usage":{"DHCP_LEASED":7}}]}`)
	})
	ranges := client.Ranges(7)
	span := ipam.AddressRange{Start: netip.MustParseAddr("192.0.2.10"), End: netip.MustParseAddr("192.0.2.50")}
	item, err := ranges.FindByRange(t.Context(), span)
	if err != nil || item.ID != 10 || item.Usage["DHCP_LEASED"] != 7 {
		t.Fatalf("range=%+v error=%v", item, err)
	}
	found := false
	for containing, err := range ranges.Containing(t.Context(), netip.MustParseAddr("192.0.2.20"), bam.ListOptions{}) {
		if err != nil {
			t.Fatal(err)
		}
		if containing.ID != 10 || len(containing.ExclusionRanges) != 1 {
			t.Fatalf("containment lost exclusions: %+v", containing)
		}
		found = true
	}
	if !found {
		t.Fatal("range containment did not exercise a result")
	}
}

func TestRanges_FindByRange_UnsetRange(t *testing.T) {
	t.Parallel()
	client := newClient(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid range lookup made a request") })
	if _, err := client.Ranges(7).FindByRange(t.Context(), ipam.AddressRange{}); err == nil {
		t.Fatal("accepted unset range")
	}
}

func TestRanges_Containing_InvalidInput(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		address netip.Addr
	}{
		{"unset address", netip.Addr{}},
		{"ipv6 address", netip.MustParseAddr("2001:db8::1")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := newClient(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid range lookup made a request") })
			errorsSeen := 0
			for _, err := range client.Ranges(7).Containing(t.Context(), tt.address, bam.ListOptions{}) {
				if err == nil {
					t.Fatal("invalid containment did not return an error")
				}
				errorsSeen++
			}
			if errorsSeen != 1 {
				t.Fatalf("errors=%d", errorsSeen)
			}
		})
	}
}
