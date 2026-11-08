package deployment_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gatework/bam"
)

func TestServersDiscovery(t *testing.T) {
	t.Parallel()
	paths := make(chan string, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		if r.Method != http.MethodGet {
			t.Errorf("server discovery mutated state: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/hal+json")
		if r.URL.Path == "/api/v2/servers/42" {
			_, _ = io.WriteString(w, `{"id":42,"type":"HAServer","name":"primary-dns","configuration":{"id":11},"profile":"LEGACY","state":"ENABLED","connected":true}`)
			return
		}
		_, _ = io.WriteString(w, `{"count":0,"data":[]}`)
	}))
	t.Cleanup(server.Close)
	c := newClient(t, server)
	if _, err := c.GlobalServers().Page(context.Background(), bam.ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Servers(11).Page(context.Background(), bam.ListOptions{}); err != nil {
		t.Fatal(err)
	}
	target, err := c.Servers(11).Get(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if target.Configuration.ID != 11 || target.Name != "primary-dns" || target.Profile != "LEGACY" || !target.Connected {
		t.Fatalf("server = %+v", target)
	}
	for _, expected := range []string{"/api/v2/servers", "/api/v2/configurations/11/servers", "/api/v2/servers/42"} {
		if got := <-paths; got != expected {
			t.Errorf("path = %s, want %s", got, expected)
		}
	}
}
