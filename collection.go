package bam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Collection shares JSON CRUD and pagination across domain modules. List and
// Create use the selected parent scope; Get, Document, Replace, and Delete
// address a resource by its globally unique ID, regardless of that parent.
// Not every BAM collection supports every verb; consult the device's OpenAPI
// schema. Relationship collections require their own linking operations.
type Collection[T any] struct {
	client     *Client
	path       string
	collection string
	err        error
	filter     Filter
	isLinked   bool
}

// NewCollection selects a root collection, such as "configurations".
func NewCollection[T any](client *Client, collection string) Collection[T] {
	c := Collection[T]{client: client, collection: collection, path: collection}
	if client == nil || !validCollection(collection) {
		c.err = fmt.Errorf("bam: collection requires a client and an alphanumeric collection name")
	}
	return c
}

// ChildCollection selects a parent's child resources. Invalid IDs are reported
// by operations without making a request, keeping scope selection side-effect free.
func ChildCollection[T any](client *Client, parent string, parentID int64, collection string) Collection[T] {
	c := NewCollection[T](client, collection)
	if c.err == nil && (!validCollection(parent) || parentID <= 0) {
		c.err = fmt.Errorf("bam: parent scope requires an alphanumeric collection name and positive ID")
	}
	c.path = parent + "/" + strconv.FormatInt(parentID, 10) + "/" + collection
	return c
}

// LinkedCollection selects a relationship whose item operations use the nested
// path. Deleting an item removes the link rather than the globally identified
// resource. Domain handles expose only the relationship's supported verbs.
func LinkedCollection[T any](client *Client, parent string, parentID int64, collection string) Collection[T] {
	c := ChildCollection[T](client, parent, parentID, collection)
	c.isLinked = true
	return c
}

// WithFilter returns a copy with a fixed collection predicate. Page and All
// combine it with caller filters, so callers cannot override the selected scope.
// Item operations still use resource IDs and do not apply collection filters.
func (c Collection[T]) WithFilter(filter Filter) Collection[T] {
	c.filter = And(c.filter, filter)
	return c
}

func validCollection(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func (c Collection[T]) check() error {
	if c.err != nil {
		return c.err
	}
	if c.client == nil {
		return fmt.Errorf("bam: uninitialized collection")
	}
	return nil
}

func (c Collection[T]) itemPath(id int64) (string, error) {
	if err := c.check(); err != nil {
		return "", err
	}
	if id <= 0 {
		return "", fmt.Errorf("bam: resource ID must be positive")
	}
	path := c.collection
	if c.isLinked {
		path = c.path
	}
	return path + "/" + strconv.FormatInt(id, 10), nil
}

// Get reads the resource with a globally unique ID.
func (c Collection[T]) Get(ctx context.Context, id int64) (T, error) {
	path, err := c.itemPath(id)
	if err != nil {
		var zero T
		return zero, err
	}
	return c.resource(ctx, http.MethodGet, path, nil, id)
}

// Create submits a resource specification. Domain specs infer the resource type
// and validate business values. Document may be used for device-specific fields.
func (c Collection[T]) Create(ctx context.Context, spec any, options ...WriteOption) (T, error) {
	var zero T
	if err := c.check(); err != nil {
		return zero, err
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return zero, fmt.Errorf("bam: encode resource specification: %w", err)
	}
	var object Document
	if json.Unmarshal(data, &object) != nil || object == nil {
		return zero, fmt.Errorf("bam: resource specification must be a JSON object")
	}
	return c.resource(ctx, http.MethodPost, c.path, json.RawMessage(data), 0, options...)
}

// Document reads the complete resource without field projection. Unknown fields
// survive edits made through Document.Set and a subsequent Replace.
func (c Collection[T]) Document(ctx context.Context, id int64) (Document, error) {
	path, err := c.itemPath(id)
	if err != nil {
		return nil, err
	}
	var document Document
	if err := c.client.Do(ctx, Request{Method: http.MethodGet, Path: path}, &document); err != nil {
		return nil, err
	}
	if err := validateResource(document, http.MethodGet, path, id); err != nil {
		return nil, err
	}
	return document, nil
}

// Replace performs PUT with a full representation, not a partial patch. Obtain
// the document using Document; incomplete bodies can reset server-side fields.
// The input is not mutated. HAL links are omitted from the replacement.
func (c Collection[T]) Replace(ctx context.Context, id int64, document Document, options ...WriteOption) (T, error) {
	var zero T
	path, err := c.itemPath(id)
	if err != nil {
		return zero, err
	}
	var identity Resource
	if document.Decode(&identity) != nil || identity.ID != id || strings.TrimSpace(identity.Type) == "" {
		return zero, fmt.Errorf("bam: replacement requires a complete resource document with matching id and nonempty type")
	}
	body := document.Clone()
	delete(body, "_links")
	return c.resource(ctx, http.MethodPut, path, body, id, options...)
}

// Delete removes the resource identified by its globally unique ID.
func (c Collection[T]) Delete(ctx context.Context, id int64, options ...WriteOption) error {
	path, err := c.itemPath(id)
	if err != nil {
		return err
	}
	header, err := writeHeaders(options)
	if err != nil {
		return err
	}
	return c.client.Do(ctx, Request{Method: http.MethodDelete, Path: path, Header: header}, nil)
}

func (c Collection[T]) resource(ctx context.Context, method, path string, body any, expectedID int64, options ...WriteOption) (T, error) {
	var result T
	var document Document
	header, err := writeHeaders(options)
	if err != nil {
		return result, err
	}
	if err := c.client.Do(ctx, Request{Method: method, Path: path, Body: body, Header: header}, &document); err != nil {
		return result, err
	}
	if err := validateResource(document, method, path, expectedID); err != nil {
		return result, err
	}
	if err := document.Decode(&result); err != nil {
		return result, &ProtocolError{Method: method, Path: apiPath + "/" + path, Detail: "decode resource", Err: err}
	}
	return result, nil
}

func validateResource(document Document, method, path string, expectedID int64) error {
	var resource Resource
	if err := document.Decode(&resource); err != nil || resource.ID <= 0 || strings.TrimSpace(resource.Type) == "" {
		return &ProtocolError{Method: method, Path: apiPath + "/" + path, Detail: "expected a resource with positive id and nonempty type", Err: err}
	}
	if expectedID != 0 && resource.ID != expectedID {
		return &ProtocolError{Method: method, Path: apiPath + "/" + path, Detail: "response resource ID does not match requested ID"}
	}
	return nil
}
