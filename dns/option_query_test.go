package dns_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dns"
)

func TestOptions_Find_ServerScopes(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		build func() (dns.OptionQuery, error)
		id    int64
	}{
		{"all servers", func() (dns.OptionQuery, error) { return dns.NewOptionQuery("allow-query", 0) }, 10},
		{"specific server", func() (dns.OptionQuery, error) { return dns.NewOptionQuery("allow-query", 20) }, 11},
		{"definition", func() (dns.OptionQuery, error) { return dns.NewOptionQueryByDefinition(30, 20) }, 11},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v2/zones/7/deploymentOptions" || r.URL.Query().Has("fields") {
					t.Errorf("unexpected lookup request %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Query().Get("offset") == "1" {
					_, _ = io.WriteString(w, `{"count":1,"data":[{"id":11,"type":"DNSOption","name":"allow-query","definition":{"id":30},"serverScope":{"id":20},"value":[]}]}`)
				} else {
					_, _ = io.WriteString(w, `{"count":1,"data":[{"id":10,"type":"DNSOption","name":"allow-query","definition":{"id":30},"serverScope":null,"value":[]}],"_links":{"next":{"href":"?offset=1"}}}`)
				}
			}))
			t.Cleanup(server.Close)
			query, err := tt.build()
			if err != nil {
				t.Fatal(err)
			}
			got, err := dns.New(newClient(t, server)).ZoneOptions(7).Find(t.Context(), query)
			if err != nil || got.ID != tt.id {
				t.Fatalf("option=%+v error=%v", got, err)
			}
		})
	}
}

func TestOptions_Find_LaterFailures(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"duplicate", "failure", "missing"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if name == "missing" {
					_, _ = io.WriteString(w, `{"count":0,"data":[]}`)
				} else if r.URL.Query().Get("offset") == "1" && name == "failure" {
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = io.WriteString(w, `{"message":"unavailable"}`)
				} else {
					_, _ = io.WriteString(w, `{"count":1,"data":[{"id":10,"type":"DNSOption","name":"allow-query","serverScope":null,"value":[]}],"_links":{"next":{"href":"?offset=1"}}}`)
				}
			}))
			t.Cleanup(server.Close)
			query, err := dns.NewOptionQuery("allow-query", 0)
			if err != nil {
				t.Fatal(err)
			}
			got, err := dns.New(newClient(t, server)).ZoneOptions(7).Find(t.Context(), query)
			if got.ID != 0 || err == nil {
				t.Fatalf("incomplete lookup returned %+v, %v", got, err)
			}
			var api *bam.APIError
			if name == "duplicate" && !errors.Is(err, bam.ErrAmbiguous) || name == "missing" && !errors.Is(err, bam.ErrNotFound) || name == "failure" && !errors.As(err, &api) {
				t.Fatalf("unexpected error %v", err)
			}
		})
	}
}

func TestNewOptionQuery_InvalidInput(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		build func() (dns.OptionQuery, error)
	}{
		{"empty name", func() (dns.OptionQuery, error) { return dns.NewOptionQuery("", 0) }},
		{"blank name", func() (dns.OptionQuery, error) { return dns.NewOptionQuery(" ", 0) }},
		{"name negative scope", func() (dns.OptionQuery, error) { return dns.NewOptionQuery("allow-query", -1) }},
		{"zero definition", func() (dns.OptionQuery, error) { return dns.NewOptionQueryByDefinition(0, 0) }},
		{"negative definition", func() (dns.OptionQuery, error) { return dns.NewOptionQueryByDefinition(-1, 0) }},
		{"definition negative scope", func() (dns.OptionQuery, error) { return dns.NewOptionQueryByDefinition(30, -1) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			query, err := tt.build()
			if err == nil || query.IsValid() || query != (dns.OptionQuery{}) {
				t.Fatalf("invalid constructor returned %+v, %v", query, err)
			}
		})
	}
}

func TestOptions_Find_UnsetQuery(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("unset query made a request")
	}))
	t.Cleanup(server.Close)
	if _, err := dns.New(newClient(t, server)).ZoneOptions(7).Find(t.Context(), dns.OptionQuery{}); err == nil {
		t.Fatal("accepted unset query")
	}
}
