package qbitflow

import "iter"

// Page is one page of a cursor-paginated list, newest first (members by when they joined).
// Bounded lists (currencies, wallets, members' held funds, the accounting export) answer the
// same envelope in one page; their methods return its Items.
type Page[T any] struct {
	// Items are the page's rows; empty when there are none.
	Items []T `json:"items"`
	// HasMore reports whether another page follows: pass NextCursor as the params' Cursor to
	// read it. A full last page answers false: there is never an extra empty page.
	HasMore bool `json:"hasMore"`
	// NextCursor is the next page's cursor, opaque: pass it back verbatim, never build or parse
	// it. "" (absent) when HasMore is false.
	NextCursor string `json:"nextCursor,omitempty"`
}

// iteratePages walks a cursor-paginated list lazily, one request per page, from start ("" =
// the first page), anew on every range of the sequence. fetch requests one page with the given
// cursor, keeping the caller's filters and limit. The sequence stops when the caller stops, on
// the last page (HasMore false), or on an error, which it yields as (zero, err) before stopping.
// Defensively, it also stops on a page claiming more without a NextCursor, on one whose
// NextCursor is the cursor just requested (which would loop on the same page forever), and on an
// empty page that claims more (the API never answers one: a server fault, which could otherwise
// cycle between cursors).
//
// The Iterate methods are built on it:
//
//	func (s *CustomersService) Iterate(ctx context.Context, params *CustomerListParams, opts ...RequestOption) iter.Seq2[Customer, error] {
//		p := derefOrZero(params)
//		return iteratePages(p.Cursor, func(cursor string) (*Page[Customer], error) {
//			page := p // a copy per request: ranges never share params
//			page.Cursor = cursor
//			return s.List(ctx, &page, opts...)
//		})
//	}
func iteratePages[T any](start string, fetch func(cursor string) (*Page[T], error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		// Each range starts from start: ranging the sequence again (or from two goroutines)
		// walks the list again, never from where another range stopped.
		cursor := start
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
			if !page.HasMore || page.NextCursor == "" || page.NextCursor == cursor || len(page.Items) == 0 {
				return
			}
			cursor = page.NextCursor
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
