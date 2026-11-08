package dhcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dhcp"
	"github.com/gatework/bam/ipam"
)

func TestReservationBindingLifecycle(t *testing.T) {
	t.Parallel()
	var mutations atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/addresses/8" {
			_, _ = fmt.Fprint(w, `{"id":8,"type":"IPv4Address","address":"192.0.2.20","state":"DHCP_RESERVED","macAddress":{"id":19,"type":"MACAddress"},"opaque":9007199254740993}`)
			return
		}
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v2/addresses/8" {
			mutations.Add(1)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path != "/api/v2/networks/7/addresses" && r.URL.Path != "/api/v2/addresses/8" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body bam.Document
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if string(body["type"]) != `"IPv4Address"` || string(body["state"]) != `"DHCP_RESERVED"` {
			t.Errorf("not a DHCP binding: %s", body)
		}
		var mac struct{ ID int64 }
		if err := json.Unmarshal(body["macAddress"], &mac); err != nil {
			t.Error(err)
		}
		if r.Method == http.MethodPut {
			if mac.ID != 20 || string(body["opaque"]) != "9007199254740993" {
				t.Errorf("update lost binding or opaque field: %s", body)
			}
		} else if mac.ID != 19 {
			t.Errorf("create MAC = %d", mac.ID)
		}
		body["id"] = json.RawMessage("8")
		mutations.Add(1)
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	bindings := client.Reservations(7)
	if _, err := bindings.Create(ctx, dhcp.ReservationSpec{Address: netip.MustParseAddr("192.0.2.20"), MAC: ipam.MACReference{ID: 19}}); err != nil {
		t.Fatal(err)
	}
	if _, err := bindings.Update(ctx, 8, dhcp.ReservationChanges{MAC: bam.Set(ipam.MACReference{ID: 20})}); err != nil {
		t.Fatal(err)
	}
	if err := bindings.Delete(ctx, 8); err != nil {
		t.Fatal(err)
	}
	if mutations.Load() != 3 {
		t.Fatalf("mutations=%d", mutations.Load())
	}
}

func TestReservationsRejectOtherAllocationStates(t *testing.T) {
	t.Parallel()
	var mutations atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mutations.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":8,"type":"IPv4Address","address":"192.0.2.20","state":"STATIC"}`)
	})
	ctx := context.Background()
	bindings := client.Reservations(7)
	if _, err := bindings.Get(ctx, 8); err == nil {
		t.Error("read accepted a static allocation")
	}
	if _, err := bindings.Update(ctx, 8, dhcp.ReservationChanges{Name: bam.Set("unexpected")}); err == nil {
		t.Error("update accepted a static allocation")
	}
	if err := bindings.Delete(ctx, 8); err == nil {
		t.Error("delete accepted a static allocation")
	}
	if _, err := bindings.Create(ctx, dhcp.ReservationSpec{Address: netip.MustParseAddr("192.0.2.20")}); err == nil {
		t.Error("create accepted missing MAC")
	}
	if mutations.Load() != 0 {
		t.Fatalf("unexpected mutations=%d", mutations.Load())
	}
}
