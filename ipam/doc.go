// Package ipam manages IPv4 and IPv6 blocks, networks, and addresses using the
// Address Manager RESTful v2 API documented for BAM 9.6.
//
// Parent IDs scope collection reads and creates; item operations address the
// global resource ID. BlockSpec and NetworkSpec create explicit CIDR ranges,
// while AddressSpec assigns an explicit IP address. The specs infer the v2
// resource type from net/netip values and validate them before a request is sent.
//
// These models cover documented common fields, not the entire server schema.
// Use Collection.Document and Replace to preserve unknown fields during full
// replacement. Collection.Create also accepts a bam.Document for fields or
// allocation modes documented by the target server's /api/openapi.json. In
// particular, this package does not infer an atomic next-address allocation
// request from the migration guide's endpoint mapping alone.
//
// Endpoint and field references:
//
//   - https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/v1-REST-API-to-RESTful-v2-API-migration-guide/9.6.0
//   - https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/Collections/9.6.0
//   - https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/Filter/9.6.0
//   - https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/Fields/9.6.0
package ipam
