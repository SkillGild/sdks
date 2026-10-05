package skillgild

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

type recorded struct {
	method, uri string
	header      http.Header
	body        string
}

// fakeAPI answers every request with status/body/headers and records what it saw.
func fakeAPI(t *testing.T, status int, body string, headers map[string]string) (*Client, *[]recorded) {
	t.Helper()
	var calls []recorded
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		calls = append(calls, recorded{r.Method, r.URL.RequestURI(), r.Header.Clone(), string(raw)})
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL+"/v1/", WithAPIKey("sk"))
	if err != nil {
		t.Fatal(err)
	}
	return client, &calls
}

func TestListSkillsUnwrapsEnvelopeWithoutKey(t *testing.T) {
	client, calls := fakeAPI(t, 200, `{"data":{"items":[{"slug":"a","page":{"what_you_get":["x"]}}],"next_cursor":"c2"}}`, nil)
	page, err := client.ListSkills(context.Background(), ListOptions{Query: " design ", Limit: 10, Cursor: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextCursor != "c2" || page.Items[0].Page.WhatYouGet[0] != "x" {
		t.Fatalf("page = %+v", page)
	}
	got := (*calls)[0]
	if got.uri != "/v1/skills?cursor=c1&limit=10&q=design" || got.header.Get("Authorization") != "" {
		t.Fatalf("request = %+v", got)
	}
}

func TestRunSkillSendsKeyAndIdempotencyKey(t *testing.T) {
	client, calls := fakeAPI(t, 200, `{"data":{"output":"ok"}}`, nil)
	run, err := client.RunSkill(context.Background(), "a b", map[string]any{"prompt": "x"}, WithIdempotencyKey("k1"))
	if err != nil || run.Output != "ok" {
		t.Fatalf("run = %+v, %v", run, err)
	}
	got := (*calls)[0]
	if got.method != "POST" || got.uri != "/v1/skills/a%20b/run" || got.header.Get("Authorization") != "Bearer sk" || got.header.Get("Idempotency-Key") != "k1" {
		t.Fatalf("request = %+v", got)
	}
	if got.body != `{"input":{"prompt":"x"}}` {
		t.Fatalf("body = %s", got.body)
	}
}

func TestKeyRequiredBeforeRequest(t *testing.T) {
	client, err := NewClient("https://api.test/v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Me(context.Background()); !errors.Is(err, ErrAPIKeyRequired) {
		t.Fatalf("err = %v", err)
	}
}

func TestQuotaErrorCarriesRetryAfter(t *testing.T) {
	client, _ := fakeAPI(t, 429, `{"error":{"code":"quota_exceeded","message":"Monthly runs used"}}`, map[string]string{"Retry-After": "120"})
	_, err := client.RunSkill(context.Background(), "a", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 429 || apiErr.Code != "quota_exceeded" || apiErr.RetryAfter != 120*time.Second {
		t.Fatalf("err = %#v", err)
	}
}

func TestUnauthorizedAndProxyErrors(t *testing.T) {
	client, _ := fakeAPI(t, 401, `{"error":{"code":"unauthorized","message":"Invalid API key"}}`, nil)
	var apiErr *APIError
	if _, err := client.Me(context.Background()); !errors.As(err, &apiErr) || apiErr.Status != 401 || apiErr.Code != "unauthorized" {
		t.Fatalf("err = %#v", err)
	}
	proxy, _ := fakeAPI(t, 502, `<html>bad gateway</html>`, nil)
	if _, err := proxy.GetSkill(context.Background(), "a"); !errors.As(err, &apiErr) || apiErr.Status != 502 || apiErr.Code != "" {
		t.Fatalf("err = %#v", err)
	}
}

func TestNoContent(t *testing.T) {
	client, calls := fakeAPI(t, 204, "", nil)
	if err := client.EndSession(context.Background(), "s1"); err != nil {
		t.Fatal(err)
	}
	if (*calls)[0].method != "DELETE" || (*calls)[0].uri != "/v1/skill-sessions/s1" {
		t.Fatalf("request = %+v", (*calls)[0])
	}
}

func TestCatalogDeviceAndKeyEndpoints(t *testing.T) {
	client, calls := fakeAPI(t, 200, `{"data":null}`, nil)
	ctx := context.Background()
	_, _ = client.ListCategories(ctx)
	_, _ = client.ListTags(ctx)
	_, _ = client.ListCollections(ctx)
	_, _ = client.GetCollection(ctx, "starter kit")
	_, _ = client.FeaturedSkills(ctx, "")
	_, _ = client.StartDeviceAuthorization(ctx, "laptop", "")
	_, _ = client.PollDeviceAuthorization(ctx, "dc")
	_ = client.RevokeCurrentKey(ctx)
	var got []string
	for _, call := range *calls {
		got = append(got, call.method+" "+call.uri)
	}
	want := []string{
		"GET /v1/categories",
		"GET /v1/tags",
		"GET /v1/collections",
		"GET /v1/collections/starter%20kit",
		"GET /v1/featured-skills?slot=home",
		"POST /v1/device-authorizations",
		"POST /v1/device-authorizations/token",
		"DELETE /v1/me/api-keys/current",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requests = %v", got)
	}
	var body map[string]string
	_ = json.Unmarshal([]byte((*calls)[5].body), &body)
	if body["client_type"] != "skillgild-sdk" || body["device_name"] != "laptop" {
		t.Fatalf("device body = %v", body)
	}
}

func TestRejectsBadBaseURLs(t *testing.T) {
	for _, raw := range []string{"https://u:p@api.test", "ftp://api.test", "api.test"} {
		if _, err := NewClient(raw); err == nil {
			t.Errorf("NewClient(%q) accepted", raw)
		}
	}
}
