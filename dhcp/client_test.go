package dhcp_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dhcp"
)

func newClient(t *testing.T, handler http.HandlerFunc) *dhcp.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := bam.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return dhcp.New(client)
}

func TestClientCollectionScope(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path, kind string
		page             func(*dhcp.Client) error
	}{
		{"ranges", "networks/42/ranges", "IPv4DHCPRange", func(c *dhcp.Client) error {
			_, e := c.Ranges(42).Page(context.Background(), bam.ListOptions{})
			return e
		}},
		{"reservations", "networks/42/addresses", "DHCP_RESERVED", func(c *dhcp.Client) error {
			_, e := c.Reservations(42).Page(context.Background(), bam.ListOptions{})
			return e
		}},
		{"configuration options", "configurations/42/deploymentOptions", "DHCPv4ClientOption", func(c *dhcp.Client) error {
			_, e := c.ConfigurationOptions(42).Page(context.Background(), bam.ListOptions{})
			return e
		}},
		{"network options", "networks/42/deploymentOptions", "DHCPv4ClientOption", func(c *dhcp.Client) error {
			_, e := c.NetworkOptions(42).Page(context.Background(), bam.ListOptions{})
			return e
		}},
		{"block roles", "blocks/42/deploymentRoles", "DHCPDeploymentRole", func(c *dhcp.Client) error {
			_, e := c.BlockRoles(42).Page(context.Background(), bam.ListOptions{})
			return e
		}},
		{"network roles", "networks/42/deploymentRoles", "DHCPDeploymentRole", func(c *dhcp.Client) error {
			_, e := c.NetworkRoles(42).Page(context.Background(), bam.ListOptions{})
			return e
		}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/api/v2/"+tt.path {
					t.Errorf("request=%s %s", r.Method, r.URL.Path)
				}
				if !strings.Contains(r.URL.Query().Get("filter"), tt.kind) {
					t.Errorf("missing domain restriction: %s", r.URL.RawQuery)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"count":0,"data":[]}`)
			})
			if err := tt.page(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}
