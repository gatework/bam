package configuration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/configuration"
)

func TestClientCreatesConfiguration(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/configurations" {
			t.Errorf("request = %s %s, want POST /api/v2/configurations", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body["type"] != "Configuration" || body["name"] != "Warehouse" ||
			body["description"] != "Inventory" || body["configurationGroup"] != "West Coast" {
			t.Errorf("unexpected create body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/hal+json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":11,"type":"Configuration","name":"Warehouse","description":null,"configurationGroup":"West Coast"}`))
	}))
	t.Cleanup(server.Close)
	client, err := bam.New(server.URL, bam.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.CloseIdleConnections)

	got, err := configuration.New(client).Configurations().Create(context.Background(), configuration.Spec{
		Name:               "Warehouse",
		Description:        "Inventory",
		ConfigurationGroup: "West Coast",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 11 || got.Type != "Configuration" || got.Name != "Warehouse" || got.Description != nil ||
		got.ConfigurationGroup == nil || *got.ConfigurationGroup != "West Coast" {
		t.Fatalf("unexpected configuration: %+v", got)
	}
}

func TestSpecRequiresName(t *testing.T) {
	t.Parallel()
	if _, err := json.Marshal(configuration.Spec{}); err == nil {
		t.Fatal("expected an error for a configuration without a name")
	}
}
