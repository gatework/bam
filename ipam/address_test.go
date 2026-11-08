package ipam_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/ipam"
)

func TestAddressAllocationModes(t *testing.T) {
	t.Parallel()
	mac, err := ipam.ParseMAC("aabb.cc11.2233")
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != "POST" || r.URL.Path != "/api/v2/networks/30/addresses" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		switch string(body["name"]) {
		case `"fixed"`:
			if string(body["state"]) != `"STATIC"` || string(body["address"]) != `"192.0.2.10"` {
				t.Errorf("static assignment: %s", body)
			}
			var link struct {
				Type    string
				Address string
			}
			if err := json.Unmarshal(body["macAddress"], &link); err != nil {
				t.Error(err)
			}
			if link.Type != "MACAddress" || link.Address != "AA-BB-CC-11-22-33" {
				t.Errorf("mac link: %+v", link)
			}
		case `"held"`:
			if string(body["state"]) != `"RESERVED"` {
				t.Errorf("reservation: %s", body)
			}
			if _, ok := body["macAddress"]; ok {
				t.Error("administrative reservation unexpectedly binds a MAC")
			}
		case `"next"`:
			if _, ok := body["address"]; ok {
				t.Error("next allocation submitted an address instead of server-side selection")
			}
			if r.Header.Get("x-bcn-excluded-addresses") != "192.0.2.1,192.0.2.10-192.0.2.20" || r.Header.Get("x-bcn-ping-before-assignment") != "false" {
				t.Errorf("allocation headers: %v", r.Header)
			}
			body["address"] = json.RawMessage(`"192.0.2.21"`)
		default:
			t.Errorf("unexpected assignment %s", body)
		}
		body["id"] = json.RawMessage(`42`)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	addresses := client.Addresses(30)
	if _, err := addresses.AssignStatic(ctx, ipam.AddressSpec{Name: "fixed", Address: netip.MustParseAddr("192.0.2.10"), MACAddress: &ipam.MACReference{Address: mac}}); err != nil {
		t.Fatal(err)
	}
	if _, err := addresses.Reserve(ctx, ipam.AddressSpec{Name: "held", Address: netip.MustParseAddr("192.0.2.11")}); err != nil {
		t.Fatal(err)
	}
	got, err := addresses.AllocateNext(ctx, ipam.NextAddressSpec{Name: "next"}, ipam.WithExcludedAddresses(netip.MustParseAddr("192.0.2.1")), ipam.WithExcludedRanges(ipam.AddressRange{Start: netip.MustParseAddr("192.0.2.10"), End: netip.MustParseAddr("192.0.2.20")}), ipam.WithPingBeforeAssignment(false))
	if err != nil {
		t.Fatal(err)
	}
	if got.Address.String() != "192.0.2.21" || requests.Load() != 3 {
		t.Fatalf("allocation=%+v requests=%d", got, requests.Load())
	}
}

func TestAddressUpdateAndReleasePreserveResourceOwnership(t *testing.T) {
	t.Parallel()
	const original = `{"id":42,"type":"IPv4Address","state":"STATIC","address":"192.0.2.8","name":"old","opaque":{"n":9007199254740993,"enabled":false}}`
	var writes atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/addresses/42" {
			t.Errorf("request escaped address ownership: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = fmt.Fprint(w, original)
		case http.MethodPut:
			writes.Add(1)
			var body bam.Document
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if string(body["opaque"]) != `{"n":9007199254740993,"enabled":false}` {
				t.Errorf("unknown field lost: %s", body["opaque"])
			}
			if string(body["state"]) != `"DHCP_RESERVED"` || string(body["name"]) != `""` {
				t.Errorf("changes lost: %s", body)
			}
			if r.Header.Get("x-bcn-change-control-comment") != "bind workstation" {
				t.Error("missing change control comment")
			}
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Error(err)
			}
		case http.MethodDelete:
			writes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	ctx := context.Background()
	_, err := client.GlobalAddresses().Update(ctx, 42, ipam.AddressChanges{Name: bam.Set(""), State: bam.Set(ipam.StateDHCPReserved), MACAddress: bam.Set(ipam.MACReference{ID: 99})}, bam.WithChangeControlComment("bind workstation"))
	if err != nil {
		t.Fatal(err)
	}
	if err := client.GlobalAddresses().Release(ctx, 42); err != nil {
		t.Fatal(err)
	}
	if writes.Load() != 2 {
		t.Fatalf("writes=%d", writes.Load())
	}
}

func TestIPAndMACLookupScopes(t *testing.T) {
	t.Parallel()
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/addresses":
			filter := r.URL.Query().Get("filter")
			if !strings.Contains(filter, "configuration.id:eq(7)") || !strings.Contains(filter, `address:eq("192.0.2.8")`) {
				t.Errorf("unsafe lookup scope: %q", filter)
			}
			_, _ = fmt.Fprint(w, `{"count":1,"data":[{"id":42,"type":"IPv4Address","address":"192.0.2.8","state":"STATIC"}]}`)
		case "/api/v2/configurations/7/macAddresses":
			if !strings.Contains(r.URL.Query().Get("filter"), `address:eq("AA-BB-CC-11-22-33")`) {
				t.Error("MAC was not normalized")
			}
			_, _ = fmt.Fprint(w, `{"count":1,"data":[{"id":99,"type":"MACAddress","address":"aa:bb:cc:11:22:33"}]}`)
		case "/api/v2/macAddresses/99/addresses":
			_, _ = fmt.Fprint(w, `{"count":1,"data":[{"id":42,"type":"IPv4Address","address":"192.0.2.8","state":"DHCP_LEASED","leaseExpirationDateTime":"2026-11-08 00:00:00 UTC"}]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	ctx := context.Background()
	address, err := client.LookupIP(ctx, 7, netip.MustParseAddr("192.0.2.8"))
	if err != nil || address.ID != 42 {
		t.Fatalf("lookup=%+v, %v", address, err)
	}
	mac, err := ipam.ParseMAC("aa:bb:cc:11:22:33")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for address, err := range client.AddressesByMAC(ctx, 7, mac, bam.ListOptions{}) {
		if err != nil {
			t.Fatal(err)
		}
		count++
		if address.State != ipam.StateDHCPLeased || address.LeaseExpirationDateTime == "" {
			t.Errorf("lease metadata lost: %+v", address)
		}
	}
	if count != 1 {
		t.Fatalf("addresses=%d", count)
	}
}

func TestLookupRejectsAmbiguousIP(t *testing.T) {
	t.Parallel()
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"count":2,"data":[{"id":1,"type":"IPv4Address"},{"id":2,"type":"IPv4Address"}]}`)
	})
	_, err := client.LookupIP(context.Background(), 7, netip.MustParseAddr("192.0.2.8"))
	if !errors.Is(err, bam.ErrAmbiguous) {
		t.Fatalf("error=%v", err)
	}
}

func TestMACValue(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"aa:bb:cc:11:22:33", "AA-BB-CC-11-22-33", "aabb.cc11.2233"} {
		value, err := ipam.ParseMAC(input)
		if err != nil || value.String() != "AA-BB-CC-11-22-33" {
			t.Errorf("parse %q = %s, %v", input, value, err)
		}
	}
	for _, input := range []string{"", "aa:bb:cc:11:22", "aa:bb:cc:11:22:33:44:55"} {
		if _, err := ipam.ParseMAC(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}
