package responsepolicy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Zone binds a response policy to a DNS view. DNS server responsibility is
// assigned separately through dns.Client.ZoneRoles.
type Zone struct {
	bam.Resource
	Configuration bam.Reference `json:"configuration"`
	View          bam.Reference `json:"view"`
	ZoneType      string        `json:"responsePolicyZoneType"`
	Policy        bam.Reference `json:"responsePolicy"`
}

// ZoneSpec creates a LOCAL response-policy zone. External security feed
// subscription and licensing are outside this local-policy interface.
type ZoneSpec struct {
	Name     string
	PolicyID int64
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s ZoneSpec) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(s.Name) == "" || s.PolicyID <= 0 {
		return nil, fmt.Errorf("responsepolicy: local zone requires a name and positive policy id")
	}
	return json.Marshal(struct {
		Type     string        `json:"type"`
		Name     string        `json:"name"`
		ZoneType string        `json:"responsePolicyZoneType"`
		Policy   bam.Reference `json:"responsePolicy"`
	}{"ResponsePolicyZone", s.Name, "LOCAL", bam.Reference{ID: s.PolicyID}})
}

// ZoneChanges updates explicitly set local response policy zone fields.
type ZoneChanges struct {
	Name     bam.Field[string]
	PolicyID bam.Field[int64]
}

// Apply validates and applies explicitly set fields to the complete resource document.
func (c ZoneChanges) Apply(d bam.Document) error {
	var current Zone
	if err := d.Decode(&current); err != nil {
		return err
	}
	if current.ZoneType != "LOCAL" {
		return fmt.Errorf("responsepolicy: only local policy zones can be changed through this interface")
	}
	if c.Name.IsNull() || c.PolicyID.IsNull() {
		return fmt.Errorf("responsepolicy: local zone fields cannot be null")
	}
	if name, ok := c.Name.Value(); ok && strings.TrimSpace(name) == "" {
		return fmt.Errorf("responsepolicy: zone name is required")
	}
	if err := c.Name.Apply(d, "name"); err != nil {
		return err
	}
	if id, ok := c.PolicyID.Value(); ok {
		if id <= 0 {
			return fmt.Errorf("responsepolicy: positive policy id required")
		}
		return d.Set("responsePolicy", bam.Reference{ID: id})
	}
	return nil
}

// Zones provides local response policy zone creation under a DNS view.
type Zones struct {
	resource.Collection[Zone, ZoneSpec, ZoneChanges]
}

// ZoneItems provides existing local response policy zone operations by global ID.
type ZoneItems struct {
	resource.Items[Zone, ZoneChanges]
}

// Zones selects local response policy zones under the DNS view.
func (c *Client) Zones(viewID int64) Zones {
	collection := bam.ChildCollection[Zone](c.client, "views", viewID, "zones").WithFilter(bam.Match("type", "ResponsePolicyZone"))
	return Zones{resource.NewCollection[Zone, ZoneSpec, ZoneChanges](collection, "ResponsePolicyZone")}
}

// GlobalZones selects existing response policy zones by global resource ID.
func (c *Client) GlobalZones() ZoneItems {
	collection := bam.NewCollection[Zone](c.client, "zones").WithFilter(bam.Match("type", "ResponsePolicyZone"))
	return ZoneItems{resource.NewItems[Zone, ZoneChanges](collection, "ResponsePolicyZone")}
}
