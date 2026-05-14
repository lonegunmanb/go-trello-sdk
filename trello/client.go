// Package trello provides a Go SDK for the Trello REST API.
//
// The bulk of this package — both the data models and the HTTP client — is
// generated from the official Trello OpenAPI 3.0 specification using
// [oapi-codegen]. This file contains hand-written helpers that wire up the
// API key/token query-parameter authentication scheme that Trello uses.
//
// Typical usage:
//
//	c, err := trello.New(trello.WithCredentials(apiKey, apiToken))
//	if err != nil { ... }
//	resp, err := c.GetMembersIdWithResponse(ctx, "me", &trello.GetMembersIdParams{})
//
// [oapi-codegen]: https://github.com/oapi-codegen/oapi-codegen
package trello

import (
	"context"
	"errors"
	"net/http"
)

// DefaultServer is the production Trello REST API base URL.
const DefaultServer = "https://api.trello.com/1"

// Option configures the [ClientWithResponses] built by [New].
type Option func(*config) error

type config struct {
	server         string
	apiKey         string
	apiToken       string
	httpClient     HttpRequestDoer
	requestEditors []RequestEditorFn
}

// WithServer overrides the base URL of the Trello API. Defaults to
// [DefaultServer].
func WithServer(server string) Option {
	return func(c *config) error {
		if server == "" {
			return errors.New("trello: server must not be empty")
		}
		c.server = server
		return nil
	}
}

// WithCredentials sets the API key and token used to authenticate requests.
// Trello expects both to be passed as ``key`` and ``token`` query parameters
// on every request; this option installs a [RequestEditorFn] that does so.
func WithCredentials(apiKey, apiToken string) Option {
	return func(c *config) error {
		c.apiKey = apiKey
		c.apiToken = apiToken
		return nil
	}
}

// WithHTTPDoer overrides the underlying HTTP client used to issue requests.
// The default is [http.DefaultClient].
func WithHTTPDoer(doer HttpRequestDoer) Option {
	return func(c *config) error {
		if doer == nil {
			return errors.New("trello: http doer must not be nil")
		}
		c.httpClient = doer
		return nil
	}
}

// WithRequestEditor appends a [RequestEditorFn] that is invoked for every
// request issued by the client. Editors run after the credentials editor
// installed by [WithCredentials].
func WithRequestEditor(fn RequestEditorFn) Option {
	return func(c *config) error {
		if fn == nil {
			return errors.New("trello: request editor must not be nil")
		}
		c.requestEditors = append(c.requestEditors, fn)
		return nil
	}
}

// New builds a high-level [ClientWithResponses] backed by the generated
// client. It applies all of the supplied [Option]s and wires Trello's
// API-key/token authentication.
func New(opts ...Option) (*ClientWithResponses, error) {
	cfg := &config{server: DefaultServer}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	clientOpts := []ClientOption{
		WithRequestEditorFn(credentialsEditor(cfg.apiKey, cfg.apiToken)),
	}
	if cfg.httpClient != nil {
		clientOpts = append(clientOpts, WithHTTPClient(cfg.httpClient))
	}
	for _, ed := range cfg.requestEditors {
		clientOpts = append(clientOpts, WithRequestEditorFn(ed))
	}
	return NewClientWithResponses(cfg.server, clientOpts...)
}

// credentialsEditor returns a [RequestEditorFn] that appends Trello's ``key``
// and ``token`` query parameters when they are set. When both are empty the
// editor is a no-op so the client can still be used against test servers.
func credentialsEditor(apiKey, apiToken string) RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		if apiKey == "" && apiToken == "" {
			return nil
		}
		q := req.URL.Query()
		if apiKey != "" {
			q.Set("key", apiKey)
		}
		if apiToken != "" {
			q.Set("token", apiToken)
		}
		req.URL.RawQuery = q.Encode()
		return nil
	}
}
