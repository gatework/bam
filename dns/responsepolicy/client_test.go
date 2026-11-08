package responsepolicy_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/deployment"
	"github.com/gatework/bam/dns"
	"github.com/gatework/bam/dns/responsepolicy"
)

func TestLocalPolicyConfigurationAndExplicitDeployment(t *testing.T) {
	t.Parallel()
	var deployments atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body bam.Document
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v2/configurations/1/responsePolicies":
			if string(body["type"]) != `"ResponsePolicy"` || string(body["policyType"]) != `"BLOCKLIST"` {
				t.Errorf("policy body=%v", body)
			}
			w.WriteHeader(201)
			_, _ = io.WriteString(w, `{"id":10,"type":"ResponsePolicy","name":"deny","policyType":"BLOCKLIST"}`)
		case "POST /api/v2/responsePolicies/10/policyItems":
			if string(body["name"]) != `"bad.example"` {
				t.Errorf("item body=%v", body)
			}
			w.WriteHeader(201)
			_, _ = io.WriteString(w, `{"id":20,"type":"ResponsePolicyItem","name":"bad.example"}`)
		case "GET /api/v2/policyItems/20":
			_, _ = io.WriteString(w, `{"id":20,"type":"ResponsePolicyItem","name":"bad.example","future":{"large":9007199254740993}}`)
		case "PUT /api/v2/policyItems/20":
			if string(body["name"]) != `"worse.example"` || string(body["future"]) != `{"large":9007199254740993}` {
				t.Errorf("item update lost fields: %v", body)
			}
			_ = json.NewEncoder(w).Encode(body)
		case "POST /api/v2/views/2/zones":
			if string(body["type"]) != `"ResponsePolicyZone"` || string(body["responsePolicyZoneType"]) != `"LOCAL"` || string(body["responsePolicy"]) != `{"id":10}` {
				t.Errorf("zone body=%v", body)
			}
			w.WriteHeader(201)
			_, _ = io.WriteString(w, `{"id":30,"type":"ResponsePolicyZone","name":"deny.rpz","responsePolicyZoneType":"LOCAL","responsePolicy":{"id":10}}`)
		case "POST /api/v2/zones/30/deploymentRoles":
			if string(body["type"]) != `"DNSDeploymentRole"` || string(body["roleType"]) != `"PRIMARY"` {
				t.Errorf("role body=%v", body)
			}
			w.WriteHeader(201)
			_, _ = io.WriteString(w, `{"id":40,"type":"DNSDeploymentRole","roleType":"PRIMARY"}`)
		case "POST /api/v2/servers/5/deployments":
			deployments.Add(1)
			if string(body["type"]) != `"FullDeployment"` || string(body["service"]) != `"DNS"` {
				t.Errorf("deployment body=%v", body)
			}
			w.WriteHeader(201)
			_, _ = io.WriteString(w, `{"id":50,"type":"FullDeployment","state":"QUEUED"}`)
		case "GET /api/v2/deployments/50":
			_, _ = io.WriteString(w, `{"id":50,"type":"FullDeployment","state":"COMPLETED"}`)
		case "DELETE /api/v2/policyItems/20":
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	c, err := bam.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	policies := responsepolicy.New(c)
	policy, err := policies.Policies(1).Create(ctx, responsepolicy.Spec{Name: "deny", Action: responsepolicy.Blocklist})
	if err != nil {
		t.Fatal(err)
	}
	item, err := policies.Items(policy.ID).Create(ctx, responsepolicy.ItemSpec{Name: "bad.example"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := policies.Items(policy.ID).Update(ctx, item.ID, responsepolicy.ItemChanges{Name: bam.Set("worse.example")})
	if err != nil || updated.Name != "worse.example" {
		t.Fatalf("update=%+v %v", updated, err)
	}
	zone, err := policies.Zones(2).Create(ctx, responsepolicy.ZoneSpec{Name: "deny.rpz", PolicyID: policy.ID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = dns.New(c).ZoneRoles(zone.ID).Create(ctx, dns.RoleSpec{RoleType: dns.RolePrimary, Interfaces: []dns.RoleInterface{{ID: 6, Type: "NetworkInterface"}}})
	if err != nil {
		t.Fatal(err)
	}
	if deployments.Load() != 0 {
		t.Fatal("configuration implicitly submitted deployment")
	}
	task, err := deployment.New(c).ForServer(5).Full(ctx, deployment.DNS)
	if err != nil {
		t.Fatal(err)
	}
	task, err = deployment.New(c).Tasks().Wait(ctx, task.ID, deployment.WaitOptions{})
	if err != nil || task.State != deployment.StateCompleted {
		t.Fatalf("deployment=%+v %v", task, err)
	}
	if err := policies.Items(policy.ID).Delete(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if deployments.Load() != 1 {
		t.Fatal("unexpected extra deployment")
	}
}

func TestInvalidPolicyDoesNotMutate(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid policy reached server")
		w.WriteHeader(400)
	}))
	t.Cleanup(server.Close)
	c, err := bam.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	policies := responsepolicy.New(c).Policies(1)
	for _, spec := range []responsepolicy.Spec{{Name: "deny", Action: responsepolicy.Redirect}, {Name: "deny", Action: "INVENTED"}, {Action: responsepolicy.Blocklist}} {
		if _, err := policies.Create(context.Background(), spec); err == nil {
			t.Fatal("invalid policy accepted")
		}
	}
}
