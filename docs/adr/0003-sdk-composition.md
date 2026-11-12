# SDK composition and dependency layers

Status: accepted for v0.4.0. Refines ADR 0002 without changing wire contracts.

## Context

BAM is an external-system SDK. Its public packages are named for the capabilities
they adapt from REST, not for an application's pure business-domain layer.
Calling every package a domain model obscured the dependency roles in ARC-1 and
ARC-2. Exported concrete embedding also exposed implementation handles and let
changes in their method sets silently expand the public API, contrary to CMP-1.

## Decision

Apply the standard's inward dependency rule to these SDK layers:

1. `bam` owns HTTP, authentication, JSON documents, references and pagination.
   It imports no other package in this module.
2. `internal/resource` implements typed collection mechanics on the protocol
   kernel. It knows no endpoint-specific model and imports only `bam` locally.
3. `configuration`, `deployment`, `dns`, `dns/responsepolicy` and `ipam` adapt
   endpoint contracts using those two layers. They do not import each other.
4. `dhcp` composes IPAM address, MAC and range contracts for DHCP operations.
   `dns/accesscontrol` composes DNS option contracts for ACL configuration.
   Neither lower layer may depend on these composed capabilities.

These are adapter and adapter-composition layers, not an exception allowing a
pure business domain to depend on I/O. Applications define their own consumer
ports and business rules outside the SDK. New independent business-domain
packages must remain free of transport and adapter dependencies.

`architecture.yaml` records the allowed graph and also rejects process,
filesystem and database imports in the SDK adapters. Naming a new package does
not grant it access to any layer; it must be declared and constrained first.

Public handles store shared resource mechanics in private named fields. They
declare each supported operation and delegate it explicitly, so changes to an
internal type cannot add a public method or expose a replaceable implementation
field. Existing method signatures and HTTP behavior are preserved. This is API
ownership, not a second implementation of CRUD or a compatibility layer.

Read DTOs declare their JSON fields directly. Request data that composes other
request data uses named fields. Query selectors with mutual-exclusion invariants
use private fields and constructors; an unset query fails before a request.

## Verification and consequences

Public HTTP contract tests constrain scopes, payloads and mutation behavior.
`TestPublicAPI_Encapsulation` protects the concrete-composition boundary. Before
switching handles, compare their value and pointer method sets and run the same
HTTP contracts against both implementations.

Direct access to embedded implementation fields is intentionally removed in
v0.4.0. Follow the [migration guide](../migration-v0.4.0.md). No resource is
deployed, moved or changed on an appliance by this structural migration.
