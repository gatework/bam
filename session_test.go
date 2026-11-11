package bam_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gatework/bam"
)

func TestSessionsAreIsolatedAndDoNotUseCookies(t *testing.T) {
	var cookieSeen atomic.Bool
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" {
			cookieSeen.Store(true)
		}
		switch r.URL.Path {
		case "/api/v2/sessions":
			if r.Header.Get("Authorization") != "" {
				t.Error("login used preexisting credentials")
			}
			var body struct{ Username, Password string }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Password != "password" {
				t.Error("invalid login payload")
			}
			http.SetCookie(w, &http.Cookie{Name: "legacy", Value: "must-not-be-replayed"})
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"id":1,"type":"UserSession","basicAuthenticationCredentials":%q,"apiTokenExpirationDateTime":"2026-11-08T00:00:00Z"}`, base64.StdEncoding.EncodeToString([]byte(body.Username+":api-token")))
		case "/api/v2/sessions/current":
			var patch struct{ State string }
			if r.Method != http.MethodPatch || r.Header.Get("Content-Type") != "application/merge-patch+json" || json.NewDecoder(r.Body).Decode(&patch) != nil || patch.State != "LOGGED_OUT" {
				t.Error("logout did not use the v2 session transition")
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			user, token, ok := r.BasicAuth()
			if !ok || token != "api-token" || user != r.URL.Query().Get("expectedUser") {
				t.Error("request used another client's session")
			}
			_, _ = fmt.Fprint(w, `{"id":1,"type":"Configuration"}`)
		}
	}
	clients := []*bam.Client{testClient(t, handler), testClient(t, handler)}
	var wg sync.WaitGroup
	for i, client := range clients {
		wg.Go(func() {
			username := fmt.Sprintf("user%d", i)
			session, err := client.Login(context.Background(), username, "password")
			if err != nil || session.ID != 1 || session.ExpiresAt.IsZero() {
				t.Errorf("Login = %+v, %v", session, err)
				return
			}
			if strings.Contains(fmt.Sprint(session), "api-token") {
				t.Error("session metadata exposes token")
			}
			if _, err := client.Login(context.Background(), username, "password"); err == nil {
				t.Error("repeated Login silently abandoned the old session")
			}
			var reads sync.WaitGroup
			for range 12 {
				reads.Go(func() {
					var result bam.Resource
					err := client.Do(context.Background(), bam.Request{Method: http.MethodGet, Path: "configurations/1?expectedUser=" + username}, &result)
					if err != nil {
						t.Error(err)
					}
				})
			}
			reads.Wait()
			if err := client.Logout(context.Background()); err != nil {
				t.Error(err)
			}
			if err := client.Logout(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if cookieSeen.Load() {
		t.Fatal("legacy cookies escaped the transport")
	}
}

func TestFailedLoginDoesNotInstallCredentials(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("malformed login installed credentials")
		}
		calls.Add(1)
		_, _ = fmt.Fprint(w, `{"id":1,"type":"UserSession","basicAuthenticationCredentials":"not-base64"}`)
	})
	for range 2 {
		if _, err := client.Login(context.Background(), "user", "secret"); err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("invalid login result: %v", err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("failed login prevented a later attempt")
	}
}

func TestLogoutRetainsCredentialsUntilServerAccepts(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); !ok {
			t.Error("retry lost session credentials")
		}
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprint(w, `{"code":"Unavailable"}`)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"code":"InvalidAuthorizationToken"}`)
	}, bam.WithBasicAuth("user", "token"))
	if err := client.Logout(context.Background()); err == nil {
		t.Fatal("unavailable server was reported as a successful logout")
	}
	if err := client.Logout(context.Background()); err == nil {
		t.Fatal("401 must remain inspectable")
	}
	if err := client.Logout(context.Background()); err != nil || calls.Load() != 2 {
		t.Fatal("expired credentials were not cleared")
	}
}

func TestTruncatedUnauthorizedResponseStillClearsCredentials(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"code":`)
	}, bam.WithBasicAuth("user", "token"))
	err := client.Logout(context.Background())
	var apiErr *bam.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("HTTP status or read failure lost: %v", err)
	}
	if err := client.Logout(context.Background()); err != nil || calls.Load() != 1 {
		t.Fatal("401 did not clear expired credentials")
	}
}

func TestQueuedSessionOperationCanBeCanceled(t *testing.T) {
	started := make(chan struct{})
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Error(err)
		}
		close(started)
		<-r.Context().Done()
	})
	loginContext, stopLogin := context.WithCancel(context.Background())
	defer stopLogin()
	loginDone := make(chan error, 1)
	go func() {
		_, err := client.Login(loginContext, "user", "password")
		loginDone <- err
	}()
	<-started
	logoutContext, cancelLogout := context.WithCancel(context.Background())
	logoutDone := make(chan error, 1)
	go func() { logoutDone <- client.Logout(logoutContext) }()
	cancelLogout()
	select {
	case err := <-logoutDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("queued logout returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("canceled logout waited for the active login")
	}
	stopLogin()
	if err := <-loginDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("active login cancellation returned %v", err)
	}
}
