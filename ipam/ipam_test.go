package ipam_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/ipam"
)

func TestCollectionScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		page func(*ipam.Client, bam.ListOptions) error
	}{
		{
			name: "configuration blocks", path: "/api/v2/configurations/10/blocks",
			page: func(c *ipam.Client, options bam.ListOptions) error {
				_, err := c.Blocks(10).Page(context.Background(), options)
				return err
			},
		},
		{
			name: "subblocks", path: "/api/v2/blocks/20/blocks",
			page: func(c *ipam.Client, options bam.ListOptions) error {
				_, err := c.Subblocks(20).Page(context.Background(), options)
				return err
			},
		},
		{
			name: "networks", path: "/api/v2/blocks/20/networks",
			page: func(c *ipam.Client, options bam.ListOptions) error {
				_, err := c.Networks(20).Page(context.Background(), options)
				return err
			},
		},
		{
			name: "addresses", path: "/api/v2/networks/30/addresses",
			page: func(c *ipam.Client, options bam.ListOptions) error {
				_, err := c.Addresses(30).Page(context.Background(), options)
				return err
			},
		},
		{
			name: "global blocks", path: "/api/v2/blocks",
			page: func(c *ipam.Client, options bam.ListOptions) error {
				_, err := c.GlobalBlocks().Page(context.Background(), options)
				return err
			},
		},
		{
			name: "global networks", path: "/api/v2/networks",
			page: func(c *ipam.Client, options bam.ListOptions) error {
				_, err := c.GlobalNetworks().Page(context.Background(), options)
				return err
			},
		},
		{
			name: "global addresses", path: "/api/v2/addresses",
			page: func(c *ipam.Client, options bam.ListOptions) error {
				_, err := c.GlobalAddresses().Page(context.Background(), options)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			const filter = "name:'a&b'"
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != tt.path {
					t.Errorf("request = %s %s, want GET %s", r.Method, r.URL.Path, tt.path)
				}
				if got := r.URL.Query().Get("filter"); got != filter {
					t.Errorf("filter = %q, want %q", got, filter)
				}
				w.Header().Set("Content-Type", "application/hal+json")
				_, _ = fmt.Fprint(w, `{"count":0,"data":[]}`)
			})
			if err := tt.page(client, bam.ListOptions{Filter: filter}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestScopedItemUsesGlobalID(t *testing.T) {
	t.Parallel()
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/networks/41" {
			t.Errorf("request = %s %s, want GET /api/v2/networks/41", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":41,"type":"IPv6Network","name":"test","range":"2001:db8::/64"}`)
	})
	got, err := client.Networks(20).Get(context.Background(), 41)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 41 || got.Range != "2001:db8::/64" {
		t.Fatalf("network = %+v", got)
	}
}

func TestCreateRanges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		path   string
		kind   string
		prefix string
		create func(*ipam.Client, netip.Prefix) error
	}{
		{
			name: "IPv4 block", path: "/api/v2/configurations/10/blocks", kind: "IPv4Block", prefix: "192.0.2.0/24",
			create: func(c *ipam.Client, prefix netip.Prefix) error {
				_, err := c.Blocks(10).Create(context.Background(), ipam.BlockSpec{Name: "test", Range: prefix})
				return err
			},
		},
		{
			name: "IPv6 subblock", path: "/api/v2/blocks/20/blocks", kind: "IPv6Block", prefix: "2001:db8::/48",
			create: func(c *ipam.Client, prefix netip.Prefix) error {
				_, err := c.Subblocks(20).Create(context.Background(), ipam.BlockSpec{Name: "test", Range: prefix})
				return err
			},
		},
		{
			name: "IPv4 network", path: "/api/v2/blocks/20/networks", kind: "IPv4Network", prefix: "192.0.2.0/24",
			create: func(c *ipam.Client, prefix netip.Prefix) error {
				_, err := c.Networks(20).Create(context.Background(), ipam.NetworkSpec{Name: "test", Range: prefix})
				return err
			},
		},
		{
			name: "IPv6 network", path: "/api/v2/blocks/20/networks", kind: "IPv6Network", prefix: "2001:db8::/64",
			create: func(c *ipam.Client, prefix netip.Prefix) error {
				_, err := c.Networks(20).Create(context.Background(), ipam.NetworkSpec{Name: "test", Range: prefix})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != tt.path {
					t.Errorf("request = %s %s, want POST %s", r.Method, r.URL.Path, tt.path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					http.Error(w, "bad request body", http.StatusBadRequest)
					return
				}
				if body["type"] != tt.kind || body["range"] != tt.prefix || body["name"] != "test" {
					t.Errorf("body = %#v", body)
				}
				body["id"] = 42
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				if err := json.NewEncoder(w).Encode(body); err != nil {
					t.Error(err)
				}
			})
			if err := tt.create(client, netip.MustParsePrefix(tt.prefix)); err != nil {
				t.Fatal(err)
			}
			if got := requests.Load(); got != 1 {
				t.Fatalf("requests = %d, want one POST", got)
			}
		})
	}
}

func TestCreateExactAddress(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		address string
		kind    string
	}{
		{name: "IPv4", address: "192.0.2.4", kind: "IPv4Address"},
		{name: "IPv6", address: "2001:db8::4", kind: "IPv6Address"},
		{name: "IPv4-mapped IPv6 preserves its family", address: "::ffff:192.0.2.4", kind: "IPv6Address"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/api/v2/networks/30/addresses" {
					t.Errorf("request = %s %s, want POST /api/v2/networks/30/addresses", r.Method, r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					http.Error(w, "bad request body", http.StatusBadRequest)
					return
				}
				if body["type"] != tt.kind || body["address"] != tt.address || body["state"] != "STATIC" {
					t.Errorf("body = %#v", body)
				}
				body["id"] = 42
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				if err := json.NewEncoder(w).Encode(body); err != nil {
					t.Error(err)
				}
			})
			address := netip.MustParseAddr(tt.address)
			got, err := client.Addresses(30).Create(context.Background(), ipam.AddressSpec{Address: address, State: ipam.StateStatic})
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != 42 || got.Address != address || got.State != ipam.StateStatic {
				t.Errorf("address = %+v", got)
			}
			if got := requests.Load(); got != 1 {
				t.Fatalf("requests = %d, want one POST without an availability lookup", got)
			}
		})
	}
}

func TestInvalidCreateDoesNotSend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		create func(*ipam.Client) error
	}{
		{
			name: "invalid parent",
			create: func(c *ipam.Client) error {
				_, err := c.Addresses(0).Create(context.Background(), ipam.AddressSpec{Address: netip.MustParseAddr("192.0.2.4")})
				return err
			},
		},
		{
			name: "invalid block prefix",
			create: func(c *ipam.Client) error {
				_, err := c.Blocks(10).Create(context.Background(), ipam.BlockSpec{})
				return err
			},
		},
		{
			name: "network prefix has host bits",
			create: func(c *ipam.Client) error {
				_, err := c.Networks(20).Create(context.Background(), ipam.NetworkSpec{Range: netip.MustParsePrefix("192.0.2.1/24")})
				return err
			},
		},
		{
			name: "missing exact address",
			create: func(c *ipam.Client) error {
				_, err := c.Addresses(30).Create(context.Background(), ipam.AddressSpec{})
				return err
			},
		},
		{
			name: "scoped IPv6 address",
			create: func(c *ipam.Client) error {
				_, err := c.Addresses(30).Create(context.Background(), ipam.AddressSpec{Address: netip.MustParseAddr("fe80::1%en0")})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			})
			if err := tt.create(client); err == nil {
				t.Fatal("invalid input accepted")
			}
			if got := requests.Load(); got != 0 {
				t.Fatalf("requests = %d, want none", got)
			}
		})
	}
}

func TestCreateAddressDoesNotRetry(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusUnauthorized, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = fmt.Fprint(w, `{"message":"assignment failed"}`)
			})
			_, err := client.Addresses(30).Create(context.Background(), ipam.AddressSpec{
				Address: netip.MustParseAddr("192.0.2.4"),
				State:   ipam.StateReserved,
			})
			if err == nil {
				t.Fatal("expected assignment error")
			}
			if got := requests.Load(); got != 1 {
				t.Fatalf("requests = %d, want one failed assignment without replay", got)
			}
		})
	}
}

func newClient(t *testing.T, handler http.HandlerFunc) *ipam.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := bam.New(server.URL, bam.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	return ipam.New(client)
}
