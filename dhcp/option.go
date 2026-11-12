package dhcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Option is a configured DHCPv4 deployment option. Value preserves the
// definition-dependent JSON type, including boolean, integer, list, or object.
type Option struct {
	bam.Resource
	Configuration bam.Reference  `json:"configuration"`
	ServerScope   *bam.Reference `json:"serverScope"`
	Code          uint8          `json:"code"`
	Definition    *bam.Reference `json:"definition"`
	Value         bam.Value      `json:"value"`
}

// OptionDefinition identifies the appliance's named option and valid resource scopes.
type OptionDefinition struct {
	bam.Resource
	AssignableTypes []string `json:"assignableTypes"`
}

// OptionSpec is one of ClientOptionSpec, ServiceOptionSpec, or RawOptionSpec.
type OptionSpec interface {
	optionSpec()
	json.Marshaler
}

// ClientOptionSpec selects a DHCPv4 option by exactly one name, RFC code, or
// definition ID. Code 6, for example, configures the DNS servers sent to clients.
type ClientOptionSpec struct {
	Name        string         `json:"name,omitempty"`
	Code        uint8          `json:"code,omitempty"`
	Definition  *bam.Reference `json:"definition,omitempty"`
	ServerScope *bam.Reference `json:"serverScope,omitempty"`
	Value       bam.Value      `json:"value"`
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s ClientOptionSpec) MarshalJSON() ([]byte, error) {
	if err := validateOption(s.Name, s.Code, s.Definition, s.ServerScope, s.Value); err != nil {
		return nil, err
	}
	type spec ClientOptionSpec
	return json.Marshal(struct {
		spec
		Type string `json:"type"`
	}{spec: spec(s), Type: "DHCPv4ClientOption"})
}

// ServiceOptionSpec configures a DHCPv4 server behavior by name or definition ID.
type ServiceOptionSpec struct {
	Name        string         `json:"name,omitempty"`
	Definition  *bam.Reference `json:"definition,omitempty"`
	ServerScope *bam.Reference `json:"serverScope,omitempty"`
	Value       bam.Value      `json:"value"`
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s ServiceOptionSpec) MarshalJSON() ([]byte, error) {
	if err := validateOption(s.Name, 0, s.Definition, s.ServerScope, s.Value); err != nil {
		return nil, err
	}
	type spec ServiceOptionSpec
	return json.Marshal(struct {
		spec
		Type string `json:"type"`
	}{spec: spec(s), Type: "DHCPv4ServiceOption"})
}

// RawOptionSpec is the explicit escape hatch for a DHCPv4 configuration statement.
type RawOptionSpec struct {
	Value       string         `json:"value"`
	ServerScope *bam.Reference `json:"serverScope,omitempty"`
}

// MarshalJSON validates and encodes the RESTful v2 JSON representation.
func (s RawOptionSpec) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(s.Value) == "" {
		return nil, fmt.Errorf("dhcp: raw option value is required")
	}
	if s.ServerScope != nil && s.ServerScope.ID <= 0 {
		return nil, fmt.Errorf("dhcp: server scope requires a positive id")
	}
	type spec RawOptionSpec
	return json.Marshal(struct {
		spec
		Type string `json:"type"`
	}{spec: spec(s), Type: "DHCPv4RawOption"})
}
func validateOption(name string, code uint8, definition, server *bam.Reference, value bam.Value) error {
	count := 0
	if name != "" {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("dhcp: option name cannot be blank")
		}
		count++
	}
	if code != 0 {
		count++
	}
	if definition != nil {
		if definition.ID <= 0 {
			return fmt.Errorf("dhcp: option definition requires a positive id")
		}
		count++
	}
	if count != 1 {
		return fmt.Errorf("dhcp: select an option by exactly one name, code, or definition")
	}
	if server != nil && server.ID <= 0 {
		return fmt.Errorf("dhcp: server scope requires a positive id")
	}
	if value.IsZero() || value.IsNull() {
		return fmt.Errorf("dhcp: a non-null option value is required")
	}
	return nil
}

// OptionChanges replaces a local option value or server scope. Delete the local
// option to restore inherited behavior. Clear is not valid for an option value.
type OptionChanges struct {
	Value       bam.Field[bam.Value]
	ServerScope bam.Field[bam.Reference]
}

// Apply validates and applies explicitly set fields to the complete resource document.
func (c OptionChanges) Apply(d bam.Document) error {
	if c.Value.IsNull() {
		return fmt.Errorf("dhcp: option value cannot be cleared; delete the override to restore inheritance")
	}
	if value, ok := c.Value.Value(); ok && (value.IsZero() || value.IsNull()) {
		return fmt.Errorf("dhcp: a non-null option value is required")
	}
	if scope, ok := c.ServerScope.Value(); ok && scope.ID <= 0 {
		return fmt.Errorf("dhcp: server scope requires a positive id")
	}
	if string(d["type"]) == `"DHCPv4RawOption"` && c.Value.IsSet() {
		value, _ := c.Value.Value()
		var text string
		if err := value.Decode(&text); err != nil || strings.TrimSpace(text) == "" {
			return fmt.Errorf("dhcp: raw option requires a nonempty string value")
		}
	}
	return errors.Join(c.Value.Apply(d, "value"), c.ServerScope.Apply(d, "serverScope"))
}

// Options provides typed deployment option operations under a parent resource.
type Options struct {
	resource.Collection[Option, OptionSpec, OptionChanges]
}

// OptionItems provides operations on existing deployment options by global ID.
type OptionItems struct {
	resource.Items[Option, OptionChanges]
}

// Definitions provides read-only DHCP option definition discovery.
type Definitions struct {
	resource.Reader[OptionDefinition]
}

// FindByName requires a unique local option. It reports ambiguous results rather
// than accidentally choosing another option with a different server scope.
func (o Options) FindByName(ctx context.Context, name string) (Option, error) {
	if strings.TrimSpace(name) == "" {
		return Option{}, fmt.Errorf("dhcp: option name is required")
	}
	return resource.One(o.All(ctx, bam.ListOptions{Where: bam.Match("name", name), Limit: 2}))
}

// RestoreInheritance removes one local override without modifying parent options.
func (o Options) RestoreInheritance(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return o.Delete(ctx, id, options...)
}

func (ClientOptionSpec) optionSpec() {}

func (ServiceOptionSpec) optionSpec() {}

func (RawOptionSpec) optionSpec() {}
