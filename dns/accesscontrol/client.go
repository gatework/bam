// Package accesscontrol manages DNS client access control lists and typed
// query/recursion option values. It is separate from domain response policies:
// ACLs match clients, while response policies match DNS questions or responses.
package accesscontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dns"
	"github.com/gatework/bam/internal/resource"
)

type Client struct{ client *bam.Client }

func New(client *bam.Client) *Client { return &Client{client: client} }

type ACL struct {
	bam.Resource
	Configuration bam.Reference `json:"configuration"`
	Predefined    bool          `json:"predefined"`
	MatchElements []Element     `json:"matchElements"`
}

type Spec struct {
	Name          string
	MatchElements []Element
}

func (s Spec) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(s.Name) == "" {
		return nil, fmt.Errorf("accesscontrol: acl name is required")
	}
	elements, err := encodeElements(s.MatchElements)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Type          string   `json:"type"`
		Name          string   `json:"name"`
		MatchElements []string `json:"matchElements"`
	}{"ACL", s.Name, elements})
}

type Changes struct {
	Name          bam.Field[string]
	MatchElements bam.Field[[]Element]
}

func (c Changes) Apply(d bam.Document) error {
	if c.Name.IsNull() || c.MatchElements.IsNull() {
		return fmt.Errorf("accesscontrol: acl fields cannot be null")
	}
	if name, ok := c.Name.Value(); ok && strings.TrimSpace(name) == "" {
		return fmt.Errorf("accesscontrol: acl name is required")
	}
	if err := c.Name.Apply(d, "name"); err != nil {
		return err
	}
	if elements, ok := c.MatchElements.Value(); ok {
		values, err := encodeElements(elements)
		if err != nil {
			return err
		}
		return d.Set("matchElements", values)
	}
	return nil
}

type Lists struct {
	resource.Collection[ACL, Spec, Changes]
}
type ListItems struct{ resource.Items[ACL, Changes] }

func (c *Client) Lists(configurationID int64) Lists {
	collection := bam.ChildCollection[ACL](c.client, "configurations", configurationID, "accessControlLists")
	return Lists{resource.NewCollection[ACL, Spec, Changes](collection, "ACL")}
}
func (c *Client) GlobalLists() ListItems {
	return ListItems{resource.NewItems[ACL, Changes](bam.NewCollection[ACL](c.client, "accessControlLists"), "ACL")}
}
func (l Lists) Find(ctx context.Context, name string) (ACL, error) {
	return resource.One(l.All(ctx, bam.ListOptions{Where: bam.Match("name", name), Limit: 2}))
}

// AllowQuery constructs the allow-query DNS deployment option. To exclude
// clients, place Not(Address(...)) or Not(Network(...)) before Any().
// Configure it through dns.Options at the desired scope, then deploy explicitly.
func AllowQuery(elements ...Element) (dns.OptionSpec, error) { return option("allow-query", elements) }

// AllowRecursion constructs the allow-recursion DNS deployment option.
func AllowRecursion(elements ...Element) (dns.OptionSpec, error) {
	return option("allow-recursion", elements)
}

func option(name string, elements []Element) (dns.OptionSpec, error) {
	values, err := encodeElements(elements)
	if err != nil {
		return dns.OptionSpec{}, err
	}
	return dns.OptionSpec{Name: name, Value: bam.StringsValue(values...)}, nil
}
