package deployment

import (
	"context"
	"fmt"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Client submits deployments independently of configuration changes.
type Client struct {
	client *bam.Client
}

// New shares c's authentication and transport without sending a request.
func New(c *bam.Client) *Client {
	return &Client{client: c}
}

// Tasks provides read-only task access. Submission and completion are separate.
type Tasks struct {
	resource.Reader[Task]
}

func tasks(collection bam.Collection[Task]) Tasks {
	return Tasks{Reader: resource.NewReader(collection,
		string(KindFull), string(KindDifferential), string(KindQuick),
		string(KindValidation), string(KindSelective))}
}

// Tasks reads deployments across servers. It does not expose task replacement
// or deletion; Wait stops polling on cancellation without cancelling the task.
func (c *Client) Tasks() Tasks {
	return tasks(bam.NewCollection[Task](c.client, "deployments"))
}

// ServerTarget submits service deployments and reads a server's task history.
// Get and Wait use globally unique task IDs, independently of the server scope.
type ServerTarget struct {
	Tasks
	submission submission
}

// ForServer selects full, differential and validation deployments for a server.
func (c *Client) ForServer(serverID int64) ServerTarget {
	s := c.submission("servers", serverID)
	return ServerTarget{Tasks: tasks(s.collection), submission: s}
}

// QuickTarget submits changes to resource records updated by the current user
// in a zone or network. Task results are read through Client.Tasks.
type QuickTarget struct {
	submission submission
}

// ForZone selects quick deployment for a zone.
func (c *Client) ForZone(zoneID int64) QuickTarget {
	return QuickTarget{submission: c.submission("zones", zoneID)}
}

// ForNetwork selects quick deployment for a network. This does not imply that
// the network's deployment collection supports task-history queries.
func (c *Client) ForNetwork(networkID int64) QuickTarget {
	return QuickTarget{submission: c.submission("networks", networkID)}
}

// ZoneTasks reads the deployment history of a zone.
func (c *Client) ZoneTasks(zoneID int64) Tasks {
	return tasks(bam.ChildCollection[Task](c.client, "zones", zoneID, "deployments"))
}

func (c *Client) submission(parent string, id int64) submission {
	return submission{
		collection: bam.ChildCollection[Task](c.client, parent, id, "deployments"),
		path:       fmt.Sprintf("/api/v2/%s/%d/deployments", parent, id),
	}
}

// Selective submits changes to specific resource records or IP addresses.
// HTTP 202 returns bam.AcceptedError, not a completed or newly created task.
func (c *Client) Selective(ctx context.Context, spec SelectiveSpec, options ...bam.WriteOption) (Task, error) {
	s := submission{collection: bam.NewCollection[Task](c.client, "deployments"), path: "/api/v2/deployments"}
	return s.create(ctx, spec, options...)
}
