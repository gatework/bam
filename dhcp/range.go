package dhcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/netip"
	"strconv"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
	"github.com/gatework/bam/ipam"
)

// Range describes a DHCPv4 dynamic allocation range and its exclusions.
type Range struct {
	ID                         int64            `json:"id"`
	Type                       string           `json:"type"`
	Name                       string           `json:"name"`
	Configuration              bam.Reference    `json:"configuration"`
	Range                      string           `json:"range"`
	SplitAroundStaticAddresses bool             `json:"splitAroundStaticAddresses"`
	ExclusionRanges            []string         `json:"exclusionRanges"`
	LowWaterMark               *int             `json:"lowWaterMark"`
	HighWaterMark              *int             `json:"highWaterMark"`
	InheritedFields            []string         `json:"_inheritedFields"`
	Usage                      map[string]int64 `json:"usage"`
}

// Span selects an IPv4 range by endpoints or relative to its parent network.
// Constructors defer validation until the specification is encoded.
type Span struct {
	endpoints ipam.AddressRange
	offset    int64
	size      uint64
	percent   int
	mode      uint8
}

// Between constructs an inclusive IPv4 span and retains validation errors until encoding.
func Between(start, end netip.Addr) Span {
	return Span{endpoints: ipam.AddressRange{Start: start, End: end}, mode: 1}
}

// OffsetSize uses an address count. A negative offset starts from the network's end.
func OffsetSize(offset int64, size uint64) Span { return Span{offset: offset, size: size, mode: 2} }

// OffsetPercent selects a percentage of the network, from 1 through 100.
func OffsetPercent(offset int64, percent int) Span {
	return Span{offset: offset, percent: percent, mode: 3}
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s Span) MarshalJSON() ([]byte, error) {
	var value string
	switch s.mode {
	case 1:
		if err := s.endpoints.Validate(); err != nil {
			return nil, err
		}
		if !s.endpoints.Start.Is4() {
			return nil, fmt.Errorf("dhcp: range must use ipv4 addresses")
		}
		value = s.endpoints.String()
	case 2, 3:
		if s.offset <= -(1<<32) || s.offset >= 1<<32 {
			return nil, fmt.Errorf("dhcp: range offset exceeds ipv4 space")
		}
		value = strconv.FormatInt(s.offset, 10) + ","
		if s.mode == 2 {
			if s.size == 0 || s.size > 1<<32 {
				return nil, fmt.Errorf("dhcp: range size must be between 1 and 4294967296")
			}
			value += strconv.FormatUint(s.size, 10)
		} else {
			if s.percent < 1 || s.percent > 100 {
				return nil, fmt.Errorf("dhcp: range percentage must be between 1 and 100")
			}
			value += strconv.Itoa(s.percent) + "%"
		}
	default:
		return nil, fmt.Errorf("dhcp: range selection is required")
	}
	return json.Marshal(value)
}

// RangeSpec creates a dynamic range. SplitAroundStaticAddresses prevents static,
// reserved and gateway addresses in the interval from joining the dynamic pool.
type RangeSpec struct {
	Name                       string              `json:"name,omitempty"`
	Range                      Span                `json:"range"`
	SplitAroundStaticAddresses bool                `json:"splitAroundStaticAddresses"`
	ExclusionRanges            []ipam.AddressRange `json:"exclusionRanges,omitempty"`
	LowWaterMark               *int                `json:"lowWaterMark,omitempty"`
	HighWaterMark              *int                `json:"highWaterMark,omitempty"`
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s RangeSpec) MarshalJSON() ([]byte, error) {
	if _, err := s.Range.MarshalJSON(); err != nil {
		return nil, err
	}
	if err := validateExclusions(s.ExclusionRanges); err != nil {
		return nil, err
	}
	if err := validateWaterMarks(s.LowWaterMark, s.HighWaterMark); err != nil {
		return nil, err
	}
	type spec RangeSpec
	return json.Marshal(struct {
		spec
		Type string `json:"type"`
	}{spec: spec(s), Type: "IPv4DHCPRange"})
}

// RangeChanges changes a range through a complete read/edit/PUT cycle.
type RangeChanges struct {
	Name                       bam.Field[string]
	Range                      bam.Field[Span]
	SplitAroundStaticAddresses bam.Field[bool]
	ExclusionRanges            bam.Field[[]ipam.AddressRange]
	LowWaterMark               bam.Field[int]
	HighWaterMark              bam.Field[int]
}

// Apply validates and applies explicitly set fields to the complete resource document.
func (c RangeChanges) Apply(d bam.Document) error {
	if c.Range.IsNull() || c.SplitAroundStaticAddresses.IsNull() || c.ExclusionRanges.IsNull() {
		return fmt.Errorf("dhcp: range, split setting, and exclusions cannot be null")
	}
	if r, ok := c.Range.Value(); ok {
		if _, err := r.MarshalJSON(); err != nil {
			return err
		}
	}
	if exclusions, ok := c.ExclusionRanges.Value(); ok {
		if err := validateExclusions(exclusions); err != nil {
			return err
		}
		if exclusions == nil {
			c.ExclusionRanges = bam.Set([]ipam.AddressRange{})
		}
	}
	if err := errors.Join(c.Name.Apply(d, "name"), c.Range.Apply(d, "range"), c.SplitAroundStaticAddresses.Apply(d, "splitAroundStaticAddresses"), c.ExclusionRanges.Apply(d, "exclusionRanges"), c.LowWaterMark.Apply(d, "lowWaterMark"), c.HighWaterMark.Apply(d, "highWaterMark")); err != nil {
		return err
	}
	var limits struct {
		Low  *int `json:"lowWaterMark"`
		High *int `json:"highWaterMark"`
	}
	if err := d.Decode(&limits); err != nil {
		return err
	}
	return validateWaterMarks(limits.Low, limits.High)
}
func validateExclusions(ranges []ipam.AddressRange) error {
	for _, r := range ranges {
		if err := r.Validate(); err != nil {
			return err
		}
		if !r.Start.Is4() {
			return fmt.Errorf("dhcp: exclusions must use ipv4 addresses")
		}
	}
	return nil
}
func validateWaterMarks(low, high *int) error {
	for _, v := range []*int{low, high} {
		if v != nil && (*v < 0 || *v > 100) {
			return fmt.Errorf("dhcp: water marks must be between 0 and 100")
		}
	}
	if low != nil && high != nil && *low > *high {
		return fmt.Errorf("dhcp: low water mark exceeds high water mark")
	}
	return nil
}

// WithOrphanedAddressState controls the state of allocated IPv4 addresses left
// outside a resized DHCP range. The appliance validates the supported target state.
func WithOrphanedAddressState(state ipam.State) bam.WriteOption {
	return func(headers http.Header) error {
		if state == "" {
			return fmt.Errorf("dhcp: orphaned address state is required")
		}
		return bam.WriteHeader("x-bcn-orphaned-address-state", string(state))(headers)
	}
}

// Ranges provides typed DHCPv4 range creation under a network.
type Ranges struct {
	resources resource.Collection[Range, RangeSpec, RangeChanges]
}

// RangeItems provides existing DHCPv4 range operations by global ID.
type RangeItems struct {
	resources resource.Items[Range, RangeChanges]
}

// Page fetches one page within the selected collection scope.
func (r Ranges) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Range], error) {
	return r.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r Ranges) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Range, error] {
	return r.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (r Ranges) Get(ctx context.Context, id int64) (Range, error) {
	return r.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (r Ranges) Document(ctx context.Context, id int64) (bam.Document, error) {
	return r.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (r Ranges) Update(ctx context.Context, id int64, changes RangeChanges, options ...bam.WriteOption) (Range, error) {
	return r.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (r Ranges) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return r.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (r Ranges) Create(ctx context.Context, spec RangeSpec, options ...bam.WriteOption) (Range, error) {
	return r.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (r RangeItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Range], error) {
	return r.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r RangeItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Range, error] {
	return r.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (r RangeItems) Get(ctx context.Context, id int64) (Range, error) {
	return r.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (r RangeItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return r.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (r RangeItems) Update(ctx context.Context, id int64, changes RangeChanges, options ...bam.WriteOption) (Range, error) {
	return r.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (r RangeItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return r.resources.Delete(ctx, id, options...)
}
