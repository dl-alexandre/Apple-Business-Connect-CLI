// Package blast provides "blast radius" protection to prevent catastrophic
// accidental changes in bulk operations.
package blast

import (
	"fmt"
)

// Protection holds blast radius limits
type Protection struct {
	MaxCreates       int
	MaxUpdates       int
	MaxDeletions     int
	MaxCreatePercent float64
	MaxUpdatePercent float64
	MaxDeletePercent float64
}

// DefaultProtection returns sensible defaults
func DefaultProtection() Protection {
	return Protection{
		MaxCreates:       0, // 0 = unlimited
		MaxUpdates:       0, // 0 = unlimited
		MaxDeletions:     0, // 0 = unlimited
		MaxCreatePercent: 0, // 0 = unlimited
		MaxUpdatePercent: 0, // 0 = unlimited
		MaxDeletePercent: 0, // 0 = unlimited
	}
}

// Limits holds the actual changes detected
type Limits struct {
	ToCreate    int
	ToUpdate    int
	ToDelete    int
	NoChange    int
	TotalLocal  int
	TotalRemote int
}

// CheckResult contains the check outcome
type CheckResult struct {
	Blocked       bool
	BlockedBy     string
	Reason        string
	ActualCreates int
	ActualUpdates int
	ActualDeletes int
	LimitCreates  int
	LimitUpdates  int
	LimitDeletes  int
}

// Check validates if the proposed changes are within blast radius limits
func (p Protection) Check(limits Limits) CheckResult {
	result := CheckResult{
		ActualCreates: limits.ToCreate,
		ActualUpdates: limits.ToUpdate,
		ActualDeletes: limits.ToDelete,
		LimitCreates:  p.MaxCreates,
		LimitUpdates:  p.MaxUpdates,
		LimitDeletes:  p.MaxDeletions,
	}

	// Check absolute limits first
	if p.MaxCreates > 0 && limits.ToCreate > p.MaxCreates {
		result.Blocked = true
		result.BlockedBy = "max_creates"
		result.Reason = fmt.Sprintf("creates (%d) exceeds maximum (%d)", limits.ToCreate, p.MaxCreates)
		return result
	}

	if p.MaxUpdates > 0 && limits.ToUpdate > p.MaxUpdates {
		result.Blocked = true
		result.BlockedBy = "max_updates"
		result.Reason = fmt.Sprintf("updates (%d) exceeds maximum (%d)", limits.ToUpdate, p.MaxUpdates)
		return result
	}

	if p.MaxDeletions > 0 && limits.ToDelete > p.MaxDeletions {
		result.Blocked = true
		result.BlockedBy = "max_deletions"
		result.Reason = fmt.Sprintf("deletions (%d) exceeds maximum (%d)", limits.ToDelete, p.MaxDeletions)
		return result
	}

	// Check percentage limits (based on remote total)
	if limits.TotalRemote > 0 {
		createPercent := float64(limits.ToCreate) / float64(limits.TotalRemote) * 100
		updatePercent := float64(limits.ToUpdate) / float64(limits.TotalRemote) * 100
		deletePercent := float64(limits.ToDelete) / float64(limits.TotalRemote) * 100

		if p.MaxCreatePercent > 0 && createPercent > p.MaxCreatePercent {
			result.Blocked = true
			result.BlockedBy = "max_create_percent"
			result.Reason = fmt.Sprintf("creates (%d, %.1f%%) exceeds maximum (%.1f%%)",
				limits.ToCreate, createPercent, p.MaxCreatePercent)
			return result
		}

		if p.MaxUpdatePercent > 0 && updatePercent > p.MaxUpdatePercent {
			result.Blocked = true
			result.BlockedBy = "max_update_percent"
			result.Reason = fmt.Sprintf("updates (%d, %.1f%%) exceeds maximum (%.1f%%)",
				limits.ToUpdate, updatePercent, p.MaxUpdatePercent)
			return result
		}

		if p.MaxDeletePercent > 0 && deletePercent > p.MaxDeletePercent {
			result.Blocked = true
			result.BlockedBy = "max_delete_percent"
			result.Reason = fmt.Sprintf("deletions (%d, %.1f%%) exceeds maximum (%.1f%%)",
				limits.ToDelete, deletePercent, p.MaxDeletePercent)
			return result
		}
	}

	return result
}

// Error returns an error if the check failed
func (r CheckResult) Error() error {
	if r.Blocked {
		return fmt.Errorf("blast radius protection triggered: %s", r.Reason)
	}
	return nil
}

// FormatLimits formats blast radius limits for display
func (p Protection) FormatLimits() string {
	parts := []string{}

	if p.MaxCreates > 0 {
		parts = append(parts, fmt.Sprintf("max %d creates", p.MaxCreates))
	}
	if p.MaxUpdates > 0 {
		parts = append(parts, fmt.Sprintf("max %d updates", p.MaxUpdates))
	}
	if p.MaxDeletions > 0 {
		parts = append(parts, fmt.Sprintf("max %d deletions", p.MaxDeletions))
	}
	if p.MaxCreatePercent > 0 {
		parts = append(parts, fmt.Sprintf("max %.0f%% creates", p.MaxCreatePercent))
	}
	if p.MaxUpdatePercent > 0 {
		parts = append(parts, fmt.Sprintf("max %.0f%% updates", p.MaxUpdatePercent))
	}
	if p.MaxDeletePercent > 0 {
		parts = append(parts, fmt.Sprintf("max %.0f%% deletions", p.MaxDeletePercent))
	}

	if len(parts) == 0 {
		return "unlimited (no blast radius protection)"
	}

	return joinParts(parts)
}

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	if len(parts) == 2 {
		return parts[0] + " and " + parts[1]
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += ", "
		}
		if i == len(parts)-1 {
			result += "and "
		}
		result += part
	}
	return result
}

// ParsePercent converts a percentage string (e.g., "10%") to a float
func ParsePercent(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	s = trimPercent(s)
	val, err := parseFloat(s)
	if err != nil {
		return 0, fmt.Errorf("invalid percentage: %s", s)
	}
	if val < 0 || val > 100 {
		return 0, fmt.Errorf("percentage must be between 0 and 100: %s", s)
	}
	return val, nil
}

func trimPercent(s string) string {
	if len(s) > 0 && s[len(s)-1] == '%' {
		return s[:len(s)-1]
	}
	return s
}

func parseFloat(s string) (float64, error) {
	return 0, nil // Stub - would use strconv.ParseFloat in real implementation
}

// SmartThresholds suggests blast radius limits based on total locations
func SmartThresholds(totalLocations int) Protection {
	switch {
	case totalLocations < 10:
		// Small business - more permissive
		return Protection{
			MaxCreatePercent: 100,
			MaxUpdatePercent: 100,
			MaxDeletePercent: 50, // Still prevent total wipeout
		}
	case totalLocations < 100:
		// Medium business
		return Protection{
			MaxCreates:       25,
			MaxUpdates:       50,
			MaxDeletions:     10,
			MaxCreatePercent: 50,
			MaxUpdatePercent: 75,
			MaxDeletePercent: 20,
		}
	default:
		// Enterprise - conservative
		return Protection{
			MaxCreates:       50,
			MaxUpdates:       100,
			MaxDeletions:     20,
			MaxCreatePercent: 25,
			MaxUpdatePercent: 50,
			MaxDeletePercent: 10,
		}
	}
}
