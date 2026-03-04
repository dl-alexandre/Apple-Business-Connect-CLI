// Package svg provides SVG validation for BIMI (Brand Indicators for Message Identification)
// compliance with the Tiny Portable/Secure (Tiny-PS) profile.
//
// BIMI requires logos to be SVG files conforming to the Tiny-PS specification,
// which ensures they render correctly and safely across all email clients.
//
// Reference: https://bimigroup.org/implementation-guide/
package svg

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Validator performs SVG validation for BIMI compliance
type Validator struct {
	errors   []ValidationError
	warnings []ValidationWarning
}

// ValidationError represents a critical SVG validation error
type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// ValidationWarning represents a non-blocking SVG issue
type ValidationWarning struct {
	Field   string
	Message string
}

// ValidationResult holds SVG validation findings
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

// Dimensions represents SVG size information
type Dimensions struct {
	Width       float64
	Height      float64
	AspectRatio float64
	Valid       bool
}

// NewValidator creates a new SVG validator
func NewValidator() *Validator {
	return &Validator{
		errors:   make([]ValidationError, 0),
		warnings: make([]ValidationWarning, 0),
	}
}

// ValidateFile validates an SVG file for BIMI compliance
func (v *Validator) ValidateFile(filePath string) ValidationResult {
	result := ValidationResult{
		FilePath: filePath,
		Valid:    true,
	}

	// Check if it's a base64 embedded SVG
	if strings.HasPrefix(filePath, "data:image/svg+xml;base64,") {
		result.IsBase64 = true
		svgData, err := extractBase64SVG(filePath)
		if err != nil {
			v.addError("encoding", fmt.Sprintf("Failed to decode base64 SVG: %v", err), "INVALID_BASE64")
			result.Valid = false
			result.Errors = v.errors
			return result
		}
		return v.ValidateBytes(svgData, filePath)
	}

	// For remote URLs
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		svgData, err := downloadSVG(filePath)
		if err != nil {
			v.addError("download", fmt.Sprintf("Failed to download SVG: %v", err), "DOWNLOAD_ERROR")
			result.Valid = false
			result.Errors = v.errors
			return result
		}
		return v.ValidateBytes(svgData, filePath)
	}

	result.Errors = v.errors
	result.Warnings = v.warnings
	return result
}

// ValidateBytes validates SVG data from a byte slice
func (v *Validator) ValidateBytes(data []byte, source string) ValidationResult {
	result := ValidationResult{
		FilePath: source,
		Valid:    true,
	}

	// Check if it looks like SVG
	if !isSVG(data) {
		v.addError("format", "File does not appear to be a valid SVG", "INVALID_FORMAT")
		result.Valid = false
		result.Errors = v.errors
		return result
	}

	// Parse XML structure
	svg, err := parseSVG(data)
	if err != nil {
		v.addError("xml", fmt.Sprintf("Failed to parse SVG XML: %v", err), "XML_PARSE_ERROR")
		result.Valid = false
		result.Errors = v.errors
		return result
	}

	// Validate Tiny-PS compliance
	v.validateTinyPS(svg, data)

	// Check dimensions
	result.Dimensions = v.validateDimensions(svg)

	// Check for scripts and security issues
	result.HasScripts = v.checkForScripts(svg, data)
	result.HasExternalRefs = v.checkExternalReferences(svg, data)

	// Check aspect ratio (must be square for BIMI)
	if result.Dimensions.Valid {
		if result.Dimensions.AspectRatio < 0.99 || result.Dimensions.AspectRatio > 1.01 {
			v.addError("aspect_ratio",
				fmt.Sprintf("BIMI requires square aspect ratio (got %.2f:1)", result.Dimensions.AspectRatio),
				"INVALID_ASPECT_RATIO")
		}
	}

	result.TinyPSCompliant = len(v.errors) == 0
	result.Valid = len(v.errors) == 0
	result.Errors = v.errors
	result.Warnings = v.warnings

	return result
}

// SVGDocument represents the parsed SVG structure
type SVGDocument struct {
	XMLName xml.Name
	Width   string `xml:"width,attr"`
	Height  string `xml:"height,attr"`
	ViewBox string `xml:"viewBox,attr"`
}

func parseSVG(data []byte) (*SVGDocument, error) {
	var svg SVGDocument
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	if err := decoder.Decode(&svg); err != nil {
		return nil, err
	}
	return &svg, nil
}

func isSVG(data []byte) bool {
	content := strings.ToLower(string(data[:min(200, len(data))]))
	return strings.Contains(content, "<svg") || strings.Contains(content, "<?xml")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// validateTinyPS checks for Tiny-PS profile compliance
func (v *Validator) validateTinyPS(svg *SVGDocument, data []byte) {
	content := string(data)

	// Forbidden elements in Tiny-PS
	forbiddenElements := []string{
		"<script", "<foreignObject", "<iframe", "<embed", "<object",
		"<use", // <use> can reference external resources
	}

	for _, element := range forbiddenElements {
		if strings.Contains(content, element) {
			v.addError("security",
				fmt.Sprintf("Forbidden element found: %s (not allowed in Tiny-PS)", element),
				"FORBIDDEN_ELEMENT")
		}
	}

	// Check for external references
	externalPatterns := []string{
		`href="http`,
		`xlink:href="http`,
		`url\(https?://`,
		`@import`,
	}

	for _, pattern := range externalPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			v.addError("external_ref",
				fmt.Sprintf("External reference detected (pattern: %s)", pattern),
				"EXTERNAL_REFERENCE")
		}
	}

	// Check for inline styles that might contain CSS expressions
	if strings.Contains(content, "expression(") {
		v.addError("security", "CSS expression found (security risk)", "CSS_EXPRESSION")
	}

	// Check for entities (XXE risk)
	if strings.Contains(content, "<!ENTITY") {
		v.addError("security", "XML entity declaration found (XXE risk)", "XML_ENTITY")
	}
}

// validateDimensions extracts and validates SVG dimensions
func (v *Validator) validateDimensions(svg *SVGDocument) Dimensions {
	dim := Dimensions{Valid: false}

	// Parse width
	if svg.Width != "" {
		width, err := parseLength(svg.Width)
		if err == nil {
			dim.Width = width
		}
	}

	// Parse height
	if svg.Height != "" {
		height, err := parseLength(svg.Height)
		if err == nil {
			dim.Height = height
		}
	}

	// Try to get dimensions from viewBox
	if !dim.Valid && svg.ViewBox != "" {
		parts := strings.Fields(svg.ViewBox)
		if len(parts) == 4 {
			w, err1 := strconv.ParseFloat(parts[2], 64)
			h, err2 := strconv.ParseFloat(parts[3], 64)
			if err1 == nil && err2 == nil {
				dim.Width = w
				dim.Height = h
			}
		}
	}

	if dim.Width > 0 && dim.Height > 0 {
		dim.Valid = true
		dim.AspectRatio = dim.Width / dim.Height
	} else {
		v.addError("dimensions", "Could not determine SVG dimensions", "MISSING_DIMENSIONS")
	}

	// BIMI recommends minimum size
	if dim.Width < 32 || dim.Height < 32 {
		v.addWarning("dimensions", "SVG dimensions are very small (BIMI recommends at least 32x32)")
	}

	// Maximum size check (some email clients have limits)
	if dim.Width > 10000 || dim.Height > 10000 {
		v.addWarning("dimensions", "SVG dimensions are very large (may cause rendering issues)")
	}

	return dim
}

func parseLength(s string) (float64, error) {
	// Remove units (px, pt, em, etc.)
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "px")
	s = strings.TrimSuffix(s, "pt")
	s = strings.TrimSuffix(s, "em")
	s = strings.TrimSuffix(s, "rem")
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimSuffix(s, "in")
	s = strings.TrimSuffix(s, "cm")
	s = strings.TrimSuffix(s, "mm")

	return strconv.ParseFloat(s, 64)
}

// checkForScripts detects script content in SVG
func (v *Validator) checkForScripts(svg *SVGDocument, data []byte) bool {
	content := string(data)

	// Check for script tags
	if strings.Contains(content, "<script") {
		return true
	}

	// Check for event handlers
	eventHandlers := []string{
		"onclick=", "onload=", "onerror=", "onmouseover=",
		"onfocus=", "onblur=", "onchange=", "onsubmit=",
	}

	for _, handler := range eventHandlers {
		if strings.Contains(content, handler) {
			return true
		}
	}

	// Check for javascript: URLs
	if strings.Contains(content, "javascript:") {
		return true
	}

	return false
}

// checkExternalReferences detects external resource references
func (v *Validator) checkExternalReferences(svg *SVGDocument, data []byte) bool {
	content := string(data)
	lowerContent := strings.ToLower(content)

	// Check for http/https references
	if strings.Contains(lowerContent, "href=\"http") ||
		strings.Contains(lowerContent, "xlink:href=\"http") ||
		strings.Contains(lowerContent, "url(http") {
		return true
	}

	// Check for data URIs with external content
	if strings.Contains(content, "data:text/html") {
		return true
	}

	return false
}

// extractBase64SVG extracts SVG data from base64 data URI
func extractBase64SVG(dataURI string) ([]byte, error) {
	// Remove data URI prefix
	prefix := "data:image/svg+xml;base64,"
	if !strings.HasPrefix(dataURI, prefix) {
		return nil, fmt.Errorf("not a valid base64 SVG data URI")
	}

	base64Data := strings.TrimPrefix(dataURI, prefix)
	return base64.StdEncoding.DecodeString(base64Data)
}

// downloadSVG fetches SVG from remote URL
func downloadSVG(urlStr string) ([]byte, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "svg") && !strings.Contains(contentType, "xml") {
		// Some servers might not set correct content type, so we don't error here
		// but we could warn about it
	}

	return io.ReadAll(resp.Body)
}

func (v *Validator) addError(field, message, code string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

func (v *Validator) addWarning(field, message string) {
	v.warnings = append(v.warnings, ValidationWarning{
		Field:   field,
		Message: message,
	})
}

// PrintResults outputs SVG validation results
func (r ValidationResult) PrintResults() {
	fmt.Printf("\n🎨 SVG Validation for %s\n", r.FilePath)
	fmt.Println(strings.Repeat("─", 50))

	// Format
	if r.IsBase64 {
		fmt.Println("Format: Base64 embedded")
	} else {
		fmt.Println("Format: File/URL")
	}

	// Dimensions
	if r.Dimensions.Valid {
		fmt.Printf("\n📐 Dimensions: %.0f x %.0f (aspect: %.2f:1)\n",
			r.Dimensions.Width, r.Dimensions.Height, r.Dimensions.AspectRatio)
	}

	// Security checks
	fmt.Println("\n🔒 Security Scan:")
	if r.HasScripts {
		fmt.Printf("  ❌ Contains scripts/event handlers (NOT BIMI compliant)\n")
	} else {
		fmt.Printf("  ✅ No scripts detected\n")
	}

	if r.HasExternalRefs {
		fmt.Printf("  ❌ Contains external references (NOT BIMI compliant)\n")
	} else {
		fmt.Printf("  ✅ No external references\n")
	}

	// Tiny-PS compliance
	fmt.Println("\n📋 Tiny-PS Compliance:")
	if r.TinyPSCompliant {
		fmt.Println("  ✅ SVG is Tiny-PS compliant (BIMI ready)")
	} else {
		fmt.Println("  ❌ SVG is NOT Tiny-PS compliant")
	}

	// Errors and warnings
	if len(r.Errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for _, err := range r.Errors {
			fmt.Printf("   [%s] %s\n", err.Code, err.Message)
		}
	}

	if len(r.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, warn := range r.Warnings {
			fmt.Printf("   [%s] %s\n", warn.Field, warn.Message)
		}
	}

	fmt.Println("\n" + strings.Repeat("─", 50))
	if r.Valid {
		fmt.Println("✅ SVG is valid for BIMI usage")
	} else {
		fmt.Println("❌ SVG validation failed - fix errors before BIMI submission")
	}
}

// Summary returns a one-line summary
func (r ValidationResult) Summary() string {
	if !r.Valid {
		return fmt.Sprintf("Invalid: %d error(s), %d warning(s)", len(r.Errors), len(r.Warnings))
	}
	if len(r.Warnings) > 0 {
		return fmt.Sprintf("Valid with %d warning(s)", len(r.Warnings))
	}
	return "BIMI compliant"
}

// IsTinyPS checks if an SVG string is likely Tiny-PS compliant
func IsTinyPS(svgContent string) bool {
	v := NewValidator()
	result := v.ValidateBytes([]byte(svgContent), "inline")
	return result.TinyPSCompliant
}

// GetBIMIRequirements returns BIMI SVG requirements
func GetBIMIRequirements() []string {
	return []string{
		"Format: SVG Tiny Portable/Secure (Tiny-PS) profile",
		"Aspect Ratio: Must be square (1:1)",
		"Dimensions: Recommended minimum 32x32, maximum reasonable size",
		"No scripts: JavaScript or event handlers not allowed",
		"No external references: All resources must be embedded",
		"No animations: CSS or SMIL animations not allowed",
		"Security: Must not contain <foreignObject>, <iframe>, etc.",
		"Base64: Must use proper data:image/svg+xml;base64 encoding",
	}
}
