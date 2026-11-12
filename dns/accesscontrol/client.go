// Package accesscontrol manages DNS client access control lists and typed
// query/recursion option values. It is separate from domain response policies:
// ACLs match clients, while response policies match DNS questions or responses.
package accesscontrol

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dns"
	"github.com/gatework/bam/internal/resource"
)

// Client shares the authenticated BAM transport for this domain.
type Client struct{ client *bam.Client }

// New creates a domain client that shares the supplied BAM transport.
func New(client *bam.Client) *Client { return &Client{client: client} }

// ACL is an ordered DNS client access-control list.
type ACL struct {
	ID            int64         `json:"id"`
	Type          string        `json:"type"`
	Name          string        `json:"name"`
	Configuration bam.Reference `json:"configuration"`
	Predefined    bool          `json:"predefined"`
	MatchElements []Element     `json:"matchElements"`
}

// Spec specifies the fields required to create a resource in this domain.
type Spec struct {
	Name          string
	MatchElements []Element
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
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

// Changes updates explicitly set resource fields while preserving unmodeled data.
type Changes struct {
	Name          bam.Field[string]
	MatchElements bam.Field[[]Element]
}

// Apply validates and applies explicitly set fields to the complete resource document.
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

// Lists provides ACL creation and mutation under a configuration.
type Lists struct {
	resources resource.Collection[ACL, Spec, Changes]
}

// ListItems provides existing ACL operations by global ID.
type ListItems struct{ resources resource.Items[ACL, Changes] }

// Lists selects client access-control lists under the configuration.
func (c *Client) Lists(configurationID int64) Lists {
	collection := bam.ChildCollection[ACL](c.client, "configurations", configurationID, "accessControlLists")
	return Lists{resource.NewCollection[ACL, Spec, Changes](collection, "ACL")}
}

// GlobalLists selects existing client access-control lists by global resource ID.
func (c *Client) GlobalLists() ListItems {
	return ListItems{resource.NewItems[ACL, Changes](bam.NewCollection[ACL](c.client, "accessControlLists"), "ACL")}
}

// Find requires exactly one resource matching the requested name.
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

// Page fetches one page within the selected collection scope.
func (l Lists) Page(ctx context.Context, options bam.ListOptions) (bam.Page[ACL], error) {
	return l.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (l Lists) All(ctx context.Context, options bam.ListOptions) iter.Seq2[ACL, error] {
	return l.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (l Lists) Get(ctx context.Context, id int64) (ACL, error) {
	return l.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (l Lists) Document(ctx context.Context, id int64) (bam.Document, error) {
	return l.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (l Lists) Update(ctx context.Context, id int64, changes Changes, options ...bam.WriteOption) (ACL, error) {
	return l.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (l Lists) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return l.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (l Lists) Create(ctx context.Context, spec Spec, options ...bam.WriteOption) (ACL, error) {
	return l.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (l ListItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[ACL], error) {
	return l.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (l ListItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[ACL, error] {
	return l.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (l ListItems) Get(ctx context.Context, id int64) (ACL, error) {
	return l.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (l ListItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return l.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (l ListItems) Update(ctx context.Context, id int64, changes Changes, options ...bam.WriteOption) (ACL, error) {
	return l.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (l ListItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return l.resources.Delete(ctx, id, options...)
}
