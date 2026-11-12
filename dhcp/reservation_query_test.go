package dhcp_test

import (
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/ipam"
)

func TestReservations_FindByIPAndMAC(t *testing.T) {
	t.Parallel()
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/networks/7/addresses" {
			t.Errorf("request=%s %s", r.Method, r.URL)
		}
		for _, clause := range []string{`type:eq("IPv4Address")`, `state:eq("DHCP_RESERVED")`} {
			if !strings.Contains(r.URL.Query().Get("filter"), clause) {
				t.Errorf("missing %s", clause)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"count":1,"data":[{"id":10,"type":"IPv4Address","address":"192.0.2.10","state":"DHCP_RESERVED","macAddress":{"id":20,"address":"00-11-22-33-44-55"}}]}`)
	})
	mac, err := ipam.ParseMAC("0011.2233.4455")
	if err != nil {
		t.Fatal(err)
	}
	byMAC, err := client.Reservations(7).FindByMAC(t.Context(), mac)
	if err != nil || byMAC.ID != 10 {
		t.Fatalf("MAC lookup=%+v err=%v", byMAC, err)
	}
	byIP, err := client.Reservations(7).FindByIP(t.Context(), netip.MustParseAddr("192.0.2.10"))
	if err != nil || byIP.ID != 10 {
		t.Fatalf("IP lookup=%+v err=%v", byIP, err)
	}
}

func TestReservations_FindByMAC_Ambiguity(t *testing.T) {
	t.Parallel()
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("offset") == "1" {
			_, _ = io.WriteString(w, `{"count":1,"data":[{"id":11,"type":"IPv4Address","address":"192.0.2.11","state":"DHCP_RESERVED","macAddress":{"address":"00:11:22:33:44:55"}}]}`)
		} else {
			_, _ = io.WriteString(w, `{"count":1,"data":[{"id":10,"type":"IPv4Address","address":"192.0.2.10","state":"DHCP_RESERVED","macAddress":{"address":"00-11-22-33-44-55"}}],"_links":{"next":{"href":"?offset=1"}}}`)
		}
	})
	mac, err := ipam.ParseMAC("00:11:22:33:44:55")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Reservations(7).FindByMAC(t.Context(), mac); !errors.Is(err, bam.ErrAmbiguous) {
		t.Fatalf("wanted ambiguity, got %v", err)
	}
}

func TestReservations_FindByIP_InvalidInput(t *testing.T) {
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
			client := newClient(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid selector made a request") })
			if _, err := client.Reservations(7).FindByIP(t.Context(), tt.address); err == nil {
				t.Fatal("accepted invalid IPv4 address")
			}
		})
	}
}

func TestReservations_FindByMAC_UnsetMAC(t *testing.T) {
	t.Parallel()
	client := newClient(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid selector made a request") })
	if _, err := client.Reservations(7).FindByMAC(t.Context(), ipam.MAC{}); err == nil {
		t.Fatal("accepted unset MAC")
	}
}
