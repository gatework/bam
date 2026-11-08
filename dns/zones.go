package dns

import (
	"context"
	"fmt"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Zone is a DNS namespace beneath a view or another zone.
type Zone struct {
	bam.Resource
	Configuration        bam.Reference     `json:"configuration"`
	View                 bam.Reference     `json:"view"`
	AbsoluteName         string            `json:"absoluteName"`
	DeploymentEnabled    bool              `json:"deploymentEnabled"`
	DynamicUpdateEnabled bool              `json:"dynamicUpdateEnabled"`
	UserDefinedFields    map[string]string `json:"userDefinedFields"`
}

// Zones supports the documented ordinary zone resource operations.
type Zones struct {
	resource.Collection[Zone, ZoneSpec, ZoneChanges]
}

// ZoneItems reads and modifies existing ordinary zones across views.
type ZoneItems struct {
	resource.Items[Zone, ZoneChanges]
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
