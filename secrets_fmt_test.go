package qbitflow

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// TestSecretsNeverFormatted: printing a client, a router or an error with fmt (a log line, a
// debug dump) never shows the API key or the webhook secret, whatever the verb.
func TestSecretsNeverFormatted(t *testing.T) {
	const key = "sk_live_secretvalue_never_printed"
	const secret = "whsec_secretvalue_never_printed"

	client, err := NewClient(key, WithBaseURL("https://api.example.invalid/v2"), WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	router := NewWebhookRouter(secret)
	_, verifyErr := ConstructEvent([]byte(`{}`), "t=1,v1=00", secret)
	_, netErr := client.Me(t.Context()) // a .invalid host never resolves: only the error's text matters
	values := map[string]any{
		"client":       client,
		"*client":      *client,
		"member":       client.OnBehalfOf("01a0f22f-9600-7000-8000-000000000001"),
		"products":     client.Products,
		"router":       router,
		"*router":      reflect.ValueOf(router).Elem(), // a copy without copying its lock
		"verify error": verifyErr,
		"empty secret": VerifyWebhook([]byte(`{}`), "", ""),
		"ctor error":   errors.New(fmt.Sprint(NewClient("sk_\x00" + key))),
		"me error":     netErr,
	}
	for name, v := range values {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%d"} {
			out := fmt.Sprintf(verb, v)
			if strings.Contains(out, "secretvalue") {
				t.Errorf("%s with %s leaks a secret: %s", name, verb, out)
			}
		}
	}
}
