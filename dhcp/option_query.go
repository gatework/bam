package dhcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// OptionQuery is an immutable selector for one local option and server scope.
// The zero value is invalid; use NewOptionQuery, NewOptionQueryByCode, or
// NewOptionQueryByDefinition. Code selectors match only DHCPv4 client options.
type OptionQuery struct {
	name          string
	code          uint8
	definitionID  int64
	serverScopeID int64
}

// NewOptionQuery selects an option by exact name. Server scope zero selects an
// all-server override; a positive ID selects one server or server group.
func NewOptionQuery(name string, serverScopeID int64) (OptionQuery, error) {
	if strings.TrimSpace(name) == "" || serverScopeID < 0 {
		return OptionQuery{}, fmt.Errorf("dhcp: option name and nonnegative server scope id are required")
	}
	return OptionQuery{name: name, serverScopeID: serverScopeID}, nil
}

// NewOptionQueryByCode selects a nonzero DHCPv4 client-option code at the exact
// server scope. Scope zero selects an all-server override, not inherited values.
func NewOptionQueryByCode(code uint8, serverScopeID int64) (OptionQuery, error) {
	if code == 0 || serverScopeID < 0 {
		return OptionQuery{}, fmt.Errorf("dhcp: nonzero option code and nonnegative server scope id are required")
	}
	return OptionQuery{code: code, serverScopeID: serverScopeID}, nil
}

// NewOptionQueryByDefinition selects a positive definition ID at the exact
// server scope, with zero selecting an all-server override.
func NewOptionQueryByDefinition(definitionID, serverScopeID int64) (OptionQuery, error) {
	if definitionID <= 0 || serverScopeID < 0 {
		return OptionQuery{}, fmt.Errorf("dhcp: positive option definition and nonnegative server scope ids are required")
	}
	return OptionQuery{definitionID: definitionID, serverScopeID: serverScopeID}, nil
}

// IsValid reports whether a constructor supplied a valid selection.
func (q OptionQuery) IsValid() bool { return q.name != "" || q.code != 0 || q.definitionID > 0 }

func (q OptionQuery) isMatch(option Option) bool {
	if option.Type != "DHCPv4ClientOption" && option.Type != "DHCPv4ServiceOption" && option.Type != "DHCPv4RawOption" {
		return false
	}
	if q.name != "" && option.Name != q.name {
		return false
	}
	if q.code != 0 && (option.Type != "DHCPv4ClientOption" || option.Code != q.code) {
		return false
	}
	if q.definitionID != 0 && (option.Definition == nil || option.Definition.ID != q.definitionID) {
		return false
	}
	if option.ServerScope == nil {
		return q.serverScopeID == 0
	}
	return q.serverScopeID > 0 && option.ServerScope.ID == q.serverScopeID
}

// Find requires a unique option at the requested local and server scope. It
// matches complete paginated resources locally; it does not invent nested-field
// filters or compute inherited configuration. Ambiguity never selects a winner.
func (o Options) Find(ctx context.Context, query OptionQuery) (Option, error) {
	if !query.IsValid() {
		return Option{}, fmt.Errorf("dhcp: option query is unset; use an option query constructor")
	}
	return resource.OneMatching(o.All(ctx, bam.ListOptions{}), query.isMatch)
}
