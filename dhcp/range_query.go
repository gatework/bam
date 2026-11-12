package dhcp

import (
	"context"
	"fmt"
	"iter"
	"net/netip"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
	"github.com/gatework/bam/ipam"
)

// FindByRange requires one DHCPv4 range with these inclusive endpoints in this
// network. Offset/size and percentage expressions are creation inputs, not IDs.
func (r Ranges) FindByRange(ctx context.Context, span ipam.AddressRange) (Range, error) {
	if err := span.Validate(); err != nil {
		return Range{}, err
	}
	if !span.Start.Is4() {
		return Range{}, fmt.Errorf("dhcp: range lookup requires ipv4 endpoints")
	}
	return resource.One(r.All(ctx, bam.ListOptions{Where: bam.Match("range", span.String()), Limit: 2}))
}

// Containing lists DHCPv4 ranges containing address in this network. It does
// not assert that the address is free or outside the range's exclusion list.
func (r Ranges) Containing(ctx context.Context, address netip.Addr, options bam.ListOptions) iter.Seq2[Range, error] {
	options.Where = bam.And(options.Where, bam.Contains("range", address.String()))
	items := r.All(ctx, options)
	return func(yield func(Range, error) bool) {
		if !address.Is4() {
			yield(Range{}, fmt.Errorf("dhcp: range lookup requires an ipv4 address"))
			return
		}
		for item, err := range items {
			if !yield(item, err) || err != nil {
				return
			}
		}
	}
}
