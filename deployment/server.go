package deployment

import (
	"context"
	"iter"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Server contains the metadata needed to discover deployment targets. Profiles
// and states remain open strings because server families vary between releases.
type Server struct {
	ID            int64         `json:"id"`
	Type          string        `json:"type"`
	Name          string        `json:"name"`
	Configuration bam.Reference `json:"configuration"`
	Profile       string        `json:"profile"`
	State         string        `json:"state"`
	Connected     bool          `json:"connected"`
}

// Servers provides read-only discovery, without appliance lifecycle operations.
type Servers struct {
	resources resource.Reader[Server]
}

// Servers selects deployment targets in a configuration.
func (c *Client) Servers(configurationID int64) Servers {
	return Servers{resources: resource.NewReader(
		bam.ChildCollection[Server](c.client, "configurations", configurationID, "servers"), "Server", "HAServer")}
}

// GlobalServers discovers deployment targets across configurations.
func (c *Client) GlobalServers() Servers {
	return Servers{resources: resource.NewReader(bam.NewCollection[Server](c.client, "servers"), "Server", "HAServer")}
}

// Page fetches one page within the selected collection scope.
func (s Servers) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Server], error) {
	return s.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (s Servers) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Server, error] {
	return s.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (s Servers) Get(ctx context.Context, id int64) (Server, error) {
	return s.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (s Servers) Document(ctx context.Context, id int64) (bam.Document, error) {
	return s.resources.Document(ctx, id)
}
