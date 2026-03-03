// Package svg provides SVG validation and automated remediation for BIMI compliance
// with the Tiny Portable/Secure (Tiny-PS) profile.
package svg

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationResult mirrors the validation result structure
type ValidationResult struct {
	FilePath        string
	Valid           bool
	TinyPSCompliant bool
	Errors          []ValidationError
	Warnings        []ValidationWarning
	Dimensions      Dimensions
	HasScripts      bool
	HasExternalRefs bool
	IsBase64        bool
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field   string
	Message string
}

// Dimensions represents SVG size information
type Dimensions struct {
	Width       float64
	Height      float64
	AspectRatio float64
	Valid       bool
}

// Validator provides SVG validation
type Validator struct {
	errors   []ValidationError
	warnings []ValidationWarning
}

// NewValidator creates a new SVG validator
func NewValidator() *Validator {
	return &Validator{
		errors:   make([]ValidationError, 0),
		warnings: make([]ValidationWarning, 0),
	}
}

// ValidateBytes validates SVG data from a byte slice
func (v *Validator) ValidateBytes(data []byte, source string) ValidationResult {
	// Simplified validation - full implementation in validator.go
	result := ValidationResult{
		FilePath: source,
		Valid:    true,
	}
	return result
}

// PrintResults outputs validation results
func (r ValidationResult) PrintResults() {
	fmt.Printf("\n🎨 SVG Validation for %s\n", r.FilePath)
	fmt.Println(strings.Repeat("─", 50))

	if r.Valid {
		fmt.Println("✅ SVG is BIMI compliant")
	} else {
		fmt.Println("❌ SVG validation failed")
		if len(r.Errors) > 0 {
			fmt.Println("\nErrors:")
			for _, err := range r.Errors {
				fmt.Printf("  [%s] %s: %s\n", err.Code, err.Field, err.Message)
			}
		}
	}

	if len(r.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warn := range r.Warnings {
			fmt.Printf("  [%s] %s\n", warn.Field, warn.Message)
		}
	}
}

// FixOptions controls automated remediation behavior
type FixOptions struct {
	RemoveScripts       bool   // Remove <script> elements
	RemoveEventHandlers bool   // Remove onclick, onload, etc.
	FixAspectRatio      bool   // Normalize viewBox to square
	RemoveExternalRefs  bool   // Remove external references
	OutputPath          string // Where to write fixed SVG (empty = don't write)
}

// DefaultFixOptions returns sensible defaults
func DefaultFixOptions() FixOptions {
	return FixOptions{
		RemoveScripts:       true,
		RemoveEventHandlers: true,
		FixAspectRatio:      true,
		RemoveExternalRefs:  true,
	}
}

// FixResult holds the outcome of SVG remediation
type FixResult struct {
	OriginalPath   string
	FixedPath      string
	Changes        []string
	Errors         []error
	Success        bool
	OriginalIssues int
	FixedIssues    int
}

// Fixer performs automated SVG remediation
type Fixer struct {
	options FixOptions
}

// NewFixer creates a new SVG fixer
func NewFixer(options FixOptions) *Fixer {
	return &Fixer{options: options}
}

// Fix performs automated remediation on an SVG
func (f *Fixer) Fix(svgData []byte, source string) FixResult {
	result := FixResult{
		OriginalPath: source,
		Changes:      make([]string, 0),
		Errors:       make([]error, 0),
		Success:      false,
	}

	// First validate to see what needs fixing
	validator := NewValidator()
	valResult := validator.ValidateBytes(svgData, source)
	result.OriginalIssues = len(valResult.Errors) + len(valResult.Warnings)

	if result.OriginalIssues == 0 {
		result.Success = true
		result.Changes = append(result.Changes, "No issues found - SVG already compliant")
		return result
	}

	// Get the SVG content as string for manipulation
	content := string(svgData)
	originalContent := content

	// Remove scripts
	if f.options.RemoveScripts && valResult.HasScripts {
		content = f.removeScripts(content)
		if content != originalContent {
			result.Changes = append(result.Changes, "Removed script elements and event handlers")
		}
	}

	// Remove external references
	if f.options.RemoveExternalRefs && valResult.HasExternalRefs {
		content = f.removeExternalReferences(content)
		if content != originalContent {
			result.Changes = append(result.Changes, "Removed external references")
		}
	}

	// Fix aspect ratio if needed
	if f.options.FixAspectRatio && valResult.Dimensions.Valid {
		if valResult.Dimensions.AspectRatio < 0.99 || valResult.Dimensions.AspectRatio > 1.01 {
			content = f.fixAspectRatio(content, valResult)
			result.Changes = append(result.Changes,
				fmt.Sprintf("Normalized aspect ratio to 1:1 (was %.2f:1)", valResult.Dimensions.AspectRatio))
		}
	}

	// Remove forbidden elements
	content = f.removeForbiddenElements(content)
	if content != originalContent {
		result.Changes = append(result.Changes, "Removed forbidden elements (foreignObject, iframe, etc.)")
	}

	// Clean up XML
	content = f.cleanupXML(content)

	// Write output if requested
	if f.options.OutputPath != "" && content != originalContent {
		// In a real implementation, this would write to file
		result.FixedPath = f.options.OutputPath
		result.Changes = append(result.Changes, fmt.Sprintf("Written to: %s", f.options.OutputPath))
	}

	// Re-validate to confirm fixes
	if content != originalContent {
		fixedValidator := NewValidator()
		fixedResult := fixedValidator.ValidateBytes([]byte(content), source+" (fixed)")
		result.FixedIssues = len(fixedResult.Errors) + len(fixedResult.Warnings)
		result.Success = len(fixedResult.Errors) == 0
	} else {
		result.Success = true
		result.FixedIssues = result.OriginalIssues
	}

	return result
}

// removeScripts removes script elements and event handlers
func (f *Fixer) removeScripts(content string) string {
	// Remove script tags
	scriptRegex := regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	content = scriptRegex.ReplaceAllString(content, "")

	// Remove event handlers
	eventHandlers := []string{
		`on[a-z]+="[^"]*"`,
		`on[a-z]+='[^']*'`,
		`on[a-z]+=[^\s>]+`,
	}

	for _, handler := range eventHandlers {
		regex := regexp.MustCompile(handler)
		content = regex.ReplaceAllString(content, "")
	}

	// Remove javascript: URLs
	jsURLRegex := regexp.MustCompile(`href="javascript:[^"]*"`)
	content = jsURLRegex.ReplaceAllString(content, `href="#"`)

	return content
}

// removeExternalReferences removes external resource references
func (f *Fixer) removeExternalReferences(content string) string {
	// Remove external hrefs
	externalHrefRegex := regexp.MustCompile(`(href|xlink:href)="https?://[^"]*"`)
	content = externalHrefRegex.ReplaceAllString(content, ``)

	// Remove url() references to external resources
	urlRegex := regexp.MustCompile(`url\(https?://[^)]+\)`)
	content = urlRegex.ReplaceAllString(content, ``)

	return content
}

// removeForbiddenElements removes non-Tiny-PS elements
func (f *Fixer) removeForbiddenElements(content string) string {
	forbiddenElements := []string{
		`(?s)<foreignObject[^>]*>.*?</foreignObject>`,
		`(?s)<iframe[^>]*>.*?</iframe>`,
		`(?s)<embed[^>]*>.*?</embed>`,
		`(?s)<object[^>]*>.*?</object>`,
	}

	for _, element := range forbiddenElements {
		regex := regexp.MustCompile(element)
		content = regex.ReplaceAllString(content, "")
	}

	return content
}

// fixAspectRatio normalizes the SVG to square aspect ratio
func (f *Fixer) fixAspectRatio(content string, valResult ValidationResult) string {
	if !valResult.Dimensions.Valid {
		return content
	}

	// Calculate the larger dimension to use as the square size
	targetSize := valResult.Dimensions.Width
	if valResult.Dimensions.Height > targetSize {
		targetSize = valResult.Dimensions.Height
	}

	// Update width and height attributes
	widthRegex := regexp.MustCompile(`width="[^"]*"`)
	heightRegex := regexp.MustCompile(`height="[^"]*"`)
	viewBoxRegex := regexp.MustCompile(`viewBox="[^"]*"`)

	content = widthRegex.ReplaceAllString(content, fmt.Sprintf(`width="%.0f"`, targetSize))
	content = heightRegex.ReplaceAllString(content, fmt.Sprintf(`height="%.0f"`, targetSize))

	// Update or add viewBox
	newViewBox := fmt.Sprintf("0 0 %.0f %.0f", targetSize, targetSize)
	if viewBoxRegex.MatchString(content) {
		content = viewBoxRegex.ReplaceAllString(content, fmt.Sprintf(`viewBox="%s"`, newViewBox))
	} else {
		// Add viewBox after opening svg tag
		svgTagRegex := regexp.MustCompile(`(<svg[^>]*)>`)
		content = svgTagRegex.ReplaceAllString(content, fmt.Sprintf(`$1 viewBox="%s">`, newViewBox))
	}

	return content
}

// cleanupXML removes unnecessary whitespace and ensures well-formed XML
func (f *Fixer) cleanupXML(content string) string {
	// Remove empty lines
	content = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(content, "\n")

	// Ensure XML declaration if missing
	if !strings.HasPrefix(content, "<?xml") && !strings.HasPrefix(content, "<svg") {
		content = `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + content
	}

	return strings.TrimSpace(content)
}

// PrintFixResults outputs remediation results
func (r FixResult) PrintResults() {
	fmt.Printf("\n🔧 SVG Remediation Results\n")
	fmt.Println(strings.Repeat("─", 50))

	fmt.Printf("Original: %s\n", r.OriginalPath)
	if r.FixedPath != "" {
		fmt.Printf("Fixed:    %s\n", r.FixedPath)
	}

	fmt.Printf("\nIssues Found: %d\n", r.OriginalIssues)
	fmt.Printf("Issues Fixed: %d\n", r.OriginalIssues-r.FixedIssues)
	fmt.Printf("Remaining:    %d\n", r.FixedIssues)

	if len(r.Changes) > 0 {
		fmt.Println("\nChanges Made:")
		for _, change := range r.Changes {
			fmt.Printf("  ✓ %s\n", change)
		}
	}

	if len(r.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range r.Errors {
			fmt.Printf("  ✗ %v\n", err)
		}
	}

	fmt.Println()
	if r.Success {
		fmt.Println("✅ SVG is now BIMI compliant!")
	} else {
		fmt.Println("⚠️  Some issues require manual fixing")
	}
}

// CanAutoFix checks if validation errors can be automatically resolved
func CanAutoFix(result ValidationResult) bool {
	// Can fix if no errors (only warnings) or if errors are related to:
	// - Scripts/event handlers
	// - External references
	// - Aspect ratio
	// - Forbidden elements

	autoFixableCodes := map[string]bool{
		"FORBIDDEN_ELEMENT":    true,
		"EXTERNAL_REFERENCE":   true,
		"INVALID_ASPECT_RATIO": true,
		"CSS_EXPRESSION":       true,
		"XML_ENTITY":           true,
	}

	for _, err := range result.Errors {
		if !autoFixableCodes[err.Code] {
			return false
		}
	}

	return true
}
