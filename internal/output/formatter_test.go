package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/dl-alexandre/abc/internal/api"
)

func captureOutput(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	callErr := fn()
	_ = w.Close()
	os.Stdout = old
	var out bytes.Buffer
	_, readErr := io.Copy(&out, r)
	_ = r.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	return out.String(), callErr
}

func TestPrintLocationsJSON(t *testing.T) {
	location := api.Location{ID: "loc-1", LocationName: api.LocalizedString{Default: "Cafe"}}
	got, err := captureOutput(t, func() error { return NewPrinter("json", false).PrintLocations([]api.Location{location}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"id": "loc-1"`) || !strings.Contains(got, `"default": "Cafe"`) {
		t.Fatalf("unexpected JSON output: %s", got)
	}
}

func TestValidateFormat(t *testing.T) {
	if err := ValidateFormat("markdown", []string{"table", "json", "markdown"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFormat("xml", []string{"table", "json", "markdown"}); err == nil {
		t.Fatal("ValidateFormat accepted xml")
	}
}
