package deployment

import (
	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Server contains the metadata needed to discover deployment targets. Profiles
// and states remain open strings because server families vary between releases.
type Server struct {
	bam.Resource
	Configuration bam.Reference `json:"configuration"`
	Profile       string        `json:"profile"`
	State         string        `json:"state"`
	Connected     bool          `json:"connected"`
}

// Servers provides read-only discovery, without appliance lifecycle operations.
type Servers struct {
	resource.Reader[Server]
}

// Servers selects deployment targets in a configuration.
func (c *Client) Servers(configurationID int64) Servers {
	return Servers{Reader: resource.NewReader(
		bam.ChildCollection[Server](c.client, "configurations", configurationID, "servers"), "Server", "HAServer")}
}

// GlobalServers discovers deployment targets across configurations.
func (c *Client) GlobalServers() Servers {
	return Servers{Reader: resource.NewReader(bam.NewCollection[Server](c.client, "servers"), "Server", "HAServer")}
}
