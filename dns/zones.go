package dns

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Zone is a DNS namespace beneath a view or another zone.
type Zone struct {
	ID                   int64             `json:"id"`
	Type                 string            `json:"type"`
	Name                 string            `json:"name"`
	Configuration        bam.Reference     `json:"configuration"`
	View                 bam.Reference     `json:"view"`
	AbsoluteName         string            `json:"absoluteName"`
	DeploymentEnabled    bool              `json:"deploymentEnabled"`
	DynamicUpdateEnabled bool              `json:"dynamicUpdateEnabled"`
	UserDefinedFields    map[string]string `json:"userDefinedFields"`
}

// Zones supports the documented ordinary zone resource operations.
type Zones struct {
	resources resource.Collection[Zone, ZoneSpec, ZoneChanges]
}

// ZoneItems reads and modifies existing ordinary zones across views.
type ZoneItems struct {
	resources resource.Items[Zone, ZoneChanges]
}

// ExternalHostsZone finds the system-managed external-host zone in a view.
// It is read-only and returns bam.ErrNotFound or bam.ErrAmbiguous unless exactly
// one zone matches. Use its ID with Records to create external host records.
func (c *Client) ExternalHostsZone(ctx context.Context, viewID int64) (Zone, error) {
	zones := bam.ChildCollection[Zone](c.client, "views", viewID, "zones").
		WithFilter(bam.Match("type", "ExternalHostsZone"))
	zone, err := resource.One(zones.All(ctx, bam.ListOptions{Limit: 2}))
	if err != nil {
		return Zone{}, err
	}
	if zone.Type != "ExternalHostsZone" {
		return Zone{}, fmt.Errorf("dns: external host zone lookup returned unexpected type %q", zone.Type)
	}
	return zone, nil
}

// Find looks up one zone by relative name within the selected parent.
func (z Zones) Find(ctx context.Context, name string) (Zone, error) {
	return resource.One(z.All(ctx, bam.ListOptions{Where: bam.Match("name", name), Limit: 2}))
}

// ZoneSpec creates a zone by relative Name or AbsoluteName. BAM applies its
// naming policy. DynamicUpdateEnabled enables server-side selective deployment;
// leave it unset or false when configuration is deployed separately.
type ZoneSpec struct {
	Name                 string            `json:"name,omitempty"`
	AbsoluteName         string            `json:"absoluteName,omitempty"`
	DeploymentEnabled    *bool             `json:"deploymentEnabled,omitempty"`
	DynamicUpdateEnabled *bool             `json:"dynamicUpdateEnabled,omitempty"`
	UserDefinedFields    map[string]string `json:"userDefinedFields,omitempty"`
}

// MarshalJSON validates a zone and supplies its resource type.
func (s ZoneSpec) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(s.Name) == "" && strings.TrimSpace(s.AbsoluteName) == "" {
		return nil, fmt.Errorf("dns: zone name or absolute name is required")
	}
	type fields ZoneSpec
	return marshalSpec("Zone", fields(s))
}

// ZoneChanges updates zone settings while retaining unmodeled fields.
type ZoneChanges struct {
	Name                 bam.Field[string]
	DeploymentEnabled    bam.Field[bool]
	DynamicUpdateEnabled bam.Field[bool]
	UserDefinedFields    bam.Field[map[string]string]
}

// Apply preserves fields outside the requested changes.
func (c ZoneChanges) Apply(d bam.Document) error {
	if err := requiredName(c.Name); err != nil {
		return err
	}
	if c.DeploymentEnabled.IsNull() || c.DynamicUpdateEnabled.IsNull() {
		return fmt.Errorf("dns: zone deployment settings cannot be null")
	}
	return applyFields(
		c.Name.Apply(d, "name"),
		c.DeploymentEnabled.Apply(d, "deploymentEnabled"),
		c.DynamicUpdateEnabled.Apply(d, "dynamicUpdateEnabled"),
		c.UserDefinedFields.Apply(d, "userDefinedFields"),
	)
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
