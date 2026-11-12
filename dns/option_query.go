package dns

import (
	"context"
	"fmt"
	"strings"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// OptionQuery is an immutable selector for one local override and server scope.
// The zero value is invalid; use NewOptionQuery or NewOptionQueryByDefinition.
type OptionQuery struct {
	name          string
	definitionID  int64
	serverScopeID int64
}

// NewOptionQuery selects an option by its exact name. Server scope zero selects
// an all-server override; a positive ID selects one server or server group.
func NewOptionQuery(name string, serverScopeID int64) (OptionQuery, error) {
	if strings.TrimSpace(name) == "" || serverScopeID < 0 {
		return OptionQuery{}, fmt.Errorf("dns: option name and nonnegative server scope id are required")
	}
	return OptionQuery{name: name, serverScopeID: serverScopeID}, nil
}

// NewOptionQueryByDefinition selects a positive definition ID at the exact
// server scope. Scope zero selects an all-server override, not inherited values.
func NewOptionQueryByDefinition(definitionID, serverScopeID int64) (OptionQuery, error) {
	if definitionID <= 0 || serverScopeID < 0 {
		return OptionQuery{}, fmt.Errorf("dns: positive option definition and nonnegative server scope ids are required")
	}
	return OptionQuery{definitionID: definitionID, serverScopeID: serverScopeID}, nil
}

// IsValid reports whether a constructor supplied a valid selection.
func (q OptionQuery) IsValid() bool { return q.name != "" || q.definitionID > 0 }

func (q OptionQuery) isMatch(option Option) bool {
	if option.Type != "DNSOption" || q.name != "" && option.Name != q.name || q.definitionID != 0 && option.Definition.ID != q.definitionID {
		return false
	}
	if option.ServerScope == nil {
		return q.serverScopeID == 0
	}
	return q.serverScopeID > 0 && option.ServerScope.ID == q.serverScopeID
}

// Find retrieves one local override with an exact server scope. It inspects
// complete paginated responses in this parent, avoiding unverified nested-field
// filters. Missing or duplicate matches return bam.ErrNotFound or ErrAmbiguous.
func (o Options) Find(ctx context.Context, query OptionQuery) (Option, error) {
	if !query.IsValid() {
		return Option{}, fmt.Errorf("dns: option query is unset; use an option query constructor")
	}
	return resource.OneMatching(o.All(ctx, bam.ListOptions{}), query.isMatch)
}
