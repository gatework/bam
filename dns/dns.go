// Package dns manages DNS views, zones, records and address associations using
// BlueCat Address Manager RESTful v2. Configuration changes and deployment are
// separate operations.
package dns

import (
	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Client shares a BAM client's authentication and transport.
type Client struct{ client *bam.Client }

// New creates a DNS client without making a network request.
func New(client *bam.Client) *Client { return &Client{client: client} }

// Views selects views within a configuration.
func (c *Client) Views(configurationID int64) Views {
	return Views{resource.NewCollection[View, ViewSpec, ViewChanges](
		bam.ChildCollection[View](c.client, "configurations", configurationID, "views"), "View")}
}

// GlobalViews selects existing views across configurations. Creation requires
// the parent selected by Views.
func (c *Client) GlobalViews() ViewItems {
	return ViewItems{resource.NewItems[View, ViewChanges](bam.NewCollection[View](c.client, "views"), "View")}
}

// Zones selects ordinary top-level zones within a view. System external-host
// zones and response-policy zones have separate query entry points.
func (c *Client) Zones(viewID int64) Zones {
	return Zones{resource.NewCollection[Zone, ZoneSpec, ZoneChanges](
		bam.ChildCollection[Zone](c.client, "views", viewID, "zones").WithFilter(bam.Match("type", "Zone")), "Zone")}
}

// Subzones selects ordinary zones directly beneath another zone.
func (c *Client) Subzones(zoneID int64) Zones {
	return Zones{resource.NewCollection[Zone, ZoneSpec, ZoneChanges](
		bam.ChildCollection[Zone](c.client, "zones", zoneID, "zones").WithFilter(bam.Match("type", "Zone")), "Zone")}
}

// GlobalZones selects existing ordinary zones across views.
func (c *Client) GlobalZones() ZoneItems {
	return ZoneItems{resource.NewItems[Zone, ZoneChanges](
		bam.NewCollection[Zone](c.client, "zones").WithFilter(bam.Match("type", "Zone")), "Zone")}
}

// Records selects records within a zone. Use an ExternalHostsZone when creating
// external host records; BAM creates that zone when its view is created.
func (c *Client) Records(zoneID int64) Records {
	return Records{newRecordItems(c.client,
		bam.ChildCollection[Record](c.client, "zones", zoneID, "resourceRecords"))}
}

// GlobalRecords selects existing records across zones and configurations.
func (c *Client) GlobalRecords() RecordItems {
	return newRecordItems(c.client, bam.NewCollection[Record](c.client, "resourceRecords"))
}

// Addresses lists IP address resources associated with a host record.
// HostChanges updates a host's complete address association set.
func (c *Client) Addresses(recordID int64) RecordAddresses {
	return RecordAddresses{resource.NewList(bam.ChildCollection[Address](c.client, "resourceRecords", recordID, "addresses"))}
}

// DependentRecords lists records that depend on a resource record.
func (c *Client) DependentRecords(recordID int64) RecordDependencies {
	return RecordDependencies{resource.NewReader(bam.LinkedCollection[Record](c.client, "resourceRecords", recordID, "dependentRecords"))}
}

// RecordsForAddress selects the links between an address and DNS records.
// Removing a link is distinct from deleting the record itself.
func (c *Client) RecordsForAddress(addressID int64) AddressRecords {
	return AddressRecords{
		List:   resource.NewList(bam.ChildCollection[Record](c.client, "addresses", addressID, "resourceRecords")),
		client: c.client, addressID: addressID,
	}
}
