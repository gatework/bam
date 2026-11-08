package dns_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dns"
)

func TestClientScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		list func(context.Context, *dns.Client) error
	}{
		{
			name: "configuration views",
			path: "/api/v2/configurations/11/views",
			list: func(ctx context.Context, c *dns.Client) error {
				_, err := c.Views(11).Page(ctx, bam.ListOptions{})
				return err
			},
		},
		{
			name: "view zones",
			path: "/api/v2/views/22/zones",
			list: func(ctx context.Context, c *dns.Client) error {
				_, err := c.Zones(22).Page(ctx, bam.ListOptions{})
				return err
			},
		},
		{
			name: "zone subzones",
			path: "/api/v2/zones/33/zones",
			list: func(ctx context.Context, c *dns.Client) error {
				_, err := c.Subzones(33).Page(ctx, bam.ListOptions{})
				return err
			},
		},
		{
			name: "zone records",
			path: "/api/v2/zones/33/resourceRecords",
			list: func(ctx context.Context, c *dns.Client) error {
				_, err := c.Records(33).Page(ctx, bam.ListOptions{})
				return err
			},
		},
		{
			name: "global views",
			path: "/api/v2/views",
			list: func(ctx context.Context, c *dns.Client) error {
				_, err := c.GlobalViews().Page(ctx, bam.ListOptions{})
				return err
			},
		},
		{
			name: "global zones",
			path: "/api/v2/zones",
			list: func(ctx context.Context, c *dns.Client) error {
				_, err := c.GlobalZones().Page(ctx, bam.ListOptions{})
				return err
			},
		},
		{
			name: "global records",
			path: "/api/v2/resourceRecords",
			list: func(ctx context.Context, c *dns.Client) error {
				_, err := c.GlobalRecords().Page(ctx, bam.ListOptions{})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != tt.path {
					t.Errorf("request = %s %s, want GET %s", r.Method, r.URL.Path, tt.path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/hal+json")
				_, _ = w.Write([]byte(`{"count":0,"data":[]}`))
			}))
			t.Cleanup(server.Close)
			if err := tt.list(context.Background(), dns.New(newClient(t, server))); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClientRecordUsesGlobalItemPath(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/resourceRecords/44" {
			t.Errorf("request = %s %s, want GET /api/v2/resourceRecords/44", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/hal+json")
		_, _ = w.Write([]byte(`{"id":44,"type":"HostRecord","name":"www","absoluteName":"www.example.test","configuration":{"id":11},"_embedded":{"addresses":[{"id":55,"type":"IPv4Address","address":"192.0.2.8"}]}}`))
	}))
	t.Cleanup(server.Close)

	record, err := dns.New(newClient(t, server)).Records(33).Get(context.Background(), 44)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != 44 || record.AbsoluteName != "www.example.test" || record.Configuration.ID != 11 {
		t.Fatalf("unexpected record: %+v", record)
	}
	host, ok := record.Details.(*dns.HostDetails)
	if !ok || len(host.Addresses) != 1 || host.Addresses[0].Address != netip.MustParseAddr("192.0.2.8") {
		t.Fatalf("unexpected host details: %+v", record.Details)
	}
}

func TestClientCreateViewAndZones(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		path   string
		kind   string
		body   map[string]string
		create func(context.Context, *dns.Client) error
	}{
		{
			name: "view",
			path: "/api/v2/configurations/11/views",
			kind: "View",
			body: map[string]string{"type": "View", "name": "default"},
			create: func(ctx context.Context, c *dns.Client) error {
				_, err := c.Views(11).Create(ctx, dns.ViewSpec{Name: "default"})
				return err
			},
		},
		{
			name: "zone by absolute name",
			path: "/api/v2/views/22/zones",
			kind: "Zone",
			body: map[string]string{"type": "Zone", "absoluteName": "example.test"},
			create: func(ctx context.Context, c *dns.Client) error {
				_, err := c.Zones(22).Create(ctx, dns.ZoneSpec{AbsoluteName: "example.test"})
				return err
			},
		},
		{
			name: "subzone by relative name",
			path: "/api/v2/zones/33/zones",
			kind: "Zone",
			body: map[string]string{"type": "Zone", "name": "dev"},
			create: func(ctx context.Context, c *dns.Client) error {
				_, err := c.Subzones(33).Create(ctx, dns.ZoneSpec{Name: "dev"})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != tt.path {
					t.Errorf("request = %s %s, want POST %s", r.Method, r.URL.Path, tt.path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if len(body) != len(tt.body) {
					t.Errorf("body = %#v, want %#v", body, tt.body)
				}
				for key, want := range tt.body {
					if body[key] != want {
						t.Errorf("body[%q] = %q, want %q", key, body[key], want)
					}
				}
				w.Header().Set("Content-Type", "application/hal+json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "type": tt.kind, "name": body["name"]})
			}))
			t.Cleanup(server.Close)
			if err := tt.create(context.Background(), dns.New(newClient(t, server))); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func newClient(t *testing.T, server *httptest.Server) *bam.Client {
	t.Helper()
	client, err := bam.New(server.URL, bam.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.CloseIdleConnections)
	return client
}
