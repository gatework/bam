// Package responsepolicy manages local DNS response policies, their domain
// items and local response-policy zones. Configuration and deployment are
// separate operations; use dns deployment roles and the deployment package to
// apply a configured policy zone to servers.
package responsepolicy

import (
	"context"
	"iter"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Client shares the authenticated BAM transport for this domain.
type Client struct{ client *bam.Client }

// New creates a domain client that shares the supplied BAM transport.
func New(client *bam.Client) *Client { return &Client{client: client} }

// Policies provides response policy creation and mutation under a DNS view.
type Policies struct {
	resources resource.Collection[Policy, Spec, Changes]
}

// PolicyItems provides existing response policy operations by global ID.
type PolicyItems struct {
	resources resource.Items[Policy, Changes]
}

// Items provides response policy item creation and mutation under a policy.
type Items struct {
	resources resource.Collection[Item, ItemSpec, ItemChanges]
}

// GlobalItems provides existing response policy item operations by global ID.
type GlobalItems struct {
	resources resource.Items[Item, ItemChanges]
}

// Policies selects response policies under the DNS view.
func (c *Client) Policies(configurationID int64) Policies {
	return Policies{resource.NewCollection[Policy, Spec, Changes](bam.ChildCollection[Policy](c.client, "configurations", configurationID, "responsePolicies"), "ResponsePolicy")}
}

// GlobalPolicies selects existing response policies by global resource ID.
func (c *Client) GlobalPolicies() PolicyItems {
	return PolicyItems{resource.NewItems[Policy, Changes](bam.NewCollection[Policy](c.client, "responsePolicies"), "ResponsePolicy")}
}

// Items selects items under the response policy.
func (c *Client) Items(policyID int64) Items {
	return Items{resource.NewCollection[Item, ItemSpec, ItemChanges](bam.ChildCollection[Item](c.client, "responsePolicies", policyID, "policyItems"), "ResponsePolicyItem")}
}

// GlobalItems selects existing response policy items by global resource ID.
func (c *Client) GlobalItems() GlobalItems {
	return GlobalItems{resource.NewItems[Item, ItemChanges](bam.NewCollection[Item](c.client, "policyItems"), "ResponsePolicyItem")}
}

// Find requires exactly one policy with this name in the selected configuration.
func (p Policies) Find(ctx context.Context, name string) (Policy, error) {
	return resource.One(p.All(ctx, bam.ListOptions{Where: bam.Match("name", name), Limit: 2}))
}

// Find requires exactly one matching domain item in this response policy.
func (i Items) Find(ctx context.Context, name string) (Item, error) {
	return resource.One(i.All(ctx, bam.ListOptions{Where: bam.Match("name", name), Limit: 2}))
}

// Page fetches one page within the selected collection scope.
func (p Policies) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Policy], error) {
	return p.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (p Policies) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Policy, error] {
	return p.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (p Policies) Get(ctx context.Context, id int64) (Policy, error) {
	return p.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (p Policies) Document(ctx context.Context, id int64) (bam.Document, error) {
	return p.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (p Policies) Update(ctx context.Context, id int64, changes Changes, options ...bam.WriteOption) (Policy, error) {
	return p.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (p Policies) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return p.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (p Policies) Create(ctx context.Context, spec Spec, options ...bam.WriteOption) (Policy, error) {
	return p.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (p PolicyItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Policy], error) {
	return p.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (p PolicyItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Policy, error] {
	return p.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (p PolicyItems) Get(ctx context.Context, id int64) (Policy, error) {
	return p.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (p PolicyItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return p.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (p PolicyItems) Update(ctx context.Context, id int64, changes Changes, options ...bam.WriteOption) (Policy, error) {
	return p.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (p PolicyItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return p.resources.Delete(ctx, id, options...)
}

// Page fetches one page within the selected collection scope.
func (i Items) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Item], error) {
	return i.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (i Items) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Item, error] {
	return i.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (i Items) Get(ctx context.Context, id int64) (Item, error) {
	return i.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (i Items) Document(ctx context.Context, id int64) (bam.Document, error) {
	return i.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (i Items) Update(ctx context.Context, id int64, changes ItemChanges, options ...bam.WriteOption) (Item, error) {
	return i.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (i Items) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return i.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (i Items) Create(ctx context.Context, spec ItemSpec, options ...bam.WriteOption) (Item, error) {
	return i.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (g GlobalItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Item], error) {
	return g.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (g GlobalItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Item, error] {
	return g.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (g GlobalItems) Get(ctx context.Context, id int64) (Item, error) {
	return g.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (g GlobalItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return g.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (g GlobalItems) Update(ctx context.Context, id int64, changes ItemChanges, options ...bam.WriteOption) (Item, error) {
	return g.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (g GlobalItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return g.resources.Delete(ctx, id, options...)
}
