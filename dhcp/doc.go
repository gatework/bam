// Package dhcp manages DHCPv4 dynamic ranges, reserved MAC bindings, MAC pools,
// deployment options, and deployment roles through Address Manager RESTful v2.
//
// Ranges use IPv4DHCPRange, and reservations use IPv4Address resources with
// DHCP_RESERVED state. MAC resources belong to package ipam; MACMembers changes
// only a pool relationship and never deletes the MAC resource on unlink.
//
// Option and role collections enforce DHCP type filters. Options represent
// configured local overrides; deleting an override restores parent inheritance.
// The package does not synthesize an effective inherited configuration. Typed
// option helpers cover common RFC DHCPv4 codes; server-specific option names and
// definition IDs come from OptionDefinitions on the target appliance.
//
// Mutations do not submit deployment jobs. New MAC pools disable instant
// deployment, while existing appliance instant-deployment settings are retained
// unless changed explicitly. Deploy configured services through package deployment.
//
// Contracts use the vendor-exported BAM 9.5 OpenAPI schema plus documented BAM
// 9.6 changes. RawRanges provides explicit access to DHCPv6 and version-specific
// range fields outside this DHCPv4 model.
package dhcp
