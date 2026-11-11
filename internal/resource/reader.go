package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/gatework/bam"
)

// Reader exposes collection queries and individual resource reads.
type Reader[T any] struct {
	List[T]
	kinds []string
}

// NewReader constructs read-only resource operations restricted to the supplied types.
func NewReader[T any](collection bam.Collection[T], kinds ...string) Reader[T] {
	return Reader[T]{List: NewList(collection), kinds: slices.Clone(kinds)}
}

// Get reads and validates the resource with the supplied ID.
func (r Reader[T]) Get(ctx context.Context, id int64) (T, error) {
	result, err := r.collection.Get(ctx, id)
	if err != nil {
		return result, err
	}
	if err := r.checkValue(result); err != nil {
		var zero T
		return zero, err
	}
	return result, nil
}

// Document reads the complete resource without projecting away unknown fields.
func (r Reader[T]) Document(ctx context.Context, id int64) (bam.Document, error) {
	document, err := r.collection.Document(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := r.checkType(document); err != nil {
		return nil, err
	}
	return document, nil
}

func (r Reader[T]) checkType(document bam.Document) error {
	if len(r.kinds) == 0 {
		return nil
	}
	var kind string
	if err := json.Unmarshal(document["type"], &kind); err != nil {
		return fmt.Errorf("bam: decode resource type: %w", err)
	}
	if !slices.Contains(r.kinds, kind) {
		return fmt.Errorf("bam: unexpected resource type %q", kind)
	}
	return nil
}

func (r Reader[T]) checkValue(value T) error {
	if len(r.kinds) == 0 {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("bam: inspect resource type: %w", err)
	}
	var document bam.Document
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("bam: inspect resource type: %w", err)
	}
	return r.checkType(document)
}
