package deployment_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/gatework/bam"
)

func TestInterfaceDiscovery(t *testing.T) {
	t.Parallel()
	paths := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		if r.Method != http.MethodGet {
			t.Errorf("interface discovery mutated state: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/hal+json")
		switch r.URL.Path {
		case "/api/v2/interfaces":
			_, _ = io.WriteString(w, `{"count":0,"data":[]}`)
		case "/api/v2/servers/42/interfaces":
			_, _ = io.WriteString(w, `{"count":1,"data":[{"id":101,"type":"NetworkInterface","name":"eth0","server":{"id":42},"managementAddress":"192.0.2.8","servicesIpv4Address":"192.0.2.9","servicesIpv4PrefixLength":24}]}`)
		case "/api/v2/interfaces/102":
			_, _ = io.WriteString(w, `{"id":102,"type":"PublishedInterface","name":"dns","server":{"id":42},"primaryAddress":"198.51.100.9","secondaryAddress":"2001:db8::9"}`)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	c := newClient(t, server)
	if _, err := c.GlobalInterfaces().Page(context.Background(), bam.ListOptions{}); err != nil {
		t.Fatal(err)
	}
	page, err := c.ServerInterfaces(42).Page(context.Background(), bam.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != 101 || page.Data[0].Server.ID != 42 ||
		page.Data[0].ServicesIPv4Address != netip.MustParseAddr("192.0.2.9") || page.Data[0].ServicesIPv4PrefixLength != 24 {
		t.Fatalf("network interfaces = %+v", page.Data)
	}
	iface, err := c.ServerInterfaces(42).Get(context.Background(), 102)
	if err != nil {
		t.Fatal(err)
	}
	if iface.PrimaryAddress != netip.MustParseAddr("198.51.100.9") || iface.SecondaryAddress != netip.MustParseAddr("2001:db8::9") {
		t.Fatalf("published interface = %+v", iface)
	}
	for _, expected := range []string{"/api/v2/interfaces", "/api/v2/servers/42/interfaces", "/api/v2/interfaces/102"} {
		if got := <-paths; got != expected {
			t.Errorf("path = %s, want %s", got, expected)
		}
	}
}
