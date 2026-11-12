package dhcp_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dhcp"
)

func TestOptions_Find_KindAndScope(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		build func() (dhcp.OptionQuery, error)
		id    int64
		err   error
	}{
		{"client code", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQueryByCode(6, 0) }, 10, nil},
		{"server code", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQueryByCode(6, 20) }, 11, nil},
		{"service definition", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQueryByDefinition(31, 0) }, 12, nil},
		{"ambiguous name", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQuery("example", 0) }, 0, bam.ErrAmbiguous},
		{"missing scope", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQueryByCode(6, 99) }, 0, bam.ErrNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v2/networks/7/deploymentOptions" || r.URL.Query().Has("fields") {
					t.Errorf("unexpected lookup request %s %s", r.Method, r.URL)
				}
				if !strings.Contains(r.URL.Query().Get("filter"), "DHCPv4ClientOption") {
					t.Error("lookup dropped DHCP type scope")
				}
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Query().Get("offset") == "1" {
					_, _ = io.WriteString(w, `{"count":2,"data":[{"id":11,"type":"DHCPv4ClientOption","name":"example","code":6,"definition":{"id":30},"serverScope":{"id":20},"value":[]},{"id":12,"type":"DHCPv4ServiceOption","name":"example","code":6,"definition":{"id":31},"serverScope":null,"value":false}]}`)
				} else {
					_, _ = io.WriteString(w, `{"count":1,"data":[{"id":10,"type":"DHCPv4ClientOption","name":"example","code":6,"definition":{"id":30},"serverScope":null,"value":[]}],"_links":{"next":{"href":"?offset=1"}}}`)
				}
			})
			query, err := tt.build()
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.NetworkOptions(7).Find(t.Context(), query)
			if !errors.Is(err, tt.err) || got.ID != tt.id {
				t.Fatalf("option=%+v error=%v", got, err)
			}
		})
	}
}

func TestNewOptionQuery_InvalidInput(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		build func() (dhcp.OptionQuery, error)
	}{
		{"empty name", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQuery("", 0) }},
		{"blank name", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQuery(" ", 0) }},
		{"name negative scope", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQuery("dns-server", -1) }},
		{"zero code", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQueryByCode(0, 0) }},
		{"code negative scope", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQueryByCode(6, -1) }},
		{"zero definition", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQueryByDefinition(0, 0) }},
		{"negative definition", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQueryByDefinition(-1, 0) }},
		{"definition negative scope", func() (dhcp.OptionQuery, error) { return dhcp.NewOptionQueryByDefinition(30, -1) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			query, err := tt.build()
			if err == nil || query.IsValid() || query != (dhcp.OptionQuery{}) {
				t.Fatalf("invalid constructor returned %+v, %v", query, err)
			}
		})
	}
}

func TestOptions_Find_UnsetQuery(t *testing.T) {
	t.Parallel()
	client := newClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("unset query made a request")
	})
	if _, err := client.NetworkOptions(7).Find(t.Context(), dhcp.OptionQuery{}); err == nil {
		t.Fatal("accepted unset query")
	}
}
