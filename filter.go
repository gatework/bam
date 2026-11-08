package bam

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Filter is a predicate constructed without interpolating values into the
// filter grammar. Its zero value selects all resources. Invalid field names or
// values are reported by the collection before making an HTTP request.
type Filter struct {
	expression string
	err        error
}

// Match selects resources whose field equals value.
func Match(field, value string) Filter { return predicate(field, "eq", value) }

// Contains selects resources whose field contains value. BAM also supports
// address containment when field is a block or network range.
func Contains(field, value string) Filter { return predicate(field, "contains", value) }

// MatchID selects resources by a positive identifier, including references such
// as configuration.id. Invalid identifiers fail when the query is executed.
func MatchID(field string, id int64) Filter {
	if id <= 0 {
		return Filter{err: fmt.Errorf("bam: filter identifier must be positive")}
	}
	if !validField(field) {
		return Filter{err: fmt.Errorf("bam: invalid filter field")}
	}
	return Filter{expression: field + ":eq(" + strconv.FormatInt(id, 10) + ")"}
}

func predicate(field, operator, value string) Filter {
	if !validField(field) {
		return Filter{err: fmt.Errorf("bam: invalid filter field")}
	}
	// Both quote styles are documented. Use the style absent from the value
	// instead of assuming an undocumented escape sequence is accepted by BAM.
	quote := `"`
	if strings.Contains(value, quote) {
		quote = "'"
	}
	if strings.Contains(value, quote) || strings.ContainsRune(value, '\\') || strings.ContainsFunc(value, unicode.IsControl) {
		return Filter{err: fmt.Errorf("bam: filter value contains unsupported quoting or control characters")}
	}
	return Filter{expression: field + ":" + operator + "(" + quote + value + quote + ")"}
}

func validField(field string) bool {
	for _, part := range strings.Split(field, ".") {
		if part == "" {
			return false
		}
		for i, r := range part {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || i > 0 && r >= '0' && r <= '9' {
				continue
			}
			return false
		}
	}
	return true
}

// And requires every nonempty predicate to match.
func And(filters ...Filter) Filter { return combine("and", filters) }

// Or requires at least one nonempty predicate to match.
func Or(filters ...Filter) Filter { return combine("or", filters) }

func combine(operator string, filters []Filter) Filter {
	parts := make([]string, 0, len(filters))
	for _, filter := range filters {
		if filter.err != nil {
			return filter
		}
		if filter.expression != "" {
			parts = append(parts, "("+filter.expression+")")
		}
	}
	return Filter{expression: strings.Join(parts, " "+operator+" ")}
}

// String returns the encoded predicate. Use ListOptions.Where to retain input
// validation; String alone does not report an invalid predicate.
func (f Filter) String() string { return f.expression }

// Err reports a predicate construction error without sending a request.
func (f Filter) Err() error { return f.err }
