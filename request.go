package bam

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// Request describes a JSON operation on RESTful v2. Path may be API-relative or
// a same-origin /api/v2 URL. Header supports BAM x-bcn-* and conditional headers;
// authorization, cookies, media types, and routing remain owned by Client.
type Request struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   any
}

// Do provides access to documented endpoints outside the domain modules. It
// decodes one JSON value into dst, or discards the body when dst is nil. HTTP 202
// returns AcceptedError without decoding the pending resource into dst.
func (c *Client) Do(ctx context.Context, request Request, dst any) error {
	req, err := c.prepareRequest(ctx, request)
	if err != nil {
		return err
	}
	path := req.URL.Path
	response, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("bam: %s %s: %w", req.Method, path, err)
	}
	defer func() {
		// Read failures are reported below; closing releases the response body.
		_ = response.Body.Close()
	}()
	data, readErr := io.ReadAll(io.LimitReader(response.Body, c.responseLimit+1))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: response.StatusCode, Method: req.Method, Path: path, Err: readErr}
		if readErr == nil && int64(len(data)) <= c.responseLimit {
			var fields struct{ Code, Reason, Message string }
			if json.Unmarshal(data, &fields) == nil {
				apiErr.Code, apiErr.Reason, apiErr.Message = fields.Code, fields.Reason, fields.Message
			}
		}
		return apiErr
	}
	if readErr != nil {
		return fmt.Errorf("bam: read %s %s response: %w", req.Method, path, readErr)
	}
	protocolError := func(detail string, err error) error {
		return &ProtocolError{Method: req.Method, Path: path, Detail: detail, Err: err}
	}
	if int64(len(data)) > c.responseLimit {
		return protocolError("response exceeds configured size limit", nil)
	}
	if response.StatusCode == http.StatusNoContent && dst == nil {
		return nil
	}
	if dst == nil && response.StatusCode != http.StatusAccepted && len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || (mediaType != "application/json" && mediaType != "application/hal+json") {
		return protocolError("expected a JSON response content type", err)
	}
	if !json.Valid(data) || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return protocolError("expected one non-null JSON value", nil)
	}
	if response.StatusCode == http.StatusAccepted {
		var pending Document
		if err := json.Unmarshal(data, &pending); err != nil {
			return protocolError("expected an accepted resource object", err)
		}
		return &AcceptedError{Resource: pending, Location: response.Header.Get("Location")}
	}
	if dst == nil {
		return nil
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return protocolError("decode response", err)
	}
	return nil
}

func (c *Client) resolve(path string) (*url.URL, error) {
	u, err := url.Parse(path)
	if err != nil || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return nil, fmt.Errorf("bam: invalid resource URL")
	}
	if u.IsAbs() || u.Host != "" {
		if u.Scheme != c.baseURL.Scheme || !strings.EqualFold(u.Host, c.baseURL.Host) {
			return nil, fmt.Errorf("bam: resource URL must use the configured BAM origin")
		}
		if u.Path != apiPath && !strings.HasPrefix(u.Path, apiPath+"/") {
			return nil, fmt.Errorf("bam: resource URL must remain under /api/v2")
		}
	} else {
		u.Scheme, u.Host = c.baseURL.Scheme, c.baseURL.Host
		if u.Path != apiPath && !strings.HasPrefix(u.Path, apiPath+"/") {
			u.Path = apiPath + "/" + strings.TrimPrefix(u.Path, "/")
		}
	}
	if strings.Contains(u.Path, "\\") || strings.Contains(u.Path, "//") || u.RawPath != "" {
		return nil, fmt.Errorf("bam: resource URL contains an ambiguous path")
	}
	for segment := range strings.SplitSeq(u.Path, "/") {
		if segment == "." || segment == ".." {
			return nil, fmt.Errorf("bam: resource URL contains a traversal segment")
		}
	}
	return u, nil
}

func (c *Client) prepareRequest(ctx context.Context, request Request) (*http.Request, error) {
	u, err := c.resolve(request.Path)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	for key, values := range request.Query {
		query[key] = append([]string(nil), values...)
	}
	u.RawQuery = query.Encode()
	var body io.Reader
	if request.Body != nil {
		encoded, err := json.Marshal(request.Body)
		if err != nil {
			return nil, fmt.Errorf("bam: encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, request.Method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("bam: create request: %w", err)
	}
	req.Header = request.Header.Clone()
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	for key := range req.Header {
		switch strings.ToLower(key) {
		case "authorization", "cookie", "host", "content-length", "transfer-encoding", "accept", "content-type":
			delete(req.Header, key)
		}
	}
	req.Header.Set("Accept", "application/hal+json")
	if body != nil {
		contentType := "application/json"
		if req.Method == http.MethodPatch {
			contentType = "application/merge-patch+json"
		}
		req.Header.Set("Content-Type", contentType)
	}
	if auth := c.credentials(); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	return req, nil
}
