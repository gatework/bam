package dhcp

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Role determines a server's responsibility for DHCP in a block or network.
type Role struct {
	bam.Resource
	Configuration bam.Reference `json:"configuration"`
	RoleType      RoleType      `json:"roleType"`
}

type RoleType string

const (
	RolePrimary RoleType = "PRIMARY"
	RoleNone    RoleType = "NONE"
)

// RoleInterface identifies a network or published server interface. Failover
// roles distinguish PRIMARY and SECONDARY through DeploymentRoleInterfaceType.
type RoleInterface struct {
	ID                          int64  `json:"id"`
	Type                        string `json:"type"`
	DeploymentRoleInterfaceType string `json:"deploymentRoleInterfaceType,omitempty"`
}

type RoleSpec struct {
	RoleType   RoleType        `json:"roleType"`
	Interfaces []RoleInterface `json:"interfaces"`
}

func (s RoleSpec) MarshalJSON() ([]byte, error) {
	if err := validateRoleType(s.RoleType); err != nil {
		return nil, err
	}
	if err := validateRoleInterfaces(s.Interfaces); err != nil {
		return nil, err
	}
	type spec RoleSpec
	return json.Marshal(struct {
		Type string `json:"type"`
		spec
	}{"DHCPDeploymentRole", spec(s)})
}

type RoleChanges struct {
	RoleType   bam.Field[RoleType]
	Interfaces bam.Field[[]RoleInterface]
}

func (c RoleChanges) Apply(d bam.Document) error {
	if c.RoleType.IsNull() || c.Interfaces.IsNull() {
		return fmt.Errorf("dhcp: role type and interfaces cannot be null")
	}
	if role, ok := c.RoleType.Value(); ok {
		if err := validateRoleType(role); err != nil {
			return err
		}
	}
	if interfaces, ok := c.Interfaces.Value(); ok {
		if err := validateRoleInterfaces(interfaces); err != nil {
			return err
		}
	}
	return errors.Join(c.RoleType.Apply(d, "roleType"), c.Interfaces.Apply(d, "interfaces"))
}
func validateRoleType(role RoleType) error {
	if role != RolePrimary && role != RoleNone {
		return fmt.Errorf("dhcp: role must be PRIMARY or NONE")
	}
	return nil
}
func validateRoleInterfaces(interfaces []RoleInterface) error {
	if len(interfaces) == 0 {
		return fmt.Errorf("dhcp: at least one deployment interface is required")
	}
	seen := make(map[int64]struct{}, len(interfaces))
	for _, item := range interfaces {
		if item.ID <= 0 {
			return fmt.Errorf("dhcp: interface id must be positive")
		}
		if item.Type != "NetworkInterface" && item.Type != "PublishedInterface" {
			return fmt.Errorf("dhcp: interface type must be NetworkInterface or PublishedInterface")
		}
		switch item.DeploymentRoleInterfaceType {
		case "", "PRIMARY", "SECONDARY":
		default:
			return fmt.Errorf("dhcp: interface role must be PRIMARY or SECONDARY")
		}
		if _, exists := seen[item.ID]; exists {
			return fmt.Errorf("dhcp: repeated deployment interface")
		}
		seen[item.ID] = struct{}{}
	}
	return nil
}

type Roles struct {
	resource.Collection[Role, RoleSpec, RoleChanges]
}
type RoleItems struct {
	resource.Items[Role, RoleChanges]
}
type RoleInterfaces struct{ resource.List[RoleInterface] }
