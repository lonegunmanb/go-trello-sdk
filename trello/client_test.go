package trello

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNew_AppendsCredentials verifies that the credentials set via
// WithCredentials are added as ``key`` and ``token`` query parameters on
// every outgoing request.
func TestNew_AppendsCredentials(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := New(
		WithServer(srv.URL),
		WithCredentials("test-key", "test-token"),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp, err := c.GetMembersIdWithResponse(context.Background(), "me", &GetMembersIdParams{})
	if err != nil {
		t.Fatalf("GetMembersIdWithResponse: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode())
	}
	if got == nil {
		t.Fatal("server did not receive request")
	}
	if got.URL.Query().Get("key") != "test-key" {
		t.Errorf("key query param: got %q want %q", got.URL.Query().Get("key"), "test-key")
	}
	if got.URL.Query().Get("token") != "test-token" {
		t.Errorf("token query param: got %q want %q", got.URL.Query().Get("token"), "test-token")
	}
	if !strings.HasPrefix(got.URL.Path, "/members/me") {
		t.Errorf("path: got %q", got.URL.Path)
	}
}

// TestNew_NoCredentialsLeavesQueryAlone verifies that omitting credentials
// does not add any auth query parameters.
func TestNew_NoCredentialsLeavesQueryAlone(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := New(WithServer(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.GetMembersIdWithResponse(context.Background(), "me", &GetMembersIdParams{}); err != nil {
		t.Fatalf("GetMembersIdWithResponse: %v", err)
	}
	if got.URL.Query().Get("key") != "" || got.URL.Query().Get("token") != "" {
		t.Errorf("expected no credentials, got %q", got.URL.RawQuery)
	}
}

// TestNew_RequestEditorRunsAfterCredentials verifies that user-provided
// editors run after the credentials editor and can observe/override its work.
func TestNew_RequestEditorRunsAfterCredentials(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := New(
		WithServer(srv.URL),
		WithCredentials("k", "t"),
		WithRequestEditor(func(_ context.Context, req *http.Request) error {
			req.Header.Set("X-Custom", "yes")
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := c.GetMembersIdWithResponse(context.Background(), "me", &GetMembersIdParams{}); err != nil {
		t.Fatalf("call: %v", err)
	}
	if got.Header.Get("X-Custom") != "yes" {
		t.Errorf("custom header missing")
	}
	if got.URL.Query().Get("key") != "k" {
		t.Errorf("credentials editor not applied")
	}
}

// TestWithServer_RejectsEmpty checks input validation.
func TestWithServer_RejectsEmpty(t *testing.T) {
	if _, err := New(WithServer("")); err == nil {
		t.Fatal("expected error for empty server")
	}
}
