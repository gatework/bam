package dns

import (
	"fmt"
	"strings"
)

// zoneName is a normalized, nonempty zone lookup key. Its zero value is unset.
// BAM remains responsible for DNS naming policy beyond this lookup syntax.
type zoneName struct{ value string }

func parseZoneName(name string) (zoneName, error) {
	name = strings.TrimSuffix(name, ".")
	if name == "" || strings.TrimSpace(name) != name || strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") || strings.Contains(name, "..") {
		return zoneName{}, fmt.Errorf("dns: nonempty absolute zone name without empty labels is required")
	}
	return zoneName{value: strings.ToLower(name)}, nil
}

func (n zoneName) isBelow(parent zoneName) bool {
	return strings.HasSuffix(n.value, "."+parent.value)
}
