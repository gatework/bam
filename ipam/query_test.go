package ipam_test

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/ipam"
)

func TestQueryIteratorsSnapshotOptionsAndSupportConcurrentReuse(t *testing.T) {
	t.Parallel()
	mac, err := ipam.ParseMAC("aa:bb:cc:00:11:22")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, tt := range []struct {
		name, path string
		iterator   func(*ipam.Client, bam.ListOptions) func() error
	}{
		{"address search", "/api/v2/addresses", func(c *ipam.Client, options bam.ListOptions) func() error {
			return consume(c.GlobalAddresses().Search(ctx, ipam.AddressQuery{ConfigurationID: 7, State: ipam.StateStatic}, options))
		}},
		{"containing networks", "/api/v2/networks", func(c *ipam.Client, options bam.ListOptions) func() error {
			return consume(c.NetworksContaining(ctx, 7, netip.MustParseAddr("192.0.2.8"), options))
		}},
		{"MAC associations", "/api/v2/macAddresses/19/addresses", func(c *ipam.Client, options bam.ListOptions) func() error {
			return consume(c.AddressesByMAC(ctx, 7, mac, options))
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			var queryMu sync.Mutex
			var firstQuery string
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/api/v2/configurations/7/macAddresses" {
					_, _ = fmt.Fprint(w, `{"count":1,"data":[{"id":19,"type":"MACAddress","address":"aa:bb:cc:00:11:22"}]}`)
					return
				}
				if r.URL.Path != tt.path {
					t.Errorf("path=%s, want %s", r.URL.Path, tt.path)
				}
				query := r.URL.Query()
				if query.Get("fields") != "id,name" || query.Get("orderBy") != "id" {
					t.Errorf("iterator retained caller's mutable slices: %s", r.URL.RawQuery)
				}
				queryMu.Lock()
				if firstQuery == "" {
					firstQuery = query.Encode()
				} else if query.Encode() != firstQuery {
					t.Errorf("reusing iterator changed query: %s; original %s", query.Encode(), firstQuery)
				}
				queryMu.Unlock()
				calls.Add(1)
				_, _ = fmt.Fprint(w, `{"count":0,"data":[]}`)
			})
			fields, order := []string{"id", "name"}, []string{"id"}
			run := tt.iterator(client, bam.ListOptions{Where: bam.Match("name", "workstation"), Fields: fields, OrderBy: order})
			fields[0], order[0] = "configuration", "name"
			for range 2 {
				if err := run(); err != nil {
					t.Fatal(err)
				}
			}
			const workers = 8
			errors := make(chan error, workers)
			start := make(chan struct{})
			var workersDone sync.WaitGroup
			workersDone.Add(workers)
			for range workers {
				go func() {
					defer workersDone.Done()
					<-start
					errors <- run()
				}()
			}
			close(start)
			workersDone.Wait()
			close(errors)
			for err := range errors {
				if err != nil {
					t.Error(err)
				}
			}
			if calls.Load() != workers+2 {
				t.Fatalf("queries=%d, want %d", calls.Load(), workers+2)
			}
		})
	}
}

func consume[T any](sequence iter.Seq2[T, error]) func() error {
	return func() error {
		for _, err := range sequence {
			if err != nil {
				return err
			}
		}
		return nil
	}
}
