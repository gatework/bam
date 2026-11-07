# Domain vocabulary

| Term | Meaning in this library |
| --- | --- |
| Configuration | Top-level administrative container for IP space, DNS views, and managed resources. |
| View | A DNS namespace within a configuration. |
| Zone | A DNS namespace beneath a view or another zone. |
| Resource record | A DNS record belonging to a zone; a host record associates a DNS name with IP addresses. |
| Block | An IPv4 or IPv6 address-space container beneath a configuration or another block. |
| Network | An address-space resource beneath a block, containing individual addresses and DHCP ranges. |
| Address | An individual IPv4 or IPv6 resource with a server-defined allocation state. |
| DHCP range | A resource describing an address range within a network for DHCP use. |
| MAC pool | A grouping of MAC address resources; pool membership and resource lifetime are distinct. |
| Deployment option | DNS or DHCP configuration attached to a scope; options may be inherited from a parent. |
| Deployment role | An assignment that determines a server's responsibility for a managed scope. |
| Deployment | A request/task to apply managed configuration to DNS/DHCP servers. Submission is distinct from completion. |
| Workflow request | A pending change subject to the server's workflow. HTTP 202 does not mean that the requested resource exists yet. |
| Resource ID | A globally unique BAM identifier. Parent scopes select collections, not separate ID namespaces. |
