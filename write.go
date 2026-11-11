package bam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WriteOption changes one mutation's request options. Options never change the
// shared client's authentication, defaults or subsequent requests.
type WriteOption func(http.Header) error

// WriteHeader supplies a BlueCat operation header. Domain-specific constructors
// should be preferred when available. Only x-bcn- headers are accepted.
func WriteHeader(name, value string) WriteOption {
	return func(header http.Header) error {
		if !strings.HasPrefix(strings.ToLower(name), "x-bcn-") || strings.ContainsAny(name+value, "\r\n") {
			return fmt.Errorf("bam: invalid bluecat write header")
		}
		for _, r := range name {
			if r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
				continue
			}
			return fmt.Errorf("bam: invalid bluecat write header name")
		}
		header.Set(name, value)
		return nil
	}
}

// WithChangeControlComment records a change-control comment for a mutation.
func WithChangeControlComment(comment string) WriteOption {
	return WriteHeader("x-bcn-change-control-comment", comment)
}

func writeHeaders(options []WriteOption) (http.Header, error) {
	if len(options) == 0 {
		return nil, nil //nolint:nilnil // nil headers mean that no write options were supplied
	}
	header := make(http.Header)
	for _, option := range options {
		if option == nil {
			return nil, fmt.Errorf("bam: nil write option")
		}
		if err := option(header); err != nil {
			return nil, err
		}
	}
	return header, nil
}

// Update reads a complete resource, applies edit once, and replaces it. It never
// writes a projected model and never retries the mutation. Read/edit/PUT is not
// an atomic transaction with other clients editing the same resource.
// Redacted resources cannot safely be replaced and are rejected.
func (c Collection[T]) Update(ctx context.Context, id int64, edit func(Document) error, options ...WriteOption) (T, error) {
	var zero T
	if edit == nil {
		return zero, fmt.Errorf("bam: update requires an edit function")
	}
	document, err := c.Document(ctx, id)
	if err != nil {
		return zero, err
	}
	var redacted []string
	if raw, ok := document["_redactedFields"]; ok {
		if err := json.Unmarshal(raw, &redacted); err != nil || len(redacted) > 0 {
			return zero, fmt.Errorf("bam: cannot replace a resource containing redacted fields")
		}
	}
	originalType := string(document["type"])
	if err := edit(document); err != nil {
		return zero, err
	}
	if string(document["type"]) != originalType {
		return zero, fmt.Errorf("bam: update cannot change resource type")
	}
	return c.Replace(ctx, id, document, options...)
}
