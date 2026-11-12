# Migrating to v0.4.0

The module path, package paths and existing handle method signatures are
unchanged. Public handles now own their API through private composition.

- Obtain handles from the relevant `Client` selector. Stop constructing or
  replacing their embedded `Collection`, `Items`, `Reader` or `List` fields.
- Call record operations directly on `Records` instead of selecting its former
  `RecordItems` field. Call task reads and `Wait` directly on `ServerTarget`
  instead of selecting its former `Tasks` field.
- Read DTOs retain `ID`, `Type`, `Name` and their JSON field names. Fixture
  literals now set these fields directly, instead of a nested `Resource`.
- DNS changes retain their named `MetadataChanges` field. Assign common fields
  through it, such as `changes.MetadataChanges.TTL = bam.Set(int64(60))`.
- `RecordQuery.ListOptions` is a named field. Set pagination through
  `query.ListOptions.Limit`, rather than the former promoted `query.Limit`.

New option queries are immutable. Use `dns.NewOptionQuery(name, serverScopeID)`
or `dns.NewOptionQueryByDefinition(definitionID, serverScopeID)`. DHCP adds
`dhcp.NewOptionQueryByCode(code, serverScopeID)` alongside the name and definition
constructors. Check the constructor error before calling `Options.Find`.
Scope zero selects an all-server local override; it never means every scope.

DNS lookup rejects empty labels, including two trailing dots, before HTTP.
DNS/DHCP option writes reject explicit JSON null, including typed nil values.
Delete an override to restore inheritance. Explicit false, zero and empty lists
retain their existing meaning under the selected option definition.

The HTTP wire formats, full-document update behavior and explicit deployment
boundary remain unchanged. These changes do not establish live BAM acceptance.
