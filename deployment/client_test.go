package deployment_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/deployment"
)

func TestClientSubmissions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		path   string
		body   string
		submit func(*deployment.Client, ...bam.WriteOption) (deployment.Task, error)
	}{
		{"full dns", "/api/v2/servers/42/deployments", `{"type":"FullDeployment","service":"DNS"}`,
			func(c *deployment.Client, o ...bam.WriteOption) (deployment.Task, error) {
				return c.ForServer(42).Full(context.Background(), deployment.DNS, o...)
			}},
		{"full dhcp", "/api/v2/servers/42/deployments", `{"type":"FullDeployment","service":"DHCPv4"}`,
			func(c *deployment.Client, o ...bam.WriteOption) (deployment.Task, error) {
				return c.ForServer(42).Full(context.Background(), deployment.DHCPv4, o...)
			}},
		{"differential", "/api/v2/servers/42/deployments", `{"type":"DifferentialDeployment","service":"DNS"}`,
			func(c *deployment.Client, o ...bam.WriteOption) (deployment.Task, error) {
				return c.ForServer(42).Differential(context.Background(), deployment.DNS, o...)
			}},
		{"validation", "/api/v2/servers/42/deployments", `{"type":"ValidationDeployment","service":"DHCPv4"}`,
			func(c *deployment.Client, o ...bam.WriteOption) (deployment.Task, error) {
				return c.ForServer(42).Validate(context.Background(), deployment.DHCPv4, o...)
			}},
		{"quick zone", "/api/v2/zones/42/deployments", `{"type":"QuickDeployment"}`,
			func(c *deployment.Client, o ...bam.WriteOption) (deployment.Task, error) {
				return c.ForZone(42).Quick(context.Background(), o...)
			}},
		{"quick network", "/api/v2/networks/42/deployments", `{"type":"QuickDeployment"}`,
			func(c *deployment.Client, o ...bam.WriteOption) (deployment.Task, error) {
				return c.ForNetwork(42).Quick(context.Background(), o...)
			}},
		{"selective", "/api/v2/deployments", `{"type":"SelectiveDeployment","resources":[{"id":12},{"id":34}],"batchMode":"BATCH_BY_SERVER","recordScope":"SPECIFIC","continueOnFailureEnabled":false,"dynamicRecordDeploymentStrategy":"ABORT"}`,
			func(c *deployment.Client, o ...bam.WriteOption) (deployment.Task, error) {
				return c.Selective(context.Background(), deployment.SelectiveSpec{
					Resources: []bam.Reference{{ID: 12}, {ID: 34}}, BatchMode: deployment.BatchByServer,
					RecordScope: deployment.RecordScopeSpecific, DynamicStrategy: deployment.DynamicAbort,
				}, o...)
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != tt.path {
					t.Errorf("request = %s %s, want POST %s", r.Method, r.URL.Path, tt.path)
				}
				if r.Header.Get("x-bcn-change-control-comment") != "change-42" {
					t.Errorf("missing change-control comment")
				}
				var got, expected map[string]any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Error(err)
				}
				if err := json.Unmarshal([]byte(tt.body), &expected); err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("body = %#v, want %#v", got, expected)
				}
				w.Header().Set("Content-Type", "application/hal+json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id": 77, "type": got["type"], "state": "QUEUED", "percentComplete": 0,
				})
			}))
			t.Cleanup(server.Close)
			task, err := tt.submit(newClient(t, server), bam.WithChangeControlComment("change-42"))
			if err != nil {
				t.Fatal(err)
			}
			if task.ID != 77 || task.State != deployment.StateQueued || requests.Load() != 1 {
				t.Fatalf("submission = %+v, requests = %d; submission must not poll", task, requests.Load())
			}
		})
	}
}

func TestClientSubmissionFailures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		body   string
		check  func(*testing.T, error)
	}{
		{"pending workflow", http.StatusAccepted, `{"id":77,"type":"WorkflowRequest","state":"PENDING"}`,
			func(t *testing.T, err error) {
				t.Helper()
				var accepted *bam.AcceptedError
				if !errors.As(err, &accepted) || string(accepted.Resource["type"]) != `"WorkflowRequest"` {
					t.Fatalf("error = %v, want retained pending workflow", err)
				}
			}},
		{"wrong resource", http.StatusCreated, `{"id":77,"type":"WorkflowRequest","state":"PENDING"}`,
			func(t *testing.T, err error) {
				t.Helper()
				if _, ok := errors.AsType[*bam.ProtocolError](err); !ok {
					t.Fatalf("error = %v, want protocol error", err)
				}
			}},
		{"conflict", http.StatusConflict, `{"status":409,"code":"Conflict","message":"already deploying"}`,
			func(t *testing.T, err error) {
				t.Helper()
				var api *bam.APIError
				if !errors.As(err, &api) || api.StatusCode != http.StatusConflict {
					t.Fatalf("error = %v, want conflict", err)
				}
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "application/hal+json")
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			t.Cleanup(server.Close)
			task, err := newClient(t, server).ForServer(42).Full(context.Background(), deployment.DNS)
			tt.check(t, err)
			if task.ID != 0 || requests.Load() != 1 {
				t.Fatalf("task = %+v, requests = %d; failed writes must not be retried", task, requests.Load())
			}
		})
	}
}

//nolint:tparallel // subtests remain sequential so the final request count covers every invalid call
func TestClientRejectsInvalidSubmissions(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	c := newClient(t, server)
	tests := []struct {
		name string
		run  func() (deployment.Task, error)
	}{
		{"missing service", func() (deployment.Task, error) { return c.ForServer(42).Full(context.Background(), "") }},
		{"unsupported service", func() (deployment.Task, error) { return c.ForServer(42).Full(context.Background(), "DHCP") }},
		{"unsupported validation", func() (deployment.Task, error) {
			return c.ForServer(42).Validate(context.Background(), deployment.TFTP)
		}},
		{"invalid server", func() (deployment.Task, error) { return c.ForServer(0).Full(context.Background(), deployment.DNS) }},
		{"invalid zone", func() (deployment.Task, error) { return c.ForZone(-1).Quick(context.Background()) }},
		{"missing selective resources", func() (deployment.Task, error) { return c.Selective(context.Background(), deployment.SelectiveSpec{}) }},
		{"invalid selective resource", func() (deployment.Task, error) {
			return c.Selective(context.Background(), deployment.SelectiveSpec{Resources: []bam.Reference{{ID: 0}}})
		}},
		{"invalid batch mode", func() (deployment.Task, error) {
			return c.Selective(context.Background(), deployment.SelectiveSpec{Resources: []bam.Reference{{ID: 42}}, BatchMode: "INVALID"})
		}},
		{"invalid record scope", func() (deployment.Task, error) {
			return c.Selective(context.Background(), deployment.SelectiveSpec{Resources: []bam.Reference{{ID: 42}}, RecordScope: "INVALID"})
		}},
		{"invalid dynamic strategy", func() (deployment.Task, error) {
			return c.Selective(context.Background(), deployment.SelectiveSpec{Resources: []bam.Reference{{ID: 42}}, DynamicStrategy: "INVALID"})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.run(); err == nil {
				t.Fatal("invalid submission succeeded")
			}
		})
	}
	if requests.Load() != 0 {
		t.Errorf("invalid submissions made %d requests", requests.Load())
	}
}

func TestClient_PageScopes(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, path string
		page       func(*deployment.Client, context.Context) error
	}{
		{"global", "/api/v2/deployments", func(c *deployment.Client, ctx context.Context) error {
			_, err := c.Tasks().Page(ctx, bam.ListOptions{})
			return err
		}},
		{"server", "/api/v2/servers/42/deployments", func(c *deployment.Client, ctx context.Context) error {
			_, err := c.ForServer(42).Page(ctx, bam.ListOptions{})
			return err
		}},
		{"zone", "/api/v2/zones/43/deployments", func(c *deployment.Client, ctx context.Context) error {
			_, err := c.ZoneTasks(43).Page(ctx, bam.ListOptions{})
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != tt.path {
					t.Errorf("request = %s %s, want GET %s", r.Method, r.URL.Path, tt.path)
				}
				w.Header().Set("Content-Type", "application/hal+json")
				_, _ = io.WriteString(w, `{"count":0,"data":[]}`)
			}))
			t.Cleanup(server.Close)
			if err := tt.page(newClient(t, server), t.Context()); err != nil {
				t.Fatal(err)
			}
			if requests.Load() != 1 {
				t.Fatalf("requests=%d", requests.Load())
			}
		})
	}
}

func TestClient_GetAndDocument(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/deployments/77" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/hal+json")
		_, _ = io.WriteString(w, `{"id":77,"type":"FullDeployment","state":"RUNNING","service":"DNS","opaque":{"integer":9007199254740993}}`)
	}))
	t.Cleanup(server.Close)
	client := newClient(t, server)
	task, err := client.ForServer(42).Get(t.Context(), 77)
	if err != nil || task.ID != 77 {
		t.Fatalf("task=%+v error=%v", task, err)
	}
	document, err := client.Tasks().Document(t.Context(), 77)
	if err != nil {
		t.Fatal(err)
	}
	if string(document["opaque"]) != `{"integer":9007199254740993}` {
		t.Fatalf("full task lost opaque fields: %s", document["opaque"])
	}
	if requests.Load() != 2 {
		t.Fatalf("requests=%d", requests.Load())
	}
}

func TestForceZoneTransfer(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-bcn-force-zone-transfer"); got != "true" {
			t.Errorf("force-zone-transfer = %q, want true", got)
		}
		w.Header().Set("Content-Type", "application/hal+json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":77,"type":"FullDeployment","state":"QUEUED"}`)
	}))
	t.Cleanup(server.Close)
	if _, err := newClient(t, server).ForServer(42).Full(context.Background(), deployment.DNS, deployment.WithForceZoneTransfer(true)); err != nil {
		t.Fatal(err)
	}
}

func newClient(t *testing.T, server *httptest.Server) *deployment.Client {
	t.Helper()
	client, err := bam.New(server.URL, bam.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.CloseIdleConnections)
	return deployment.New(client)
}
