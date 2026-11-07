// Package configuration manages BlueCat Address Manager configurations through
// the RESTful v2 interface.
package configuration

import (
	"encoding/json"
	"fmt"

	"github.com/gatework/bam"
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
func (c *Client) Configurations() bam.Collection[Configuration] {
	return bam.NewCollection[Configuration](c.client, "configurations")
}

// Configuration is a typed projection of a configuration resource. Use
// Collection.Document to retain every server field for a replacement.
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
	if s.Name == "" {
		return nil, fmt.Errorf("configuration: name is required")
	}
	type spec Spec
	return json.Marshal(struct {
		Type string `json:"type"`
		spec
	}{Type: "Configuration", spec: spec(s)})
}
