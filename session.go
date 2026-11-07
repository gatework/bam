package bam

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Session contains metadata only; passwords and API tokens are not retained in
// the returned value and cannot accidentally appear when it is logged.
type Session struct {
	ID        int64
	ExpiresAt time.Time
}

// Login creates and installs a RESTful v2 UserSession. An already authenticated
// client must Logout first, so repeated Login cannot silently abandon a session.
// A failed Login leaves the client unauthenticated.
func (c *Client) Login(ctx context.Context, username, password string) (Session, error) {
	if err := c.lockSession(ctx); err != nil {
		return Session{}, err
	}
	defer c.unlockSession()
	if c.credentials() != "" {
		return Session{}, fmt.Errorf("bam: client is already authenticated; log out before logging in again")
	}
	if username == "" || strings.Contains(username, ":") || password == "" {
		return Session{}, fmt.Errorf("bam: login requires a username without ':' and a nonempty password")
	}
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{username, password}
	var response struct {
		Resource
		Credentials string    `json:"basicAuthenticationCredentials"`
		ExpiresAt   time.Time `json:"apiTokenExpirationDateTime"`
	}
	if err := c.Do(ctx, Request{Method: http.MethodPost, Path: "sessions", Body: body}, &response); err != nil {
		return Session{}, err
	}
	decoded, err := base64.StdEncoding.DecodeString(response.Credentials)
	user, token, ok := strings.Cut(string(decoded), ":")
	if err != nil || !ok || user == "" || token == "" || response.ID <= 0 || response.Type != "UserSession" {
		return Session{}, &ProtocolError{Method: http.MethodPost, Path: apiPath + "/sessions", Detail: "invalid UserSession credentials or identity"}
	}
	c.authMu.Lock()
	c.authorization = "Basic " + response.Credentials
	c.authMu.Unlock()
	return Session{ID: response.ID, ExpiresAt: response.ExpiresAt}, nil
}

// Logout invalidates the current BAM session via a merge patch. It is a no-op
// when unauthenticated. Credentials are cleared on success or HTTP 401, and
// retained on other failures so the caller can retry logout. For externally
// managed OAuth tokens, revocation remains the authorization server's concern.
func (c *Client) Logout(ctx context.Context) error {
	if err := c.lockSession(ctx); err != nil {
		return err
	}
	defer c.unlockSession()
	if c.credentials() == "" {
		return nil
	}
	err := c.Do(ctx, Request{
		Method: http.MethodPatch,
		Path:   "sessions/current",
		Body: struct {
			State string `json:"state"`
		}{"LOGGED_OUT"},
	}, nil)
	var apiErr *APIError
	if err == nil || (errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized) {
		c.authMu.Lock()
		c.authorization = ""
		c.authMu.Unlock()
	}
	return err
}

func (c *Client) lockSession(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.sessionGate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			c.unlockSession()
			return err
		}
		return nil
	}
}

func (c *Client) unlockSession() { <-c.sessionGate }
