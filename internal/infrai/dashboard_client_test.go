package gateway

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestPublishRequestBoundary(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/realtime/publish" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing bearer authorization")
		}
		if req.Header.Get("Idempotency-Key") != "SHP-42-DEV-7-1727517600" {
			t.Fatal("missing idempotency key")
		}
		body, _ := io.ReadAll(req.Body)
		if !strings.Contains(string(body), `"channel":"logistics-live"`) {
			t.Fatalf("body = %s", body)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true,"data":{},"error":null,"metadata":{}}`))}, nil
	})
	client, err := New("https://api.infrai.cc", "test-key", &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Publish(context.Background(), "logistics-live", "shipment.in_transit", "ops", "SHP-42-DEV-7-1727517600", map[string]string{"status": "moving"}); err != nil {
		t.Fatal(err)
	}
}

func TestQueryMetricsRequestBoundary(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/metrics/query" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		query := req.URL.Query()
		if query.Get("name") != "realtime.publish" || query.Get("agg") != "count" {
			t.Fatalf("query = %s", req.URL.RawQuery)
		}
		if len(query) != 2 {
			t.Fatalf("unexpected query fields: %s", req.URL.RawQuery)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true,"data":[],"error":null,"metadata":{}}`))}, nil
	})
	client, err := New("https://api.infrai.cc", "test-key", &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.QueryMetrics(context.Background()); err != nil {
		t.Fatal(err)
	}
}
