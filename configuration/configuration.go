// Package configuration manages BlueCat Address Manager configurations through
// the RESTful v2 interface.
package configuration

import (
	"encoding/json"
	"fmt"
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
	resource.Collection[Configuration, Spec, Changes]
}

// Configuration is a typed projection of a configuration resource. Update
// preserves settings outside this projection through a full read/edit/PUT.
type Configuration struct {
	bam.Resource
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
		Type string `json:"type"`
		spec
	}{Type: "Configuration", spec: spec(s)})
}

// Changes edits configuration metadata without resetting DNS, DHCP or other
// server settings. Empty descriptions and groups are distinct from no change.
type Changes struct {
	Name               bam.Field[string]
	Description        bam.Field[string]
	ConfigurationGroup bam.Field[string]
}

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
