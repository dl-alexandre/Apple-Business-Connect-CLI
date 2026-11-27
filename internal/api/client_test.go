package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(ClientOptions{BaseURL: server.URL, ClientID: "test-id", ClientSecret: "test-secret", Timeout: 5})
	if err != nil {
		t.Fatal(err)
	}
	c.client.SetTransport(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "businessconnect.apple.com" {
			target, err := url.Parse(server.URL + req.URL.RequestURI())
			if err != nil {
				return nil, err
			}
			req.URL = target
			req.Host = target.Host
		}
		return http.DefaultTransport.RoundTrip(req)
	}))
	return c
}

func TestListLocationsPerformsOAuthAndDecodesResponse(t *testing.T) {
	tokenRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/v1/token":
			tokenRequests++
			body, _ := io.ReadAll(r.Body)
			if r.Method != http.MethodPost || !strings.Contains(string(body), "grant_type=client_credentials") || !strings.Contains(string(body), "client_id=test-id") {
				t.Errorf("unexpected token request: %s %s body=%q", r.Method, r.URL, body)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
		case "/locations":
			if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
				t.Errorf("Authorization = %q, want Bearer test-token", got)
			}
			if r.URL.Query().Get("companyId") != "co-1" || r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("pageToken") != "next" {
				t.Errorf("unexpected query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"locations":[{"id":"loc-1","locationName":{"default":"Shop"}}],"nextPageToken":"more"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	got, err := newTestClient(t, server).ListLocations(context.Background(), "co-1", 2, "next")
	if err != nil {
		t.Fatal(err)
	}
	if tokenRequests != 1 || len(got.Locations) != 1 || got.Locations[0].ID != "loc-1" || got.NextPageToken != "more" {
		t.Fatalf("unexpected API result: token requests=%d response=%+v", tokenRequests, got)
	}
}

func TestGetLocationMapsNotFoundResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/v1/token" {
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"missing"}}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).GetLocation(context.Background(), "missing-id")
	if got, ok := err.(*NotFoundError); !ok || got.Resource != "missing-id" {
		t.Fatalf("GetLocation() error = %#v, want NotFoundError", err)
	}
}

func TestAPIErrorResponse(t *testing.T) {
	err := &APIErrorResponse{
		StatusCode: 404,
		ErrorDetails: ErrorDetails{
			Code:    "NOT_FOUND",
			Message: "Resource not found",
		},
	}

	expected := "API error 404: NOT_FOUND - Resource not found"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

func TestAPIErrorResponse_NoCode(t *testing.T) {
	err := &APIErrorResponse{
		StatusCode: 500,
		ErrorDetails: ErrorDetails{
			Message: "Internal server error",
		},
	}

	expected := "API error 500: Internal server error"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{
		Resource: "test-resource",
	}

	expected := "resource not found: test-resource"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

func TestNewClient(t *testing.T) {
	client, err := NewClient(ClientOptions{
		BaseURL:      "https://api.businessconnect.apple.com/v3.0",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Timeout:      30,
		Verbose:      false,
		Debug:        false,
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if client == nil {
		t.Error("client should not be nil")
	}
}

func TestNewClient_DefaultBaseURL(t *testing.T) {
	client, err := NewClient(ClientOptions{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if client == nil {
		t.Error("client should not be nil")
	}
}
