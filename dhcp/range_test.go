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

func TestRangeCreateAndResize(t *testing.T) {
	t.Parallel()
	var writes atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/ranges/9" {
			_, _ = fmt.Fprint(w, `{"id":9,"type":"IPv4DHCPRange","range":"192.0.2.10-192.0.2.200","splitAroundStaticAddresses":true,"exclusionRanges":["192.0.2.50-192.0.2.60"],"lowWaterMark":10,"highWaterMark":90,"extension":{"opaque":true}}`)
			return
		}
		var body bam.Document
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/networks/7/ranges":
			if string(body["type"]) != `"IPv4DHCPRange"` || string(body["range"]) != `"10,50%"` || string(body["splitAroundStaticAddresses"]) != "false" {
				t.Errorf("create payload: %s", body)
			}
			body["id"] = json.RawMessage("9")
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/ranges/9":
			if string(body["range"]) != `"192.0.2.10-192.0.2.99"` || string(body["splitAroundStaticAddresses"]) != "false" || string(body["exclusionRanges"]) != "[]" {
				t.Errorf("update payload: %s", body)
			}
			if string(body["extension"]) != `{"opaque":true}` || string(body["lowWaterMark"]) != "0" {
				t.Errorf("preserved fields or zero lost: %s", body)
			}
			if r.Header.Get("x-bcn-orphaned-address-state") != "STATIC" {
				t.Error("missing orphan state header")
			}
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
		writes.Add(1)
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	ranges := client.Ranges(7)
	if _, err := ranges.Create(ctx, dhcp.RangeSpec{Name: "pool", Range: dhcp.OffsetPercent(10, 50)}); err != nil {
		t.Fatal(err)
	}
	_, err := ranges.Update(ctx, 9, dhcp.RangeChanges{
		Range:                      bam.Set(dhcp.Between(netip.MustParseAddr("192.0.2.10"), netip.MustParseAddr("192.0.2.99"))),
		SplitAroundStaticAddresses: bam.Set(false),
		ExclusionRanges:            bam.Set([]ipam.AddressRange{}),
		LowWaterMark:               bam.Set(0),
	}, dhcp.WithOrphanedAddressState(ipam.StateStatic))
	if err != nil {
		t.Fatal(err)
	}
	if writes.Load() != 2 {
		t.Fatalf("writes = %d", writes.Load())
	}
}

func TestRangeValidation(t *testing.T) {
	t.Parallel()
	for _, span := range []dhcp.Span{
		{}, dhcp.OffsetSize(0, 0), dhcp.OffsetPercent(10, 101),
		dhcp.Between(netip.MustParseAddr("192.0.2.10"), netip.MustParseAddr("192.0.2.1")),
		dhcp.Between(netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("2001:db8::2")),
	} {
		if _, err := json.Marshal(dhcp.RangeSpec{Range: span}); err == nil {
			t.Errorf("accepted invalid span %+v", span)
		}
	}
	var document bam.Document
	if err := json.Unmarshal([]byte(`{"lowWaterMark":20,"highWaterMark":80}`), &document); err != nil {
		t.Fatal(err)
	}
	if err := (dhcp.RangeChanges{HighWaterMark: bam.Set(10)}).Apply(document); err == nil {
		t.Fatal("accepted water marks that conflict after merging")
	}
}
