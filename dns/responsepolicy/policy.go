package responsepolicy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gatework/bam"
)

// Action is the policyType discriminator within a ResponsePolicy resource.
type Action string

const (
	Allowlist Action = "ALLOWLIST"
	Blackhole Action = "BLACKHOLE"
	Blocklist Action = "BLOCKLIST"
	Redirect  Action = "REDIRECT"
)

func (a Action) validate() error {
	switch a {
	case Allowlist, Blackhole, Blocklist, Redirect:
		return nil
	}
	return fmt.Errorf("responsepolicy: invalid policy action")
}

type Policy struct {
	bam.Resource
	Configuration  bam.Reference `json:"configuration"`
	Action         Action        `json:"policyType"`
	TTL            int64         `json:"ttl"`
	RedirectTarget string        `json:"redirectTarget"`
}

// Spec creates a local policy. TTL is seconds; nil lets BAM select its default.
type Spec struct {
	Name           string
	Action         Action
	TTL            *int64
	RedirectTarget string
}

func (s Spec) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(s.Name) == "" {
		return nil, fmt.Errorf("responsepolicy: policy name is required")
	}
	if err := s.Action.validate(); err != nil {
		return nil, err
	}
	if s.TTL != nil && (*s.TTL < 0 || *s.TTL > 2147483647) {
		return nil, fmt.Errorf("responsepolicy: invalid ttl")
	}
	if s.Action == Redirect && strings.TrimSpace(s.RedirectTarget) == "" {
		return nil, fmt.Errorf("responsepolicy: redirect target is required")
	}
	if s.Action != Redirect && s.RedirectTarget != "" {
		return nil, fmt.Errorf("responsepolicy: redirect target requires a redirect policy")
	}
	return json.Marshal(struct {
		Type           string `json:"type"`
		Name           string `json:"name"`
		Action         Action `json:"policyType"`
		TTL            *int64 `json:"ttl,omitempty"`
		RedirectTarget string `json:"redirectTarget,omitempty"`
	}{"ResponsePolicy", s.Name, s.Action, s.TTL, s.RedirectTarget})
}

type Changes struct {
	Name           bam.Field[string]
	Action         bam.Field[Action]
	TTL            bam.Field[int64]
	RedirectTarget bam.Field[string]
}

func (c Changes) Apply(d bam.Document) error {
	if c.Name.IsNull() || c.Action.IsNull() || c.TTL.IsNull() || c.RedirectTarget.IsNull() {
		return fmt.Errorf("responsepolicy: policy fields cannot be null")
	}
	if name, ok := c.Name.Value(); ok && strings.TrimSpace(name) == "" {
		return fmt.Errorf("responsepolicy: policy name is required")
	}
	if a, ok := c.Action.Value(); ok {
		if err := a.validate(); err != nil {
			return err
		}
	}
	if ttl, ok := c.TTL.Value(); ok && (ttl < 0 || ttl > 2147483647) {
		return fmt.Errorf("responsepolicy: invalid ttl")
	}
	if err := c.Name.Apply(d, "name"); err != nil {
		return err
	}
	if err := c.Action.Apply(d, "policyType"); err != nil {
		return err
	}
	if err := c.TTL.Apply(d, "ttl"); err != nil {
		return err
	}
	if err := c.RedirectTarget.Apply(d, "redirectTarget"); err != nil {
		return err
	}
	var policy Policy
	if err := d.Decode(&policy); err != nil {
		return err
	}
	if policy.Action == Redirect && strings.TrimSpace(policy.RedirectTarget) == "" {
		return fmt.Errorf("responsepolicy: redirect target is required")
	}
	return nil
}

type Item struct{ bam.Resource }

// ItemSpec names a domain or server-supported RPZ trigger. BAM validates its
// naming policy; no implicit wildcard expansion or client-side normalization is
// performed, since these would change which DNS names match the policy.
type ItemSpec struct {
	Name string `json:"name"`
}

func (s ItemSpec) MarshalJSON() ([]byte, error) {
	if strings.TrimSpace(s.Name) == "" {
		return nil, fmt.Errorf("responsepolicy: domain name is required")
	}
	return json.Marshal(struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}{"ResponsePolicyItem", s.Name})
}

type ItemChanges struct{ Name bam.Field[string] }

func (c ItemChanges) Apply(d bam.Document) error {
	if c.Name.IsNull() {
		return fmt.Errorf("responsepolicy: domain name cannot be null")
	}
	if name, ok := c.Name.Value(); ok && strings.TrimSpace(name) == "" {
		return fmt.Errorf("responsepolicy: domain name is required")
	}
	return c.Name.Apply(d, "name")
}
