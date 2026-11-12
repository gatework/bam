package dhcp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"strconv"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dhcp"
)

func TestCommonProvisioningOptionsUseTypedWireValues(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		build func() (dhcp.ClientOptionSpec, error)
		code  int
		value string
	}{
		{"ntp", func() (dhcp.ClientOptionSpec, error) {
			return dhcp.NTPServers(netip.MustParseAddr("192.0.2.42"), netip.MustParseAddr("192.0.2.43"))
		}, 42, `["192.0.2.42","192.0.2.43"]`},
		{"search domains", func() (dhcp.ClientOptionSpec, error) {
			return dhcp.DomainSearch("ops.example.test", "example.test")
		}, 119, `["ops.example.test","example.test"]`},
		{"tftp addresses", func() (dhcp.ClientOptionSpec, error) {
			return dhcp.TFTPServers(netip.MustParseAddr("192.0.2.150"))
		}, 150, `["192.0.2.150"]`},
		{"tftp name", func() (dhcp.ClientOptionSpec, error) {
			return dhcp.TFTPServerName("boot.example.test")
		}, 66, `"boot.example.test"`},
		{"boot file", func() (dhcp.ClientOptionSpec, error) {
			return dhcp.BootFileName("pxelinux.0")
		}, 67, `"pxelinux.0"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/v2/networks/7/deploymentOptions" {
					t.Errorf("unexpected mutation %s %s", r.Method, r.URL)
				}
				var body bam.Document
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if string(body["type"]) != `"DHCPv4ClientOption"` || string(body["code"]) != strconv.Itoa(tt.code) || string(body["value"]) != tt.value {
					t.Errorf("unexpected option body %s", body)
				}
				body["id"] = json.RawMessage("8")
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(body)
			})
			spec, err := tt.build()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.NetworkOptions(7).Create(t.Context(), spec); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProvisioningOptions_InvalidInput(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		build func() (dhcp.ClientOptionSpec, error)
	}{
		{"empty domain list", func() (dhcp.ClientOptionSpec, error) { return dhcp.DomainSearch() }},
		{"blank domain", func() (dhcp.ClientOptionSpec, error) { return dhcp.DomainSearch("example.test", " ") }},
		{"ipv6 ntp server", func() (dhcp.ClientOptionSpec, error) { return dhcp.NTPServers(netip.MustParseAddr("2001:db8::1")) }},
		{"empty tftp list", func() (dhcp.ClientOptionSpec, error) { return dhcp.TFTPServers() }},
		{"blank tftp name", func() (dhcp.ClientOptionSpec, error) { return dhcp.TFTPServerName(" ") }},
		{"empty boot file", func() (dhcp.ClientOptionSpec, error) { return dhcp.BootFileName("") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tt.build(); err == nil {
				t.Fatal("accepted invalid provisioning option")
			}
		})
	}
}

func ExampleDomainSearch() {
	spec, err := dhcp.DomainSearch("ops.example.test", "example.test")
	if err != nil {
		return
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return
	}
	fmt.Println(string(data))
	// Output:
	// {"code":119,"value":["ops.example.test","example.test"],"type":"DHCPv4ClientOption"}
}
