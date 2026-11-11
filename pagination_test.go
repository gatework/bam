package bam_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
)

func TestPaginationFollowsLinksAndStopsOnBreak(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("filter") != `name:'a & b'` || r.URL.Query().Get("fields") != "id,type" {
			t.Error("query was corrupted or lost on the next page")
		}
		if r.URL.Query().Get("offset") == "0" {
			// A server may clamp the page size below the requested limit.
			_, _ = fmt.Fprint(w, `{"count":1,"data":[{"id":10,"type":"Configuration"}],"_links":{"next":{"href":"?offset=1&limit=1"}}}`)
		} else {
			_, _ = fmt.Fprint(w, `{"count":1,"data":[{"id":11,"type":"Configuration"}],"_links":{}}`)
		}
	})
	collection := bam.NewCollection[bam.Resource](client, "configurations")
	options := bam.ListOptions{Limit: 100, Filter: `name:'a & b'`, Fields: []string{"id", "type"}}
	var ids []int64
	for resource, err := range collection.All(context.Background(), options) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, resource.ID)
	}
	if !reflect.DeepEqual(ids, []int64{10, 11}) || calls.Load() != 2 {
		t.Fatalf("ids = %v, calls = %d", ids, calls.Load())
	}
	for _, err := range collection.All(context.Background(), options) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if calls.Load() != 3 {
		t.Fatal("early break fetched another page")
	}
}

func TestPaginationRejectsCyclesForeignLinksAndScopeChanges(t *testing.T) {
	var foreignCalls atomic.Int32
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { foreignCalls.Add(1) }))
	defer foreign.Close()
	for _, link := range []string{
		"?offset=0&limit=1000",
		foreign.URL + "/api/v2/configurations?offset=1",
		"/api/v2/users?offset=1",
		"?offset=1&filter=name:'somebody-else'",
	} {
		t.Run(link, func(t *testing.T) {
			var calls atomic.Int32
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_, _ = fmt.Fprintf(w, `{"count":1,"data":[{"id":1}],"_links":{"next":{"href":%q}}}`, link)
			}, bam.WithBearerToken("secret"))
			var gotErr error
			for _, err := range bam.NewCollection[bam.Resource](client, "configurations").All(context.Background(), bam.ListOptions{Filter: "name:'mine'"}) {
				if err != nil {
					gotErr = err
				}
			}
			if gotErr == nil || calls.Load() != 1 {
				t.Fatalf("err = %v, requests = %d", gotErr, calls.Load())
			}
		})
	}
	if foreignCalls.Load() != 0 {
		t.Fatal("pagination leaked credentials")
	}
}

func TestMalformedCollectionsDoNotLookEmpty(t *testing.T) {
	for _, body := range []string{`{}`, `{"count":0,"data":null}`, `{"count":8,"data":[]}`, `{"count":1,"data":[null]}`, `{"count":1,"data":[1]}`} {
		t.Run(body, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, body) })
			_, err := bam.NewCollection[bam.Resource](client, "networks").Page(context.Background(), bam.ListOptions{})
			if _, ok := errors.AsType[*bam.ProtocolError](err); !ok {
				t.Fatalf("malformed collection accepted: %v", err)
			}
		})
	}
}

func TestPaginationCannotIntroduceQueryConstraints(t *testing.T) {
	for _, key := range []string{"filter", "fields", "orderBy"} {
		t.Run(key, func(t *testing.T) {
			var calls atomic.Int32
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_, _ = fmt.Fprintf(w, `{"count":0,"data":[],"_links":{"next":{"href":"?offset=1&%s=changed"}}}`, key)
			})
			var terminal error
			for _, err := range bam.NewCollection[bam.Resource](client, "networks").All(context.Background(), bam.ListOptions{}) {
				terminal = err
			}
			if terminal == nil || calls.Load() != 1 {
				t.Fatalf("constraint introduced: err=%v, calls=%d", terminal, calls.Load())
			}
		})
	}
}

func TestPaginationBudgetAndCancellation(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"count":1,"data":[{"id":1}],"_links":{"next":{"href":"?offset=1"}}}`)
	})
	collection := bam.NewCollection[bam.Resource](client, "networks")
	var terminal error
	for _, err := range collection.All(context.Background(), bam.ListOptions{MaxPages: 1}) {
		terminal = err
	}
	if terminal == nil {
		t.Fatal("page budget silently truncated the list")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, err := range collection.All(ctx, bam.ListOptions{}) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled iteration: %v", err)
		}
	}
}
