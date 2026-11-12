package bam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// Value retains a deployment option's JSON value without converting numbers
// through float64. The option definition determines whether it is a scalar,
// list or object. Domain constructors provide typed values for common options.
// Copies are safe to share; UnmarshalJSON replaces the receiver's storage.
type Value struct{ data json.RawMessage }

// ValueOf encodes a value at the JSON extension point. Prefer a domain's typed
// option constructors for known DNS and DHCP options.
func ValueOf(value any) (Value, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return Value{}, fmt.Errorf("bam: encode option value: %w", err)
	}
	return Value{data: data}, nil
}

// StringValue creates a JSON string value.
func StringValue(value string) Value {
	data, _ := json.Marshal(value) // JSON string encoding cannot fail.
	return Value{data: data}
}

// StringsValue creates a JSON string array, including [] for an empty input.
func StringsValue(values ...string) Value {
	if values == nil {
		values = []string{}
	}
	data, _ := json.Marshal(values) // A string slice always has a JSON encoding.
	return Value{data: data}
}

// BoolValue creates a JSON boolean value.
func BoolValue(value bool) Value { return Value{data: []byte(strconv.FormatBool(value))} }

// IntValue creates a JSON integer value without losing precision.
func IntValue(value int64) Value { return Value{data: []byte(strconv.FormatInt(value, 10))} }

// IsZero reports whether no JSON value has been supplied.
func (v Value) IsZero() bool { return len(v.data) == 0 }

// IsNull reports an explicitly supplied JSON null, including a typed nil passed
// to ValueOf. An unset Value is distinct from null and reports false.
func (v Value) IsNull() bool { return bytes.Equal(bytes.TrimSpace(v.data), []byte("null")) }

// MarshalJSON returns an independent JSON representation. An unset value
// encodes as null; request specifications reject it where a value is required.
func (v Value) MarshalJSON() ([]byte, error) {
	if v.IsZero() {
		return []byte("null"), nil
	}
	return bytes.Clone(v.data), nil
}

// UnmarshalJSON retains a copy of any valid JSON value, including null.
func (v *Value) UnmarshalJSON(data []byte) error {
	if !json.Valid(data) {
		return fmt.Errorf("bam: invalid JSON option value")
	}
	v.data = bytes.Clone(data)
	return nil
}

// Decode reads a typed option value without altering its JSON representation.
func (v Value) Decode(dst any) error {
	if v.IsZero() {
		return fmt.Errorf("bam: option value is unset")
	}
	if err := json.Unmarshal(v.data, dst); err != nil {
		return fmt.Errorf("bam: decode option value: %w", err)
	}
	return nil
}
