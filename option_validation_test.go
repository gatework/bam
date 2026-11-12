package bam_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dhcp"
	"github.com/gatework/bam/dns"
)

func TestDomainOptionsRejectExplicitNullBeforeMutation(t *testing.T) {
	t.Parallel()
	var null bam.Value
	if err := null.UnmarshalJSON([]byte(" \n null \t")); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, kind string
		write      func(context.Context, *bam.Client, bam.Value) error
	}{
		{"dns create", "DNSOption", func(ctx context.Context, c *bam.Client, value bam.Value) error {
			_, err := dns.New(c).ZoneOptions(7).Create(ctx, dns.OptionSpec{Name: "allow-query", Value: value})
			return err
		}},
		{"dns update", "DNSOption", func(ctx context.Context, c *bam.Client, value bam.Value) error {
			_, err := dns.New(c).ZoneOptions(7).Update(ctx, 8, dns.OptionChanges{Value: bam.Set(value)})
			return err
		}},
		{"dhcp client create", "DHCPv4ClientOption", func(ctx context.Context, c *bam.Client, value bam.Value) error {
			_, err := dhcp.New(c).NetworkOptions(7).Create(ctx, dhcp.ClientOptionSpec{Code: 6, Value: value})
			return err
		}},
		{"dhcp service create", "DHCPv4ServiceOption", func(ctx context.Context, c *bam.Client, value bam.Value) error {
			_, err := dhcp.New(c).NetworkOptions(7).Create(ctx, dhcp.ServiceOptionSpec{Name: "ddns-ttl", Value: value})
			return err
		}},
		{"dhcp update", "DHCPv4ClientOption", func(ctx context.Context, c *bam.Client, value bam.Value) error {
			_, err := dhcp.New(c).NetworkOptions(7).Update(ctx, 8, dhcp.OptionChanges{Value: bam.Set(value)})
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var writes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					writes.Add(1)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(struct {
					ID    int    `json:"id"`
					Type  string `json:"type"`
					Value string `json:"value"`
				}{8, tt.kind, "existing"})
			}))
			t.Cleanup(server.Close)
			client, err := bam.New(server.URL, bam.WithHTTPClient(server.Client()))
			if err != nil {
				t.Fatal(err)
			}
			if err := tt.write(t.Context(), client, null); err == nil {
				t.Error("explicit JSON null was accepted as an option value")
			}
			if writes.Load() != 0 {
				t.Errorf("sent %d mutations with an invalid option value", writes.Load())
			}
		})
	}
}
