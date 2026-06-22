package dns

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckBIMIWithValidRecord(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/logo.svg" {
			t.Fatalf("unexpected logo request path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewChecker()
	checker.httpClient = server.Client()
	checker.lookupTXT = func(name string) ([]string, error) {
		if name != "default._bimi.example.com" {
			t.Fatalf("unexpected lookup %s", name)
		}
		return []string{"v=BIMI1; l=" + server.URL + "/logo.svg; a=https://example.com/vmc.pem"}, nil
	}

	record := checker.checkBIMI("example.com")
	if !record.Present {
		t.Fatal("expected BIMI record to be present")
	}
	if record.LogoURL != server.URL+"/logo.svg" {
		t.Fatalf("unexpected logo URL %q", record.LogoURL)
	}
	if record.VMCURL != "https://example.com/vmc.pem" {
		t.Fatalf("unexpected VMC URL %q", record.VMCURL)
	}
	if !record.URLAccessible || record.StatusCode != http.StatusOK {
		t.Fatalf("expected accessible logo, got accessible=%v status=%d", record.URLAccessible, record.StatusCode)
	}
	if record.ContentType != "image/svg+xml" {
		t.Fatalf("unexpected content type %q", record.ContentType)
	}
	if len(checker.errors) != 0 {
		t.Fatalf("expected no errors, got %+v", checker.errors)
	}
	if len(checker.warnings) != 0 {
		t.Fatalf("expected no warnings, got %+v", checker.warnings)
	}
}

func TestCheckBIMIMissingRecordIsOptional(t *testing.T) {
	checker := NewChecker()
	checker.lookupTXT = func(name string) ([]string, error) {
		if name != "default._bimi.example.com" {
			t.Fatalf("unexpected lookup %s", name)
		}
		return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
	}

	record := checker.checkBIMI("example.com")
	if record.Present {
		t.Fatalf("expected missing BIMI record, got %+v", record)
	}
	if len(checker.errors) != 0 {
		t.Fatalf("expected no errors for optional missing BIMI, got %+v", checker.errors)
	}
	if len(checker.warnings) != 0 {
		t.Fatalf("expected no warnings for optional missing BIMI, got %+v", checker.warnings)
	}
}

func TestCheckBIMIMalformedRecordWarns(t *testing.T) {
	checker := NewChecker()
	checker.lookupTXT = func(name string) ([]string, error) {
		if name != "default._bimi.example.com" {
			t.Fatalf("unexpected lookup %s", name)
		}
		return []string{"v=BIMI1; l=not-a-url"}, nil
	}

	record := checker.checkBIMI("example.com")
	if !record.Present {
		t.Fatal("expected malformed BIMI record to be marked present")
	}
	if record.LogoURL != "" {
		t.Fatalf("expected malformed logo URL to be rejected, got %q", record.LogoURL)
	}
	if !strings.Contains(record.Error, "missing a valid HTTPS logo URL") {
		t.Fatalf("expected actionable malformed error, got %q", record.Error)
	}
	if len(checker.errors) != 0 {
		t.Fatalf("expected malformed BIMI to warn, not error, got %+v", checker.errors)
	}
	if len(checker.warnings) != 1 {
		t.Fatalf("expected one warning, got %+v", checker.warnings)
	}
	if checker.warnings[0].Record != "BIMI" {
		t.Fatalf("expected BIMI warning, got %+v", checker.warnings[0])
	}
}
