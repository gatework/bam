package dhcp

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/gatework/bam"
)

// Routers configures DHCP option 3, preserving router preference order.
func Routers(addresses ...netip.Addr) (ClientOptionSpec, error) { return addressOption(3, addresses) }

// DNSServers configures DHCP option 6, preserving resolver preference order.
func DNSServers(addresses ...netip.Addr) (ClientOptionSpec, error) {
	return addressOption(6, addresses)
}
func addressOption(code uint8, addresses []netip.Addr) (ClientOptionSpec, error) {
	if len(addresses) == 0 {
		return ClientOptionSpec{}, fmt.Errorf("dhcp: at least one ipv4 address is required")
	}
	values := make([]string, len(addresses))
	for i, address := range addresses {
		if !address.Is4() {
			return ClientOptionSpec{}, fmt.Errorf("dhcp: client option requires ipv4 addresses")
		}
		values[i] = address.String()
	}
	return ClientOptionSpec{Code: code, Value: bam.StringsValue(values...)}, nil
}

// DomainName configures DHCP option 15. BAM applies DNS naming policy validation.
func DomainName(name string) (ClientOptionSpec, error) {
	if strings.TrimSpace(name) == "" {
		return ClientOptionSpec{}, fmt.Errorf("dhcp: domain name is required")
	}
	return ClientOptionSpec{Code: 15, Value: bam.StringValue(name)}, nil
}

// LeaseTime configures the IP Address Lease Time client option (RFC 2132, code 51).
// Fractions of a second are rejected rather than silently truncated.
func LeaseTime(duration time.Duration) (ClientOptionSpec, error) {
	if duration <= 0 || duration%time.Second != 0 || duration/time.Second > 1<<32-1 {
		return ClientOptionSpec{}, fmt.Errorf("dhcp: lease time must be whole seconds between 1 and 4294967295")
	}
	return ClientOptionSpec{Code: 51, Value: bam.IntValue(int64(duration / time.Second))}, nil
}
