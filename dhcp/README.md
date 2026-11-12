# DHCPv4 management

The DHCP client targets BAM 9.6 and shares the root client's authentication,
timeouts, pagination and write behavior. Configuration changes and deployment
are separate operations. No write is automatically retried.

## Reservations and pools

Reservations bind an IPv4 address to a MAC in `DHCP_RESERVED` state. They are
distinct from IPAM `STATIC` assignments and `RESERVED` administrative holds.

```go
clientDHCP := dhcp.New(client)
reservations := clientDHCP.Reservations(networkID)
binding, err := reservations.FindByIP(ctx, netip.MustParseAddr("192.0.2.10"))
if err != nil {
    return err
}
mac, err := ipam.ParseMAC("00:11:22:33:44:55")
if err != nil {
    return err
}
_, err = reservations.Update(ctx, binding.ID, dhcp.ReservationChanges{
    MAC: bam.Set(ipam.MACReference{Address: mac}),
})
return err
```

`FindByMAC` searches reservations within the selected network. Multiple matching
bindings return `bam.ErrAmbiguous`; a missing binding returns `bam.ErrNotFound`.
It reads every required page and compares normalized MAC addresses locally.
For a configuration-wide address search, use `ipam.AddressesByMAC` and inspect
each returned address state.

`Create` requires an exact IPv4 address and a MAC. `AllocateNext` lets BAM select
and reserve an address in one POST; do not query a free address and assume it
has been claimed. `Delete` removes the reservation, not the MAC resource.

`MACPools(configurationID).Find(ctx, name)` locates an ordinary MAC pool.
`MACMembers(poolID).Link` and `Unlink` change membership without creating or
deleting the MAC resource. Deny pools remain a separate, unmodeled contract.

## Dynamic ranges and observations

`Ranges(networkID).FindByRange` selects an exact inclusive `ipam.AddressRange`.
`Containing` lists ranges containing an IPv4 address and retains usage and
exclusion information. Containment does not mean the address is available.
`RangeSpec` and `RangeChanges` configure interval size, exclusions and watermarks;
resizing with orphaned addresses still follows the appliance's state rules.

`LeasedAddresses(ctx, networkID, options)` lazily lists addresses whose BAM state
is `DHCP_LEASED`. Read `MACAddress`, `ClientIdentifier`, `LeaseDateTime` and
`LeaseExpirationDateTime` on each result. This observes BAM's recorded state,
not live DHCP server state or historical lease events. It does not renew or
release leases. The standalone `/leases` schema remains incomplete and is not
represented by this helper.

## Local options

Constructors cover routers (3), DNS servers (6), domain name (15), NTP servers
(42), lease time (51), TFTP server name (66), boot filename (67), domain search
(119), and TFTP addresses (150). Address and domain lists preserve preference
order. `DomainSearch` encodes a JSON string array; BAM performs DNS policy
validation. Boot filename configuration does not upload a file.

```go
options := clientDHCP.NetworkOptions(networkID)
query, err := dhcp.NewOptionQueryByCode(6, serverID)
if err != nil {
    return err
}
option, err := options.Find(ctx, query)
if err != nil {
    return err
}
spec, err := dhcp.DNSServers(netip.MustParseAddr("192.0.2.53"))
if err != nil {
    return err
}
_, err = options.Update(ctx, option.ID, dhcp.OptionChanges{Value: bam.Set(spec.Value)})
return err
```

Build immutable queries with `NewOptionQuery`, `NewOptionQueryByCode`, or
`NewOptionQueryByDefinition`; each selects exactly one name, code, or definition.
Server scope zero selects an all-server override; a positive ID selects a
server or server group. An unset query fails before HTTP. Code lookup never selects a service option with the
same number. Lookup reads complete paginated resources at the chosen parent,
matches locally, and returns ambiguity rather than selecting the first result.
It does not calculate inherited values or silently create a missing option.

Delete a local override with `RestoreInheritance` to restore inheritance.
Unset changes preserve existing values. Explicit null option values are rejected;
false, zero and empty lists remain distinct valid JSON values, subject to each
option definition. Full-document updates preserve unknown fields but are not
transactions with concurrent writers.

## Evidence

The [capability matrix](../docs/rest-v2-coverage.md) pins endpoint and resource
schemas. The [BlueCat DHCPv4 option reference](https://docs.bluecatnetworks.com/r/Address-Manager-Administration-Guide/Reference-DHCPv4-Client-options/9.6.0)
documents common option codes and value meanings. Synthetic HTTP tests exercise
scope, payloads, pagination, ambiguity, invalid inputs and lossless changes.
They do not establish live BAM appliance acceptance or server deployment.
