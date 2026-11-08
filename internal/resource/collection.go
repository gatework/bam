package resource

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gatework/bam"
)

// Changes applies a domain-specific update to a complete document.
type Changes interface{ Apply(bam.Document) error }

// Items manages existing resources. It deliberately has no Create method.
type Items[T any, U Changes] struct{ Reader[T] }

func NewItems[T any, U Changes](collection bam.Collection[T], kinds ...string) Items[T, U] {
	return Items[T, U]{Reader: NewReader(collection, kinds...)}
}

func (r Items[T, U]) Update(ctx context.Context, id int64, changes U, options ...bam.WriteOption) (T, error) {
	return r.collection.Update(ctx, id, func(document bam.Document) error {
		if err := r.checkType(document); err != nil {
			return err
		}
		return changes.Apply(document)
	}, options...)
}

// Delete checks the resource type before deleting a globally identified item.
func (r Items[T, U]) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	if len(r.kinds) > 0 {
		if _, err := r.Document(ctx, id); err != nil {
			return err
		}
	}
	return r.collection.Delete(ctx, id, options...)
}

// Collection adds type-safe creation at a supported parent scope.
type Collection[T any, S any, U Changes] struct{ Items[T, U] }

func NewCollection[T any, S any, U Changes](collection bam.Collection[T], kinds ...string) Collection[T, S, U] {
	return Collection[T, S, U]{Items: NewItems[T, U](collection, kinds...)}
}

func (r Collection[T, S, U]) Create(ctx context.Context, spec S, options ...bam.WriteOption) (T, error) {
	var zero T
	// Validate the discriminator before the mutation as well as in its reply.
	data, err := json.Marshal(spec)
	if err != nil {
		return zero, fmt.Errorf("bam: encode resource specification: %w", err)
	}
	var document bam.Document
	if err := json.Unmarshal(data, &document); err != nil || document == nil {
		return zero, fmt.Errorf("bam: resource specification must be an object")
	}
	if err := r.checkType(document); err != nil {
		return zero, err
	}
	result, err := r.collection.Create(ctx, document, options...)
	if err != nil {
		return zero, err
	}
	if err := r.checkValue(result); err != nil {
		return zero, err
	}
	return result, nil
}
