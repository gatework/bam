package dhcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Option is a configured DHCPv4 deployment option. Value preserves the
// definition-dependent JSON type, including boolean, integer, list, or object.
type Option struct {
	ID            int64          `json:"id"`
	Type          string         `json:"type"`
	Name          string         `json:"name"`
	Configuration bam.Reference  `json:"configuration"`
	ServerScope   *bam.Reference `json:"serverScope"`
	Code          uint8          `json:"code"`
	Definition    *bam.Reference `json:"definition"`
	Value         bam.Value      `json:"value"`
}

// OptionDefinition identifies the appliance's named option and valid resource scopes.
type OptionDefinition struct {
	ID              int64    `json:"id"`
	Type            string   `json:"type"`
	Name            string   `json:"name"`
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
	resources resource.Collection[Option, OptionSpec, OptionChanges]
}

// OptionItems provides operations on existing deployment options by global ID.
type OptionItems struct {
	resources resource.Items[Option, OptionChanges]
}

// Definitions provides read-only DHCP option definition discovery.
type Definitions struct {
	resources resource.Reader[OptionDefinition]
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

// Page fetches one page within the selected collection scope.
func (o Options) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Option], error) {
	return o.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (o Options) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Option, error] {
	return o.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (o Options) Get(ctx context.Context, id int64) (Option, error) {
	return o.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (o Options) Document(ctx context.Context, id int64) (bam.Document, error) {
	return o.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (o Options) Update(ctx context.Context, id int64, changes OptionChanges, options ...bam.WriteOption) (Option, error) {
	return o.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (o Options) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return o.resources.Delete(ctx, id, options...)
}

// Create submits a typed specification within the selected parent scope.
func (o Options) Create(ctx context.Context, spec OptionSpec, options ...bam.WriteOption) (Option, error) {
	return o.resources.Create(ctx, spec, options...)
}

// Page fetches one page within the selected collection scope.
func (o OptionItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Option], error) {
	return o.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (o OptionItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Option, error] {
	return o.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (o OptionItems) Get(ctx context.Context, id int64) (Option, error) {
	return o.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (o OptionItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return o.resources.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (o OptionItems) Update(ctx context.Context, id int64, changes OptionChanges, options ...bam.WriteOption) (Option, error) {
	return o.resources.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (o OptionItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return o.resources.Delete(ctx, id, options...)
}

// Page fetches one page within the selected collection scope.
func (d Definitions) Page(ctx context.Context, options bam.ListOptions) (bam.Page[OptionDefinition], error) {
	return d.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (d Definitions) All(ctx context.Context, options bam.ListOptions) iter.Seq2[OptionDefinition, error] {
	return d.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (d Definitions) Get(ctx context.Context, id int64) (OptionDefinition, error) {
	return d.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (d Definitions) Document(ctx context.Context, id int64) (bam.Document, error) {
	return d.resources.Document(ctx, id)
}
