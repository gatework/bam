// Package ipam manages address space, IP allocations, MAC resources, and lookups
// through the Address Manager RESTful v2 API.
//
// Blocks, networks, and addresses use typed specifications for creation and
// explicit changes for updates. Updates read the full resource before applying
// changes, preserving server fields absent from the read models. Parent IDs
// scope collection reads and creates; item operations use global resource IDs.
//
// STATIC assigns a fixed address, RESERVED holds an address administratively,
// and DHCP_RESERVED binds an address to a MAC for DHCP. AllocateNext delegates
// selection to a single server-side POST. An Available query does not reserve an
// address. Release deletes the allocation and does not delete its MAC resource.
//
// Exact IP and MAC lookups require a configuration scope and report ambiguous
// results. MAC values normalize 48-bit hardware addresses. IPv4 is the primary
// workflow; explicit IPv6 block, network, and address operations remain available.
//
// The resource contracts come from the vendor-exported BAM 9.5 OpenAPI schema,
// supplemented by documented BAM 9.6 field and HTTP-header changes. Appliances
// remain authoritative for allocation policy, inheritance, and reference checks.
package ipam
