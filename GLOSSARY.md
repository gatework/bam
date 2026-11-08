# Domain vocabulary

| Term | Meaning in this library |
| --- | --- |
| Configuration | Administrative container for IP space, DNS views and managed resources. Required scope for unambiguous cross-network IP/MAC lookups. |
| View / Zone | A DNS namespace within a configuration / a namespace beneath a view or another zone. Ordinary zones and response-policy zones are distinct resource types. |
| Resource record | A DNS record in a zone. A host record associates a name with IP addresses; its reverse-record setting controls PTR generation. |
| Block / Network | An IPv4 or IPv6 address-space container / a resource beneath a block containing individual addresses and DHCP ranges. |
| Static allocation | An address in `STATIC` state, optionally associated with a MAC. It does not by itself create a DHCP reservation. |
| Administrative reservation | An IPv4 address held as `RESERVED` for administrative use, without creating a DHCP binding. |
| DHCP reservation | An address bound to a client as `DHCP_RESERVED`. The DHCPv4 interface requires a MAC reference. |
| Available address | An unassigned IPAM candidate. A query does not claim it, and does not mean it is available in a DHCP dynamic pool. |
| Next-address allocation | A single POST in which BAM selects and allocates an IPv4 address. It is not a client-side query-then-create transaction. |
| DHCP range | An IPv4 dynamic allocation interval, with optional exclusions and usage thresholds. The BAM discriminator is `IPv4DHCPRange`. |
| Lease data | Observed address lease state and timestamps. Reading them does not renew, release or otherwise manage a lease. |
| MAC resource / MAC pool membership | A hardware address stored in a configuration / its association with a pool. Removing membership does not delete the MAC resource or its IP allocations. |
| Response policy | An allowlist, blocklist, blackhole or redirect policy applied to DNS questions/responses through policy items. |
| Local RPZ | A `LOCAL` response-policy zone linking a response policy to a view. DNS deployment roles assign server responsibility. |
| Client ACL | An ordered list of address, CIDR, ACL-name or TSIG-key matches used by DNS query/recursion access options. It is distinct from a domain response policy. |
| Deployment option | A DNS/DHCP value configured at a scope, optionally limited to a server. Deleting the local override restores inheritance; empty and inherited are different states. |
| Deployment role | Server-interface responsibility for a managed scope. Defining a role and deploying its configuration are separate operations. |
| Deployment task | A submitted job applying configuration or validating it. Queued/running does not mean completed; waiting can end without cancelling the task. |
| Workflow request | A pending change returned with HTTP 202. It is not a completed resource mutation or an instruction to retry. |
| Resource ID | A global BAM identifier. Parent handles scope collections, while relationship handles retain nested item paths. |
| Change field | An explicit unchanged, set or clear operation. `Set(false)` and `Set(0)` retain their meaning; clear is supported only by nullable contracts. |
