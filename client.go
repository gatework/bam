package bam

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const apiPath = "/api/v2"

// Client owns an independent authentication session. It is safe for concurrent
// requests. Login and Logout serialize session changes; in-flight requests may
// still be using the previous credentials when Logout completes.
type Client struct {
	baseURL       *url.URL
	http          *http.Client
	responseLimit int64
	authMu        sync.RWMutex
	authorization string
	sessionGate   chan struct{}
}

type settings struct {
	http          *http.Client
	timeout       time.Duration
	responseLimit int64
	authorization string
}

// Option configures a Client before its first use.
type Option func(*settings) error

// New accepts a BAM origin or its /api/v2 URL. HTTPS is assumed when the scheme
// is omitted. The default request timeout is 30 seconds and response limit is
// 16 MiB. Redirects are rejected, including redirects to another BAM host.
func New(endpoint string, options ...Option) (*Client, error) {
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("bam: invalid endpoint")
	}
	if (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" ||
		u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" ||
		(u.Path != "" && u.Path != "/" && u.Path != apiPath && u.Path != apiPath+"/") {
		return nil, fmt.Errorf("bam: endpoint must be an HTTP(S) origin or /api/v2 URL without credentials, query, or fragment")
	}
	u.Path, u.RawPath = apiPath, ""
	s := settings{timeout: 30 * time.Second, responseLimit: 16 << 20}
	for _, option := range options {
		if option == nil {
			return nil, fmt.Errorf("bam: nil client option")
		}
		if err := option(&s); err != nil {
			return nil, err
		}
	}
	hc := http.Client{Transport: http.DefaultTransport}
	if s.http != nil {
		hc = *s.http
	}
	hc.Timeout = s.timeout
	hc.Jar = nil // RESTful v2 authenticates with tokens, never shared cookies.
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{
		baseURL: u, http: &hc, responseLimit: s.responseLimit,
		authorization: s.authorization, sessionGate: make(chan struct{}, 1),
	}, nil
}

// WithHTTPClient copies the supplied client without modifying it. Its transport
// must be safe for concurrent use. Gobam owns the timeout, redirect and cookie
// policies; use WithTimeout to change the timeout.
func WithHTTPClient(client *http.Client) Option {
	return func(s *settings) error {
		if client == nil {
			return fmt.Errorf("bam: nil HTTP client")
		}
		s.http = client
		return nil
	}
}

// WithTimeout sets the per-request timeout. The caller's context can impose a
// shorter deadline. A positive duration is required.
func WithTimeout(timeout time.Duration) Option {
	return func(s *settings) error {
		if timeout <= 0 {
			return fmt.Errorf("bam: timeout must be positive")
		}
		s.timeout = timeout
		return nil
	}
}

// WithResponseLimit bounds the bytes read from each response, including errors.
func WithResponseLimit(bytes int64) Option {
	return func(s *settings) error {
		if bytes <= 0 || bytes == 1<<63-1 {
			return fmt.Errorf("bam: response limit must be positive and less than MaxInt64")
		}
		s.responseLimit = bytes
		return nil
	}
}

// WithBasicAuth uses an existing RESTful v2 API token, not a password or v1 token.
func WithBasicAuth(username, apiToken string) Option {
	return func(s *settings) error {
		if username == "" || strings.Contains(username, ":") || apiToken == "" {
			return fmt.Errorf("bam: Basic authentication requires a username without ':' and a nonempty API token")
		}
		return setAuthorization(s, "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+apiToken)))
	}
}

// WithBearerToken uses an externally managed OAuth access token.
func WithBearerToken(token string) Option {
	return func(s *settings) error {
		if token == "" || strings.ContainsAny(token, " \t\r\n") {
			return fmt.Errorf("bam: invalid bearer token")
		}
		return setAuthorization(s, "Bearer "+token)
	}
}

func setAuthorization(s *settings, value string) error {
	if s.authorization != "" {
		return fmt.Errorf("bam: only one authentication option is allowed")
	}
	s.authorization = value
	return nil
}

// CloseIdleConnections closes idle transport connections; it does not log out.
func (c *Client) CloseIdleConnections() { c.http.CloseIdleConnections() }

func (c *Client) credentials() string {
	c.authMu.RLock()
	defer c.authMu.RUnlock()
	return c.authorization
}
