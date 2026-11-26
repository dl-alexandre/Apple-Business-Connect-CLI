package cli

import (
	"io"
	"testing"

	"github.com/alecthomas/kong"
)

func TestKongLocationsListParsesFlags(t *testing.T) {
	var app CLI
	parser, err := kong.New(&app, kong.Writers(io.Discard, io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	_, err = parser.Parse([]string{"--format", "json", "locations", "list", "--company-id", "company-7", "--limit", "12", "--page-token", "next", "--output-format", "markdown"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if app.Format != "json" || app.Locations.List.CompanyID != "company-7" || app.Locations.List.Limit != 12 || app.Locations.List.PageToken != "next" || app.Locations.List.OutputFormat != "markdown" {
		t.Fatalf("unexpected parsed flags: globals=%+v list=%+v", app.Globals, app.Locations.List)
	}
}

func TestKongAuthLoginParsesCredentialFlags(t *testing.T) {
	var app CLI
	parser, err := kong.New(&app, kong.Writers(io.Discard, io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	_, err = parser.Parse([]string{"auth", "login", "--client-id", "abc-id", "--client-secret", "abc-secret"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if app.Auth.Login.ClientID != "abc-id" || app.Auth.Login.ClientSecret != "abc-secret" {
		t.Fatalf("unexpected auth flags: %+v", app.Auth.Login)
	}
}
