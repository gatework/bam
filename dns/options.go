package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// Option is a DNS deployment option. Delete the locally configured option to
// restore inheritance; assigning an empty value is a different operation.
type Option struct {
	bam.Resource
	Configuration bam.Reference  `json:"configuration"`
	ServerScope   *bam.Reference `json:"serverScope"`
	Definition    bam.Reference  `json:"definition"`
	Value         bam.Value      `json:"value"`
}

// OptionDefinition identifies a server-supported option and its legal scopes.
type OptionDefinition struct {
	bam.Resource
	AssignableTypes []string `json:"assignableTypes"`
}

// OptionSpec configures a DNSOption selected by Name or Definition. Common
// client access options can be built with dns/accesscontrol constructors.
type OptionSpec struct {
	Name        string
	Definition  *bam.Reference
	Value       bam.Value
	ServerScope *bam.Reference
}

func (s OptionSpec) MarshalJSON() ([]byte, error) {
	if (strings.TrimSpace(s.Name) == "") == (s.Definition == nil) {
		return nil, fmt.Errorf("dns: option requires either a name or a positive definition id")
	}
	if s.Definition != nil && s.Definition.ID <= 0 {
		return nil, fmt.Errorf("dns: option definition requires a positive id")
	}
	if s.Value.IsZero() {
		return nil, fmt.Errorf("dns: option value is required")
	}
	if s.ServerScope != nil && s.ServerScope.ID <= 0 {
		return nil, fmt.Errorf("dns: server scope requires a positive id")
	}
	return json.Marshal(struct {
		Type        string         `json:"type"`
		Name        string         `json:"name,omitempty"`
		Definition  *bam.Reference `json:"definition,omitempty"`
		Value       bam.Value      `json:"value"`
		ServerScope *bam.Reference `json:"serverScope,omitempty"`
	}{"DNSOption", s.Name, s.Definition, s.Value, s.ServerScope})
}

// OptionChanges edits a locally configured option. Clear ServerScope to apply
// it to all servers. Delete the option to restore inherited configuration.
type OptionChanges struct {
	Value       bam.Field[bam.Value]
	ServerScope bam.Field[bam.Reference]
}

func (c OptionChanges) Apply(d bam.Document) error {
	if c.Value.IsNull() {
		return fmt.Errorf("dns: delete an option to restore inheritance")
	}
	if value, ok := c.Value.Value(); ok && value.IsZero() {
		return fmt.Errorf("dns: option value is required")
	}
	if scope, ok := c.ServerScope.Value(); ok && scope.ID <= 0 {
		return fmt.Errorf("dns: server scope requires a positive id")
	}
	if err := c.Value.Apply(d, "value"); err != nil {
		return err
	}
	return c.ServerScope.Apply(d, "serverScope")
}

type Options struct {
	resource.Collection[Option, OptionSpec, OptionChanges]
}
type OptionItems struct {
	resource.Items[Option, OptionChanges]
}
type OptionDefinitions struct {
	resource.Reader[OptionDefinition]
}

// FindByName requires one local option, reporting ambiguity across server scopes.
func (o Options) FindByName(ctx context.Context, name string) (Option, error) {
	if strings.TrimSpace(name) == "" {
		return Option{}, fmt.Errorf("dns: option name is required")
	}
	return resource.One(o.All(ctx, bam.ListOptions{Where: bam.Match("name", name), Limit: 2}))
}

// RestoreInheritance deletes a local override without modifying its parent.
func (o Options) RestoreInheritance(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return o.Delete(ctx, id, options...)
}

func (c *Client) options(parent string, id int64) Options {
	collection := bam.ChildCollection[Option](c.client, parent, id, "deploymentOptions").WithFilter(bam.Match("type", "DNSOption"))
	return Options{resource.NewCollection[Option, OptionSpec, OptionChanges](collection, "DNSOption")}
}

func (c *Client) ConfigurationOptions(id int64) Options { return c.options("configurations", id) }
func (c *Client) ViewOptions(id int64) Options          { return c.options("views", id) }
func (c *Client) ZoneOptions(id int64) Options          { return c.options("zones", id) }
func (c *Client) ServerOptions(id int64) Options        { return c.options("servers", id) }
func (c *Client) BlockOptions(id int64) Options         { return c.options("blocks", id) }
func (c *Client) NetworkOptions(id int64) Options       { return c.options("networks", id) }

func (c *Client) GlobalOptions() OptionItems {
	collection := bam.NewCollection[Option](c.client, "deploymentOptions").WithFilter(bam.Match("type", "DNSOption"))
	return OptionItems{resource.NewItems[Option, OptionChanges](collection, "DNSOption")}
}

func (c *Client) OptionDefinitions() OptionDefinitions {
	collection := bam.NewCollection[OptionDefinition](c.client, "deploymentOptionDefinitions").WithFilter(bam.Match("type", "DNSOptionDefinition"))
	return OptionDefinitions{resource.NewReader(collection, "DNSOptionDefinition")}
}
