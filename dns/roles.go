package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"slices"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// RoleType determines a server's DNS responsibility for a view, zone or IP
// space. NONE clears the deployed data for that role on the next deployment.
type RoleType string

// Supported deployment role types.
const (
	RolePrimary          RoleType = "PRIMARY"
	RoleHiddenPrimary    RoleType = "HIDDEN_PRIMARY"
	RoleSecondary        RoleType = "SECONDARY"
	RoleStealthSecondary RoleType = "STEALTH_SECONDARY"
	RoleForwarding       RoleType = "FORWARDING"
	RoleStub             RoleType = "STUB"
	RoleRecursive        RoleType = "RECURSIVE"
	RoleNone             RoleType = "NONE"
)

func (r RoleType) validate() error {
	switch r {
	case RolePrimary, RoleHiddenPrimary, RoleSecondary, RoleStealthSecondary, RoleForwarding, RoleStub, RoleRecursive, RoleNone:
		return nil
	}
	return fmt.Errorf("dns: invalid deployment role type")
}

// RoleInterface identifies a service or zone-transfer interface. Type is
// NetworkInterface or PublishedInterface; Role and Protocol are optional.
type RoleInterface struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Role     string `json:"deploymentRoleInterfaceType,omitempty"`
	Protocol string `json:"deploymentRoleInterfaceProtocol,omitempty"`
}

func validateInterfaces(interfaces []RoleInterface) error {
	if len(interfaces) == 0 {
		return fmt.Errorf("dns: at least one role interface is required")
	}
	for _, v := range interfaces {
		if v.ID <= 0 || v.Type != "NetworkInterface" && v.Type != "PublishedInterface" {
			return fmt.Errorf("dns: invalid role interface")
		}
		if !slices.Contains([]string{"", "SERVICE", "ZONE_TRANSFER", "TARGET"}, v.Role) {
			return fmt.Errorf("dns: invalid role interface purpose")
		}
		if !slices.Contains([]string{"", "IPV4", "IPV6"}, v.Protocol) {
			return fmt.Errorf("dns: invalid zone transfer protocol")
		}
	}
	return nil
}

// Role is a DNS deployment role returned by BAM.
type Role struct {
	ID            int64         `json:"id"`
	Type          string        `json:"type"`
	Name          string        `json:"name"`
	Configuration bam.Reference `json:"configuration"`
	RoleType      RoleType      `json:"roleType"`
	View          bam.Reference `json:"view"`
	NSRecordTTL   int64         `json:"nsRecordTtl"`
}

// RoleSpec assigns server interfaces to a scope. ViewID supplies the DNS view
// when assigning a reverse-DNS role to IP space.
type RoleSpec struct {
	RoleType    RoleType
	Interfaces  []RoleInterface
	ViewID      int64
	NSRecordTTL *int64
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s RoleSpec) MarshalJSON() ([]byte, error) {
	if err := s.RoleType.validate(); err != nil {
		return nil, err
	}
	if err := validateInterfaces(s.Interfaces); err != nil {
		return nil, err
	}
	if s.ViewID < 0 {
		return nil, fmt.Errorf("dns: invalid view id")
	}
	if s.NSRecordTTL != nil && (*s.NSRecordTTL < 0 || *s.NSRecordTTL > 2147483647) {
		return nil, fmt.Errorf("dns: invalid name server record ttl")
	}
	var view *bam.Reference
	if s.ViewID > 0 {
		view = &bam.Reference{ID: s.ViewID}
	}
	return json.Marshal(struct {
		Type        string          `json:"type"`
		RoleType    RoleType        `json:"roleType"`
		Interfaces  []RoleInterface `json:"interfaces"`
		View        *bam.Reference  `json:"view,omitempty"`
		NSRecordTTL *int64          `json:"nsRecordTtl,omitempty"`
	}{"DNSDeploymentRole", s.RoleType, s.Interfaces, view, s.NSRecordTTL})
}

// RoleChanges updates explicitly set deployment role fields.
type RoleChanges struct {
	RoleType    bam.Field[RoleType]
	Interfaces  bam.Field[[]RoleInterface]
	NSRecordTTL bam.Field[int64]
}

// Apply validates and applies explicitly set fields to the complete resource document.
func (c RoleChanges) Apply(d bam.Document) error {
	if c.RoleType.IsNull() || c.Interfaces.IsNull() || c.NSRecordTTL.IsNull() {
		return fmt.Errorf("dns: deployment role fields cannot be null")
	}
	if value, ok := c.RoleType.Value(); ok {
		if err := value.validate(); err != nil {
			return err
		}
	}
	if value, ok := c.Interfaces.Value(); ok {
		if err := validateInterfaces(value); err != nil {
			return err
		}
	}
	if value, ok := c.NSRecordTTL.Value(); ok && (value < 0 || value > 2147483647) {
		return fmt.Errorf("dns: invalid name server record ttl")
	}
	if err := c.RoleType.Apply(d, "roleType"); err != nil {
		return err
	}
	if err := c.Interfaces.Apply(d, "interfaces"); err != nil {
		return err
	}
	return c.NSRecordTTL.Apply(d, "nsRecordTtl")
}

// Roles provides deployment role creation and mutation under a parent resource.
type Roles struct {
	resources resource.Collection[Role, RoleSpec, RoleChanges]
}

// RoleItems provides existing deployment role operations by global ID.
type RoleItems struct {
	resources resource.Items[Role, RoleChanges]
}

// RoleInterfaces provides read-only discovery of interfaces assigned to a role.
type RoleInterfaces struct{ resources resource.List[RoleInterface] }

// ViewRoles selects deployment roles directly beneath the supplied view.
func (c *Client) ViewRoles(id int64) Roles { return c.roles("views", id) }

// ZoneRoles selects deployment roles directly beneath the supplied zone.
func (c *Client) ZoneRoles(id int64) Roles { return c.roles("zones", id) }

// BlockRoles selects deployment roles directly beneath the supplied block.
func (c *Client) BlockRoles(id int64) Roles { return c.roles("blocks", id) }

// NetworkRoles selects deployment roles directly beneath the supplied network.
func (c *Client) NetworkRoles(id int64) Roles { return c.roles("networks", id) }

// GlobalRoles selects existing deployment roles by global resource ID.
func (c *Client) GlobalRoles() RoleItems {
	collection := bam.NewCollection[Role](c.client, "deploymentRoles").WithFilter(bam.Match("type", "DNSDeploymentRole"))
	return RoleItems{resource.NewItems[Role, RoleChanges](collection, "DNSDeploymentRole")}
}

// RoleInterfaces selects server interface associations for a deployment role.
func (c *Client) RoleInterfaces(roleID int64) RoleInterfaces {
	return RoleInterfaces{resource.NewList(bam.ChildCollection[RoleInterface](c.client, "deploymentRoles", roleID, "interfaces"))}
}

func (c *Client) roles(parent string, id int64) Roles {
	collection := bam.ChildCollection[Role](c.client, parent, id, "deploymentRoles").WithFilter(bam.Match("type", "DNSDeploymentRole"))
	return Roles{resource.NewCollection[Role, RoleSpec, RoleChanges](collection, "DNSDeploymentRole")}
}

// Page fetches one page within the selected collection scope.
func (r Roles) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Role], error) {
	return r.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r Roles) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Role, error] {
	return r.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (r Roles) Get(ctx context.Context, id int64) (Role, error) {
	return r.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (r Roles) Document(ctx context.Context, id int64) (bam.Document, error) {
	return r.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (r Roles) Update(ctx context.Context, id int64, changes RoleChanges, options ...bam.WriteOption) (Role, error) {
	return r.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (r Roles) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return r.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (r Roles) Create(ctx context.Context, spec RoleSpec, options ...bam.WriteOption) (Role, error) {
	return r.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (r RoleItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Role], error) {
	return r.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r RoleItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Role, error] {
	return r.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (r RoleItems) Get(ctx context.Context, id int64) (Role, error) {
	return r.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (r RoleItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return r.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (r RoleItems) Update(ctx context.Context, id int64, changes RoleChanges, options ...bam.WriteOption) (Role, error) {
	return r.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (r RoleItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return r.resources.Delete(ctx, id, options...)
}

// Page fetches one page within the selected collection scope.
func (r RoleInterfaces) Page(ctx context.Context, options bam.ListOptions) (bam.Page[RoleInterface], error) {
	return r.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r RoleInterfaces) All(ctx context.Context, options bam.ListOptions) iter.Seq2[RoleInterface, error] {
	return r.resources.All(ctx, options)
}
