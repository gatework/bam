package bam_test

import (
	"reflect"
	"testing"

	"github.com/gatework/bam/configuration"
	"github.com/gatework/bam/deployment"
	"github.com/gatework/bam/dhcp"
	"github.com/gatework/bam/dns"
	"github.com/gatework/bam/dns/accesscontrol"
	"github.com/gatework/bam/dns/responsepolicy"
	"github.com/gatework/bam/ipam"
)

// Keep internal handles private and public DTO fields explicit (CMP-1).
func TestPublicAPI_Encapsulation(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		typ    reflect.Type
		handle bool
	}{
		{reflect.TypeFor[configuration.Configurations](), true},
		{reflect.TypeFor[configuration.Configuration](), false},
		{reflect.TypeFor[deployment.Tasks](), true},
		{reflect.TypeFor[deployment.ServerTarget](), true},
		{reflect.TypeFor[deployment.Interface](), false},
		{reflect.TypeFor[deployment.Interfaces](), true},
		{reflect.TypeFor[deployment.Server](), false},
		{reflect.TypeFor[deployment.Servers](), true},
		{reflect.TypeFor[dhcp.Option](), false},
		{reflect.TypeFor[dhcp.OptionDefinition](), false},
		{reflect.TypeFor[dhcp.Options](), true},
		{reflect.TypeFor[dhcp.OptionItems](), true},
		{reflect.TypeFor[dhcp.Definitions](), true},
		{reflect.TypeFor[dhcp.MACPool](), false},
		{reflect.TypeFor[dhcp.MACPools](), true},
		{reflect.TypeFor[dhcp.MACPoolItems](), true},
		{reflect.TypeFor[dhcp.MACMembers](), true},
		{reflect.TypeFor[dhcp.Range](), false},
		{reflect.TypeFor[dhcp.Ranges](), true},
		{reflect.TypeFor[dhcp.RangeItems](), true},
		{reflect.TypeFor[dhcp.Reservations](), true},
		{reflect.TypeFor[dhcp.Role](), false},
		{reflect.TypeFor[dhcp.Roles](), true},
		{reflect.TypeFor[dhcp.RoleItems](), true},
		{reflect.TypeFor[dhcp.RoleInterfaces](), true},
		{reflect.TypeFor[dns.Option](), false},
		{reflect.TypeFor[dns.OptionDefinition](), false},
		{reflect.TypeFor[dns.Options](), true},
		{reflect.TypeFor[dns.OptionItems](), true},
		{reflect.TypeFor[dns.OptionDefinitions](), true},
		{reflect.TypeFor[dns.Record](), false},
		{reflect.TypeFor[dns.Address](), false},
		{reflect.TypeFor[dns.HostChanges](), false},
		{reflect.TypeFor[dns.TXTChanges](), false},
		{reflect.TypeFor[dns.HINFOChanges](), false},
		{reflect.TypeFor[dns.NAPTRChanges](), false},
		{reflect.TypeFor[dns.GenericChanges](), false},
		{reflect.TypeFor[dns.AliasChanges](), false},
		{reflect.TypeFor[dns.MXChanges](), false},
		{reflect.TypeFor[dns.SRVChanges](), false},
		{reflect.TypeFor[dns.RecordAddresses](), true},
		{reflect.TypeFor[dns.RecordDependencies](), true},
		{reflect.TypeFor[dns.AddressRecords](), true},
		{reflect.TypeFor[dns.RecordQuery](), false},
		{reflect.TypeFor[dns.RecordItems](), true},
		{reflect.TypeFor[dns.Records](), true},
		{reflect.TypeFor[dns.Role](), false},
		{reflect.TypeFor[dns.Roles](), true},
		{reflect.TypeFor[dns.RoleItems](), true},
		{reflect.TypeFor[dns.RoleInterfaces](), true},
		{reflect.TypeFor[dns.View](), false},
		{reflect.TypeFor[dns.Views](), true},
		{reflect.TypeFor[dns.ViewItems](), true},
		{reflect.TypeFor[dns.Zone](), false},
		{reflect.TypeFor[dns.Zones](), true},
		{reflect.TypeFor[dns.ZoneItems](), true},
		{reflect.TypeFor[accesscontrol.ACL](), false},
		{reflect.TypeFor[accesscontrol.Lists](), true},
		{reflect.TypeFor[accesscontrol.ListItems](), true},
		{reflect.TypeFor[responsepolicy.Policies](), true},
		{reflect.TypeFor[responsepolicy.PolicyItems](), true},
		{reflect.TypeFor[responsepolicy.Items](), true},
		{reflect.TypeFor[responsepolicy.GlobalItems](), true},
		{reflect.TypeFor[responsepolicy.Policy](), false},
		{reflect.TypeFor[responsepolicy.Item](), false},
		{reflect.TypeFor[responsepolicy.Zone](), false},
		{reflect.TypeFor[responsepolicy.Zones](), true},
		{reflect.TypeFor[responsepolicy.ZoneItems](), true},
		{reflect.TypeFor[ipam.Addresses](), true},
		{reflect.TypeFor[ipam.AddressItems](), true},
		{reflect.TypeFor[ipam.Blocks](), true},
		{reflect.TypeFor[ipam.Networks](), true},
		{reflect.TypeFor[ipam.BlockItems](), true},
		{reflect.TypeFor[ipam.NetworkItems](), true},
		{reflect.TypeFor[ipam.MACAddress](), false},
		{reflect.TypeFor[ipam.MACAddresses](), true},
		{reflect.TypeFor[ipam.MACItems](), true},
		{reflect.TypeFor[ipam.AddressLinks](), true},
		{reflect.TypeFor[ipam.Block](), false},
		{reflect.TypeFor[ipam.Network](), false},
		{reflect.TypeFor[ipam.Address](), false},
		{reflect.TypeFor[dns.OptionQuery](), true},
		{reflect.TypeFor[dhcp.OptionQuery](), true},
	} {
		t.Run(tt.typ.String(), func(t *testing.T) {
			t.Parallel()
			for field := range tt.typ.Fields() {
				if field.Anonymous && field.IsExported() && field.Type.Kind() != reflect.Interface {
					t.Errorf("concrete embedding exposes %s.%s", tt.typ, field.Name)
				}
				if tt.handle && field.IsExported() {
					t.Errorf("mutable implementation field exposed: %s.%s", tt.typ, field.Name)
				}
			}
		})
	}
}
