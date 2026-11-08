package ipam_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/ipam"
)

func TestIPv6RejectsIPv4OnlyAllocationStates(t *testing.T) {
	t.Parallel()
	var reads, mutations atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			reads.Add(1)
		} else {
			mutations.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":8,"type":"IPv6Address","address":"2001:db8::8","state":"STATIC"}`)
	})
	ctx := context.Background()
	for _, state := range []ipam.State{ipam.StateReserved, ipam.StateGateway} {
		_, err := client.Addresses(7).Create(ctx, ipam.AddressSpec{Address: netip.MustParseAddr("2001:db8::8"), State: state})
		if err == nil {
			t.Errorf("create accepted IPv6 state %s", state)
		}
		_, err = client.GlobalAddresses().Update(ctx, 8, ipam.AddressChanges{State: bam.Set(state)})
		if err == nil {
			t.Errorf("update accepted IPv6 state %s", state)
		}
	}
	if reads.Load() != 2 || mutations.Load() != 0 {
		t.Fatalf("reads=%d mutations=%d", reads.Load(), mutations.Load())
	}
}

func TestIPv6DHCPReservationWithMAC(t *testing.T) {
	t.Parallel()
	var creates, updates atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/addresses/8" {
			_, _ = fmt.Fprint(w, `{"id":8,"type":"IPv6Address","address":"2001:db8::8","state":"STATIC","opaque":{"n":9007199254740993}}`)
			return
		}
		var body bam.Document
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if string(body["type"]) != `"IPv6Address"` || string(body["state"]) != `"DHCP_RESERVED"` || string(body["address"]) != `"2001:db8::8"` {
			t.Errorf("reservation body=%s", body)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/networks/7/addresses":
			creates.Add(1)
			if string(body["macAddress"]) != `{"type":"MACAddress","address":"AA-BB-CC-00-11-22"}` {
				t.Errorf("MAC=%s", body["macAddress"])
			}
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/addresses/8":
			updates.Add(1)
			if string(body["opaque"]) != `{"n":9007199254740993}` || string(body["macAddress"]) != `{"type":"MACAddress","id":19}` {
				t.Errorf("update lost MAC or opaque data: %s", body)
			}
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		body["id"] = json.RawMessage("8")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	mac, err := ipam.ParseMAC("aa:bb:cc:00:11:22")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Addresses(7).Create(ctx, ipam.AddressSpec{
		Address: netip.MustParseAddr("2001:db8::8"), State: ipam.StateDHCPReserved,
		MACAddress: &ipam.MACReference{Address: mac},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GlobalAddresses().Update(ctx, 8, ipam.AddressChanges{
		State: bam.Set(ipam.StateDHCPReserved), MACAddress: bam.Set(ipam.MACReference{ID: 19}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if creates.Load() != 1 || updates.Load() != 1 {
		t.Fatalf("creates=%d updates=%d", creates.Load(), updates.Load())
	}
}
