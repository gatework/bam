package accesscontrol_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"reflect"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dns"
	"github.com/gatework/bam/dns/accesscontrol"
)

func TestACLOrderBindingAndInheritance(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body bam.Document
		if r.Method == "POST" || r.Method == "PUT" {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v2/configurations/1/accessControlLists":
			var got []string
			if err := json.Unmarshal(body["matchElements"], &got); err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(got, []string{"!192.0.2.0/24", "any"}) {
				t.Errorf("match order=%v", got)
			}
			w.WriteHeader(201)
			_, _ = io.WriteString(w, `{"id":11,"type":"ACL","name":"allowed-clients","matchElements":["!192.0.2.0/24","any"]}`)
		case "GET /api/v2/accessControlLists/11":
			_, _ = io.WriteString(w, `{"id":11,"type":"ACL","name":"allowed-clients","matchElements":["!192.0.2.0/24","any"],"future":false}`)
		case "PUT /api/v2/accessControlLists/11":
			if string(body["future"]) != "false" || string(body["matchElements"]) != `["!192.0.2.8","any"]` {
				t.Errorf("ACL update=%v", body)
			}
			_ = json.NewEncoder(w).Encode(body)
		case "POST /api/v2/views/2/deploymentOptions":
			if string(body["type"]) != `"DNSOption"` || string(body["name"]) != `"allow-query"` || string(body["value"]) != `["allowed-clients"]` {
				t.Errorf("binding=%v", body)
			}
			w.WriteHeader(201)
			_, _ = io.WriteString(w, `{"id":12,"type":"DNSOption","name":"allow-query","value":["allowed-clients"],"serverScope":{"id":5}}`)
		case "GET /api/v2/deploymentOptions/12":
			_, _ = io.WriteString(w, `{"id":12,"type":"DNSOption","name":"allow-query","value":["allowed-clients"],"serverScope":{"id":5},"future":9007199254740993}`)
		case "PUT /api/v2/deploymentOptions/12":
			if string(body["serverScope"]) != "null" || string(body["value"]) != `["allowed-clients"]` || string(body["future"]) != "9007199254740993" {
				t.Errorf("option update=%v", body)
			}
			_ = json.NewEncoder(w).Encode(body)
		case "DELETE /api/v2/deploymentOptions/12", "DELETE /api/v2/accessControlLists/11":
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
	lists := accesscontrol.New(c).Lists(1)
	acl, err := lists.Create(ctx, accesscontrol.Spec{Name: "allowed-clients", MatchElements: []accesscontrol.Element{accesscontrol.Not(accesscontrol.Network(netip.MustParsePrefix("192.0.2.0/24"))), accesscontrol.Any()}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = lists.Update(ctx, acl.ID, accesscontrol.Changes{MatchElements: bam.Set([]accesscontrol.Element{accesscontrol.Not(accesscontrol.Address(netip.MustParseAddr("192.0.2.8"))), accesscontrol.Any()})})
	if err != nil {
		t.Fatal(err)
	}
	spec, err := accesscontrol.AllowQuery(accesscontrol.Named(acl.Name))
	if err != nil {
		t.Fatal(err)
	}
	options := dns.New(c).ViewOptions(2)
	option, err := options.Create(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	_, err = options.Update(ctx, option.ID, dns.OptionChanges{ServerScope: bam.Clear[bam.Reference]()})
	if err != nil {
		t.Fatal(err)
	}
	if err := options.RestoreInheritance(ctx, option.ID); err != nil {
		t.Fatal(err)
	}
	if err := lists.Delete(ctx, acl.ID); err != nil {
		t.Fatal(err)
	}
}

func TestACLInvalidElementsAreRejected(t *testing.T) {
	t.Parallel()
	for _, element := range []accesscontrol.Element{{}, accesscontrol.Address(netip.Addr{}), accesscontrol.Network(netip.MustParsePrefix("192.0.2.3/24")), accesscontrol.Named("bad;any")} {
		if _, err := accesscontrol.AllowQuery(element); err == nil {
			t.Fatal("invalid match accepted")
		}
	}
	if _, err := accesscontrol.AllowQuery(); err == nil {
		t.Fatal("empty query ACL accepted")
	}
}
