# Changelog

## v0.4.0

### Added

- Find ordinary DNS zones by absolute name within a view, including nested and
  multi-label zones, without crossing views or choosing an ambiguous match.
- Find local DNS/DHCP options by name or definition and exact server scope;
  DHCP client options can also be selected by RFC code.
- Find DHCP reservations by IP or MAC, MAC pools by name, and DHCP ranges by
  endpoints or containment. Observe network addresses in `DHCP_LEASED` state
  with their reported client identifiers and lease timestamps.
- Add DHCPv4 NTP, domain search, TFTP server and boot filename option constructors.

### Changed

- Keep public handle implementations private and declare supported operations
  explicitly. Existing method signatures are retained; direct embedded-field
  access changes. Read DTOs expose ID/type/name directly and DNS metadata and
  pagination use named fields. See [migration](docs/migration-v0.4.0.md).
- Construct immutable DNS/DHCP option queries by name, definition or client code;
  reject invalid inputs during construction and unset queries before HTTP.
- Document SDK adapter layers and enforce their inward dependencies. Add an API
  encapsulation check and bounded parser/value fuzz smoke tests to `make check`.

### Fixed

- Reject zone lookup names with empty labels, including two trailing dots, before
  HTTP; reject malformed names in hierarchy responses as protocol errors.
- Reject explicit JSON null option values before DNS/DHCP mutations, including
  typed nil values. Empty arrays, false, zero and untouched fields retain their
  existing meaning; remove a local override to restore inheritance.
- Correct the documented minimum-toolchain check to Go 1.26.

Runtime dependencies remain standard-library-only. Live BAM acceptance is separate.

## v0.3.0

- Require Go 1.26 or later and validate Go 1.26.x and 1.27.x patches.
- Adopt go-standards v1.0.2 through shared formatting, lint, architecture and `make check` gates; runtime dependencies remain standard-library-only.
- Document typed domain operations and split request preparation and iterator delivery into internal helpers, preserving HTTP, pagination and mutation contracts.

Public Go API names and RESTful v2 capabilities are unchanged. Live appliance acceptance remains separate.

## v0.2.0

Breaking domain API expansion; module path remains `github.com/gatework/bam`.

- Typed CRUD and changes for configurations, DNS views/zones/record variants,
  IPAM blocks/networks/addresses/MACs, DHCPv4 ranges, reservations, options and roles.
- Scoped IP/MAC/name queries, atomic server-selected IPv4 allocation, and explicit
  distinction between static, administrative and DHCP reservations.
- Separate response policies/local RPZs and ordered DNS client ACLs, with DNS
  deployment options and roles.
- Relationship Link/Unlink operations preserve resource lifetime; typed updates
  retain unknown JSON fields and complete host-address associations.
- Explicit full, differential, quick, validation and selective deployments,
  read-only target/task discovery and bounded task waiting.
- Source-backed capability matrix, executable examples and HTTP contract tests.
  BAM 9.6 schema gaps and live appliance acceptance remain explicitly documented.

Known limits include Response Policies view binding/ordering, HTTPS/SVCB/URI
typed writes and multi-primary roles. See the
[versioned capability matrix](https://github.com/gatework/bam/blob/v0.2.0/docs/rest-v2-coverage.md).

## v0.1.0

Initial release of `github.com/gatework/bam`, a standard-library-only Go client
for BlueCat Address Manager RESTful v2, using the BAM 9.6 protocol documentation.

- Separate configuration, DNS, IPAM, DHCP, and deployment packages.
- Per-client Basic/OAuth authentication, cancellable session changes, bounded
  JSON responses, and typed HTTP/protocol/pending-workflow errors.
- Generic resource collections with lazy HAL pagination and lossless full
  document replacement.
- Typed configuration, DNS, and IPAM read models and common creation specs using
  `net/netip`; raw documents support additional device-specific fields.
- Contract tests for authentication isolation, cancellation, resource identity,
  malformed responses, pagination safety, and HTTP 202 workflow handling.

Requires Go 1.23 or later; recommends the patched Go 1.27.3 toolchain.
No live BAM appliance acceptance has been performed.
DHCP/deployment request schemas and additional allocation modes are supplied as
documents based on the target appliance's OpenAPI specification.
