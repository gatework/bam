package dhcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dhcp"
)

func TestMACPoolMembershipOnlyChangesRelationship(t *testing.T) {
	t.Parallel()
	var links, unlinks atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/configurations/7/macPools":
			var body struct {
				Type                     string
				InstantDeploymentEnabled *bool
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Type != "MACPool" || body.InstantDeploymentEnabled == nil || *body.InstantDeploymentEnabled {
				t.Errorf("new pool must require explicit deployment: %+v", body)
			}
			_, _ = fmt.Fprint(w, `{"id":8,"type":"MACPool","name":"workstations","instantDeploymentEnabled":false}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/macPools/8/macAddresses":
			var body struct {
				ID   int64
				Type string
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.ID != 19 || body.Type != "MACAddress" {
				t.Errorf("link=%+v", body)
			}
			links.Add(1)
			_, _ = fmt.Fprint(w, `{"id":19,"type":"MACAddress","address":"aa:bb:cc:00:11:22"}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v2/macPools/8/macAddresses/19":
			unlinks.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("membership changed wrong resource: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	ctx := context.Background()
	if _, err := client.MACPools(7).Create(ctx, dhcp.MACPoolSpec{Name: "workstations"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.MACMembers(8).Link(ctx, 19); err != nil {
		t.Fatal(err)
	}
	if err := client.MACMembers(8).Unlink(ctx, 19); err != nil {
		t.Fatal(err)
	}
	if links.Load() != 1 || unlinks.Load() != 1 {
		t.Fatalf("links=%d unlinks=%d", links.Load(), unlinks.Load())
	}
}

func TestMACPoolLinkRejectsDifferentMember(t *testing.T) {
	t.Parallel()
	for name, reply := range map[string]string{
		"different id":   `{"id":20,"type":"MACAddress"}`,
		"different type": `{"id":19,"type":"MACPool"}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, reply)
			})
			_, err := client.MACMembers(8).Link(context.Background(), 19)
			if _, ok := errors.AsType[*bam.ProtocolError](err); !ok {
				t.Fatalf("link returned %v, want protocol error", err)
			}
		})
	}
}
