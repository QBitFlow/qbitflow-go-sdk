package qbitflow

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
)

// pagedServer serves /customer/all in pages of 2 over 5 customers (c0…c4), keyed by cursor.
func pagedServer(t *testing.T, failOn string) *testServer {
	return newTestServer(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		cursor := r.URL.Query().Get("cursor")
		if cursor == failOn && failOn != "" {
			jsonReply(w, 400, `{"error":"bad cursor","code":"validation_failed","details":{"errors":[{"field":"cursor","message":"cursor is unknown"}]}}`)
			return
		}
		pages := map[string]string{
			"":   `{"items":[{"uuid":"c0"},{"uuid":"c1"}],"nextCursor":"c1"}`,
			"c1": `{"items":[{"uuid":"c2"},{"uuid":"c3"}],"nextCursor":"c3"}`,
			"c3": `{"items":[{"uuid":"c4"}],"nextCursor":null}`,
		}
		jsonReply(w, 200, pages[cursor])
	})
}

// iterateCustomers walks /customer/all with Customers.Iterate.
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
	for i, wantQuery := range []string{"email=a%40b.co&limit=2", "cursor=c1&email=a%40b.co&limit=2", "cursor=c3&email=a%40b.co&limit=2"} {
		if reqs[i].query != wantQuery {
			t.Errorf("page %d query %q, want %q (filters and limit kept)", i, reqs[i].query, wantQuery)
		}
	}
}

func TestIterateFromCursorAndNil(t *testing.T) {
	ts := pagedServer(t, "")
	c, _ := newTestClient(t, ts)
	var got []string
	for cust, err := range iterateCustomers(c, context.Background(), &CustomerListParams{Cursor: "c3"}) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, cust.UUID)
	}
	if !slices.Equal(got, []string{"c4"}) || len(ts.recorded()) != 1 {
		t.Errorf("from cursor c3: %v, %d requests", got, len(ts.recorded()))
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
	ts := pagedServer(t, "c3")
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

func TestIterateStopsOnEmptyOrStuckPage(t *testing.T) {
	calls := 0
	seq := iteratePages("", func(cursor string) (*Page[int], error) {
		calls++
		next := fmt.Sprint("n", calls)
		if calls == 2 {
			return &Page[int]{Items: nil, NextCursor: &next}, nil // empty page with a cursor
		}
		return &Page[int]{Items: []int{calls}, NextCursor: &next}, nil
	})
	var got []int
	for v, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, v)
	}
	if !slices.Equal(got, []int{1}) || calls != 2 {
		t.Errorf("empty page: %v after %d calls", got, calls)
	}

	stuck := "same"
	calls = 0
	for range iteratePages("same", func(string) (*Page[int], error) {
		calls++
		return &Page[int]{Items: []int{1}, NextCursor: &stuck}, nil
	}) {
	}
	if calls != 1 {
		t.Errorf("a cursor that does not move must stop the walk (%d calls)", calls)
	}
}

func TestPageHasMore(t *testing.T) {
	next := "x"
	if !(&Page[int]{NextCursor: &next}).HasMore() || (&Page[int]{}).HasMore() || (*Page[int])(nil).HasMore() {
		t.Error("HasMore")
	}
	p, err := decodeResponse[Page[Customer]](&response{status: 200, body: []byte(`{"items":null,"nextCursor":null}`)})
	if err != nil || p.Items != nil || p.HasMore() {
		t.Errorf("null page: %+v %v", p, err)
	}
}
