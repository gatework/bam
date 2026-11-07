# Changelog

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
