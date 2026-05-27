package netbox

import (
	"context"
	"errors"
	"fmt"
)

// PageFetcher is the shape any list endpoint exposes for iteration: given
// offset+limit, return one page of T. Typed methods (ListSites, ListTenants,
// ...) build closures of this type when they want auto-pagination.
type PageFetcher[T any] func(ctx context.Context, offset, limit int) (Page[T], error)

// IterateOptions tunes pagination. Zero values pick reasonable defaults.
type IterateOptions struct {
	// PageSize is the per-request limit. Defaults to 50 when zero.
	PageSize int

	// MaxPages caps the number of pages fetched as a safety belt for huge
	// datasets. Zero means no cap — useful with a ctx.WithTimeout instead.
	MaxPages int
}

// ErrMaxPages signals that MaxPages was hit before the result set was exhausted.
// The caller still gets the partial result; this lets `nbcli show ... limit 0`
// distinguish "truncated" from "complete".
var ErrMaxPages = errors.New("netbox: max pages reached before exhausting result set")

// ListAll walks fetch until the Next link is nil (or MaxPages hits) and
// returns the concatenated results. Pagination is offset/limit per Netbox v2.
//
// Use this from commands that handle `limit 0` (or some other "give me all"
// sentinel). For interactive views, prefer Iterate so the UI can render rows
// as they arrive instead of blocking on the full walk.
func ListAll[T any](ctx context.Context, fetch PageFetcher[T], opts IterateOptions) ([]T, error) {
	size := opts.PageSize
	if size <= 0 {
		size = 50
	}
	var out []T
	offset, pages := 0, 0
	for {
		if err := ctx.Err(); err != nil {
			return out, fmt.Errorf("paginate: %w", err)
		}
		page, err := fetch(ctx, offset, size)
		if err != nil {
			return out, err
		}
		out = append(out, page.Results...)
		pages++

		if page.Next == nil || len(page.Results) == 0 {
			return out, nil
		}
		offset += len(page.Results)
		if page.Count > 0 && offset >= page.Count {
			return out, nil
		}
		if opts.MaxPages > 0 && pages >= opts.MaxPages {
			return out, ErrMaxPages
		}
	}
}

// Iterate is the streaming variant: fn is called once per row as pages arrive,
// so a UI (or a streaming JSON writer) can render incrementally. Returning a
// non-nil error from fn stops iteration with that error.
func Iterate[T any](ctx context.Context, fetch PageFetcher[T], opts IterateOptions, fn func(T) error) error {
	size := opts.PageSize
	if size <= 0 {
		size = 50
	}
	offset, pages := 0, 0
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("paginate: %w", err)
		}
		page, err := fetch(ctx, offset, size)
		if err != nil {
			return err
		}
		for _, row := range page.Results {
			if err := fn(row); err != nil {
				return err
			}
		}
		pages++
		if page.Next == nil || len(page.Results) == 0 {
			return nil
		}
		offset += len(page.Results)
		if page.Count > 0 && offset >= page.Count {
			return nil
		}
		if opts.MaxPages > 0 && pages >= opts.MaxPages {
			return ErrMaxPages
		}
	}
}
