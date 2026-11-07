// Package dhcp selects Address Manager RESTful v2 DHCP resource collections.
//
// Resources use bam.Document so fields outside this package's knowledge survive
// reads and replacements. Construct request bodies from the target Address
// Manager's /api/openapi.json schema. Resource types, required fields, and allowed
// operations depend on that schema; the selectors do not invent DHCP enums or
// translate legacy property strings.
//
// Page, All, and Create use the selected parent collection. Get, Document,
// Replace, and Delete address the resource's global collection by ID. Replace
// requires a complete document. Deployment option and role collections can also
// contain resources for other services; callers can supply a schema-supported
// filter. Editing an inherited option updates its original resource; overriding
// it requires creating an option at the desired child scope.
//
// MACPools and MACAddresses select resources, not pool membership. Deleting a MAC
// address deletes the resource; it does not merely unlink it from a pool.
//
// Endpoint mappings are based on the BlueCat Integrity 9.6 migration guide and
// its HAL collection examples:
//
// https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/v1-REST-API-to-RESTful-v2-API-migration-guide/9.6.0
//
// https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/HAL-links/9.6.0
//
// Option and role scopes follow the documented DHCP hierarchy:
//
// https://docs.bluecatnetworks.com/r/Address-Manager-Administration-Guide/Adding-DHCPv4-client-deployment-options/9.6.0
//
// https://docs.bluecatnetworks.com/r/Address-Manager-Administration-Guide/DHCP-deployment-roles/9.6.0
package dhcp
