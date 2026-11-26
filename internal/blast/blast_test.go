package blast

import "testing"

func TestParsePercent(t *testing.T) {
	for input, want := range map[string]float64{"": 0, "12": 12, "12.5%": 12.5, "100%": 100} {
		got, err := ParsePercent(input)
		if err != nil || got != want {
			t.Errorf("ParsePercent(%q) = %v, %v; want %v, nil", input, got, err, want)
		}
	}
	for _, input := range []string{"abc", "-1", "100.1", "1,5"} {
		if _, err := ParsePercent(input); err == nil {
			t.Errorf("ParsePercent(%q) succeeded, want validation error", input)
		}
	}
}

func TestCheckEnforcesAbsoluteAndPercentageLimits(t *testing.T) {
	limits := Limits{ToCreate: 3, ToUpdate: 4, ToDelete: 1, TotalRemote: 10}
	got := (Protection{MaxCreates: 2}).Check(limits)
	if !got.Blocked || got.BlockedBy != "max_creates" {
		t.Fatalf("absolute limit result = %+v", got)
	}
	got = (Protection{MaxDeletePercent: 5}).Check(limits)
	if !got.Blocked || got.BlockedBy != "max_delete_percent" {
		t.Fatalf("percentage limit result = %+v", got)
	}
}
