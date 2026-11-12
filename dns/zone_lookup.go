package dns

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// LookupZone finds an ordinary zone by absolute name within one view, including
// subzones and zones with multi-label names. Names are compared without case or
// a final root dot. It traverses only branches that can contain the name, using
// documented child collections rather than scanning every view. Missing and
// duplicate matches return bam.ErrNotFound and bam.ErrAmbiguous respectively.
func (c *Client) LookupZone(ctx context.Context, viewID int64, name string) (Zone, error) {
	if viewID <= 0 {
		return Zone{}, fmt.Errorf("dns: view id must be positive")
	}
	absolute, err := parseZoneName(name)
	if err != nil {
		return Zone{}, err
	}
	return resource.One(func(yield func(Zone, error) bool) {
		c.lookupZone(ctx, viewID, absolute, yield)
	})
}

type zoneBranch struct {
	zones Zones
	name  zoneName
}

func (c *Client) lookupZone(ctx context.Context, viewID int64, name zoneName, yield func(Zone, error) bool) {
	pending := []zoneBranch{{zones: c.Zones(viewID)}}
	visited := make(map[int64]bool)
	for len(pending) > 0 {
		branch := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		for zone, err := range branch.zones.All(ctx, bam.ListOptions{}) {
			if err != nil {
				yield(Zone{}, err)
				return
			}
			absolute, err := branch.check(zone, viewID)
			if err != nil {
				yield(Zone{}, err)
				return
			}
			if absolute == name {
				if !yield(zone, nil) {
					return
				}
			} else if name.isBelow(absolute) {
				if visited[zone.ID] {
					yield(Zone{}, &bam.ProtocolError{Method: http.MethodGet, Detail: "zone hierarchy repeats a resource"})
					return
				}
				visited[zone.ID] = true
				pending = append(pending, zoneBranch{zones: c.Subzones(zone.ID), name: absolute})
			}
		}
	}
}

func (b zoneBranch) check(zone Zone, viewID int64) (zoneName, error) {
	absolute, err := parseZoneName(zone.AbsoluteName)
	if err != nil || zone.ID <= 0 || zone.Type != "Zone" || zone.View.ID != viewID {
		return zoneName{}, &bam.ProtocolError{Method: http.MethodGet, Detail: "zone lookup requires a complete ordinary zone in the selected view", Err: err}
	}
	if b.name.value != "" && !absolute.isBelow(b.name) {
		return zoneName{}, &bam.ProtocolError{Method: http.MethodGet, Detail: "subzone does not descend from its parent"}
	}
	return absolute, nil
}
