package deployment_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gatework/bam"
	"github.com/gatework/bam/deployment"
)

func ExampleTasks_Wait() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/hal+json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":77,"type":"FullDeployment","state":"QUEUED"}`)
			return
		}
		_, _ = io.WriteString(w, `{"id":77,"type":"FullDeployment","state":"COMPLETED"}`)
	}))
	defer server.Close()
	client, err := bam.New(server.URL)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer client.CloseIdleConnections()
	ctx := context.Background()
	deployments := deployment.New(client)
	task, err := deployments.ForServer(42).Full(ctx, deployment.DNS)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("submitted:", task.State)
	task, err = deployments.Tasks().Wait(ctx, task.ID, deployment.WaitOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("observed:", task.State)
	// Output:
	// submitted: QUEUED
	// observed: COMPLETED
}
