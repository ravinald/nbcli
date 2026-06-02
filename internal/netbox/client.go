// Package netbox is a small hand-rolled client for the Netbox v2 API.
//
// It targets the endpoints nbcli actually uses. The base Client handles auth,
// JSON encoding, pagination, and context cancellation; resource-specific files
// (dcim.go, ipam.go, ...) build typed methods on top.
//
// Token format follows the project convention "nbt_${KEY}.${TOKEN}".
package netbox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

// UserAgent is sent on every request. Useful for Netbox access logs.
const UserAgent = "nbcli/dev (+https://github.com/ravinald/nbcli)"

// AuthScheme names a Netbox token authorization style.
//
//   - AuthSchemeV2 sends "Authorization: Bearer nbt_KEY.TOKEN" — preferred,
//     used by Netbox tokens created with the v2 hashing scheme.
//   - AuthSchemeV1 sends "Authorization: Token <token>" — legacy, for
//     deployments that still use plaintext-stored tokens.
//
// Default is v2. Override with config `auth_scheme: v1` or `--auth-scheme v1`.
type AuthScheme string

// Supported auth scheme values. Compared case-insensitively in New().
const (
	AuthSchemeV1 AuthScheme = "v1"
	AuthSchemeV2 AuthScheme = "v2"
)

// Client is a thread-safe Netbox API client.
type Client struct {
	baseURL    *url.URL
	authHeader string // pre-built "Bearer <tok>" or "Token <tok>"
	httpClient *http.Client

	// searchUsesREST is set true after the GraphQL endpoint returns 404,
	// indicating the operator disabled GraphQL on this Netbox. Subsequent
	// Search calls skip the GraphQL probe and go straight to REST fan-out.
	searchUsesREST atomic.Bool
}

// Options configures a Client.
type Options struct {
	// BaseURL is the Netbox root, e.g. "https://netbox.example.com".
	// Trailing slash is fine; the client normalizes it.
	BaseURL string

	// Token is the Netbox API token ("nbt_KEY.TOKEN" for v2; raw token for v1).
	Token string

	// AuthScheme selects the Authorization header style. Empty defaults to v2.
	AuthScheme AuthScheme

	// Timeout caps a single request including body read. Zero = 30s default.
	Timeout time.Duration

	// InsecureSkipVerify disables TLS cert verification. Off by default —
	// only enable for known self-signed dev Netbox instances.
	InsecureSkipVerify bool
}

// New constructs a Client. It validates the URL and token shape eagerly so
// failures surface at startup rather than at first request.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, errors.New("netbox: BaseURL is required")
	}
	if opts.Token == "" {
		return nil, errors.New("netbox: Token is required")
	}
	u, err := url.Parse(strings.TrimRight(opts.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("netbox: parse BaseURL %q: %w", opts.BaseURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("netbox: BaseURL must be http(s), got %q", u.Scheme)
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if opts.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in
	}

	authHeader, err := authHeaderFor(opts.AuthScheme, opts.Token)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:    u,
		authHeader: authHeader,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}

// V2Prefix is the wire-format prefix every v2 Netbox token carries
// ("nbt_KEY.TOKEN"). authHeaderFor auto-prepends it when missing so users
// who store the bare KEY in their env file don't have to remember to add it.
const V2Prefix = "nbt_"

// authHeaderFor builds the Authorization header value for the given scheme.
// Empty scheme defaults to v2; comparison is case-insensitive. v2 tokens
// require an "nbt_" prefix; if the supplied token doesn't have one, we add
// it. Unknown scheme values error rather than silently falling through.
func authHeaderFor(scheme AuthScheme, token string) (string, error) {
	switch AuthScheme(strings.ToLower(string(scheme))) {
	case "", AuthSchemeV2:
		if !strings.HasPrefix(token, V2Prefix) {
			token = V2Prefix + token
		}
		return "Bearer " + token, nil
	case AuthSchemeV1:
		return "Token " + token, nil
	default:
		return "", fmt.Errorf("netbox: unknown AuthScheme %q (want %q or %q)",
			scheme, AuthSchemeV1, AuthSchemeV2)
	}
}

// BaseURL returns the configured root URL. Useful for diagnostics and for
// plugin passthrough that wants to log the target.
func (c *Client) BaseURL() string { return c.baseURL.String() }

// Page is a Netbox v2 paginated list response.
type Page[T any] struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []T     `json:"results"`
}

// APIError is returned for non-2xx responses. The body is captured raw so the
// caller can format it. Netbox typically returns JSON {"detail": "..."}.
type APIError struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("netbox: %s %d: %s", e.URL, e.StatusCode, truncate(e.Body, 240))
}

// Do performs an authenticated request. Path is joined to BaseURL; query is
// merged into the URL. Body is JSON-encoded when non-nil. The response body
// is decoded into out when non-nil and the status is 2xx.
//
// Use Do directly for endpoints not covered by typed methods (e.g. plugins).
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("netbox: marshal body: %w", err)
		}
		reqBody = strings.NewReader(string(buf))
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return fmt.Errorf("netbox: build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	slog.DebugContext(ctx, "netbox request", "method", method, "url", u.String())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("netbox: %s %s: %w", method, u.String(), err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("netbox: read body: %w", err)
	}
	slog.DebugContext(ctx, "netbox response",
		"method", method,
		"url", u.String(),
		"status", resp.StatusCode,
		"bytes", len(raw),
		"elapsed", time.Since(start).String(),
	)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, URL: u.String(), Body: string(raw)}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("netbox: decode %s: %w", u.String(), err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
