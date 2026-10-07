package qbitflow

import "iter"

// Page is one page of a cursor-paginated list.
type Page[T any] struct {
	// Items are the page's rows.
	Items []T `json:"items"`
	// NextCursor is the cursor of the next page (pass it back as the params' Cursor, verbatim);
	// nil on the last page.
	NextCursor *string `json:"nextCursor"`
}

// HasMore reports whether another page follows.
func (p *Page[T]) HasMore() bool {
	return p != nil && p.NextCursor != nil
}

// iteratePages walks a cursor-paginated list lazily, one request per page, from cursor ("" =
// the first page). fetch requests one page with the given cursor, keeping the caller's filters
// and limit. The sequence stops when the caller stops, on the last page (nextCursor null),
// on an empty page, or on an error, which it yields as (zero, err) before stopping.
//
// The Iterate methods are built on it:
//
//	func (s *CustomersService) Iterate(ctx context.Context, params *CustomerListParams, opts ...RequestOption) iter.Seq2[Customer, error] {
//		p := derefOrZero(params)
//		return iteratePages(p.Cursor, func(cursor string) (*Page[Customer], error) {
//			p.Cursor = cursor
//			return s.List(ctx, &p, opts...)
//		})
//	}
func iteratePages[T any](cursor string, fetch func(cursor string) (*Page[T], error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for {
			page, err := fetch(cursor)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			if page == nil {
				return
			}
			for _, item := range page.Items {
				if !yield(item, nil) {
					return
				}
			}
			// The last page, an empty page, or a cursor that does not move (which would loop
			// on the same page forever) ends the walk.
			if page.NextCursor == nil || len(page.Items) == 0 || *page.NextCursor == cursor {
				return
			}
			cursor = *page.NextCursor
		}
	}
}

// derefOrZero returns *p, or T's zero value for a nil p (optional params).
func derefOrZero[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
