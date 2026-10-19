package qbitflow

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
)

// pagedServer serves /customers in pages of 2 over 5 customers (c0…c4), keyed by an opaque
// cursor; the last page answers hasMore false (a full last page never adds an empty one).
func pagedServer(t *testing.T, failOn string) *testServer {
	return newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		cursor := r.URL.Query().Get("cursor")
		if cursor == failOn && failOn != "" {
			jsonReply(w, 400, `{"error":"bad cursor","code":"validation_failed","details":{"errors":[{"field":"cursor","message":"cursor is unknown"}]}}`)
			return
		}
		pages := map[string]string{
			"":        `{"items":[{"uuid":"c0"},{"uuid":"c1"}],"hasMore":true,"nextCursor":"djF8YzE"}`,
			"djF8YzE": `{"items":[{"uuid":"c2"},{"uuid":"c3"}],"hasMore":true,"nextCursor":"djF8YzM"}`,
			"djF8YzM": `{"items":[{"uuid":"c4"}],"hasMore":false}`,
		}
		jsonReply(w, 200, pages[cursor])
	})
}

// iterateCustomers walks /customers with Customers.Iterate.
func iterateCustomers(c *Client, ctx context.Context, params *CustomerListParams) func(func(Customer, error) bool) {
	return c.Customers.Iterate(ctx, params)
}

func TestIterateAllPages(t *testing.T) {
	ts := pagedServer(t, "")
	c, _ := newTestClient(t, ts)

	var got []string
	for cust, err := range iterateCustomers(c, context.Background(), &CustomerListParams{Limit: 2, Email: "a@b.co"}) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, cust.UUID)
	}
	if want := []string{"c0", "c1", "c2", "c3", "c4"}; !slices.Equal(got, want) {
		t.Errorf("items %v", got)
	}
	reqs := ts.recorded()
	if len(reqs) != 3 {
		t.Fatalf("requests %d", len(reqs))
	}
	for i, wantQuery := range []string{"email=a%40b.co&limit=2", "cursor=djF8YzE&email=a%40b.co&limit=2", "cursor=djF8YzM&email=a%40b.co&limit=2"} {
		if reqs[i].query != wantQuery {
			t.Errorf("page %d query %q, want %q (filters and limit kept)", i, reqs[i].query, wantQuery)
		}
	}
}

func TestIterateFromCursorAndNil(t *testing.T) {
	ts := pagedServer(t, "")
	c, _ := newTestClient(t, ts)
	var got []string
	for cust, err := range iterateCustomers(c, context.Background(), &CustomerListParams{Cursor: "djF8YzM"}) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, cust.UUID)
	}
	if !slices.Equal(got, []string{"c4"}) || len(ts.recorded()) != 1 {
		t.Errorf("from the last cursor: %v, %d requests", got, len(ts.recorded()))
	}

	n := 0
	for _, err := range iterateCustomers(c, context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 5 {
		t.Errorf("nil params: %d items", n)
	}
}

func TestIterateEarlyBreak(t *testing.T) {
	ts := pagedServer(t, "")
	c, _ := newTestClient(t, ts)
	var got []string
	for cust, err := range iterateCustomers(c, context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, cust.UUID)
		if cust.UUID == "c2" {
			break
		}
	}
	if !slices.Equal(got, []string{"c0", "c1", "c2"}) {
		t.Errorf("items %v", got)
	}
	if len(ts.recorded()) != 2 {
		t.Errorf("lazy: %d requests, want 2", len(ts.recorded()))
	}
}

func TestIterateErrorPropagation(t *testing.T) {
	ts := pagedServer(t, "djF8YzM")
	c, _ := newTestClient(t, ts)
	var got []string
	var errs []error
	for cust, err := range iterateCustomers(c, context.Background(), nil) {
		if err != nil {
			errs = append(errs, err)
			if cust.UUID != "" {
				t.Error("the error comes with a zero item")
			}
			continue
		}
		got = append(got, cust.UUID)
	}
	if !slices.Equal(got, []string{"c0", "c1", "c2", "c3"}) {
		t.Errorf("items %v", got)
	}
	var ve *ValidationError
	if len(errs) != 1 || !errors.As(errs[0], &ve) || ve.FieldErrors[0].Field != "cursor" {
		t.Errorf("errors %v", errs)
	}
}

// The walk stops on hasMore false; the defensive guards also stop a page claiming more without a
// cursor, with the cursor just requested, or with no rows (the API never answers an empty page
// that has more: a server fault that could otherwise cycle between cursors forever).
func TestIterateStopRules(t *testing.T) {
	calls := 0
	seq := iteratePages("", func(cursor string) (*Page[int], error) {
		calls++
		switch calls {
		case 1:
			return &Page[int]{Items: []int{1}, HasMore: true, NextCursor: "a"}, nil
		case 2:
			return &Page[int]{Items: []int{2}, HasMore: true, NextCursor: "b"}, nil
		case 3:
			return &Page[int]{Items: []int{3}, HasMore: false}, nil
		}
		t.Fatal("requested a page past hasMore false")
		return nil, nil
	})
	var got []int
	for v, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, v)
	}
	if !slices.Equal(got, []int{1, 2, 3}) || calls != 3 {
		t.Errorf("walk: %v after %d calls", got, calls)
	}

	// hasMore false stops even with a cursor present (the API never sends one then).
	calls = 0
	for range iteratePages("", func(string) (*Page[int], error) {
		calls++
		return &Page[int]{Items: []int{1}, HasMore: false, NextCursor: "x"}, nil
	}) {
	}
	if calls != 1 {
		t.Errorf("hasMore false must stop the walk (%d calls)", calls)
	}

	// hasMore true without a nextCursor: defensive stop.
	calls = 0
	for range iteratePages("", func(string) (*Page[int], error) {
		calls++
		return &Page[int]{Items: []int{1}, HasMore: true}, nil
	}) {
	}
	if calls != 1 {
		t.Errorf("hasMore without a cursor must stop the walk (%d calls)", calls)
	}

	// An empty page that claims more, with cursors that keep changing (a <-> b): defensive stop.
	calls = 0
	for range iteratePages("", func(string) (*Page[int], error) {
		calls++
		return &Page[int]{HasMore: true, NextCursor: []string{"a", "b"}[calls%2]}, nil
	}) {
	}
	if calls != 1 {
		t.Errorf("an empty page that has more must stop the walk (%d calls)", calls)
	}

	// A cursor that does not move: defensive stop.
	calls = 0
	for range iteratePages("same", func(string) (*Page[int], error) {
		calls++
		return &Page[int]{Items: []int{1}, HasMore: true, NextCursor: "same"}, nil
	}) {
	}
	if calls != 1 {
		t.Errorf("a cursor that does not move must stop the walk (%d calls)", calls)
	}
}

func TestPageDecoding(t *testing.T) {
	p, err := decodeResponse[Page[Customer]](&response{status: 200, body: []byte(`{"items":[{"uuid":"c0"}],"hasMore":true,"nextCursor":"djF8eA"}`)})
	if err != nil || !p.HasMore || p.NextCursor != "djF8eA" || len(p.Items) != 1 {
		t.Errorf("page: %+v %v", p, err)
	}
	p, err = decodeResponse[Page[Customer]](&response{status: 200, body: []byte(`{"items":[],"hasMore":false}`)})
	if err != nil || p.HasMore || p.NextCursor != "" || p.Items == nil || len(p.Items) != 0 {
		t.Errorf("last page: %+v %v", p, err)
	}
	// Lenient on absence and null: zero values.
	p, err = decodeResponse[Page[Customer]](&response{status: 200, body: []byte(`{"items":null}`)})
	if err != nil || p.Items != nil || p.HasMore {
		t.Errorf("null page: %+v %v", p, err)
	}
	// Strict on type.
	for _, body := range []string{`{"items":[],"hasMore":"true"}`, `{"items":{},"hasMore":false}`, `{"items":[],"nextCursor":5}`, `[]`} {
		if _, err := decodeResponse[Page[Customer]](&response{status: 200, body: []byte(body)}); !errors.As(err, new(*ServerError)) {
			t.Errorf("%s: want *ServerError, got %v", body, err)
		}
	}
}

// Bounded lists (one page, hasMore false) are returned as their items.
func TestBoundedListsUnwrapItems(t *testing.T) {
	ts := newTestServer(t, staticReply(200, `{"items":[{"id":8,"code":"BASE:USDC"},{"id":1,"code":"ETH"}],"hasMore":false}`))
	c, _ := newTestClient(t, ts)
	list, err := c.Currencies.List(context.Background(), &CurrencyListParams{Main: true})
	if err != nil || len(list) != 2 || list[1].Code != "ETH" {
		t.Fatalf("currencies %+v %v", list, err)
	}
	empty := newTestServer(t, staticReply(200, `{"items":[],"hasMore":false}`))
	c2, _ := newTestClient(t, empty)
	wallets, err := c2.Wallets.List(context.Background(), nil)
	if err != nil || wallets == nil || len(wallets) != 0 {
		t.Errorf("empty bounded list: %#v %v", wallets, err)
	}
}
