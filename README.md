# bam

A standard-library-only Go client for BlueCat Address Manager **RESTful v2**.
Requires Go 1.26 or later. There is no SOAP or
REST v1 compatibility layer.

Install:

```sh
go get github.com/gatework/bam@v0.4.0
```

Version `v0.4.0` makes handle implementations private and adds immutable option
queries. Existing handle method signatures and HTTP contracts are retained;
direct access to former embedded fields requires the
[v0.4.0 migration](docs/migration-v0.4.0.md). The module path remains
`github.com/gatework/bam`. The earlier
[typed API decision](docs/adr/0002-typed-domain-capabilities.md) covers v0.1 migration.

## Domains

| Package | Responsibility |
| --- | --- |
| `bam` | JSON transport, authentication, errors, pagination, lossless documents and explicit field changes |
| `configuration` | Configuration creation and metadata updates |
| `dns` | Views, zones, typed records, IP associations, searches, DNS options and roles |
| `dns/responsepolicy` | Domain response policies, policy items and local RPZs |
| `dns/accesscontrol` | Ordered client IP/CIDR ACLs and query/recursion access options |
| `ipam` | Blocks, networks, static assignments, administrative reservations, next-address allocation, MAC resources and IP/MAC queries |
| `dhcp` | DHCPv4 ranges, reservations, MAC pool membership, lease observations, options and roles |
| `deployment` | Server/interface discovery, explicit deployment submission and bounded task observation |

The protocol baseline is **BAM 9.6**. Resource contracts come from a preserved
vendor 9.5 OpenAPI export cross-checked with the official 9.6 changes. This is
not a complete 9.6 appliance schema or a live acceptance result. The
[capability matrix](docs/rest-v2-coverage.md) records implemented operations,
schema provenance, version differences and remaining gaps.

## Client and allocation

Create a shared client with `bam.New("https://bam.example")`, then authenticate
using `client.Login(ctx, username, password)`. An existing v2 token can be supplied
with `bam.WithBasicAuth(username, apiToken)`; `bam.WithBearerToken(token)` supports
externally managed OAuth credentials. Credentials are isolated per client.

All network operations take a caller-owned context. Logout changes the remote
session; `CloseIdleConnections` only closes idle connections. Use a separate,
bounded logout context if the operation's context has already expired.

```go
import (
    "context"
    "net/netip"

    "github.com/gatework/bam"
    "github.com/gatework/bam/ipam"
)

func assignServer(ctx context.Context, client *bam.Client, networkID int64) (ipam.Address, error) {
    return ipam.New(client).Addresses(networkID).AssignStatic(ctx, ipam.AddressSpec{
        Name:    "application",
        Address: netip.MustParseAddr("192.0.2.10"),
    })
}
```

`AssignStatic` creates a `STATIC` allocation. `Reserve` holds an IPv4 address as
`RESERVED` without a DHCP binding. `dhcp.Reservations(networkID).Create` creates
`DHCP_RESERVED` with a MAC. These are different business operations; see the
[executable allocation example](ipam/example_test.go).

`AllocateNext` sends one allocation request and lets BAM choose the address.
`Available` only queries unassigned IPAM candidates; it does not reserve them or
report DHCP lease availability. `LookupIP` and `LookupMAC` require a configuration,
and `AddressesByMAC` traverses the MAC's address association. Exact lookups return
`bam.ErrNotFound` or `bam.ErrAmbiguous` instead of choosing an arbitrary result.

## Reads and updates

Domain handles expose supported operations with typed specifications and changes.
Global selectors modify existing resources; creation selects a parent. Parent
scopes constrain listing and creation, while ordinary item operations use global
IDs. Relationship operations retain their nested scope: `Unlink` removes a
membership, while `Delete` removes a resource.

`Page` reads one page. `All` is a lazy `iter.Seq2`; breaking iteration stops further
requests. Filters compose with `bam.Match`, `MatchID`, `Contains`, `And` and `Or`:

```go
for address, err := range ipam.New(client).Addresses(networkID).All(ctx, bam.ListOptions{
    Where:   bam.Match("state", string(ipam.StateStatic)),
    OrderBy: []string{"id"},
    Limit:   200,
}) {
    if err != nil {
        return err
    }
    // Use address.Address, address.State and address.MACAddress.
}
```

Typed `Update` reads a complete document, applies selected changes and sends PUT.
It retains unmodeled fields and large integers. A zero `bam.Field` leaves a value
unchanged; `bam.Set(false)` and `bam.Set(0)` explicitly write zero values.
`bam.Clear[T]()` writes null only where that resource contract permits it.
Deleting a local DNS/DHCP option restores inheritance; an empty option value does
not. Updates are not atomic with concurrent writers and are never retried.

```go
_, err := dns.New(client).Records(zoneID).Update(ctx, recordID, dns.HostChanges{
    Addresses:     bam.Set([]netip.Addr{netip.MustParseAddr("192.0.2.11")}),
    ReverseRecord: bam.Set(false),
}, bam.WithChangeControlComment("Move application address"))
```

[DNS documentation](dns/README.md) describes record variants, complete host-address
preservation, view-scoped absolute zone lookup and subtype-specific updates.
[DHCP documentation](dhcp/README.md) covers reservation lookup, range discovery,
lease observations and common client options. Both domains support exact local
option lookup with a server scope, including an all-server override.
`bam.Collection` and `Client.Do` remain
explicit low-level extension points; their existence does not establish support
for an unmodeled operation.

## Blacklists and deployment

Response policies match DNS names or responses. Client ACLs match IPs, networks,
named ACLs or TSIG keys and preserve evaluation order. They have separate packages;
see the [blacklist workflows](docs/blacklists.md).

Configuration writes do not submit deployment tasks. Configure roles and options,
then explicitly submit and observe a deployment:

```go
deployments := deployment.New(client)
task, err := deployments.ForServer(serverID).Full(ctx, deployment.DNS)
if err != nil {
    return err
}
task, err = deployments.Tasks().Wait(ctx, task.ID, deployment.WaitOptions{})
```

Submission is not completion. `Wait` distinguishes pending, failure, cancellation,
completion and unknown states, and preserves completion warnings. Cancelling the
wait does not cancel the server task. Existing appliance dynamic-update and instant
deployment settings can still cause writes to take effect immediately; the client
does not silently rewrite those settings.

HTTP 202 returns `*bam.AcceptedError` with the pending workflow resource; do not
resubmit the mutation. HTTP failures use `*bam.APIError`; malformed success
responses use `*bam.ProtocolError`. Inspect errors with `errors.Is` / `errors.As`.

## Verification and release

HTTP contract tests exercise public APIs against `httptest`, without BAM
credentials or live mutations:

```sh
go test -race -shuffle=on -timeout 60s ./...
go vet ./...
go build ./...
GOTOOLCHAIN=go1.26.0 GOWORK=off go test -timeout 60s ./...
```

Redirects and cross-origin or scope-changing pagination are rejected. TLS
verification stays enabled. Default request timeout and response limit are
30 seconds and 16 MiB; configure them or inject an `http.Client` as needed.

See the [glossary](GLOSSARY.md), [domain design](docs/adr/0002-typed-domain-capabilities.md)
and [changelog](CHANGELOG.md). Releases use successful same-commit CI artifacts
and verify downloaded assets before publication. Local tests do not establish
live appliance acceptance or publication of a new version.

## Development

Run `make check` to use the published go-standards v1.0.2 lint, formatting, architecture, race and vulnerability gates, plus module verification, build and workflow checks. These development tools are not runtime module dependencies.
