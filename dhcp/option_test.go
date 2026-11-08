package dhcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dhcp"
)

func TestOptionOverrideLifecycle(t *testing.T) {
	t.Parallel()
	var mutations atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/deploymentOptions/8" {
			_, _ = fmt.Fprint(w, `{"id":8,"type":"DHCPv4ClientOption","code":6,"value":["192.0.2.53"],"serverScope":{"id":19},"opaque":{"keep":true}}`)
			return
		}
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v2/deploymentOptions/8" {
			mutations.Add(1)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var body bam.Document
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/networks/7/deploymentOptions":
			if string(body["type"]) != `"DHCPv4ClientOption"` || string(body["code"]) != "6" || string(body["value"]) != `["192.0.2.53","192.0.2.54"]` {
				t.Errorf("DNS server option: %s", body)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/deploymentOptions/8":
			if string(body["value"]) != "[]" || string(body["opaque"]) != `{"keep":true}` || string(body["serverScope"]) != "null" {
				t.Errorf("update: %s", body)
			}
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		mutations.Add(1)
		body["id"] = json.RawMessage("8")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	options := client.NetworkOptions(7)
	spec, err := dhcp.DNSServers(netip.MustParseAddr("192.0.2.53"), netip.MustParseAddr("192.0.2.54"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := options.Create(ctx, spec); err != nil {
		t.Fatal(err)
	}
	if _, err := options.Update(ctx, 8, dhcp.OptionChanges{Value: bam.Set(bam.StringsValue()), ServerScope: bam.Clear[bam.Reference]()}); err != nil {
		t.Fatal(err)
	}
	if _, err := options.Update(ctx, 8, dhcp.OptionChanges{Value: bam.Clear[bam.Value]()}); err == nil {
		t.Error("clearing an option value cannot restore inheritance")
	}
	if err := options.RestoreInheritance(ctx, 8); err != nil {
		t.Fatal(err)
	}
	if mutations.Load() != 3 {
		t.Fatalf("mutations=%d", mutations.Load())
	}
}

func TestOptionValueTypesAndValidation(t *testing.T) {
	t.Parallel()
	lease, err := dhcp.LeaseTime(2 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		spec  dhcp.OptionSpec
		value string
	}{
		{lease, "7200"},
		{dhcp.ClientOptionSpec{Name: "ip-forwarding", Value: bam.BoolValue(false)}, "false"},
		{dhcp.ServiceOptionSpec{Name: "ddns-ttl", Value: bam.IntValue(86400)}, "86400"},
		{dhcp.RawOptionSpec{Value: "ignore client-updates;"}, `"ignore client-updates;"`},
	} {
		data, err := json.Marshal(tt.spec)
		if err != nil {
			t.Fatal(err)
		}
		var body bam.Document
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatal(err)
		}
		if string(body["value"]) != tt.value {
			t.Errorf("value=%s want=%s", body["value"], tt.value)
		}
	}
	for _, spec := range []dhcp.OptionSpec{
		dhcp.ClientOptionSpec{Code: 6},
		dhcp.ClientOptionSpec{Name: "domain-name-servers", Code: 6, Value: bam.StringValue("invalid")},
		dhcp.ServiceOptionSpec{Definition: &bam.Reference{}, Value: bam.BoolValue(true)},
	} {
		if _, err := json.Marshal(spec); err == nil {
			t.Errorf("accepted invalid spec %+v", spec)
		}
	}
	if _, err := dhcp.DNSServers(netip.MustParseAddr("2001:db8::53")); err == nil {
		t.Error("DHCPv4 DNS option accepted IPv6")
	}
	if _, err := dhcp.LeaseTime(time.Millisecond); err == nil {
		t.Error("lease option accepted fractional seconds")
	}
}
