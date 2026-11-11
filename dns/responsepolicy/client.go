// Package responsepolicy manages local DNS response policies, their domain
// items and local response-policy zones. Configuration and deployment are
// separate operations; use dns deployment roles and the deployment package to
// apply a configured policy zone to servers.
package responsepolicy

import (
	"context"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Client shares the authenticated BAM transport for this domain.
type Client struct{ client *bam.Client }

// New creates a domain client that shares the supplied BAM transport.
func New(client *bam.Client) *Client { return &Client{client: client} }

// Policies provides response policy creation and mutation under a DNS view.
type Policies struct {
	resource.Collection[Policy, Spec, Changes]
}

// PolicyItems provides existing response policy operations by global ID.
type PolicyItems struct {
	resource.Items[Policy, Changes]
}

// Items provides response policy item creation and mutation under a policy.
type Items struct {
	resource.Collection[Item, ItemSpec, ItemChanges]
}

// GlobalItems provides existing response policy item operations by global ID.
type GlobalItems struct {
	resource.Items[Item, ItemChanges]
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
