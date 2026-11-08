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

func TestNetworkSettingsPreserveUnknownFields(t *testing.T) {
	t.Parallel()
	var writes atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = fmt.Fprint(w, `{"id":8,"type":"IPv4Network","range":"192.0.2.0/24","gateway":"192.0.2.1","pingBeforeAssignmentEnabled":true,"dynamicUpdateEnabled":true,"extension":{"keep":true}}`)
			return
		}
		if r.Method != http.MethodPut || r.URL.Path != "/api/v2/networks/8" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body bam.Document
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if string(body["pingBeforeAssignmentEnabled"]) != "false" || string(body["dynamicUpdateEnabled"]) != "false" || string(body["extension"]) != `{"keep":true}` {
			t.Errorf("network settings=%s", body)
		}
		if _, found := body["pingBeforeAssignEnabled"]; found {
			t.Error("used obsolete 9.5 field name")
		}
		if r.Header.Get("x-bcn-no-gateway") != "true" {
			t.Error("missing gateway removal header")
		}
		writes.Add(1)
		body["gateway"] = json.RawMessage("null")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	})
	_, err := client.GlobalNetworks().Update(context.Background(), 8, ipam.NetworkChanges{
		PingBeforeAssignmentEnabled: bam.Set(false), DynamicUpdateEnabled: bam.Set(false),
	}, ipam.WithNoGateway(true))
	if err != nil {
		t.Fatal(err)
	}
	if writes.Load() != 1 {
		t.Fatalf("writes=%d", writes.Load())
	}
}

func TestPrefixChangesRejectFamilySwitch(t *testing.T) {
	t.Parallel()
	document := bam.Document{"type": json.RawMessage(`"IPv4Network"`), "range": json.RawMessage(`"192.0.2.0/24"`)}
	if err := (ipam.NetworkChanges{Range: bam.Set(netip.MustParsePrefix("2001:db8::/64"))}).Apply(document); err == nil {
		t.Fatal("accepted family change")
	}
	if err := (ipam.NetworkChanges{Range: bam.Clear[netip.Prefix]()}).Apply(document); err == nil {
		t.Fatal("accepted clearing prefix")
	}
}
