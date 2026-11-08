package bam

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// Link is a HAL resource link.
type Link struct {
	Href string `json:"href"`
}

// Page is one collection response. Count is the number returned in this page,
// not the total number of matching resources. Links["next"] advances the list.
type Page[T any] struct {
	Count int             `json:"count"`
	Data  []T             `json:"data"`
	Links map[string]Link `json:"_links"`
}

// ListOptions controls a collection query. Filter uses BAM's documented filter
// grammar. Limit defaults to 1000, Offset to zero. All follows HAL next links
// even when the server returns fewer than Limit items. MaxPages defaults to
// 10000 and bounds an iteration; reaching it with a next link returns an error.
type ListOptions struct {
	Filter   string
	Where    Filter
	Fields   []string
	OrderBy  []string
	Limit    int
	Offset   int
	MaxPages int
}

func (o ListOptions) query() (url.Values, error) {
	if err := o.Where.Err(); err != nil {
		return nil, err
	}
	if o.Limit < 0 || o.Offset < 0 || o.MaxPages < 0 {
		return nil, fmt.Errorf("bam: pagination values cannot be negative")
	}
	limit := o.Limit
	if limit == 0 {
		limit = 1000
	}
	q := url.Values{"limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(o.Offset)}}
	filter := And(Filter{expression: o.Filter}, o.Where).String()
	if o.Where.String() == "" {
		filter = o.Filter
	}
	if o.Filter == "" {
		filter = o.Where.String()
	}
	for key, value := range map[string]string{
		"filter": filter, "fields": strings.Join(o.Fields, ","), "orderBy": strings.Join(o.OrderBy, ","),
	} {
		if value != "" {
			q.Set(key, value)
		}
	}
	return q, nil
}

// Page fetches exactly one page.
func (c Collection[T]) Page(ctx context.Context, options ListOptions) (Page[T], error) {
	if err := c.check(); err != nil {
		return Page[T]{}, err
	}
	options.Where = And(c.filter, options.Where)
	query, err := options.query()
	if err != nil {
		return Page[T]{}, err
	}
	return c.page(ctx, c.path, query)
}

// All lazily walks HAL next links. Breaking the range stops further requests;
// no goroutine is created. Iteration can be repeated, starting a fresh query.
// A changing server collection is not a snapshot; use stable ordering when
// appropriate. Authentication is never forwarded outside the configured origin.
func (c Collection[T]) All(ctx context.Context, options ListOptions) iter.Seq2[T, error] {
	options.Fields, options.OrderBy = slices.Clone(options.Fields), slices.Clone(options.OrderBy)
	options.Where = And(c.filter, options.Where)
	return func(yield func(T, error) bool) {
		var zero T
		if err := c.check(); err != nil {
			yield(zero, err)
			return
		}
		query, err := options.query()
		if err != nil {
			yield(zero, err)
			return
		}
		current, err := c.client.resolve(c.path)
		if err != nil {
			yield(zero, err)
			return
		}
		current.RawQuery = query.Encode()
		maxPages := options.MaxPages
		if maxPages == 0 {
			maxPages = 10000
		}
		seen := make(map[string]struct{})
		for pages := 0; ; pages++ {
			key := current.String()
			_, repeated := seen[key]
			if repeated || pages >= maxPages {
				yield(zero, &ProtocolError{Method: http.MethodGet, Path: current.Path, Detail: "pagination cycle or page limit reached"})
				return
			}
			seen[key] = struct{}{}
			page, err := c.page(ctx, key, nil)
			if err != nil {
				yield(zero, err)
				return
			}
			for _, item := range page.Data {
				if err := ctx.Err(); err != nil {
					yield(zero, err)
					return
				}
				if !yield(item, nil) {
					return
				}
			}
			next := page.Links["next"].Href
			if next == "" {
				return
			}
			current, err = c.nextURL(current, next, query)
			if err != nil {
				yield(zero, err)
				return
			}
		}
	}
}

func (c Collection[T]) page(ctx context.Context, path string, query url.Values) (Page[T], error) {
	var raw struct {
		Count *int              `json:"count"`
		Data  []json.RawMessage `json:"data"`
		Links map[string]Link   `json:"_links"`
	}
	if err := c.client.Do(ctx, Request{Method: http.MethodGet, Path: path, Query: query}, &raw); err != nil {
		return Page[T]{}, err
	}
	badPage := func(detail string, err error) (Page[T], error) {
		return Page[T]{}, &ProtocolError{Method: http.MethodGet, Path: apiPath + "/" + c.path, Detail: detail, Err: err}
	}
	if raw.Count == nil || *raw.Count < 0 || raw.Data == nil || *raw.Count != len(raw.Data) {
		return badPage("collection requires count matching its data array", nil)
	}
	page := Page[T]{Count: *raw.Count, Data: make([]T, len(raw.Data)), Links: raw.Links}
	for i, item := range raw.Data {
		if len(item) == 0 || item[0] != '{' {
			return badPage("collection item must be a resource object", nil)
		}
		if err := json.Unmarshal(item, &page.Data[i]); err != nil {
			return badPage("decode collection item", err)
		}
	}
	return page, nil
}

func (c Collection[T]) nextURL(current *url.URL, href string, original url.Values) (*url.URL, error) {
	reference, err := url.Parse(href)
	if err != nil {
		return nil, fmt.Errorf("bam: invalid pagination link")
	}
	next, err := c.client.resolve(current.ResolveReference(reference).String())
	if err != nil {
		return nil, err
	}
	if next.Path != current.Path {
		return nil, fmt.Errorf("bam: pagination link changed the collection scope")
	}
	query := next.Query()
	for _, key := range []string{"filter", "fields", "orderBy"} {
		if actual, ok := query[key]; ok && !slices.Equal(actual, original[key]) {
			return nil, fmt.Errorf("bam: pagination link changed query field %s", key)
		}
	}
	for key, values := range original {
		if key == "limit" || key == "offset" {
			continue
		}
		query[key] = slices.Clone(values)
	}
	next.RawQuery = query.Encode()
	return next, nil
}
