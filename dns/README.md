# DNS records

Views, ordinary zones and records have typed create and update operations.
Creation selects a parent; existing resources use their global ID. Global
selectors expose reads, updates and deletion, without a create operation.
Zones, Subzones and GlobalZones always select ordinary Zone resources. They
exclude response-policy zones and BAM's system-managed external-host zones.

```go
records := dns.New(client).Records(zoneID)
host, err := records.Create(ctx, dns.HostRecordSpec{
    Name: "app",
    Addresses: []netip.Addr{netip.MustParseAddr("192.0.2.8")},
    ReverseRecord: true,
})
if err != nil {
    return err
}

_, err = records.Update(ctx, host.ID, dns.HostChanges{
    Addresses: bam.Set([]netip.Addr{netip.MustParseAddr("192.0.2.9")}),
    ReverseRecord: bam.Set(false),
}, dns.WithServerUpdates(false))
return err
```

Each subtype has its own specification and changes: Host, Alias, MX, TXT, SRV,
HINFO, NAPTR and Generic. External hosts use ExternalHostRecordSpec in the
view's ExternalHostsZone and MetadataChanges for common updates. Generic
records retain BAM's recordType and presentation-format rdata; this covers PTR,
CAA, NS and other generic types without inventing a separate BAM resource.
External alias/MX/SRV targets use ExternalTarget, which encodes the external
host type in linkedRecord.

ExternalHostsZone looks up the system-managed zone in a view without requiring
a discriminator string from the caller. It returns ErrNotFound or ErrAmbiguous
unless exactly one zone matches:

```go
clientDNS := dns.New(client)
zone, err := clientDNS.ExternalHostsZone(ctx, viewID)
if err != nil {
    return err
}
_, err = clientDNS.Records(zone.ID).Create(ctx, dns.ExternalHostRecordSpec{
    Name: "origin.example.net",
})
return err
```

Record is the common read envelope. Its Details field contains a pointer to
HostDetails, AliasDetails, MXDetails, and the corresponding subtype details.
HostDetails.Addresses comes from HAL embed(addresses); Client.Addresses reads
the complete paginated association collection. UnknownDetails retains the
original fields of an unmodeled subtype.

RecordItems.Update reads a complete document and changes selected fields.
Host address associations are fetched through every page and retained unless
HostChanges explicitly replaces them. Unmodeled fields and large JSON integers
survive the update. This is a read/edit/PUT sequence, not a transaction with
other writers. An unset change retains its value; Set(false) and Set(0) are
explicit updates. The verified TTL schema accepts 0 through 2147483647 seconds.
Clearing TTL to null is rejected until its inheritance-reset contract is known.

RecordQuery supports name, absolute name, discriminator, generic RR type,
configuration and linked-record selection. Search is lazy; Find rejects missing
or ambiguous results. RecordsByIP first resolves an address within a required
configuration and follows its record links, avoiding unsupported nested filters.

RecordsForAddress provides Link, Get and Unlink. Unlink uses the nested
address/record relationship URI; it does not delete the DNS record.
DependentRecords also preserves its parent when reading an individual member.

The library does not submit deployment tasks from record mutations. Existing
server-side dynamic update settings can still deploy records automatically;
ZoneSpec and ZoneChanges expose that setting, and WithServerUpdates supplies
the documented update/delete request header.

## Contract evidence

Field names, subtype discriminators, required fields and relationship operations
come from the [BAM 9.5 exported OpenAPI](https://github.com/umich-vci/gobam2/blob/e894d31d00a85fb41f1ab648882600e5476ca091/api/openapi.yaml).
This is a preserved vendor schema in another repository, not a 9.6 appliance
export. The [official 9.6 changes](https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/What-s-New-in-v9.6.0/9.6.0)
confirm host-address PUT updates and additional record filters. Operation
headers follow the [9.6 header reference](https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/HTTP-headers/9.6.0).

HTTPSRecord, SVCBRecord and URIRecord are documented additions in 9.6, but
their exact fields are absent from the available schema. Their reads are
preserved as UnknownDetails and they can be deleted through the normal record
endpoint. Typed creation and updating remain a schema gap. The new HostRecord
dependentRecords create payload is also not guessed; aliases can be created
individually using the host reference.

HTTP contract tests cover subtype bodies, read details, a complete CRUD flow,
lossless updates, paginated host addresses, relationship deletion and typed
searches. They do not establish acceptance by a BAM 9.6 appliance.
