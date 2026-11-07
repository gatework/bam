package deployment_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/deployment"
)

func TestClientSubmissionScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		selectFrom func(*deployment.Client) bam.Collection[bam.Document]
		path       string
	}{
		{
			"server", func(c *deployment.Client) bam.Collection[bam.Document] { return c.ForServer(42) },
			"/api/v2/servers/42/deployments",
		},
		{
			"zone", func(c *deployment.Client) bam.Collection[bam.Document] { return c.ForZone(42) },
			"/api/v2/zones/42/deployments",
		},
		{
			"network", func(c *deployment.Client) bam.Collection[bam.Document] { return c.ForNetwork(42) },
			"/api/v2/networks/42/deployments",
		},
		{
			"selective", func(c *deployment.Client) bam.Collection[bam.Document] { return c.Tasks() },
			"/api/v2/deployments",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// This fixture tests JSON transport, not a particular server schema.
			const body = `{"opaqueSetting":{"enabled":false,"number":9007199254740993},"nullable":null}`
			const task = `{"id":77,"type":"Deployment","state":"SERVER_DEFINED","opaqueSetting":{"number":9007199254740993}}`
			requests := make(chan string, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- r.Method + " " + r.URL.Path
				w.Header().Set("Content-Type", "application/hal+json")
				if r.Method == http.MethodPost {
					got, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					if strings.TrimSpace(string(got)) != body {
						t.Errorf("submission = %s, want %s", got, body)
					}
					w.WriteHeader(http.StatusCreated)
				}
				_, _ = io.WriteString(w, task)
			}))
			t.Cleanup(server.Close)
			client, err := bam.New(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			collection := tt.selectFrom(deployment.New(client))
			created, err := collection.Create(context.Background(), json.RawMessage(body))
			if err != nil {
				t.Fatal(err)
			}
			read, err := collection.Get(context.Background(), 77)
			if err != nil {
				t.Fatal(err)
			}
			for _, expected := range []string{"POST " + tt.path, "GET /api/v2/deployments/77"} {
				if got := <-requests; got != expected {
					t.Errorf("request = %q, want %q", got, expected)
				}
			}
			for _, document := range []bam.Document{created, read} {
				if got := string(document["opaqueSetting"]); got != `{"number":9007199254740993}` {
					t.Errorf("unknown task field changed: %s", got)
				}
			}
		})
	}
}

func TestClientTaskErrorsAndCancellation(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"status":403,"code":"PermissionDenied","message":"denied"}`)
	}))
	t.Cleanup(server.Close)
	client, err := bam.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	tasks := deployment.New(client).Tasks()
	if _, err := tasks.Get(context.Background(), 77); err == nil {
		t.Fatal("Get succeeded despite forbidden response")
	} else {
		var apiError *bam.APIError
		if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusForbidden || apiError.Code != "PermissionDenied" {
			t.Fatalf("Get error = %v, want forbidden APIError", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tasks.Get(ctx, 77); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Get error = %v, want context.Canceled", err)
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("requests = %d, want only the uncancelled request", got)
	}
}

func TestClientSubmissionAcceptedIsNotCompleted(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/hal+json")
		w.Header().Set("Location", "/api/v2/workflowRequests/77")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"id":77,"type":"WorkflowRequest","state":"PENDING"}`)
	}))
	t.Cleanup(server.Close)
	client, err := bam.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = deployment.New(client).ForServer(42).Create(context.Background(), json.RawMessage(`{"type":"Deployment"}`))
	var accepted *bam.AcceptedError
	if !errors.As(err, &accepted) {
		t.Fatalf("Create error = %v, want AcceptedError", err)
	}
	if string(accepted.Resource["type"]) != `"WorkflowRequest"` || accepted.Location != "/api/v2/workflowRequests/77" {
		t.Errorf("accepted resource or location was lost: %#v", accepted)
	}
}
