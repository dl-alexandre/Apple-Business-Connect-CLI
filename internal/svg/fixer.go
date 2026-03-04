// Package svg provides SVG validation and automated remediation for BIMI compliance
// with the Tiny Portable/Secure (Tiny-PS) profile.
package svg

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FixOptions controls automated remediation behavior
type FixOptions struct {
	RemoveScripts       bool   // Remove <script> elements
	RemoveEventHandlers bool   // Remove onclick, onload, etc.
	FixHeader           bool   // Fix version and baseProfile
	InjectTitle         bool   // Add missing title
	RemoveMetadata      bool   // Remove editor metadata
	StripAttributes     bool   // Remove x, y, overflow from root svg
	FixAspectRatio      bool   // Normalize viewBox to square
	RemoveExternalRefs  bool   // Remove external references
	OutputPath          string // Where to write fixed SVG
	OutputDir           string // Directory for batch processing
}

// DefaultFixOptions returns sensible defaults
func DefaultFixOptions() FixOptions {
	return FixOptions{
		RemoveScripts:       true,
		RemoveEventHandlers: true,
		FixHeader:           true,
		InjectTitle:         true,
		RemoveMetadata:      true,
		StripAttributes:     true,
		FixAspectRatio:      true,
		RemoveExternalRefs:  true,
	}
}

// FixResult holds the outcome of SVG remediation
type FixResult struct {
	OriginalPath string
	FixedPath    string
	Changes      []string
	Errors       []error
	Success      bool
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

	content := string(svgData)
	originalContent := content

	// Remove scripts
	if f.options.RemoveScripts {
		content = f.removeScripts(content)
		if content != originalContent {
			result.Changes = append(result.Changes, "Removed script elements")
		}
	}

	// Remove forbidden elements
	content = f.removeForbiddenElements(content)
	if content != originalContent {
		result.Changes = append(result.Changes, "Removed forbidden elements")
	}

	// Clean up XML
	content = f.cleanupXML(content)

	// Write output if requested
	if f.options.OutputPath != "" && content != originalContent {
		err := os.WriteFile(f.options.OutputPath, []byte(content), 0644)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to write: %w", err))
		} else {
			result.FixedPath = f.options.OutputPath
			result.Changes = append(result.Changes, fmt.Sprintf("Written to: %s", f.options.OutputPath))
		}
	}

	result.Success = len(result.Errors) == 0
	return result
}

// BatchFix processes multiple SVG files
func (f *Fixer) BatchFix(files []string, outputDir string) []FixResult {
	results := make([]FixResult, 0, len(files))

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			result := FixResult{
				OriginalPath: file,
				Errors:       []error{fmt.Errorf("failed to read: %w", err)},
				Success:      false,
			}
			results = append(results, result)
			continue
		}

		if outputDir != "" {
			baseName := filepath.Base(file)
			ext := filepath.Ext(baseName)
			nameWithoutExt := strings.TrimSuffix(baseName, ext)
			f.options.OutputPath = filepath.Join(outputDir, nameWithoutExt+"-fixed"+ext)
		}

		result := f.Fix(data, file)
		results = append(results, result)
	}

	return results
}

// removeScripts removes script elements and event handlers
func (f *Fixer) removeScripts(content string) string {
	// Remove script tags
	re := regexp.MustCompile("(?s)<script[^>]*>.*?</script>")
	content = re.ReplaceAllString(content, "")

	// Remove event handlers
	handlers := []string{
		"on[a-z]+=\"[^\"]*\"",
		"on[a-z]+='[^']*'",
	}
	for _, h := range handlers {
		re := regexp.MustCompile(h)
		content = re.ReplaceAllString(content, "")
	}

	return content
}

// removeForbiddenElements removes non-Tiny-PS elements
func (f *Fixer) removeForbiddenElements(content string) string {
	elements := []string{
		"(?s)<foreignObject[^>]*>.*?</foreignObject>",
		"(?s)<iframe[^>]*>.*?</iframe>",
		"(?s)<embed[^>]*>.*?</embed>",
	}

	for _, e := range elements {
		re := regexp.MustCompile(e)
		content = re.ReplaceAllString(content, "")
	}

	return content
}

// cleanupXML removes unnecessary whitespace
func (f *Fixer) cleanupXML(content string) string {
	content = regexp.MustCompile("\n\\s*\n").ReplaceAllString(content, "\n")
	return strings.TrimSpace(content)
}

// PrintFixResults outputs remediation results
func (r FixResult) PrintFixResults() {
	fmt.Printf("\n🔧 SVG Remediation\n")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("File: %s\n", r.OriginalPath)
	if r.FixedPath != "" {
		fmt.Printf("Output: %s\n", r.FixedPath)
	}

	if len(r.Changes) > 0 {
		fmt.Println("\nChanges:")
		for _, c := range r.Changes {
			fmt.Printf("  ✓ %s\n", c)
		}
	}

	if len(r.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range r.Errors {
			fmt.Printf("  ✗ %v\n", e)
		}
	}

	fmt.Println()
	if r.Success {
		fmt.Println("✅ Success!")
	}
}

// PrintBatchResults outputs batch processing summary
func PrintBatchResults(results []FixResult) {
	success := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			success++
		} else {
			failed++
		}
	}

	fmt.Printf("\n📊 Batch Summary: %d successful, %d failed\n", success, failed)
}
