package bam_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
)

func TestCollectionFixedFilterAndQuotedValues(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		f := r.URL.Query().Get("filter")
		for _, part := range []string{`configuration.id:eq(9)`, `name:eq("O'Reilly")`, `state:eq("RESERVED")`, `type:'IPv4Address'`} {
			if !strings.Contains(f, part) {
				t.Errorf("filter %q omits %q", f, part)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"count":0,"data":[]}`)
	}))
	t.Cleanup(server.Close)
	c, err := bam.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	addresses := bam.NewCollection[bam.Resource](c, "addresses").WithFilter(bam.MatchID("configuration.id", 9))
	_, err = addresses.Page(context.Background(), bam.ListOptions{Filter: `type:'IPv4Address'`, Where: bam.And(bam.Match("name", "O'Reilly"), bam.Match("state", "RESERVED"))})
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []bam.Filter{bam.Match("name)", "x"), bam.Match("name", "both'\"quotes"), bam.Match("name", "bad\nvalue"), bam.MatchID("configuration.id", 0)} {
		if _, err := addresses.Page(context.Background(), bam.ListOptions{Where: invalid}); err == nil {
			t.Fatal("invalid filter succeeded")
		}
	}
	if requests.Load() != 1 {
		t.Fatal("invalid filters reached server")
	}
}
