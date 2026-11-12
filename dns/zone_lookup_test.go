package dns_test

import (
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

func TestClient_LookupZone_Traversal(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || !strings.Contains(r.URL.Query().Get("filter"), `type:eq("Zone")`) {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/views/7/zones":
			if r.URL.Query().Get("offset") == "1" {
				_, _ = io.WriteString(w, `{"count":1,"data":[{"id":10,"type":"Zone","absoluteName":"example.test","view":{"id":7}}]}`)
			} else {
				_, _ = io.WriteString(w, `{"count":1,"data":[{"id":9,"type":"Zone","absoluteName":"other.test","view":{"id":7}}],"_links":{"next":{"href":"?offset=1"}}}`)
			}
		case "/api/v2/zones/10/zones":
			_, _ = io.WriteString(w, `{"count":1,"data":[{"id":11,"type":"Zone","absoluteName":"dev.ops.example.test","view":{"id":7}}]}`)
		default:
			t.Errorf("traversed an unrelated branch: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	zone, err := dns.New(newClient(t, server)).LookupZone(t.Context(), 7, "DEV.Ops.Example.Test.")
	if err != nil || zone.ID != 11 || calls.Load() != 3 {
		t.Fatalf("zone=%+v err=%v calls=%d", zone, err, calls.Load())
	}
}

func TestClient_LookupZone_BrokenHierarchy(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, root, child string
		want              error
		protocol          bool
	}{
		{"missing", `{"count":0,"data":[]}`, "", bam.ErrNotFound, false},
		{"duplicate", `{"count":2,"data":[{"id":1,"type":"Zone","absoluteName":"ops.example.test","view":{"id":7}},{"id":2,"type":"Zone","absoluteName":"ops.example.test","view":{"id":7}}]}`, "", bam.ErrAmbiguous, false},
		{"wrong view", `{"count":1,"data":[{"id":1,"type":"Zone","absoluteName":"ops.example.test","view":{"id":8}}]}`, "", nil, true},
		{"cycle", `{"count":1,"data":[{"id":1,"type":"Zone","absoluteName":"example.test","view":{"id":7}}]}`, `{"count":1,"data":[{"id":1,"type":"Zone","absoluteName":"example.test","view":{"id":7}}]}`, nil, true},
		{"missing name", `{"count":1,"data":[{"id":1,"type":"Zone","view":{"id":7}}]}`, "", nil, true},
		{"two root dots", `{"count":1,"data":[{"id":1,"type":"Zone","absoluteName":"ops.example.test..","view":{"id":7}}]}`, "", nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				body := tt.root
				if r.URL.Path != "/api/v2/views/7/zones" {
					body = tt.child
				}
				_, _ = io.WriteString(w, body)
			}))
			t.Cleanup(server.Close)
			_, err := dns.New(newClient(t, server)).LookupZone(t.Context(), 7, "ops.example.test")
			if tt.protocol {
				if _, ok := errors.AsType[*bam.ProtocolError](err); !ok {
					t.Fatalf("want protocol error, got %v", err)
				}
			} else if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestClient_LookupZone_InvalidInput(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, input string
		viewID      int64
	}{
		{"zero view", "example.test", 0},
		{"negative view", "example.test", -1},
		{"empty name", "", 7},
		{"root only", ".", 7},
		{"surrounding space", " example.test", 7},
		{"empty label", "example..test", 7},
		{"two root dots", "example.test..", 7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("invalid lookup made an HTTP request")
			}))
			t.Cleanup(server.Close)
			client := dns.New(newClient(t, server))
			if _, err := client.LookupZone(t.Context(), tt.viewID, tt.input); err == nil {
				t.Errorf("accepted invalid lookup %+v", tt)
			}
		})
	}
}
