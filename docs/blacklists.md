# DNS blacklist workflows

Domain response policies and client access lists answer different questions.
`dns/responsepolicy` configures DNS names/responses; `dns/accesscontrol` determines
which clients may query or recurse. Neither package submits deployment tasks.

## Domain policies

The verified API supports policy and item CRUD, local response-policy zones,
and DNS role assignment. The following prepares those resources using an
authenticated `client` and a caller-owned `ctx`:

```go
policies := responsepolicy.New(client)
policy, err := policies.Policies(configurationID).Create(ctx, responsepolicy.Spec{
    Name: "blocked-domains", Action: responsepolicy.Blocklist,
})
if err != nil {
    return err
}
_, err = policies.Items(policy.ID).Create(ctx, responsepolicy.ItemSpec{
    Name: "unwanted.example",
})
if err != nil {
    return err
}
zone, err := policies.Zones(viewID).Create(ctx, responsepolicy.ZoneSpec{
    Name: "blocked.rpz.example", PolicyID: policy.ID,
})
if err != nil {
    return err
}
_, err = dns.New(client).ZoneRoles(zone.ID).Create(ctx, dns.RoleSpec{
    RoleType: dns.RolePrimary,
    Interfaces: []dns.RoleInterface{{ID: interfaceID, Type: "NetworkInterface"}},
})
if err != nil {
    return err
}
```

**View policy binding remains a schema gap.** BlueCat requires the Response
Policies view option to make selected policies deployable and defines their
evaluation order. The exact REST option value has not been verified, so the
library does not invent a typed binding helper. Configure/verify that binding
on the target BAM before explicit deployment. Creating a policy, RPZ and role
does not establish that the blacklist is active. Allowlist policies must precede
other selected policies when exceptions should win.
[BlueCat option reference](https://docs.bluecatnetworks.com/r/Address-Manager-Administration-Guide/Reference-DNS-deployment-options/9.6.0)

Use policy `Update` for TTL/action/redirect settings and item `Update`/`Delete`
for domain entries. The library does not add wildcards, reorder policies or
silently expand a supplied domain name.

## Client query and recursion access

Create an ordered ACL and bind it through a DNS option. Exclusions precede the
catch-all; sorting these elements would change the result:

```go
acl, err := accesscontrol.New(client).Lists(configurationID).Create(ctx, accesscontrol.Spec{
    Name: "query-clients",
    MatchElements: []accesscontrol.Element{
        accesscontrol.Not(accesscontrol.Network(netip.MustParsePrefix("192.0.2.0/24"))),
        accesscontrol.Any(),
    },
})
if err != nil {
    return err
}
option, err := accesscontrol.AllowQuery(accesscontrol.Named(acl.Name))
if err != nil {
    return err
}
_, err = dns.New(client).ViewOptions(viewID).Create(ctx, option)
if err != nil {
    return err
}
```

`AllowRecursion` constructs the independent recursion-access option. A named
ACL is referenced by its name, not its numeric resource ID. For an existing
local option, use `Update` rather than creating a duplicate. `FindByName` reports
ambiguity if server-specific options share the same name; query the intended
server scope explicitly. Deleting a local option with `RestoreInheritance`
restores the inherited setting; it does not necessarily allow all clients.

## Explicit deployment and observation

Once required bindings and roles are configured, submit DNS deployment and
observe its task separately:

```go
deployments := deployment.New(client)
task, err := deployments.ForServer(serverID).Full(ctx, deployment.DNS)
if err != nil {
    return err
}
task, err = deployments.Tasks().Wait(ctx, task.ID, deployment.WaitOptions{})
if err != nil {
    return err
}
// Inspect task.State and task.Message, including completion warnings.
```

A completed task establishes the reported deployment state. Verify actual DNS
answers and client query/recursion behavior separately during appliance
acceptance. The local tests validate wire contracts and independent submission;
they do not claim an observed DNS filtering result.
