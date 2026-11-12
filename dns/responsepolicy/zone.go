package responsepolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Zone binds a response policy to a DNS view. DNS server responsibility is
// assigned separately through dns.Client.ZoneRoles.
type Zone struct {
	ID            int64         `json:"id"`
	Type          string        `json:"type"`
	Name          string        `json:"name"`
	Configuration bam.Reference `json:"configuration"`
	View          bam.Reference `json:"view"`
	ZoneType      string        `json:"responsePolicyZoneType"`
	Policy        bam.Reference `json:"responsePolicy"`
}

// ZoneSpec creates a LOCAL response-policy zone. External security feed
// subscription and licensing are outside this local-policy interface.
type ZoneSpec struct {
	Name     string
	PolicyID int64
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s ZoneSpec) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(s.Name) == "" || s.PolicyID <= 0 {
		return nil, fmt.Errorf("responsepolicy: local zone requires a name and positive policy id")
	}
	return json.Marshal(struct {
		Type     string        `json:"type"`
		Name     string        `json:"name"`
		ZoneType string        `json:"responsePolicyZoneType"`
		Policy   bam.Reference `json:"responsePolicy"`
	}{"ResponsePolicyZone", s.Name, "LOCAL", bam.Reference{ID: s.PolicyID}})
}

// ZoneChanges updates explicitly set local response policy zone fields.
type ZoneChanges struct {
	Name     bam.Field[string]
	PolicyID bam.Field[int64]
}

// Apply validates and applies explicitly set fields to the complete resource document.
func (c ZoneChanges) Apply(d bam.Document) error {
	var current Zone
	if err := d.Decode(&current); err != nil {
		return err
	}
	if current.ZoneType != "LOCAL" {
		return fmt.Errorf("responsepolicy: only local policy zones can be changed through this interface")
	}
	if c.Name.IsNull() || c.PolicyID.IsNull() {
		return fmt.Errorf("responsepolicy: local zone fields cannot be null")
	}
	if name, ok := c.Name.Value(); ok && strings.TrimSpace(name) == "" {
		return fmt.Errorf("responsepolicy: zone name is required")
	}
	if err := c.Name.Apply(d, "name"); err != nil {
		return err
	}
	if id, ok := c.PolicyID.Value(); ok {
		if id <= 0 {
			return fmt.Errorf("responsepolicy: positive policy id required")
		}
		return d.Set("responsePolicy", bam.Reference{ID: id})
	}
	return nil
}

// Zones provides local response policy zone creation under a DNS view.
type Zones struct {
	resources resource.Collection[Zone, ZoneSpec, ZoneChanges]
}

// ZoneItems provides existing local response policy zone operations by global ID.
type ZoneItems struct {
	resources resource.Items[Zone, ZoneChanges]
}

// Zones selects local response policy zones under the DNS view.
func (c *Client) Zones(viewID int64) Zones {
	collection := bam.ChildCollection[Zone](c.client, "views", viewID, "zones").WithFilter(bam.Match("type", "ResponsePolicyZone"))
	return Zones{resource.NewCollection[Zone, ZoneSpec, ZoneChanges](collection, "ResponsePolicyZone")}
}

// GlobalZones selects existing response policy zones by global resource ID.
func (c *Client) GlobalZones() ZoneItems {
	collection := bam.NewCollection[Zone](c.client, "zones").WithFilter(bam.Match("type", "ResponsePolicyZone"))
	return ZoneItems{resource.NewItems[Zone, ZoneChanges](collection, "ResponsePolicyZone")}
}

// Page fetches one page within the selected collection scope.
func (z Zones) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Zone], error) {
	return z.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (z Zones) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Zone, error] {
	return z.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (z Zones) Get(ctx context.Context, id int64) (Zone, error) {
	return z.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (z Zones) Document(ctx context.Context, id int64) (bam.Document, error) {
	return z.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (z Zones) Update(ctx context.Context, id int64, changes ZoneChanges, options ...bam.WriteOption) (Zone, error) {
	return z.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (z Zones) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return z.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (z Zones) Create(ctx context.Context, spec ZoneSpec, options ...bam.WriteOption) (Zone, error) {
	return z.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (z ZoneItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Zone], error) {
	return z.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (z ZoneItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Zone, error] {
	return z.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (z ZoneItems) Get(ctx context.Context, id int64) (Zone, error) {
	return z.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (z ZoneItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return z.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (z ZoneItems) Update(ctx context.Context, id int64, changes ZoneChanges, options ...bam.WriteOption) (Zone, error) {
	return z.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (z ZoneItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return z.resources.Delete(ctx, id, options...)
}
