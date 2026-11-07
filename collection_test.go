package bam_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
)

func TestFullReplacementPreservesUnknownFields(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/configurations/17" || r.URL.RawQuery != "" {
			t.Error("full document used a projection or an incorrect resource path")
		}
		if r.Method == http.MethodGet {
			_, _ = fmt.Fprint(w, `{"id":17,"type":"Configuration","name":"old","future":{"number":9007199254740993,"enabled":false,"nullable":null},"_links":{"self":{"href":"/api/v2/configurations/17"}}}`)
			return
		}
		var body bam.Document
		if r.Method != http.MethodPut || json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Error("invalid replacement")
		}
		if string(body["future"]) != `{"number":9007199254740993,"enabled":false,"nullable":null}` || string(body["name"]) != `"new"` {
			t.Errorf("unknown fields or edit were lost: %s", body["future"])
		}
		if _, present := body["_links"]; present {
			t.Error("HAL metadata was sent in a replacement")
		}
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	})
	collection := bam.NewCollection[bam.Resource](client, "configurations")
	document, err := collection.Document(context.Background(), 17)
	if err != nil {
		t.Fatal(err)
	}
	updated := document.Clone()
	if err := updated.Set("name", "new"); err != nil {
		t.Fatal(err)
	}
	result, err := collection.Replace(context.Background(), 17, updated)
	if err != nil || result.Name != "new" {
		t.Fatalf("Replace = %+v, %v", result, err)
	}
	if string(document["name"]) != `"old"` || updated["_links"] == nil {
		t.Fatal("replacement mutated caller-owned data")
	}
	updated["future"][0] = '['
	if document["future"][0] != '{' {
		t.Fatal("Clone shares raw JSON storage")
	}
}

func TestInvalidResourceOperationsNeverSendRequests(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	collection := bam.NewCollection[bam.Resource](client, "configurations")
	ctx := context.Background()
	for _, document := range []bam.Document{
		nil,
		{"name": json.RawMessage(`"partial"`)},
		{"id": json.RawMessage(`2`), "type": json.RawMessage(`"Configuration"`)},
		{"id": json.RawMessage(`1`)},
	} {
		if _, err := collection.Replace(ctx, 1, document); err == nil {
			t.Error("incomplete or mismatching replacement accepted")
		}
	}
	if _, err := collection.Get(ctx, 0); err == nil {
		t.Error("zero resource ID accepted")
	}
	if _, err := bam.ChildCollection[bam.Resource](client, "zones", -1, "resourceRecords").Create(ctx, map[string]string{"type": "HostRecord"}); err == nil {
		t.Error("negative scope accepted")
	}
	if _, err := bam.NewCollection[bam.Resource](client, "../sessions").Page(ctx, bam.ListOptions{}); err == nil {
		t.Error("invalid collection name accepted")
	}
	if calls.Load() != 0 {
		t.Fatal("invalid operation reached server")
	}
}

func TestResourceIdentityMustMatchRequestedID(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"id":2,"type":"Configuration"}`)
	})
	collection := bam.NewCollection[bam.Resource](client, "configurations")
	if _, err := collection.Get(context.Background(), 1); err == nil {
		t.Fatal("different resource returned as requested resource")
	}
	if _, err := collection.Document(context.Background(), 1); err == nil {
		t.Fatal("different document returned as requested document")
	}
}
