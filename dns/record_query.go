package dns

import (
	"context"
	"fmt"
	"iter"
	"net/netip"
	"slices"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// RecordQuery selects records by documented RESTful v2 fields. Empty selectors
// are omitted. Use the parent Records selector to restrict a query to a zone.
type RecordQuery struct {
	ListOptions     bam.ListOptions
	ConfigurationID int64
	Name            string
	AbsoluteName    string
	Kind            RecordKind
	RecordType      string
	LinkedRecordID  int64
}

func (q RecordQuery) options() bam.ListOptions {
	options := q.ListOptions
	filters := []bam.Filter{options.Where}
	if q.ConfigurationID != 0 {
		filters = append(filters, bam.MatchID("configuration.id", q.ConfigurationID))
	}
	if q.Name != "" {
		filters = append(filters, bam.Match("name", q.Name))
	}
	if q.AbsoluteName != "" {
		filters = append(filters, bam.Match("absoluteName", q.AbsoluteName))
	}
	if q.Kind != "" {
		filters = append(filters, bam.Match("type", string(q.Kind)))
	}
	if q.RecordType != "" {
		filters = append(filters, bam.Match("recordType", q.RecordType))
	}
	if q.LinkedRecordID != 0 {
		filters = append(filters, bam.MatchID("linkedRecord.id", q.LinkedRecordID))
	}
	options.Where = bam.And(filters...)
	return options
}

// Search lazily retrieves every record matching a typed query.
func (r RecordItems) Search(ctx context.Context, query RecordQuery) iter.Seq2[Record, error] {
	return r.All(ctx, query.options())
}

// Find requires exactly one matching record. Missing and ambiguous matches
// return bam.ErrNotFound and bam.ErrAmbiguous respectively.
func (r RecordItems) Find(ctx context.Context, query RecordQuery) (Record, error) {
	query.ListOptions.Limit = 2
	return resource.One(r.Search(ctx, query))
}

// RecordsByIP resolves an IP within one configuration, then follows its record
// associations. It avoids relying on an undocumented nested-address filter.
// A record linked more than once is yielded once. No matching address is an
// empty result, rather than a mutation or a scan of all DNS records. ID is
// included in projected responses so associations can be deduplicated.
func (c *Client) RecordsByIP(ctx context.Context, configurationID int64, address netip.Addr, options bam.ListOptions) iter.Seq2[Record, error] {
	options.Fields = slices.Clone(options.Fields)
	options.OrderBy = slices.Clone(options.OrderBy)
	if len(options.Fields) > 0 && !slices.Contains(options.Fields, "id") {
		options.Fields = append(options.Fields, "id")
	}
	return func(yield func(Record, error) bool) {
		if !address.IsValid() || address.Zone() != "" {
			yield(Record{}, fmt.Errorf("dns: valid unscoped address is required"))
			return
		}
		addresses := bam.NewCollection[Address](c.client, "addresses")
		seen := make(map[int64]struct{})
		query := bam.ListOptions{Where: bam.And(
			bam.MatchID("configuration.id", configurationID),
			bam.Match("address", address.String()),
		)}
		for matched, err := range addresses.All(ctx, query) {
			if err != nil {
				yield(Record{}, err)
				return
			}
			if matched.ID <= 0 {
				yield(Record{}, fmt.Errorf("dns: address query returned an invalid resource id"))
				return
			}
			if !c.yieldAddressRecords(ctx, matched.ID, options, seen, yield) {
				return
			}

		}
	}
}

func (c *Client) yieldAddressRecords(ctx context.Context, addressID int64, options bam.ListOptions, seen map[int64]struct{}, yield func(Record, error) bool) bool {
	for record, err := range c.RecordsForAddress(addressID).All(ctx, options) {
		if err != nil {
			yield(Record{}, err)
			return false
		}
		if record.ID <= 0 {
			yield(Record{}, fmt.Errorf("dns: record association returned an invalid resource id"))
			return false
		}
		if _, exists := seen[record.ID]; exists {
			continue
		}
		seen[record.ID] = struct{}{}
		if !yield(record, nil) {
			return false
		}
	}
	return true
}
