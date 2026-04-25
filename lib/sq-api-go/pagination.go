package sqapi

import (
	"context"
	"math"
)

// PageSize is the default and maximum page size used in paginated requests.
// SonarQube Server accepts up to 500 results per page.
const PageSize = 500

// FetchPageFunc is the function signature for fetching a single page of results.
// page is 1-indexed. Returns the items on that page, the total item count
// across all pages, and any error. A non-nil error halts iteration.
type FetchPageFunc[T any] func(ctx context.Context, page, pageSize int) (items []T, total int, err error)

// Paginator iterates over all pages of a paginated SonarQube API response.
//
// Type parameter T is the item type returned by the endpoint, e.g. Project.
//
// Usage:
//
//	pag := sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]Project, int, error) {
//	    return client.fetchProjectsPage(ctx, page, pageSize, params)
//	}, 0)
//
//	all, err := pag.All(ctx)
//
// Or iterate page-by-page:
//
//	for items, err := range pag.Pages(ctx) {
//	    if err != nil { return err }
//	    for _, item := range items { ... }
//	}
type Paginator[T any] struct {
	fetch    FetchPageFunc[T]
	pageSize int
}

// NewPaginator creates a Paginator that calls fetch for each page.
// Pass pageSize=0 to use the default (500).
func NewPaginator[T any](fetch FetchPageFunc[T], pageSize int) *Paginator[T] {
	if pageSize <= 0 {
		pageSize = PageSize
	}
	return &Paginator[T]{fetch: fetch, pageSize: pageSize}
}

// All fetches every page and returns all items concatenated into a single slice.
func (p *Paginator[T]) All(ctx context.Context) ([]T, error) {
	var all []T
	for items, err := range p.Pages(ctx) {
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
	}
	return all, nil
}

// Pages returns a range-over-function iterator that yields one page of items
// per iteration. Fetches page 1 first to determine the total count, then
// fetches remaining pages in order.
//
// Example:
//
//	for items, err := range pag.Pages(ctx) {
//	    if err != nil { return err }
//	    process(items)
//	}
func (p *Paginator[T]) Pages(ctx context.Context) func(yield func([]T, error) bool) {
	return func(yield func([]T, error) bool) {
		items, total, err := p.fetch(ctx, 1, p.pageSize)
		if err != nil {
			yield(nil, err)
			return
		}

		if !yield(items, nil) {
			return
		}

		pages := totalPages(total, p.pageSize)
		for page := 2; page <= pages; page++ {
			if err := ctx.Err(); err != nil {
				yield(nil, err)
				return
			}

			items, _, err := p.fetch(ctx, page, p.pageSize)
			if err != nil {
				yield(nil, err)
				return
			}

			if !yield(items, nil) {
				return
			}
		}
	}
}

// totalPages computes the number of pages needed for total items at pageSize per page.
func totalPages(total, pageSize int) int {
	if total == 0 || pageSize <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(pageSize)))
}
