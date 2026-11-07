package bam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
)

// Resource is the common read representation. Domain models add documented
// fields. Use Collection.Document for lossless reads before a full replacement.
type Resource struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// Reference identifies an existing BAM resource in a request body.
type Reference struct {
	ID int64 `json:"id"`
}

// Document retains every JSON field, including fields unknown to this library.
// Use a full GET document for PUT. A projected collection item is not a complete
// resource. Call Clone before modifying a document shared with another goroutine.
type Document map[string]json.RawMessage

// Decode reads a typed view without changing or dropping the retained fields.
func (d Document) Decode(dst any) error {
	b, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("bam: encode document: %w", err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("bam: decode document: %w", err)
	}
	return nil
}

// Set encodes a field. Use nil for JSON null and delete(d, field) to omit it.
func (d Document) Set(field string, value any) error {
	if d == nil || field == "" {
		return fmt.Errorf("bam: Set requires a nonnil document and a field name")
	}
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("bam: encode field %s: %w", field, err)
	}
	d[field] = b
	return nil
}

// Clone makes an independent copy, including each raw JSON value.
func (d Document) Clone() Document {
	copy := maps.Clone(d)
	for key, value := range copy {
		copy[key] = bytes.Clone(value)
	}
	return copy
}
