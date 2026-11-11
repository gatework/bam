package bam_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gatework/bam"
)

func testClient(t *testing.T, handler http.HandlerFunc, options ...bam.Option) *bam.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/hal+json")
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	client, err := bam.New(server.URL, options...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.CloseIdleConnections)
	return client
}

func TestHTTPFailuresAndMalformedSuccess(t *testing.T) {
	for _, test := range []struct {
		name, body, media string
		status            int
		apiError          bool
	}{
		{"unauthorized", `{"status":200,"code":"InvalidAuthorizationToken","message":"secret"}`, "application/json", 401, true},
		{"proxy failure", "<html>secret</html>", "text/html", 502, true},
		{"redirect", "", "text/plain", 302, true},
		{"HTML success", "<html>login</html>", "text/html", 200, false},
		{"null", "null", "application/json", 200, false},
		{"trailing value", `{} {}`, "application/json", 200, false},
		{"missing resource", `{}`, "application/json", 200, false},
		{"absent response", "", "application/json", 204, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", test.media)
				w.WriteHeader(test.status)
				_, _ = fmt.Fprint(w, test.body)
			})
			_, err := bam.NewCollection[bam.Resource](client, "configurations").Get(context.Background(), 1)
			if err == nil || strings.Contains(err.Error(), "secret") {
				t.Fatalf("error = %v", err)
			}
			var apiErr *bam.APIError
			if test.apiError {
				if !errors.As(err, &apiErr) || apiErr.StatusCode != test.status {
					t.Fatalf("expected actual HTTP status %d, got %v", test.status, err)
				}
			} else if _, ok := errors.AsType[*bam.ProtocolError](err); !ok {
				t.Fatalf("expected protocol error, got %v", err)
			}
		})
	}
}

func TestAcceptedWorkflowIsNotACompletedResource(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Location", "/api/v2/workflowRequests/42")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprint(w, `{"id":42,"type":"WorkflowRequest","state":"PENDING"}`)
	})
	resource, err := bam.NewCollection[bam.Resource](client, "networks").Create(context.Background(), map[string]string{"type": "IPv4Network"})
	var accepted *bam.AcceptedError
	if !errors.As(err, &accepted) || resource.ID != 0 || calls.Load() != 1 {
		t.Fatalf("resource = %+v, error = %v, requests = %d", resource, err, calls.Load())
	}
	var pending bam.Resource
	if err := accepted.Resource.Decode(&pending); err != nil || pending.ID != 42 || accepted.Location != "/api/v2/workflowRequests/42" {
		t.Fatalf("pending response was lost: %+v (%v)", accepted, err)
	}
}

func TestRequestCancellationAndResponseLimit(t *testing.T) {
	started := make(chan struct{})
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := bam.NewCollection[bam.Resource](client, "networks").Get(ctx, 1)
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation chain lost: %v", err)
	}
	limited := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"id":1,"type":"Configuration","name":"too long"}`)
	}, bam.WithResponseLimit(16))
	_, err := bam.NewCollection[bam.Resource](limited, "configurations").Get(context.Background(), 1)
	if _, ok := errors.AsType[*bam.ProtocolError](err); !ok {
		t.Fatalf("response limit not enforced: %v", err)
	}
}

func TestRedirectAndForeignURLsNeverReceiveCredentials(t *testing.T) {
	var foreignCalls atomic.Int32
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { foreignCalls.Add(1) }))
	defer foreign.Close()
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", foreign.URL+"/api/v2/configurations")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}, bam.WithBasicAuth("user", "secret"))
	for _, path := range []string{"configurations", foreign.URL + "/api/v2/configurations", "../sessions", "%2e%2e/sessions"} {
		if err := client.Do(context.Background(), bam.Request{Method: http.MethodGet, Path: path}, nil); err == nil {
			t.Fatalf("expected rejection for %q", path)
		}
	}
	if foreignCalls.Load() != 0 {
		t.Fatal("credentials could reach foreign origin")
	}
}

func TestRequestHeadersAndQueryAreOwnedCopies(t *testing.T) {
	headers := http.Header{"authorization": {"Bearer attacker"}, "cookie": {"session=old"}, "X-Bcn-Test": {"true"}}
	originalHeaders := headers.Clone() // Deliberately include noncanonical caller keys.
	query := url.Values{"fields": {"id,type"}}
	transportClient := &http.Client{Timeout: time.Minute}
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		user, token, ok := r.BasicAuth()
		if !ok || user != "user" || token != "api-token" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Bcn-Test") != "true" {
			t.Errorf("incorrect authentication or request headers")
		}
		if r.Header.Get("Content-Type") != "application/merge-patch+json" || r.Header.Get("Accept") != "application/hal+json" || r.URL.Query().Get("fields") != "id,type" {
			t.Error("incorrect JSON contract")
		}
		w.WriteHeader(http.StatusNoContent)
	}, bam.WithHTTPClient(transportClient), bam.WithBasicAuth("user", "api-token"))
	err := client.Do(context.Background(), bam.Request{Method: http.MethodPatch, Path: "sessions/current", Header: headers, Query: query, Body: map[string]string{"state": "LOGGED_OUT"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(headers, originalHeaders) || query.Get("fields") != "id,type" || transportClient.Timeout != time.Minute || transportClient.CheckRedirect != nil {
		t.Fatal("caller-owned settings were modified")
	}
}
