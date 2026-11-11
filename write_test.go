package bam_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gatework/bam"
)

func TestCollectionUpdatePreservesDocumentAndExplicitValues(t *testing.T) {
	t.Parallel()
	const original = `{"id":7,"type":"Fixture","name":"old","enabled":true,"ttl":10,"nullable":"value","unknown":{"big":9007199254740993},"_links":{"self":{"href":"/api/v2/fixtures/7"}}}`
	methods := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods <- r.Method
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			if r.Header.Get("x-bcn-change-control-comment") != "" {
				t.Error("write header leaked into read")
			}
			_, _ = io.WriteString(w, original)
			return
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("x-bcn-change-control-comment") != "change 17" {
			t.Error("missing change comment")
		}
		var body bam.Document
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		for k, want := range map[string]string{"enabled": "false", "ttl": "0", "nullable": "null", "name": `"old"`, "unknown": `{"big":9007199254740993}`} {
			if string(body[k]) != want {
				t.Errorf("%s = %s, want %s", k, body[k], want)
			}
		}
		if _, ok := body["_links"]; ok {
			t.Error("HAL links included in replacement")
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)
	c, err := bam.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = bam.NewCollection[bam.Document](c, "fixtures").Update(context.Background(), 7, func(d bam.Document) error {
		return errors.Join(bam.Set(false).Apply(d, "enabled"), bam.Set(0).Apply(d, "ttl"), bam.Clear[string]().Apply(d, "nullable"), bam.Field[string]{}.Apply(d, "name"))
	}, bam.WithChangeControlComment("change 17"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"GET", "PUT"} {
		if got := <-methods; got != want {
			t.Fatalf("method=%s, want %s", got, want)
		}
	}
}

func TestCollectionUpdateRejectsUnsafeReplacement(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body string
		edit       func(bam.Document) error
	}{
		{"redacted", `{"id":7,"type":"Fixture","_redactedFields":["name"]}`, func(bam.Document) error { return nil }},
		{"different type", `{"id":7,"type":"Fixture"}`, func(d bam.Document) error { return d.Set("type", "Other") }},
		{"different id", `{"id":7,"type":"Fixture"}`, func(d bam.Document) error { return d.Set("id", 8) }},
		{"edit failed", `{"id":7,"type":"Fixture"}`, func(bam.Document) error { return errors.New("invalid edit") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Error("unsafe update sent a mutation")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.body)
			}))
			t.Cleanup(server.Close)
			c, err := bam.New(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := bam.NewCollection[bam.Document](c, "fixtures").Update(context.Background(), 7, tc.edit); err == nil {
				t.Fatal("unsafe update succeeded")
			}
		})
	}
}

func TestLinkedCollectionUnlinksNestedResource(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v2/macPools/12/macAddresses/7" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	c, err := bam.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := bam.LinkedCollection[bam.Document](c, "macPools", 12, "macAddresses").Delete(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
}
