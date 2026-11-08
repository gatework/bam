package ipam

import (
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"github.com/gatework/bam"
)

// WithNoGateway controls gateway creation when creating or updating a network.
func WithNoGateway(enabled bool) bam.WriteOption {
	return bam.WriteHeader("x-bcn-no-gateway", strconv.FormatBool(enabled))
}

// WithPingBeforeAssignment overrides the network's address availability probe.
func WithPingBeforeAssignment(enabled bool) bam.WriteOption {
	return bam.WriteHeader("x-bcn-ping-before-assignment", strconv.FormatBool(enabled))
}

// WithReverseRecord controls reverse records for linked hosts during assignment.
func WithReverseRecord(enabled bool) bam.WriteOption {
	return bam.WriteHeader("x-bcn-create-reverse-record", strconv.FormatBool(enabled))
}

// WithExcludedAddresses excludes exact IPv4 addresses from server-side selection.
// Multiple exclusion options accumulate rather than overwrite one another.
func WithExcludedAddresses(addresses ...netip.Addr) bam.WriteOption {
	addresses = slices.Clone(addresses)
	return func(header http.Header) error {
		values := make([]string, len(addresses))
		for i, address := range addresses {
			if !address.Is4() {
				return fmt.Errorf("ipam: exclusions require ipv4 addresses")
			}
			values[i] = address.String()
		}
		return appendExclusions(header, values)
	}
}

// WithExcludedRanges excludes inclusive IPv4 ranges from server-side selection.
func WithExcludedRanges(ranges ...AddressRange) bam.WriteOption {
	ranges = slices.Clone(ranges)
	return func(header http.Header) error {
		values := make([]string, len(ranges))
		for i, r := range ranges {
			if err := r.Validate(); err != nil {
				return err
			}
			if !r.Start.Is4() {
				return fmt.Errorf("ipam: exclusions require ipv4 ranges")
			}
			values[i] = r.String()
		}
		return appendExclusions(header, values)
	}
}
func appendExclusions(header http.Header, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("ipam: at least one exclusion is required")
	}
	const key = "x-bcn-excluded-addresses"
	if previous := header.Get(key); previous != "" {
		values = append([]string{previous}, values...)
	}
	return bam.WriteHeader(key, strings.Join(values, ","))(header)
}
