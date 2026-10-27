package qbitflow

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestSettingsList(t *testing.T) {
	// lockedBy is not modelled: a body carrying it still decodes (unknown keys are ignored).
	ts := newTestServer(t, staticReply(200, bounded(
		`{"key":"refunds.windowDays","kind":"setting","value":30,"source":"default","editable":true,"lockedBy":"plan"}`,
		`{"key":"branding","kind":"setting","value":{"displayName":"Shop"},"source":"own","editable":false}`)))
	c, _ := newTestClient(t, ts)

	got, err := c.Settings.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%d settings, want 2", len(got))
	}
	if s := got[0]; s.Key != "refunds.windowDays" || s.Kind != SettingKindSetting || string(s.Value) != "30" ||
		s.Source != SettingSourceDefault || !s.Editable {
		t.Errorf("%+v", s)
	}
	if s := got[1]; string(s.Value) != `{"displayName":"Shop"}` || s.Source != SettingSourceOwn || s.Editable {
		t.Errorf("%+v", s)
	}
	if r := ts.recorded()[0]; r.method != http.MethodGet || r.path != "/settings" || r.query != "" {
		t.Errorf("%s %s?%s", r.method, r.path, r.query)
	}
}

func TestSettingsDefinitions(t *testing.T) {
	ts := newTestServer(t, staticReply(200, bounded(fx("SettingDefinition"))))
	c, _ := newTestClient(t, ts)

	got, err := c.Settings.Definitions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Key != "refunds.windowDays" || got[0].Type != "integer" || got[0].Min == nil || *got[0].Min != 1 ||
		got[0].Max == nil || *got[0].Max != 365 || string(got[0].Default) != "30" || got[0].MembersEdit != "never" {
		t.Errorf("%+v", got)
	}
	if r := ts.recorded()[0]; r.method != http.MethodGet || r.path != "/settings/definitions" {
		t.Errorf("%s %s", r.method, r.path)
	}
}

func TestSettingsSetSendsValue(t *testing.T) {
	ts := newTestServer(t, staticReply(200, fx("SettingValue")))
	c, _ := newTestClient(t, ts)

	got, err := c.Settings.Set(context.Background(), "branding", map[string]any{"displayName": "Shop"})
	if err != nil {
		t.Fatal(err)
	}
	r := ts.recorded()[0]
	if r.method != http.MethodPut || r.path != "/settings/branding" || r.body != `{"value":{"displayName":"Shop"}}` {
		t.Errorf("%s %s %s", r.method, r.path, r.body)
	}
	if got.Key != "refunds.windowDays" || !got.Editable {
		t.Errorf("%+v", got)
	}
	for name, value := range map[string]any{"bool": false, "int": 30, "string": "full"} {
		if _, err := c.Settings.Set(context.Background(), "k", value); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A write is never retried.
	ts2 := newTestServer(t, staticReply(503, `{"error":"unavailable","message":"x"}`))
	c2, _ := newTestClient(t, ts2)
	if _, err := c2.Settings.Set(context.Background(), "k", true); err == nil || len(ts2.recorded()) != 1 {
		t.Errorf("err %v, %d requests", err, len(ts2.recorded()))
	}
}

func TestSettingsResetNoBody(t *testing.T) {
	ts := newTestServer(t, staticReply(204, ``))
	c, _ := newTestClient(t, ts)

	if err := c.Settings.Reset(context.Background(), "refunds.windowDays"); err != nil {
		t.Fatal(err)
	}
	r := ts.recorded()[0]
	if r.method != http.MethodDelete || r.path != "/settings/refunds.windowDays" || r.body != "" || r.header.Get("Content-Type") != "" {
		t.Errorf("%s %s body %q", r.method, r.path, r.body)
	}
}

func TestSettingsKeyValidation(t *testing.T) {
	ts := newTestServer(t, staticReply(204, `{}`))
	c, _ := newTestClient(t, ts)
	ctx := context.Background()

	for name, key := range map[string]string{"empty": "", "blank": "  ", "101 chars": strings.Repeat("k", 101), "dot": ".", "dotdot": ".."} {
		for op, err := range map[string]error{
			"Set":   func() error { _, err := c.Settings.Set(ctx, key, true); return err }(),
			"Reset": c.Settings.Reset(ctx, key),
		} {
			var ve *ValidationError
			if !errors.As(err, &ve) || len(ve.FieldErrors) != 1 || ve.FieldErrors[0].Field != "key" {
				t.Errorf("%s %s: %v", op, name, err)
			}
		}
	}
	if _, err := c.Settings.Set(ctx, "k", nil); err == nil {
		t.Error("a nil value must be refused")
	} else if ve := new(ValidationError); !errors.As(err, &ve) || ve.FieldErrors[0].Field != "value" {
		t.Errorf("nil value: %v", err)
	}
	if n := len(ts.recorded()); n != 0 {
		t.Fatalf("%d requests sent for invalid input", n)
	}

	if err := c.Settings.Reset(ctx, strings.Repeat("k", 100)); err != nil {
		t.Errorf("100 chars: %v", err)
	}
	if err := c.Settings.Reset(ctx, "a/b"); err != nil {
		t.Fatal(err)
	}
	if r := ts.recorded()[1]; r.path != "/settings/a%2Fb" {
		t.Errorf("path %q, want /settings/a%%2Fb", r.path)
	}
}
