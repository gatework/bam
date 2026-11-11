package ipam

import (
	"net/netip"

	"github.com/gatework/bam"
)

// Block is an IPv4 or IPv6 address-space container.
type Block struct {
	bam.Resource
	Configuration               bam.Reference `json:"configuration"`
	Range                       string        `json:"range"`
	PingBeforeAssignmentEnabled *bool         `json:"pingBeforeAssignmentEnabled"`
	InheritedFields             []string      `json:"_inheritedFields"`
}

// Network is an IPv4 or IPv6 network. InheritedFields identifies settings
// inherited from a parent rather than explicitly configured on this network.
type Network struct {
	bam.Resource
	Configuration               bam.Reference    `json:"configuration"`
	Range                       string           `json:"range"`
	Gateway                     *netip.Addr      `json:"gateway"`
	PingBeforeAssignmentEnabled *bool            `json:"pingBeforeAssignmentEnabled"`
	DynamicUpdateEnabled        *bool            `json:"dynamicUpdateEnabled"`
	DefaultView                 *bam.Reference   `json:"defaultView"`
	InheritedFields             []string         `json:"_inheritedFields"`
	Usage                       map[string]int64 `json:"usage"`
}

// Address is an allocation and its current MAC and lease information. Lease
// timestamps retain BAM's wire representation, which varies between versions.
type Address struct {
	bam.Resource
	Configuration           bam.Reference `json:"configuration"`
	Address                 netip.Addr    `json:"address"`
	State                   State         `json:"state"`
	MACAddress              *MACAddress   `json:"macAddress"`
	LeaseDateTime           string        `json:"leaseDateTime"`
	LeaseExpirationDateTime string        `json:"leaseExpirationDateTime"`
	ClientIdentifier        *ClientID     `json:"clientIdentifier"`
}

// ClientID is a DHCPv4 client identifier associated with an address.
type ClientID struct {
	ID   int64  `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	UID  string `json:"uid"`
}

// State is an open set so new server-defined read states remain observable.
type State string

// Address allocation states returned by BAM.
const (
	StateUnassigned     State = "UNASSIGNED"
	StateStatic         State = "STATIC"
	StateReserved       State = "RESERVED"
	StateGateway        State = "GATEWAY"
	StateDHCPReserved   State = "DHCP_RESERVED"
	StateDHCPAllocated  State = "DHCP_ALLOCATED"
	StateDHCPLeased     State = "DHCP_LEASED"
	StateDHCPAbandoned  State = "DHCP_ABANDONED"
	StateDHCPFree       State = "DHCP_FREE"
	StateDHCPExcluded   State = "DHCP_EXCLUDED"
	StateDHCPUnassigned State = "DHCP_UNASSIGNED"
	StateRestricted     State = "RESTRICTED"
)
