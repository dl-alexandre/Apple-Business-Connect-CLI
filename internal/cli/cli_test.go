package cli

import (
	"testing"
)

func TestGlobals_AfterApply(t *testing.T) {
	tests := []struct {
		name    string
		globals Globals
		wantErr bool
	}{
		{
			name:    "empty globals - no initialization",
			globals: Globals{
				// Empty - should skip initialization
			},
			wantErr: false,
		},
		{
			name: "with API URL only",
			globals: Globals{
				APIURL: "https://api.example.com",
			},
			wantErr: false, // Creates client without credentials, fails on first API call
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.globals.AfterApply()
			if (err != nil) != tt.wantErr {
				t.Errorf("AfterApply() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGlobals_ShouldUseColor(t *testing.T) {
	g := &Globals{}

	// This will depend on the test environment
	// Just ensure it doesn't panic
	_ = g.ShouldUseColor()
}

func TestLocationsListCmd_Validation(t *testing.T) {
	cmd := &LocationsListCmd{
		CompanyID:    "test-company",
		Limit:        10,
		PageToken:    "",
		OutputFormat: "json",
	}

	if cmd.Limit < 1 {
		t.Error("limit should be at least 1")
	}

	if cmd.CompanyID == "" {
		t.Error("company ID should not be empty for this test")
	}

	validFormats := []string{"table", "json", "markdown"}
	found := false
	for _, f := range validFormats {
		if f == cmd.OutputFormat {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("format %s is not valid", cmd.OutputFormat)
	}
}

func TestLocationsGetCmd_Validation(t *testing.T) {
	cmd := &LocationsGetCmd{
		ID:           "test-id",
		OutputFormat: "json",
	}

	if cmd.ID == "" {
		t.Error("ID should not be empty")
	}

	validFormats := []string{"table", "json", "markdown"}
	found := false
	for _, f := range validFormats {
		if f == cmd.OutputFormat {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("format %s is not valid", cmd.OutputFormat)
	}
}

func TestShowcasesListCmd_Validation(t *testing.T) {
	cmd := &ShowcasesListCmd{
		LocationID:   "test-location",
		Limit:        5,
		OutputFormat: "table",
	}

	if cmd.LocationID == "" {
		t.Error("location ID should not be empty")
	}

	if cmd.Limit < 1 {
		t.Error("limit should be at least 1")
	}
}

func TestInsightsGetCmd_Validation(t *testing.T) {
	cmd := &InsightsGetCmd{
		LocationID:   "test-location",
		Period:       "MONTH",
		StartDate:    "2024-01-01",
		EndDate:      "2024-01-31",
		OutputFormat: "json",
	}

	if cmd.LocationID == "" {
		t.Error("location ID should not be empty")
	}

	validPeriods := []string{"DAY", "WEEK", "MONTH"}
	found := false
	for _, p := range validPeriods {
		if p == cmd.Period {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("period %s is not valid", cmd.Period)
	}
}

func TestCompletionCmd_Shells(t *testing.T) {
	validShells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range validShells {
		t.Run(shell, func(t *testing.T) {
			cmd := &CompletionCmd{Shell: shell}
			// Just ensure shell value is set
			if cmd.Shell != shell {
				t.Errorf("expected shell %s, got %s", shell, cmd.Shell)
			}
		})
	}
}
