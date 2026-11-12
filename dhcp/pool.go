package dhcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
	"github.com/gatework/bam/ipam"
)

// MACPool groups MAC addresses for DHCP policy. New pools disable instant
// deployment; updates preserve the appliance setting unless explicitly changed.
type MACPool struct {
	ID                       int64         `json:"id"`
	Type                     string        `json:"type"`
	Name                     string        `json:"name"`
	Configuration            bam.Reference `json:"configuration"`
	InstantDeploymentEnabled bool          `json:"instantDeploymentEnabled"`
}

// MACPoolSpec specifies metadata for creating a MAC pool.
type MACPoolSpec struct {
	Name string `json:"name"`
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s MACPoolSpec) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(s.Name) == "" {
		return nil, fmt.Errorf("dhcp: mac pool name is required")
	}
	return json.Marshal(struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		Instant bool   `json:"instantDeploymentEnabled"`
	}{"MACPool", s.Name, false})
}

// MACPoolChanges updates explicitly set MAC pool metadata fields.
type MACPoolChanges struct {
	Name                     bam.Field[string]
	InstantDeploymentEnabled bam.Field[bool]
}

// Apply validates and applies explicitly set fields to the complete resource document.
func (c MACPoolChanges) Apply(d bam.Document) error {
	if c.Name.IsNull() {
		return fmt.Errorf("dhcp: mac pool name cannot be cleared")
	}
	if name, ok := c.Name.Value(); ok && strings.TrimSpace(name) == "" {
		return fmt.Errorf("dhcp: mac pool name is required")
	}
	if c.InstantDeploymentEnabled.IsNull() {
		return fmt.Errorf("dhcp: instant deployment setting cannot be null")
	}
	return errors.Join(c.Name.Apply(d, "name"), c.InstantDeploymentEnabled.Apply(d, "instantDeploymentEnabled"))
}

// MACPools provides MAC pool creation and mutation under a configuration.
type MACPools struct {
	resources resource.Collection[MACPool, MACPoolSpec, MACPoolChanges]
}

// Find requires a unique ordinary MAC pool by name in this configuration.
func (m MACPools) Find(ctx context.Context, name string) (MACPool, error) {
	if strings.TrimSpace(name) == "" {
		return MACPool{}, fmt.Errorf("dhcp: mac pool name is required")
	}
	return resource.One(m.All(ctx, bam.ListOptions{Where: bam.Match("name", name), Limit: 2}))
}

// MACPoolItems provides existing MAC pool operations by global ID.
type MACPoolItems struct {
	resources resource.Items[MACPool, MACPoolChanges]
}

// MACMembers exposes relationship operations. Unlink never deletes a MAC resource.
type MACMembers struct {
	resources resource.List[ipam.MACAddress]
}

// Get reads and validates the resource with the supplied ID.
func (m MACMembers) Get(ctx context.Context, id int64) (ipam.MACAddress, error) {
	return resource.CollectionOf(m.resources).Get(ctx, id)
}

// Link adds a membership relationship without creating the referenced resource.
func (m MACMembers) Link(ctx context.Context, macID int64, options ...bam.WriteOption) (ipam.MACAddress, error) {
	if macID <= 0 {
		return ipam.MACAddress{}, fmt.Errorf("dhcp: mac resource id must be positive")
	}
	member, err := resource.CollectionOf(m.resources).Create(ctx, struct {
		Type string `json:"type"`
		ID   int64  `json:"id"`
	}{"MACAddress", macID}, options...)
	if err != nil {
		return ipam.MACAddress{}, err
	}
	if member.ID != macID || member.Type != "MACAddress" {
		return ipam.MACAddress{}, &bam.ProtocolError{
			Method: http.MethodPost, Detail: "pool membership response does not match the requested mac address",
		}
	}
	return member, nil
}

// Unlink removes only the membership relationship, preserving the referenced resource.
func (m MACMembers) Unlink(ctx context.Context, macID int64, options ...bam.WriteOption) error {
	return resource.CollectionOf(m.resources).Delete(ctx, macID, options...)
}

// Page fetches one page within the selected collection scope.
func (m MACPools) Page(ctx context.Context, options bam.ListOptions) (bam.Page[MACPool], error) {
	return m.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (m MACPools) All(ctx context.Context, options bam.ListOptions) iter.Seq2[MACPool, error] {
	return m.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (m MACPools) Get(ctx context.Context, id int64) (MACPool, error) {
	return m.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (m MACPools) Document(ctx context.Context, id int64) (bam.Document, error) {
	return m.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (m MACPools) Update(ctx context.Context, id int64, changes MACPoolChanges, options ...bam.WriteOption) (MACPool, error) {
	return m.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (m MACPools) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return m.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (m MACPools) Create(ctx context.Context, spec MACPoolSpec, options ...bam.WriteOption) (MACPool, error) {
	return m.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (m MACPoolItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[MACPool], error) {
	return m.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (m MACPoolItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[MACPool, error] {
	return m.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (m MACPoolItems) Get(ctx context.Context, id int64) (MACPool, error) {
	return m.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (m MACPoolItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return m.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (m MACPoolItems) Update(ctx context.Context, id int64, changes MACPoolChanges, options ...bam.WriteOption) (MACPool, error) {
	return m.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (m MACPoolItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return m.resources.Delete(ctx, id, options...)
}

// Page fetches one page within the selected collection scope.
func (m MACMembers) Page(ctx context.Context, options bam.ListOptions) (bam.Page[ipam.MACAddress], error) {
	return m.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (m MACMembers) All(ctx context.Context, options bam.ListOptions) iter.Seq2[ipam.MACAddress, error] {
	return m.resources.All(ctx, options)
}
