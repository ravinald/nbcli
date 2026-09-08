package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/ravinald/nbcli/internal/netbox"
)

// PassthroughResult is the decoded JSON from a generic plugin call. We keep
// it as a generic map / any so the user can render it with --format json/yaml
// without us needing a typed schema per plugin.
type PassthroughResult struct {
	// StatusCode is the raw HTTP status from Netbox.
	StatusCode int

	// Body is the decoded JSON (object, array, scalar, or nil for empty body).
	Body any
}

// Passthrough calls /api/plugins/<plugin>/<subpath> on Netbox.
// subpath should NOT include the leading "/api/plugins/<plugin>/" — we add it.
// extraQuery is merged into the URL.
//
// The decoded body is returned in PassthroughResult.Body as a generic structure
// (map[string]any / []any / scalar), so the caller can pipe it to the JSON or
// YAML renderer without knowing the plugin's schema.
func Passthrough(ctx context.Context, c *netbox.Client, plugin, method, subpath string, extraQuery url.Values, body io.Reader) (PassthroughResult, error) {
	if plugin == "" {
		return PassthroughResult{}, errors.New("plugins: plugin name is required")
	}
	plugin = strings.Trim(plugin, "/")
	subpath = strings.TrimLeft(subpath, "/")
	path := fmt.Sprintf("/api/plugins/%s/%s", plugin, subpath)

	var bodyAny any
	if body != nil {
		raw, err := io.ReadAll(body)
		if err != nil {
			return PassthroughResult{}, fmt.Errorf("plugins: read body: %w", err)
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &bodyAny); err != nil {
				return PassthroughResult{}, fmt.Errorf("plugins: body must be JSON: %w", err)
			}
		}
	}

	var out any
	if err := c.Do(ctx, method, path, extraQuery, bodyAny, &out); err != nil {
		return PassthroughResult{}, err
	}
	return PassthroughResult{StatusCode: 200, Body: out}, nil
}
