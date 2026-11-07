package dhcp_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dhcp"
)

func TestClientCollectionScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		selectFrom func(*dhcp.Client, int64) bam.Collection[bam.Document]
		collection string
		item       string
	}{
		{"network ranges", (*dhcp.Client).Ranges, "networks/42/ranges", "ranges"},
		{"configuration mac pools", (*dhcp.Client).MACPools, "configurations/42/macPools", "macPools"},
		{"configuration mac addresses", (*dhcp.Client).MACAddresses, "configurations/42/macAddresses", "macAddresses"},
		{"configuration options", (*dhcp.Client).ConfigurationOptions, "configurations/42/deploymentOptions", "deploymentOptions"},
		{"block options", (*dhcp.Client).BlockOptions, "blocks/42/deploymentOptions", "deploymentOptions"},
		{"network options", (*dhcp.Client).NetworkOptions, "networks/42/deploymentOptions", "deploymentOptions"},
		{"range options", (*dhcp.Client).RangeOptions, "ranges/42/deploymentOptions", "deploymentOptions"},
		{"address options", (*dhcp.Client).AddressOptions, "addresses/42/deploymentOptions", "deploymentOptions"},
		{"block roles", (*dhcp.Client).BlockRoles, "blocks/42/deploymentRoles", "deploymentRoles"},
		{"network roles", (*dhcp.Client).NetworkRoles, "networks/42/deploymentRoles", "deploymentRoles"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			requests := make(chan string, 3)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- r.Method + " " + r.URL.Path
				w.Header().Set("Content-Type", "application/hal+json")
				if r.Method == http.MethodGet && r.URL.Path == "/api/v2/"+tt.collection {
					_, _ = io.WriteString(w, `{"count":0,"data":[]}`)
					return
				}
				if r.Method == http.MethodPost {
					w.WriteHeader(http.StatusCreated)
				}
				// The selector retains unmodeled resource types without guessing a schema.
				_, _ = io.WriteString(w, `{"id":77,"type":"FixtureResource"}`)
			}))
			t.Cleanup(server.Close)
			client, err := bam.New(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			collection := tt.selectFrom(dhcp.New(client), 42)
			if _, err := collection.Page(context.Background(), bam.ListOptions{}); err != nil {
				t.Fatal(err)
			}
			if _, err := collection.Create(context.Background(), json.RawMessage(`{"name":"example"}`)); err != nil {
				t.Fatal(err)
			}
			if _, err := collection.Get(context.Background(), 77); err != nil {
				t.Fatal(err)
			}
			for _, expected := range []string{
				"GET /api/v2/" + tt.collection,
				"POST /api/v2/" + tt.collection,
				"GET /api/v2/" + tt.item + "/77",
			} {
				if got := <-requests; got != expected {
					t.Errorf("request = %q, want %q", got, expected)
				}
			}
		})
	}
}

func TestClientRangeReplacementRetainsDocument(t *testing.T) {
	t.Parallel()
	const original = `{"id":77,"name":"old","opaqueSetting":{"integer":9007199254740993,"enabled":false,"nullable":null},"range":"192.0.2.10-192.0.2.20","type":"FixtureResource"}`
	const expected = `{"id":77,"name":"new","opaqueSetting":{"integer":9007199254740993,"enabled":false,"nullable":null},"range":"192.0.2.10-192.0.2.20","type":"FixtureResource"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/ranges/77" {
			t.Errorf("path = %q, want global range resource", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/hal+json")
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, original)
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if got := strings.TrimSpace(string(body)); got != expected {
				t.Errorf("replacement = %s, want %s", got, expected)
			}
			_, _ = io.WriteString(w, expected)
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(server.Close)
	client, err := bam.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ranges := dhcp.New(client).Ranges(42)
	document, err := ranges.Document(context.Background(), 77)
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Set("name", "new"); err != nil {
		t.Fatal(err)
	}
	updated, err := ranges.Replace(context.Background(), 77, document)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(updated["opaqueSetting"]); got != string(document["opaqueSetting"]) {
		t.Errorf("unknown JSON field changed: %s", got)
	}
}
