package dns

import (
	"strconv"

	"github.com/gatework/bam"
)

// WithNamingPolicyOverride controls BAM naming-policy validation on record
// creation and update.
func WithNamingPolicyOverride(enabled bool) bam.WriteOption {
	return bam.WriteHeader("x-bcn-override-naming-policy", strconv.FormatBool(enabled))
}

// WithSameNameAsZone creates or updates a record at the zone apex.
func WithSameNameAsZone(enabled bool) bam.WriteOption {
	return bam.WriteHeader("x-bcn-same-as-zone", strconv.FormatBool(enabled))
}

// WithMultiLabelZoneNames controls multi-label names on zone create/update.
func WithMultiLabelZoneNames(enabled bool) bam.WriteOption {
	return bam.WriteHeader("x-bcn-allow-multi-label-zone-names", strconv.FormatBool(enabled))
}

// WithServerUpdates controls server updates on record updates/deletions.
// Server-side dynamic-update configuration can also affect deployment behavior.
func WithServerUpdates(enabled bool) bam.WriteOption {
	return bam.WriteHeader("x-bcn-update-servers", strconv.FormatBool(enabled))
}
