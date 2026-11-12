// Package resource shares typed operations among domain modules. It is not an
// independently configurable public client or a registry of business resources.
package resource

import (
	"context"
	"iter"

	"github.com/gatework/bam"
)

// List exposes only collection queries, suitable for relationship endpoints.
type List[T any] struct{ collection bam.Collection[T] }

// NewList constructs read-only iteration over the supplied collection.
func NewList[T any](collection bam.Collection[T]) List[T] { return List[T]{collection: collection} }

// Page fetches one page without following its next link.
func (l List[T]) Page(ctx context.Context, options bam.ListOptions) (bam.Page[T], error) {
	return l.collection.Page(ctx, options)
}

// All lazily follows bounded pagination while the caller continues iteration.
func (l List[T]) All(ctx context.Context, options bam.ListOptions) iter.Seq2[T, error] {
	return l.collection.All(ctx, options)
}

// CollectionOf gives domain implementations access to shared operations without
// promoting a CRUD escape hatch through every public read-only handle.
func CollectionOf[T any](list List[T]) bam.Collection[T] { return list.collection }

// One reads at most two matching items and never silently chooses the first of
// an ambiguous result. Errors returned by the iterator remain unchanged.
func One[T any](items iter.Seq2[T, error]) (T, error) {
	var result, zero T
	found := false
	for item, err := range items {
		if err != nil {
			return zero, err
		}
		if found {
			return zero, bam.ErrAmbiguous
		}
		result, found = item, true
	}
	if !found {
		return zero, bam.ErrNotFound
	}
	return result, nil
}

// OneMatching finds a unique match across every page without hiding iteration
// failures. It stops once ambiguity is established, as One does.
func OneMatching[T any](items iter.Seq2[T, error], matches func(T) bool) (T, error) {
	return One(func(yield func(T, error) bool) {
		for item, err := range items {
			if err != nil || matches(item) {
				if !yield(item, err) || err != nil {
					return
				}
			}
		}
	})
}
