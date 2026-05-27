package netbox

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeFetcher returns canned pages of ints. pages[i] is delivered on call i.
func fakeFetcher(pages []Page[int]) PageFetcher[int] {
	call := 0
	return func(_ context.Context, _, _ int) (Page[int], error) {
		if call >= len(pages) {
			return Page[int]{}, errors.New("over-fetched")
		}
		p := pages[call]
		call++
		return p, nil
	}
}

func ptr(s string) *string { return &s }

func TestListAll_StopsOnNilNext(t *testing.T) {
	t.Parallel()
	pages := []Page[int]{
		{Count: 5, Next: ptr("p2"), Results: []int{1, 2, 3}},
		{Count: 5, Next: nil, Results: []int{4, 5}},
	}
	out, err := ListAll(context.Background(), fakeFetcher(pages), IterateOptions{PageSize: 3})
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, out)
}

func TestListAll_StopsOnCountReached(t *testing.T) {
	t.Parallel()
	// Next is non-nil but we've already collected Count items → stop anyway.
	pages := []Page[int]{
		{Count: 3, Next: ptr("p2"), Results: []int{1, 2, 3}},
	}
	out, err := ListAll(context.Background(), fakeFetcher(pages), IterateOptions{PageSize: 3})
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, out)
}

func TestListAll_RespectsMaxPages(t *testing.T) {
	t.Parallel()
	pages := []Page[int]{
		{Count: 99, Next: ptr("p2"), Results: []int{1}},
		{Count: 99, Next: ptr("p3"), Results: []int{2}},
		{Count: 99, Next: ptr("p4"), Results: []int{3}},
	}
	out, err := ListAll(context.Background(), fakeFetcher(pages), IterateOptions{PageSize: 1, MaxPages: 2})
	require.ErrorIs(t, err, ErrMaxPages)
	assert.Equal(t, []int{1, 2}, out, "partial result returned")
}

func TestListAll_PropagatesFetchError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("boom")
	fetch := func(_ context.Context, _, _ int) (Page[int], error) {
		return Page[int]{}, sentinel
	}
	_, err := ListAll(context.Background(), fetch, IterateOptions{})
	require.ErrorIs(t, err, sentinel)
}

func TestListAll_CancelCtx(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ListAll(ctx, fakeFetcher(nil), IterateOptions{})
	require.Error(t, err)
}

func TestIterate_StreamsRowsThenStops(t *testing.T) {
	t.Parallel()
	pages := []Page[int]{
		{Count: 4, Next: ptr("p2"), Results: []int{10, 20}},
		{Count: 4, Next: nil, Results: []int{30, 40}},
	}
	var got []int
	err := Iterate(context.Background(), fakeFetcher(pages), IterateOptions{PageSize: 2}, func(n int) error {
		got = append(got, n)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []int{10, 20, 30, 40}, got)
}

func TestIterate_StopsOnFnError(t *testing.T) {
	t.Parallel()
	pages := []Page[int]{
		{Count: 10, Next: ptr("p2"), Results: []int{1, 2, 3}},
	}
	stop := errors.New("stop")
	var seen int
	err := Iterate(context.Background(), fakeFetcher(pages), IterateOptions{PageSize: 3}, func(n int) error {
		seen = n
		if n == 2 {
			return stop
		}
		return nil
	})
	require.ErrorIs(t, err, stop)
	assert.Equal(t, 2, seen)
}

// Compile-time check that PageFetcher composes with method-call sites the way
// the show commands will compose it.
func ExampleListAll_compose() {
	var c *Client // pretend wired up
	_ = c
	fetch := PageFetcher[Site](func(ctx context.Context, offset, limit int) (Page[Site], error) {
		opts := ListSitesOptions{Offset: offset, Limit: limit}
		return c.ListSites(ctx, opts)
	})
	_ = fetch
	fmt.Println("ok")
	// Output: ok
}
