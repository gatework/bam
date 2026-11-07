# bam

A Go client for BlueCat Address Manager **RESTful v2**, using the BAM 9.6
protocol documentation as its baseline. Requires Go 1.23 or later and uses only
the standard library. This is a breaking replacement of the former SOAP client;
there is no SOAP transport, WSDL-generated interface, or v1 token compatibility.

The project recommends Go 1.27.3 through `go.mod`'s toolchain directive. Go 1.23
is tested for language compatibility; deploy with a currently patched Go release
for standard-library security fixes.

Install the published module:

```sh
go get github.com/gatework/bam@v0.1.0
```

Business modules share one authenticated client and one collection implementation:

| Package | Responsibility |
| --- | --- |
| `bam` | JSON transport, sessions, errors, HAL pagination, full resource documents |
| `configuration` | Configuration resources |
| `dns` | Views, zones, resource records, host record creation |
| `ipam` | IPv4/IPv6 blocks, networks, exact address assignment |
| `dhcp` | Ranges, MAC resources, scoped deployment options and roles |
| `deployment` | Server, zone, network and selective deployment requests; task retrieval |

Create a client with `bam.New("https://bam.example")`. Authenticate using
`client.Login(ctx, username, password)`, or supply an existing v2 token through
`bam.WithBasicAuth(username, apiToken)`. `bam.WithBearerToken(token)` supports
externally managed OAuth credentials. Login stores the returned Basic
credentials per client; the returned `Session` contains metadata only.

All network operations take a caller-owned context. `client.Logout(ctx)` sends
the v2 session state transition; `client.CloseIdleConnections()` closes idle HTTP
connections independently. Give logout its own bounded context if the operation's
context has already expired. Failed writes are never automatically retried.

For example, create a network under a block without writing JSON resource types
or HTTP request plumbing:

```go
import (
    "context"
    "net/netip"

    "github.com/gatework/bam"
    "github.com/gatework/bam/ipam"
)

func addNetwork(ctx context.Context, client *bam.Client, blockID int64) (ipam.Network, error) {
    return ipam.New(client).Networks(blockID).Create(ctx, ipam.NetworkSpec{
        Name:  "application-network",
        Range: netip.MustParsePrefix("192.0.2.0/24"),
    })
}
```

`configuration.Spec`, `dns.ViewSpec`, `dns.ZoneSpec`, `dns.HostRecordSpec`,
`ipam.BlockSpec`, `ipam.NetworkSpec`, and `ipam.AddressSpec` encode documented
create fields and infer resource types. IP values use `net/netip`; malformed or
scoped addresses and prefixes with host bits fail before sending a request.

Each domain selects a `Collection[T]`. `Page` fetches one page; `All` is a lazy
Go iterator, following HAL `next` links until exhausted. Breaking iteration
stops further requests. `count` is the current page's size, never a total.

```go
for network, err := range ipam.New(client).Networks(blockID).All(ctx, bam.ListOptions{
    Limit:   200,
    OrderBy: []string{"id"},
}) {
    if err != nil {
        return err
    }
    // Use network.ID, network.Name, and network.Range.
}
```

Parent scopes apply to `Page`, `All` and `Create`. `Get`, `Document`, `Replace`
and `Delete` use globally unique resource IDs. Global DNS/IPAM selectors support
cross-scope queries. A collection handle does not prove that every HTTP verb is
supported by that endpoint; the device's OpenAPI schema defines its capabilities.
MAC pool membership is deliberately separate from deleting a MAC resource.

`PUT` replaces a complete resource. Read projections intentionally cannot be
passed to `Replace`; obtain a lossless `Document`, edit it, and send it back:

```go
records := dns.New(client).GlobalRecords()
document, err := records.Document(ctx, recordID)
if err != nil {
    return err
}
if err := document.Set("name", "new-name"); err != nil {
    return err
}
record, err := records.Replace(ctx, recordID, document)
```

`Document` uses `json.RawMessage` to retain unknown fields, large integers, null
and false values. Replacement checks the resource ID and type, removes HAL links
from the outgoing copy, and does not mutate the input. Do not construct a partial
document for PUT or use a projected collection item as a complete resource.

Use `errors.As` to inspect `*bam.APIError` (HTTP status and BAM error code),
`*bam.ProtocolError` (malformed success response), or `*bam.AcceptedError`
(HTTP 202). An accepted workflow request is **pending**, not a completed DNS,
IPAM or deployment change. `AcceptedError.Resource` retains the pending resource;
do not resubmit the mutation. Context and transport errors remain discoverable
through `errors.Is` / `errors.As`.

The default timeout is 30 seconds and the response limit is 16 MiB, configurable
with `WithTimeout` and `WithResponseLimit`. `WithHTTPClient` injects a transport
without mutating the caller's client. Redirects, cross-origin pagination and
pagination scope changes are rejected. TLS verification stays enabled; configure
trusted certificates on an injected `http.Transport` for private certificate
authorities.

Typed models cover the fields verified in the public 9.6 guide. DHCP and
deployment use complete `Document` values rather than guessed request schemas.
Other DNS types, allocation modes, and version-specific fields can use
`Collection.Create` with a document, or `Client.Do` for additional JSON endpoints.
In particular, the library does not infer a next-address allocation payload from
a legacy endpoint mapping. Consult the target BAM's `/api/openapi.json` or
`/api/docs`; unknown request fields can be silently ignored by BAM.

Local verification uses `httptest`, with no BAM credentials or live mutations:

```sh
go test -race -timeout 60s ./...
go vet ./...
GOTOOLCHAIN=go1.23.0 GOWORK=off go test -timeout 60s ./...
```

See the [domain vocabulary](GLOSSARY.md) and
[RESTful v2 design decision](docs/adr/0001-restful-v2.md) for scope and source
references. These tests establish client protocol behavior; they do not establish
acceptance by a particular BAM appliance or complete coverage of its OpenAPI.

Releases are tagged from successful main-branch CI commits. The release workflow
reuses that CI run's immutable source artifact, verifies the revision and checksums,
and compares downloaded draft assets before publication. See [CHANGELOG.md](CHANGELOG.md)
for version history.
