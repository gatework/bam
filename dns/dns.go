// Package dns manages DNS views, zones, and resource records through the
// BlueCat Address Manager RESTful v2 interface.
package dns

import (
	"net/netip"

	"github.com/gatework/bam"
)

// Client selects DNS collections using an existing authenticated client.
type Client struct {
	client *bam.Client
}

// New creates a DNS client sharing c's authentication and transport.
func New(c *bam.Client) *Client {
	return &Client{client: c}
}

// Views selects the views belonging to a configuration.
// The configuration scopes list and create operations; item operations use
// the global /views/{id} resource path.
func (c *Client) Views(configurationID int64) bam.Collection[View] {
	return bam.ChildCollection[View](c.client, "configurations", configurationID, "views")
}

// Zones selects the top-level zones belonging to a view.
func (c *Client) Zones(viewID int64) bam.Collection[Zone] {
	return bam.ChildCollection[Zone](c.client, "views", viewID, "zones")
}

// Subzones selects the zones directly beneath a zone.
func (c *Client) Subzones(zoneID int64) bam.Collection[Zone] {
	return bam.ChildCollection[Zone](c.client, "zones", zoneID, "zones")
}

// Records selects the resource records belonging to a zone.
func (c *Client) Records(zoneID int64) bam.Collection[Record] {
	return bam.ChildCollection[Record](c.client, "zones", zoneID, "resourceRecords")
}

// GlobalViews accesses views across configurations for filtering and lookup.
// Create views through Views to supply the parent configuration.
func (c *Client) GlobalViews() bam.Collection[View] {
	return bam.NewCollection[View](c.client, "views")
}

// GlobalZones accesses zones across views for filtering and lookup.
// Create zones through Zones or Subzones to supply the parent scope.
func (c *Client) GlobalZones() bam.Collection[Zone] {
	return bam.NewCollection[Zone](c.client, "zones")
}

// GlobalRecords accesses resource records across zones for filtering and lookup.
// Create records through Records to supply the parent zone.
func (c *Client) GlobalRecords() bam.Collection[Record] {
	return bam.NewCollection[Record](c.client, "resourceRecords")
}

// View is a typed projection of a DNS view. Use Collection.Document to retain
// every server field when preparing a replacement.
type View struct {
	bam.Resource
	Configuration                   bam.Reference `json:"configuration"`
	DeviceRegistrationEnabled       bool          `json:"deviceRegistrationEnabled"`
	DeviceRegistrationPortalAddress *string       `json:"deviceRegistrationPortalAddress"`
}

// Zone is a typed projection of a DNS zone.
type Zone struct {
	bam.Resource
	Configuration bam.Reference `json:"configuration"`
	AbsoluteName  string        `json:"absoluteName"`
}

// Record contains common resource-record fields and the documented host-record
// fields. Other record-specific fields remain available through
// Collection.Document; this projection must not be used as a PUT body.
type Record struct {
	bam.Resource
	Configuration bam.Reference `json:"configuration"`
	AbsoluteName  string        `json:"absoluteName"`
	ReverseRecord bool          `json:"reverseRecord"`
	Embedded      struct {
		Addresses []Address `json:"addresses"`
	} `json:"_embedded"`
}

// Address is an IP address associated with a host record.
type Address struct {
	bam.Resource
	Address netip.Addr `json:"address"`
}
