package dhcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dhcp"
)

func TestRoleFailoverInterfacesAndUpdate(t *testing.T) {
	t.Parallel()
	var mutations atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/deploymentRoles/8" {
			// interfaces is write-only and is absent from resource GET responses.
			_, _ = fmt.Fprint(w, `{"id":8,"type":"DHCPDeploymentRole","roleType":"PRIMARY","opaque":{"keep":true}}`)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v2/deploymentRoles/8/interfaces" {
			_, _ = fmt.Fprint(w, `{"count":2,"data":[{"id":19,"type":"NetworkInterface","deploymentRoleInterfaceType":"PRIMARY"},{"id":20,"type":"PublishedInterface","deploymentRoleInterfaceType":"SECONDARY"}]}`)
			return
		}
		var body bam.Document
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/networks/7/deploymentRoles":
			var interfaces []dhcp.RoleInterface
			if err := json.Unmarshal(body["interfaces"], &interfaces); err != nil {
				t.Error(err)
			}
			if len(interfaces) != 2 || interfaces[0].DeploymentRoleInterfaceType != "PRIMARY" || interfaces[1].Type != "PublishedInterface" || interfaces[1].DeploymentRoleInterfaceType != "SECONDARY" {
				t.Errorf("interfaces=%+v", interfaces)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/api/v2/deploymentRoles/8":
			if _, exists := body["interfaces"]; exists {
				t.Error("role update unexpectedly overwrote absent write-only interfaces")
			}
			if string(body["roleType"]) != `"NONE"` || string(body["opaque"]) != `{"keep":true}` {
				t.Errorf("update=%s", body)
			}
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		mutations.Add(1)
		body["id"] = json.RawMessage("8")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Error(err)
		}
	})
	ctx := context.Background()
	roles := client.NetworkRoles(7)
	if _, err := roles.Create(ctx, dhcp.RoleSpec{RoleType: dhcp.RolePrimary, Interfaces: []dhcp.RoleInterface{
		{ID: 19, Type: "NetworkInterface", DeploymentRoleInterfaceType: "PRIMARY"},
		{ID: 20, Type: "PublishedInterface", DeploymentRoleInterfaceType: "SECONDARY"},
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := roles.Update(ctx, 8, dhcp.RoleChanges{RoleType: bam.Set(dhcp.RoleNone)}); err != nil {
		t.Fatal(err)
	}
	interfaces, err := client.RoleInterfaces(8).Page(ctx, bam.ListOptions{})
	if err != nil || len(interfaces.Data) != 2 {
		t.Fatalf("interfaces=%+v, %v", interfaces, err)
	}
	if mutations.Load() != 2 {
		t.Fatalf("mutations=%d", mutations.Load())
	}
}
