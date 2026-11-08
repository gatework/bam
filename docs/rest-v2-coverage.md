# RESTful v2 capability matrix

This matrix describes the v0.2.0 typed domain API, not every BAM endpoint.
The baseline is **BAM 9.6, IPv4 first**, with existing IPv6 block/network/address
basics retained. No live appliance acceptance has been performed.

## Contract sources

1. [Preserved BAM OpenAPI export](https://github.com/umich-vci/gobam2/blob/e894d31d00a85fb41f1ab648882600e5476ca091/api/openapi.yaml):
   vendor schema redistributed in `umich-vci/gobam2`, `info.version: 9.5.0`.
   Immutable revision: `e894d31d00a85fb41f1ab648882600e5476ca091`.
   SHA-256: `1804cd1da1b51d446d3c8fadb2f7425f5c5dc1f83414642acfdd5fc470180b09`.
   Used for endpoint methods, resource discriminators, required fields, enum
   values and examples. The entire vendor schema is not vendored into this SDK.
2. [Official 9.6 changes](https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/What-s-New-in-v9.6.0/9.6.0):
   host-address PUT payloads, field renames, dynamic
   IPv4 updates, filters, workflow responses and newly introduced resource types.
3. [HTTP headers](https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/HTTP-headers/9.6.0),
   [updates](https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/Updating-resources/9.6.0),
   [filters](https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/Filter/9.6.0)
   and [v1-to-v2 mapping](https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/v1-REST-API-to-RESTful-v2-API-migration-guide/9.6.0)
   establish operation options and protocol semantics. A legacy method mapping
   alone is insufficient evidence for a new JSON request body.
4. [DNS deployment option reference](https://docs.bluecatnetworks.com/r/Address-Manager-Administration-Guide/Reference-DNS-deployment-options/9.6.0)
   and [RPZ configuration](https://docs.bluecatnetworks.com/r/Address-Manager-Administration-Guide/Adding-Response-Policy-zones-with-BlueCat-Security-Feed/9.6.0)
   supply policy ordering and role/deployment context.

This combination does not establish the full 9.6 schema. Obtain the target
appliance's `/api/openapi.json` as described in
[Using an API client](https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/Using-an-API-client/9.6.0)
before closing the gaps below. BAM may ignore unknown request fields, so a
successful response alone cannot prove that an unverified field took effect.

## Implemented capabilities

Paths below omit `/api/v2`. `{parent}` denotes only the listed supported scopes.
CRUD means typed POST, GET/list, GET/edit/PUT, and DELETE. Global selectors expose
existing-item operations, not an unsupported global POST. Every list supports
bounded HAL pagination; item operations address global IDs except explicit links.

| Domain | Go entry point | REST resources and methods | Supported business behavior |
| --- | --- | --- | --- |
| Configuration | `Configurations` | `/configurations` and `/configurations/{id}`; CRUD | Name, description and group changes preserve the other configuration settings. |
| DNS namespace | `Views`, `Zones`, `Subzones`, global selectors; `ExternalHostsZone` | `/configurations/{id}/views`, `/views/{id}/zones`, `/zones/{id}/zones`; item CRUD; external zone GET only | View/ordinary-zone CRUD with fixed type predicates. Exact read-only external-host-zone lookup avoids treating it as an ordinary mutable zone. |
| DNS records | `Records`, `GlobalRecords` | `/zones/{id}/resourceRecords`; `/resourceRecords/{id}`; CRUD | Host (A/AAAA), Alias (CNAME), MX, TXT, SRV, HINFO, NAPTR, Generic and ExternalHost specifications and changes. Generic records use BAM's `recordType`/`rdata` for PTR, CAA, NS and other generic types. |
| DNS searches | `Search`, `Find`, `RecordsByIP` | Record collection filters; address-to-record collection GET | Name, absolute name, subtype, generic RR type, configuration, linked record and IP lookup; exact lookup rejects ambiguity. |
| DNS associations | `Addresses`, `DependentRecords`, `RecordsForAddress` | `/resourceRecords/{id}/addresses`, `/resourceRecords/{id}/dependentRecords`; `/addresses/{id}/resourceRecords[/{recordID}]` | Read complete associations; link existing hosts; unlink without deleting records. Host updates preserve all pages of addresses unless explicitly replaced. |
| DNS options | `ConfigurationOptions`, `ViewOptions`, `ZoneOptions`, `ServerOptions`, `BlockOptions`, `NetworkOptions` | Scoped `/deploymentOptions`; global `/deploymentOptions/{id}`; CRUD | `DNSOption` by name or definition, lossless JSON values, nullable server scope, deletion to restore inheritance. |
| DNS roles | `ViewRoles`, `ZoneRoles`, `BlockRoles`, `NetworkRoles`, `RoleInterfaces` | Scoped `/deploymentRoles`; `/deploymentRoles/{id}`; CRUD; interfaces GET | Known primary/secondary/forwarding/stub/recursive/none roles, interface assignment, reverse view and NS TTL. Multi-primary extension excluded below. |
| Domain blacklist | `responsepolicy.Policies`, `Items`, `Zones` | `/configurations/{id}/responsePolicies`, `/responsePolicies/{id}/policyItems`, `/views/{id}/zones`; item CRUD | ALLOWLIST, BLOCKLIST, BLACKHOLE, REDIRECT; policy items; LOCAL RPZ with policy reference, then ordinary DNS role assignment. |
| Client blacklist | `accesscontrol.Lists`, `AllowQuery`, `AllowRecursion` | `/configurations/{id}/accessControlLists`, `/accessControlLists/{id}`; CRUD; DNS option CRUD | Ordered address/CIDR/ACL/key elements, negation, named ACL binding through `allow-query` and `allow-recursion`. |
| IP space | `Blocks`, `Subblocks`, `Networks`, global selectors | Configuration/block `/blocks`, block `/networks`; `/blocks/{id}`, `/networks/{id}`; CRUD | IPv4/IPv6 CIDR resources; metadata updates; verified IPv4 gateway, ping and dynamic-update settings. |
| IP allocation | `Addresses`, `AssignStatic`, `Reserve`, `AllocateNext`, `Release` | `/networks/{id}/addresses`, `/addresses/{id}`; CRUD | Exact assignment and one-POST next-free IPv4 allocation; optional ping/exclusions; explicit STATIC, RESERVED and DHCP_RESERVED semantics. Release deletes the allocation, not the MAC. |
| IP/MAC search | `LookupIP`, `LookupMAC`, `AddressesByMAC`, `NetworksContaining`, `Available` | Configuration/address filters; `/macAddresses/{id}/addresses`; network address collection GET | Scoped exact IP/MAC lookups, containment, state, name and prefix queries. Available means network `UNASSIGNED` IPAM candidates, not DHCP_FREE. Lease timestamps remain observable. |
| MAC resources | `MACAddresses`, `GlobalMACAddresses` | `/configurations/{id}/macAddresses`, `/macAddresses/{id}`; CRUD | Validated 48-bit MAC values with normalized wire representation, metadata updates, explicit resource deletion. |
| DHCP reservations | `Reservations` | Network address POST/GET; `/addresses/{id}` GET/PUT/DELETE | Exact or next-free IPv4 DHCP reservation requiring a MAC. Reads/mutations verify `DHCP_RESERVED`; administrative reservations use IPAM. |
| DHCP ranges | `Ranges`, `GlobalRanges` | `/networks/{id}/ranges`, `/ranges/{id}`; CRUD | `IPv4DHCPRange` endpoints, offset/size or percentage, exclusion ranges, static-address splitting, usage thresholds and resize header. |
| DHCP pools | `MACPools`, `MACMembers` | `/configurations/{id}/macPools`, `/macPools/{id}`; CRUD; nested `/macAddresses[/{id}]` GET/POST/DELETE | Ordinary MACPool resources, explicit membership linking/unlinking. New pools set instant deployment false; existing settings survive updates. |
| DHCP options | Scope-specific `Options`, `GlobalOptions`, `OptionDefinitions` | Scoped `/deploymentOptions`; item CRUD; `/deploymentOptionDefinitions` GET | DHCPv4 client/service/raw options. Typed router/DNS/domain/lease-time values; local overrides and server scopes. Scopes: configuration, block, network, range, address, MAC/pool, server/group. |
| DHCP roles | `BlockRoles`, `NetworkRoles`, `MACPoolRoles`, `RoleInterfaces` | Scoped `/deploymentRoles`; item CRUD; interfaces GET | PRIMARY/NONE, network/published interfaces and failover primary/secondary associations. |
| Deployment discovery | `Servers`, `GlobalServers`, `ServerInterfaces`, `GlobalInterfaces` | Configuration/global `/servers`, server/global `/interfaces`; GET | Read-only deployment target and interface discovery. No server lifecycle mutations. |
| Deployment submission | `ForServer`, `ForZone`, `ForNetwork`, `Selective` | Server/zone/network `/deployments` POST; `/deployments` POST | Full, differential, validation, quick and selective request types. Service/type validation and selective batch/record/dynamic settings. |
| Deployment observation | `Tasks`, `ForServer`, `ZoneTasks`, `Wait` | `/deployments[/{id}]`, server/zone deployment collection GET | Pending/running/failure/completion/unknown states, warnings, bounded wait and context cancellation. No resubmission or implicit server cancellation. |

## Evidence boundaries and deferred contracts

| Capability | Current boundary |
| --- | --- |
| HTTPSRecord, SVCBRecord, URIRecord | New in 9.6; exact request schemas unavailable. Unknown read details are retained; normal deletion works, but typed creation/update is not claimed. |
| HostRecord `dependentRecords` creation | New nested alias payload is unverified. Create aliases separately through their verified host reference. |
| TTL inheritance reset | Zero TTL is supported. Null TTL reset is rejected because the available schema does not establish it. |
| Multi-primary DNS roles | 9.6 adds MULTI_PRIMARY/HIDDEN_MULTI_PRIMARY and server-group behavior. Exact group association payload remains unverified; typed writes are deferred. |
| View policy binding and ordering | Policy/local RPZ CRUD and role assignment are supported. The required Response Policies view option's exact REST value is unverified; no typed binding/ordering helper or automatic precedence claim. Creating a policy and deploying a server alone does not prove it is active. |
| External/FEED RPZs | Existing resources can be read/deleted. Typed creation/update is limited to LOCAL; feed subscription, licensing and external feeds are not modeled. |
| Effective/inherited option queries | APIs return locally configured option resources. No invented client-computed effective configuration or unsupported inherited-query endpoint. Deletion restores the parent inheritance relationship. |
| Network deployment history | Quick POST is supported; the available schema is inconsistent about history GET. Use global task reads by ID. |
| DHCPv6, DUID and DHCP leases | Basic IPv6 IPAM, including MAC-based DHCP_RESERVED where supported, remains available. DHCPv6 range/options modeling, typed DUID writes, and standalone lease mutation workflows are deferred. `RawRanges` is an explicit extension point, not typed coverage. |
| DenyMACPool | Ordinary pool handles filter and validate MACPool. DenyMACPool semantics are not silently mapped to ordinary pools. |
| Range resize orphan state | Header name is documented; allowed target-state combinations remain appliance-validated. |
| Other BAM administration | Server lifecycle, schedules, imports, backup, settings, DNSSEC, DHCP classes and other administration endpoints are outside this increment. |

## Behavioral guarantees and validation

- Domain writes accept typed specifications/changes; common transport code owns
  contexts, identity, headers, JSON, errors and pagination. No generated SDK or
  per-endpoint HTTP plumbing is added.
- Update retains unknown JSON fields, false/zero/null and integer precision;
  redacted representations cannot safely be replaced and are rejected. Host
  address write fields are hydrated from the complete association collection.
- `Set` and `Clear` distinguish explicit values from untouched fields. Nullable
  settings are checked per contract. GET/edit/PUT is not concurrency control.
- Fixed type/state predicates cannot be removed by caller list options. Exact
  lookups return not-found/ambiguous errors and propagate pagination failures.
- Mutation headers are per request. Failed writes are not automatically retried;
  HTTP 202 is a pending workflow error, not a completed resource mutation.
- Writes do not submit deployment tasks. Appliance dynamic/instant settings
  remain independently significant. Waiting stops locally on cancellation.

The HTTP suite covers CRUD wire shape, state distinctions, relationship lifetime,
lossless updates, pagination, errors/cancellation, typed discovery and deployment
state transitions. Executable examples demonstrate allocations and deployment
waiting. These are local client-contract checks, not proof of a real appliance's
permissions, schema variants, server policy, inherited values or deployed behavior.

To validate a target appliance, export its 9.6 schema, compare these request/response
contracts, and exercise the domain lifecycle in an isolated configuration: create,
read, update, relationship query, explicit deployment, observation and cleanup.
Preserve the difference between configured BAM state and observed DNS/DHCP service
behavior when recording acceptance results.
