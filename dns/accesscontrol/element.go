package accesscontrol

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
)

// Element is one ordered ACL match expression: an IP address, network, ACL
// name, predefined keyword or TSIG key. Constructors retain input errors until
// the request is encoded, so invalid values never reach the server.
type Element struct {
	value string
	err   error
}

func Address(address netip.Addr) Element {
	if !address.IsValid() || address.Zone() != "" {
		return Element{err: fmt.Errorf("accesscontrol: valid unscoped address required")}
	}
	return Element{value: address.String()}
}
func Network(prefix netip.Prefix) Element {
	if !prefix.IsValid() || prefix != prefix.Masked() {
		return Element{err: fmt.Errorf("accesscontrol: canonical network prefix required")}
	}
	return Element{value: prefix.String()}
}

// Named refers to an existing ACL by its name, not its numeric resource ID.
func Named(name string) Element {
	if !validName(name) {
		return Element{err: fmt.Errorf("accesscontrol: invalid acl reference name")}
	}
	return Element{value: name}
}

func Any() Element       { return Element{value: "any"} }
func None() Element      { return Element{value: "none"} }
func Localhost() Element { return Element{value: "localhost"} }
func Localnets() Element { return Element{value: "localnets"} }

func Key(name string) Element {
	if !validName(name) {
		return Element{err: fmt.Errorf("accesscontrol: invalid key name")}
	}
	return Element{value: "key " + name}
}

// Not negates one element. Order is significant: put exclusions before a
// catch-all such as Any. No sorting or deduplication is performed.
func Not(element Element) Element {
	if element.err != nil {
		return element
	}
	if element.value == "" {
		return Element{err: fmt.Errorf("accesscontrol: empty match element")}
	}
	if strings.HasPrefix(element.value, "!") {
		element.value = strings.TrimPrefix(element.value, "!")
	} else {
		element.value = "!" + element.value
	}
	return element
}

func validName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func (e Element) String() string { return e.value }

func (e Element) MarshalJSON() ([]byte, error) {
	if e.err != nil {
		return nil, e.err
	}
	if e.value == "" {
		return nil, fmt.Errorf("accesscontrol: empty match element")
	}
	return json.Marshal(e.value)
}

// UnmarshalJSON preserves server-provided expressions, including expressions
// added by newer BAM versions. New match expressions use the typed constructors.
func (e *Element) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	e.value, e.err = value, nil
	return nil
}

func encodeElements(elements []Element) ([]string, error) {
	if len(elements) == 0 {
		return nil, fmt.Errorf("accesscontrol: at least one ordered match element is required")
	}
	values := make([]string, len(elements))
	for i, element := range elements {
		if element.err != nil {
			return nil, element.err
		}
		if element.value == "" {
			return nil, fmt.Errorf("accesscontrol: empty match element")
		}
		values[i] = element.value
	}
	return values, nil
}
