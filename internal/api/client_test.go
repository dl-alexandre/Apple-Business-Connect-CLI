package api

import (
	"testing"
)

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
