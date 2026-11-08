package dns_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dns"
)

func TestOrdinaryZoneQueriesKeepTypeAndParentScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, path string
		query      func(context.Context, *dns.Client, bam.ListOptions) error
	}{
		{
			name: "view zones", path: "/api/v2/views/22/zones",
			query: func(ctx context.Context, c *dns.Client, options bam.ListOptions) error {
				_, err := c.Zones(22).Page(ctx, options)
				return err
			},
		},
		{
			name: "subzones", path: "/api/v2/zones/33/zones",
			query: func(ctx context.Context, c *dns.Client, options bam.ListOptions) error {
				for _, err := range c.Subzones(33).All(ctx, options) {
					if err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			name: "global zones", path: "/api/v2/zones",
			query: func(ctx context.Context, c *dns.Client, options bam.ListOptions) error {
				_, err := c.GlobalZones().Page(ctx, options)
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
				filter := r.URL.Query().Get("filter")
				for _, required := range []string{`type:eq("Zone")`, `name:eq("example.test")`} {
					if !strings.Contains(filter, required) {
						t.Errorf("filter %q missing %q", filter, required)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"count":0,"data":[]}`)
			}))
			t.Cleanup(server.Close)
			options := bam.ListOptions{Where: bam.Match("name", "example.test")}
			if err := tt.query(context.Background(), dns.New(newClient(t, server)), options); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExternalHostsZoneLookup(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, body string
		wantError  error
	}{
		{"found", `{"count":1,"data":[{"id":77,"type":"ExternalHostsZone","view":{"id":22}}]}`, nil},
		{"missing", `{"count":0,"data":[]}`, bam.ErrNotFound},
		{"ambiguous", `{"count":2,"data":[{"id":77,"type":"ExternalHostsZone"},{"id":78,"type":"ExternalHostsZone"}]}`, bam.ErrAmbiguous},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v2/views/22/zones" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				query := r.URL.Query()
				if !strings.Contains(query.Get("filter"), `type:eq("ExternalHostsZone")`) || query.Get("limit") != "2" {
					t.Errorf("external zone query = %v", query)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tt.body)
			}))
			t.Cleanup(server.Close)
			zone, err := dns.New(newClient(t, server)).ExternalHostsZone(context.Background(), 22)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("error = %v, want %v", err, tt.wantError)
			}
			if err == nil && (zone.ID != 77 || zone.Type != "ExternalHostsZone" || zone.View.ID != 22) {
				t.Fatalf("external hosts zone = %#v", zone)
			}
		})
	}
}

func TestNamespaceUpdatesPreserveUnmodeledFieldsAndExplicitFalse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, path, original, field string
		update                      func(context.Context, *dns.Client) error
	}{
		{
			name: "view registration", path: "/api/v2/views/22", field: "deviceRegistrationEnabled",
			original: `{"id":22,"type":"View","name":"default","deviceRegistrationEnabled":true,"deviceRegistrationPortalAddress":"192.0.2.8","vendorValue":9007199254740993}`,
			update: func(ctx context.Context, c *dns.Client) error {
				_, err := c.GlobalViews().Update(ctx, 22, dns.ViewChanges{DeviceRegistrationEnabled: bam.Set(false)})
				return err
			},
		},
		{
			name: "zone dynamic updates", path: "/api/v2/zones/33", field: "dynamicUpdateEnabled",
			original: `{"id":33,"type":"Zone","name":"example.test","dynamicUpdateEnabled":true,"deploymentEnabled":true,"vendorValue":9007199254740993}`,
			update: func(ctx context.Context, c *dns.Client) error {
				_, err := c.GlobalZones().Update(ctx, 33, dns.ZoneChanges{DynamicUpdateEnabled: bam.Set(false)})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var writes atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path != tt.path {
					t.Errorf("path = %s, want %s", r.URL.Path, tt.path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if r.Method == http.MethodGet {
					_, _ = io.WriteString(w, tt.original)
					return
				}
				if r.Method != http.MethodPut {
					t.Errorf("unexpected method %s", r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				writes.Add(1)
				var body, original bam.Document
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				if err := json.Unmarshal([]byte(tt.original), &original); err != nil {
					t.Error(err)
					return
				}
				for key, want := range original {
					if key == tt.field {
						want = json.RawMessage("false")
					}
					if string(body[key]) != string(want) {
						t.Errorf("field %s = %s, want %s", key, body[key], want)
					}
				}
				_ = json.NewEncoder(w).Encode(body)
			}))
			t.Cleanup(server.Close)
			if err := tt.update(context.Background(), dns.New(newClient(t, server))); err != nil {
				t.Fatal(err)
			}
			if writes.Load() != 1 {
				t.Fatalf("writes = %d, want one complete PUT", writes.Load())
			}
		})
	}
}
