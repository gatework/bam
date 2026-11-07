package deployment

import "github.com/gatework/bam"

// Client selects deployment collections using a shared RESTful v2 client.
type Client struct {
	client *bam.Client
}

// New creates deployment selectors. It does not authenticate or send requests.
func New(client *bam.Client) *Client {
	return &Client{client: client}
}

// ForServer selects deployments for a server.
func (c *Client) ForServer(serverID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "servers", serverID, "deployments")
}

// ForZone selects quick deployments for a zone.
func (c *Client) ForZone(zoneID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "zones", zoneID, "deployments")
}

// ForNetwork selects quick deployments for a network.
func (c *Client) ForNetwork(networkID int64) bam.Collection[bam.Document] {
	return bam.ChildCollection[bam.Document](c.client, "networks", networkID, "deployments")
}

// Tasks selects the global collection for task retrieval and selective deployment.
func (c *Client) Tasks() bam.Collection[bam.Document] {
	return bam.NewCollection[bam.Document](c.client, "deployments")
}
