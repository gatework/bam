// Package configuration manages BlueCat Address Manager configurations through
// the RESTful v2 interface.
package configuration

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Client selects configuration resources using an existing authenticated client.
type Client struct {
	client *bam.Client
}

// New creates a configuration client sharing c's authentication and transport.
func New(c *bam.Client) *Client {
	return &Client{client: c}
}

// Configurations selects the top-level configuration collection.
func (c *Client) Configurations() Configurations {
	return Configurations{resource.NewCollection[Configuration, Spec, Changes](bam.NewCollection[Configuration](c.client, "configurations"), "Configuration")}
}

// Configurations manages top-level configuration resources.
type Configurations struct {
	resources resource.Collection[Configuration, Spec, Changes]
}

// Configuration is a typed projection of a configuration resource. Update
// preserves settings outside this projection through a full read/edit/PUT.
type Configuration struct {
	ID                 int64   `json:"id"`
	Type               string  `json:"type"`
	Name               string  `json:"name"`
	Description        *string `json:"description"`
	ConfigurationGroup *string `json:"configurationGroup"`
}

// Spec creates a configuration. Name is required by Address Manager.
type Spec struct {
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	ConfigurationGroup string `json:"configurationGroup,omitempty"`
}

// MarshalJSON supplies the RESTful v2 resource type.
func (s Spec) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(s.Name) == "" {
		return nil, fmt.Errorf("configuration: name is required")
	}
	type spec Spec
	return json.Marshal(struct {
		spec
		Type string `json:"type"`
	}{Type: "Configuration", spec: spec(s)})
}

// Changes edits configuration metadata without resetting DNS, DHCP or other
// server settings. Empty descriptions and groups are distinct from no change.
type Changes struct {
	Name               bam.Field[string]
	Description        bam.Field[string]
	ConfigurationGroup bam.Field[string]
}

// Apply validates and applies explicitly set fields to the complete resource document.
func (c Changes) Apply(d bam.Document) error {
	if c.Name.IsNull() || c.Description.IsNull() || c.ConfigurationGroup.IsNull() {
		return fmt.Errorf("configuration: metadata fields cannot be null")
	}
	if name, ok := c.Name.Value(); ok && strings.TrimSpace(name) == "" {
		return fmt.Errorf("configuration: name is required")
	}
	if err := c.Name.Apply(d, "name"); err != nil {
		return err
	}
	if err := c.Description.Apply(d, "description"); err != nil {
		return err
	}
	return c.ConfigurationGroup.Apply(d, "configurationGroup")
}

// Page fetches one page within the selected collection scope.
func (c Configurations) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Configuration], error) {
	return c.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (c Configurations) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Configuration, error] {
	return c.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (c Configurations) Get(ctx context.Context, id int64) (Configuration, error) {
	return c.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (c Configurations) Document(ctx context.Context, id int64) (bam.Document, error) {
	return c.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (c Configurations) Update(ctx context.Context, id int64, changes Changes, options ...bam.WriteOption) (Configuration, error) {
	return c.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (c Configurations) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return c.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (c Configurations) Create(ctx context.Context, spec Spec, options ...bam.WriteOption) (Configuration, error) {
	return c.resources.Create(ctx, spec, options...)
}
