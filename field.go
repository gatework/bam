package bam

// Field describes an update to one field. Its zero value leaves the field
// unchanged; Set assigns a value (including false or zero), and Clear assigns
// JSON null. Whether null is supported is defined by the resource's contract.
type Field[T any] struct {
	value  T
	isSet  bool
	isNull bool
}

// Set assigns value when an update is applied.
func Set[T any](value T) Field[T] { return Field[T]{value: value, isSet: true} }

// Clear explicitly assigns JSON null when an update is applied.
func Clear[T any]() Field[T] { return Field[T]{isSet: true, isNull: true} }

// IsSet reports whether the field is being changed, including being cleared.
func (f Field[T]) IsSet() bool { return f.isSet }

// IsNull reports whether the field is explicitly being cleared.
func (f Field[T]) IsNull() bool { return f.isNull }

// Value returns the assigned value. The boolean is false for unchanged or
// cleared fields; use IsSet and IsNull when distinguishing those cases.
func (f Field[T]) Value() (T, bool) { return f.value, f.isSet && !f.isNull }

// Apply changes key in a complete document, preserving all other fields.
func (f Field[T]) Apply(document Document, key string) error {
	if !f.isSet {
		return nil
	}
	if f.isNull {
		return document.Set(key, nil)
	}
	return document.Set(key, f.value)
}
