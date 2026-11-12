package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// View is a DNS namespace within a configuration.
type View struct {
	ID                              int64             `json:"id"`
	Type                            string            `json:"type"`
	Name                            string            `json:"name"`
	Configuration                   bam.Reference     `json:"configuration"`
	DeviceRegistrationEnabled       bool              `json:"deviceRegistrationEnabled"`
	DeviceRegistrationPortalAddress *string           `json:"deviceRegistrationPortalAddress"`
	UserDefinedFields               map[string]string `json:"userDefinedFields"`
}

// Views supports the documented view resource operations.
type Views struct {
	resources resource.Collection[View, ViewSpec, ViewChanges]
}

// ViewItems reads and modifies existing views across configurations.
type ViewItems struct {
	resources resource.Items[View, ViewChanges]
}

// Find looks up one view by name within the selected configuration.
func (v Views) Find(ctx context.Context, name string) (View, error) {
	return resource.One(v.All(ctx, bam.ListOptions{Where: bam.Match("name", name), Limit: 2}))
}

// ViewSpec creates a DNS view.
type ViewSpec struct {
	Name                            string            `json:"name"`
	DeviceRegistrationEnabled       *bool             `json:"deviceRegistrationEnabled,omitempty"`
	DeviceRegistrationPortalAddress *string           `json:"deviceRegistrationPortalAddress,omitempty"`
	UserDefinedFields               map[string]string `json:"userDefinedFields,omitempty"`
}

// MarshalJSON validates a view and supplies its resource type.
func (s ViewSpec) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(s.Name) == "" {
		return nil, fmt.Errorf("dns: view name is required")
	}
	type fields ViewSpec
	return marshalSpec("View", fields(s))
}

// ViewChanges modifies selected view fields. The zero value changes nothing.
type ViewChanges struct {
	Name                            bam.Field[string]
	DeviceRegistrationEnabled       bam.Field[bool]
	DeviceRegistrationPortalAddress bam.Field[string]
	UserDefinedFields               bam.Field[map[string]string]
}

// Apply preserves fields outside the requested changes.
func (c ViewChanges) Apply(d bam.Document) error {
	if err := requiredName(c.Name); err != nil {
		return err
	}
	if c.DeviceRegistrationEnabled.IsNull() {
		return fmt.Errorf("dns: device registration enabled cannot be null")
	}
	return applyFields(
		c.Name.Apply(d, "name"),
		c.DeviceRegistrationEnabled.Apply(d, "deviceRegistrationEnabled"),
		c.DeviceRegistrationPortalAddress.Apply(d, "deviceRegistrationPortalAddress"),
		c.UserDefinedFields.Apply(d, "userDefinedFields"),
	)
}

func requiredName(field bam.Field[string]) error {
	if field.IsNull() {
		return fmt.Errorf("dns: name cannot be null")
	}
	if value, ok := field.Value(); ok && strings.TrimSpace(value) == "" {
		return fmt.Errorf("dns: name cannot be empty")
	}
	return nil
}

// marshalSpec adds the discriminator at the JSON boundary. Each caller passes
// an alias so custom marshalers cannot recurse.
func marshalSpec(kind string, fields any) ([]byte, error) {
	data, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	var document bam.Document
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if err := document.Set("type", kind); err != nil {
		return nil, err
	}
	return json.Marshal(document)
}

func applyFields(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Page fetches one page within the selected collection scope.
func (v Views) Page(ctx context.Context, options bam.ListOptions) (bam.Page[View], error) {
	return v.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (v Views) All(ctx context.Context, options bam.ListOptions) iter.Seq2[View, error] {
	return v.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (v Views) Get(ctx context.Context, id int64) (View, error) {
	return v.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (v Views) Document(ctx context.Context, id int64) (bam.Document, error) {
	return v.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (v Views) Update(ctx context.Context, id int64, changes ViewChanges, options ...bam.WriteOption) (View, error) {
	return v.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (v Views) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return v.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (v Views) Create(ctx context.Context, spec ViewSpec, options ...bam.WriteOption) (View, error) {
	return v.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (v ViewItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[View], error) {
	return v.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (v ViewItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[View, error] {
	return v.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (v ViewItems) Get(ctx context.Context, id int64) (View, error) {
	return v.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (v ViewItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return v.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (v ViewItems) Update(ctx context.Context, id int64, changes ViewChanges, options ...bam.WriteOption) (View, error) {
	return v.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (v ViewItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return v.resources.Delete(ctx, id, options...)
}
