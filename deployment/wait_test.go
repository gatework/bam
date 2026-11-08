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
	"time"

	"github.com/gatework/bam"
	"github.com/gatework/bam/deployment"
)

func TestTasksWaitCompletion(t *testing.T) {
	t.Parallel()
	for _, state := range []deployment.State{deployment.StateCompleted, deployment.StateCompletedWithWarnings} {
		t.Run(strings.ToLower(string(state)), func(t *testing.T) {
			t.Parallel()
			states := []deployment.State{deployment.StatePending, deployment.StateQueued, deployment.StateRunning, state}
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				index := int(requests.Add(1)) - 1
				if r.Method != http.MethodGet || r.URL.Path != "/api/v2/deployments/77" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if index >= len(states) {
					t.Error("polled after completion")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/hal+json")
				_ = json.NewEncoder(w).Encode(deployment.Task{ID: 77, Type: deployment.KindFull, State: states[index]})
			}))
			t.Cleanup(server.Close)
			task, err := newClient(t, server).Tasks().Wait(context.Background(), 77,
				deployment.WaitOptions{Interval: time.Millisecond, Timeout: 5 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			if task.ID != 77 || task.State != state || requests.Load() != int64(len(states)) {
				t.Fatalf("task = %+v, requests = %d", task, requests.Load())
			}
		})
	}
}

func TestTasksWaitFailureAndUnknownState(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		state   deployment.State
		failure bool
	}{
		{"failed", deployment.StateFailed, true},
		{"cancelled", deployment.StateCancelled, true},
		{"unknown", deployment.StateUnknown, false},
		{"future state", "NEW_SERVER_STATE", false},
		{"missing state", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "application/hal+json")
				_ = json.NewEncoder(w).Encode(deployment.Task{
					ID: 77, Type: deployment.KindFull, State: tt.state, Message: "server diagnostic detail",
				})
			}))
			t.Cleanup(server.Close)
			task, err := newClient(t, server).Tasks().Wait(context.Background(), 77, deployment.WaitOptions{})
			var failure *deployment.FailureError
			var stateError *deployment.StateError
			if tt.failure {
				if !errors.As(err, &failure) || failure.Task != task {
					t.Fatalf("error = %v, want failure retaining task %+v", err, task)
				}
			} else if !errors.As(err, &stateError) || stateError.Task != task {
				t.Fatalf("error = %v, want unrecognized state retaining task %+v", err, task)
			}
			if task.State != tt.state || task.Message != "server diagnostic detail" || requests.Load() != 1 {
				t.Fatalf("task = %+v, requests = %d", task, requests.Load())
			}
			if strings.Contains(err.Error(), task.Message) {
				t.Errorf("routine error exposes server diagnostic message: %v", err)
			}
		})
	}
}

func TestTasksWaitCancellationRetainsObservation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var requests atomic.Int64
	polling := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("cancellation must not mutate server task: %s", r.Method)
		}
		if requests.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/hal+json")
			_, _ = io.WriteString(w, `{"id":77,"type":"FullDeployment","state":"RUNNING","percentComplete":50}`)
			return
		}
		close(polling)
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	c := newClient(t, server)
	type result struct {
		task deployment.Task
		err  error
	}
	done := make(chan result, 1)
	go func() {
		task, err := c.Tasks().Wait(ctx, 77, deployment.WaitOptions{Interval: time.Millisecond})
		done <- result{task: task, err: err}
	}()
	select {
	case <-polling:
		cancel()
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("wait did not start polling")
	}
	select {
	case got := <-done:
		if !errors.Is(got.err, context.Canceled) || got.task.State != deployment.StateRunning || got.task.PercentComplete != 50 {
			t.Fatalf("cancelled wait = %+v, %v", got.task, got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("wait did not stop on cancellation")
	}
	if requests.Load() != 2 {
		t.Errorf("requests = %d, want two observations and no task mutation", requests.Load())
	}
}

func TestTasksWaitTimeoutAndPollingError(t *testing.T) {
	t.Parallel()
	t.Run("timeout", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		t.Cleanup(server.Close)
		_, err := newClient(t, server).Tasks().Wait(context.Background(), 77,
			deployment.WaitOptions{Timeout: 20 * time.Millisecond})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v, want deadline exceeded", err)
		}
	})
	t.Run("http failure retains observation", func(t *testing.T) {
		t.Parallel()
		var requests atomic.Int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/hal+json")
			if requests.Add(1) == 1 {
				_, _ = io.WriteString(w, `{"id":77,"type":"FullDeployment","state":"CANCELLING"}`)
				return
			}
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"status":403,"code":"PermissionDenied"}`)
		}))
		t.Cleanup(server.Close)
		task, err := newClient(t, server).Tasks().Wait(context.Background(), 77,
			deployment.WaitOptions{Interval: time.Millisecond})
		var api *bam.APIError
		if !errors.As(err, &api) || api.StatusCode != http.StatusForbidden || task.State != deployment.StateCancelling {
			t.Fatalf("wait = %+v, %v", task, err)
		}
		if requests.Load() != 2 {
			t.Errorf("requests = %d, want no retries", requests.Load())
		}
	})
}

func TestTasksWaitRejectsInvalidOptions(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	tasks := newClient(t, server).Tasks()
	tests := []struct {
		name    string
		options deployment.WaitOptions
	}{
		{"negative interval", deployment.WaitOptions{Interval: -time.Second}},
		{"negative timeout", deployment.WaitOptions{Timeout: -time.Second}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tasks.Wait(context.Background(), 77, tt.options); err == nil {
				t.Fatal("invalid options succeeded")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tasks.Wait(ctx, 77, deployment.WaitOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancelled wait = %v, want context canceled", err)
	}
	if requests.Load() != 0 {
		t.Errorf("invalid or cancelled wait made %d requests", requests.Load())
	}
}
