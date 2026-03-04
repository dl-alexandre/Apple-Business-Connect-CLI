// Package svg provides SVG validation and automated remediation for BIMI compliance
// with the Tiny Portable/Secure (Tiny-PS) profile.
package svg

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// FixOptions controls automated remediation behavior
type FixOptions struct {
	RemoveScripts       bool  // Remove <script> elements
	RemoveEventHandlers bool  // Remove onclick, onload, etc.
	FixHeader           bool  // Fix version, baseProfile, and attributes
	InjectTitle         bool  // Add missing title element
	RemoveMetadata      bool  // Remove editor metadata
	StripAttributes     bool  // Remove x, y, overflow from root svg
	FixAspectRatio      bool  // Normalize viewBox to square with letterboxing
	RemoveExternalRefs  bool  // Remove external references
	CompressPaths       bool  // Reduce decimal precision in path data
	CompressPrecision   int   // Decimal places for paths (default: 2)
	MaxFileSize         int64 // Max file size in bytes (default: 32768)
	OutputPath          string
	OutputDir           string
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
		CompressPaths:       true,
		CompressPrecision:   2,
		MaxFileSize:         32768,
	}
}

// FixResult holds the outcome of SVG remediation
type FixResult struct {
	OriginalPath  string
	FixedPath     string
	Changes       []string
	Errors        []error
	Success       bool
	OriginalSize  int64
	FixedSize     int64
	SizeReduction float64
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
		OriginalSize: int64(len(svgData)),
	}

	content := string(svgData)
	originalContent := content

	// 1. Fix header (version, baseProfile)
	if f.options.FixHeader {
		content = f.fixHeader(content)
		if content != originalContent {
			result.Changes = append(result.Changes, "Fixed SVG header (version=1.2, baseProfile=tiny-ps)")
		}
	}

	// 2. Strip prohibited attributes from root svg
	if f.options.StripAttributes {
		content = f.stripRootAttributes(content)
		if content != originalContent {
			result.Changes = append(result.Changes, "Removed prohibited attributes (x, y, overflow)")
		}
	}

	// 3. Inject title if missing
	if f.options.InjectTitle {
		content = f.injectTitle(content, source)
		if content != originalContent {
			result.Changes = append(result.Changes, "Added missing title element")
		}
	}

	// 4. Remove metadata and editor namespaces
	if f.options.RemoveMetadata {
		content = f.removeMetadata(content)
		if content != originalContent {
			result.Changes = append(result.Changes, "Removed metadata and editor namespaces")
		}
	}

	// 5. Remove scripts
	if f.options.RemoveScripts {
		content = f.removeScripts(content)
		if content != originalContent {
			result.Changes = append(result.Changes, "Removed script elements and event handlers")
		}
	}

	// 6. Remove external references
	if f.options.RemoveExternalRefs {
		content = f.removeExternalReferences(content)
		if content != originalContent {
			result.Changes = append(result.Changes, "Removed external references")
		}
	}

	// 7. Fix aspect ratio with letterboxing
	if f.options.FixAspectRatio {
		content = f.fixAspectRatioWithLetterboxing(content)
		if content != originalContent {
			result.Changes = append(result.Changes, "Normalized to square aspect ratio with letterboxing")
		}
	}

	// 8. Remove forbidden elements
	content = f.removeForbiddenElements(content)
	if content != originalContent {
		result.Changes = append(result.Changes, "Removed forbidden elements (foreignObject, iframe, etc.)")
	}

	// 9. Compress path data if enabled
	if f.options.CompressPaths {
		originalLen := len(content)
		content = f.compressPathData(content, f.options.CompressPrecision)
		if len(content) < originalLen {
			bytesSaved := originalLen - len(content)
			result.Changes = append(result.Changes,
				fmt.Sprintf("Compressed path data (saved %d bytes)", bytesSaved))
		}
	}

	// 10. Clean up XML
	content = f.cleanupXML(content)

	result.FixedSize = int64(len(content))
	result.SizeReduction = float64(result.OriginalSize-result.FixedSize) / float64(result.OriginalSize) * 100

	// Check file size
	if result.FixedSize > f.options.MaxFileSize {
		result.Errors = append(result.Errors,
			fmt.Errorf("file size %d exceeds BIMI 32KB limit", result.FixedSize))
	}

	// Write output if requested
	if f.options.OutputPath != "" && content != originalContent {
		err := os.WriteFile(f.options.OutputPath, []byte(content), 0644)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to write: %w", err))
		} else {
			result.FixedPath = f.options.OutputPath
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
			results = append(results, FixResult{
				OriginalPath: file,
				Errors:       []error{fmt.Errorf("failed to read: %w", err)},
				Success:      false,
			})
			continue
		}

		if outputDir != "" {
			baseName := filepath.Base(file)
			ext := filepath.Ext(baseName)
			name := strings.TrimSuffix(baseName, ext)
			f.options.OutputPath = filepath.Join(outputDir, name+"-fixed"+ext)
		}

		results = append(results, f.Fix(data, file))
	}

	return results
}

// fixHeader ensures proper version="1.2" and baseProfile="tiny-ps"
func (f *Fixer) fixHeader(content string) string {
	// Add XML declaration if missing
	if !strings.HasPrefix(content, "<?xml") {
		content = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" + content
	}

	// Fix svg tag attributes
	svgRegex := regexp.MustCompile("<svg([^>]*)>")
	content = svgRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Fix version
		if !strings.Contains(match, "version=\"") {
			match = strings.Replace(match, ">", " version=\"1.2\">", 1)
		} else {
			match = regexp.MustCompile("version=\"[^\"]*\"").ReplaceAllString(match, "version=\"1.2\"")
		}

		// Fix baseProfile
		if !strings.Contains(match, "baseProfile=\"tiny-ps\"") {
			if strings.Contains(match, "baseProfile=\"") {
				match = regexp.MustCompile("baseProfile=\"[^\"]*\"").ReplaceAllString(match, "baseProfile=\"tiny-ps\"")
			} else {
				match = strings.Replace(match, ">", " baseProfile=\"tiny-ps\">", 1)
			}
		}

		return match
	})

	return content
}

// stripRootAttributes removes x, y, overflow from root svg tag
func (f *Fixer) stripRootAttributes(content string) string {
	// Match first svg tag only (the root)
	svgRegex := regexp.MustCompile("(?s)^\\s*<svg([^>]*)>")
	return svgRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Remove prohibited attributes
		match = regexp.MustCompile("\\sx=\"[^\"]*\"").ReplaceAllString(match, "")
		match = regexp.MustCompile("\\sy=\"[^\"]*\"").ReplaceAllString(match, "")
		match = regexp.MustCompile("\\soverflow=\"[^\"]*\"").ReplaceAllString(match, "")
		return match
	})
}

// injectTitle adds a title element if missing
func (f *Fixer) injectTitle(content string, source string) string {
	if strings.Contains(content, "<title>") || strings.Contains(content, "<title ") {
		return content
	}

	// Generate title from filename
	title := filepath.Base(source)
	title = strings.TrimSuffix(title, filepath.Ext(title))
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.Title(title)

	// Insert after opening svg tag
	svgRegex := regexp.MustCompile("(<svg[^>]*>)")
	return svgRegex.ReplaceAllString(content, fmt.Sprintf("$1\n  <title>%s</title>", title))
}

// removeMetadata strips metadata, editor namespaces, and comments
func (f *Fixer) removeMetadata(content string) string {
	// Remove metadata elements
	metadataRegex := regexp.MustCompile("(?s)<metadata[^>]*>.*?</metadata>")
	content = metadataRegex.ReplaceAllString(content, "")

	// Remove editor namespaces (Adobe, Inkscape, Sketch)
	content = regexp.MustCompile("\\sxmlns:[a-z]+=\"http://ns\\.adobe\\.com/[^\"]*\"").ReplaceAllString(content, "")
	content = regexp.MustCompile("\\sxmlns:[a-z]+=\"http://www\\.inkscape\\.org/[^\"]*\"").ReplaceAllString(content, "")
	content = regexp.MustCompile("\\sxmlns:[a-z]+=\"http://www\\.sketch\\.com/[^\"]*\"").ReplaceAllString(content, "")

	// Remove editor-specific attributes
	content = regexp.MustCompile("\\s[a-z]+:[a-z]+=\"[^\"]*\"").ReplaceAllString(content, "")

	// Remove XML comments
	commentRegex := regexp.MustCompile("(?s)<!--.*?-->")
	content = commentRegex.ReplaceAllString(content, "")

	return content
}

// removeScripts removes script elements and event handlers
func (f *Fixer) removeScripts(content string) string {
	// Remove script tags
	scriptRegex := regexp.MustCompile("(?s)<script[^>]*>.*?</script>")
	content = scriptRegex.ReplaceAllString(content, "")

	// Remove event handlers
	handlers := []string{
		"on[a-z]+=\"[^\"]*\"",
		"on[a-z]+='[^']*'",
	}
	for _, h := range handlers {
		re := regexp.MustCompile(h)
		content = re.ReplaceAllString(content, "")
	}

	// Remove javascript: URLs
	content = regexp.MustCompile("href=\"javascript:[^\"]*\"").ReplaceAllString(content, "href=\"#\"")

	return content
}

// removeExternalReferences removes external resource references
func (f *Fixer) removeExternalReferences(content string) string {
	// Remove external hrefs
	content = regexp.MustCompile("(href|xlink:href)=\"https?://[^\"]*\"").ReplaceAllString(content, "")
	// Remove url() references
	content = regexp.MustCompile("url\\(https?://[^)]+\\)").ReplaceAllString(content, "")
	return content
}

// removeForbiddenElements removes non-Tiny-PS elements
func (f *Fixer) removeForbiddenElements(content string) string {
	elements := []string{
		"(?s)<foreignObject[^>]*>.*?</foreignObject>",
		"(?s)<iframe[^>]*>.*?</iframe>",
		"(?s)<embed[^>]*>.*?</embed>",
		"(?s)<object[^>]*>.*?</object>",
	}

	for _, e := range elements {
		re := regexp.MustCompile(e)
		content = re.ReplaceAllString(content, "")
	}

	return content
}

// fixAspectRatioWithLetterboxing creates square aspect ratio with letterboxing
func (f *Fixer) fixAspectRatioWithLetterboxing(content string) string {
	// Extract current dimensions from viewBox
	viewBoxRegex := regexp.MustCompile("viewBox=\"([^\"]*)\"")
	match := viewBoxRegex.FindStringSubmatch(content)

	if len(match) < 2 {
		// No viewBox, add default square one
		svgRegex := regexp.MustCompile("(<svg[^>]*)>")
		return svgRegex.ReplaceAllString(content, "$1 viewBox=\"0 0 100 100\">")
	}

	parts := strings.Fields(match[1])
	if len(parts) != 4 {
		return content // Invalid viewBox
	}

	minX, _ := strconv.ParseFloat(parts[0], 64)
	minY, _ := strconv.ParseFloat(parts[1], 64)
	width, _ := strconv.ParseFloat(parts[2], 64)
	height, _ := strconv.ParseFloat(parts[3], 64)

	if width == 0 || height == 0 {
		return content
	}

	// Check if already square
	if width == height {
		return content
	}

	// Calculate new square dimensions
	newSize := width
	if height > newSize {
		newSize = height
	}

	// Calculate offsets for centering (letterboxing)
	offsetX := (newSize - width) / 2
	offsetY := (newSize - height) / 2

	newViewBox := fmt.Sprintf("%.2f %.2f %.2f %.2f", minX-offsetX, minY-offsetY, newSize, newSize)
	content = viewBoxRegex.ReplaceAllString(content, fmt.Sprintf("viewBox=\"%s\"", newViewBox))

	// Update width and height attributes to match
	content = regexp.MustCompile("width=\"[^\"]*\"").ReplaceAllString(content, fmt.Sprintf("width=\"%.0f\"", newSize))
	content = regexp.MustCompile("height=\"[^\"]*\"").ReplaceAllString(content, fmt.Sprintf("height=\"%.0f\"", newSize))

	return content
}

// compressPathData reduces decimal precision in path data
func (f *Fixer) compressPathData(content string, precision int) string {
	// Pattern to match decimal numbers in path data
	// Matches numbers like 45.123456 in d="M45.123456 10.987654"
	pattern := fmt.Sprintf("(\\d+)\\.\\d{%d,}", precision+1)
	decimalRegex := regexp.MustCompile(pattern)

	content = decimalRegex.ReplaceAllStringFunc(content, func(match string) string {
		if val, err := strconv.ParseFloat(match, 64); err == nil {
			format := fmt.Sprintf("%%.%df", precision)
			return fmt.Sprintf(format, val)
		}
		return match
	})

	return content
}

// cleanupXML removes unnecessary whitespace
func (f *Fixer) cleanupXML(content string) string {
	content = regexp.MustCompile("\n\\s*\n").ReplaceAllString(content, "\n")
	return strings.TrimSpace(content)
}

// PrintFixResults outputs remediation results
func (r FixResult) PrintFixResults() {
	fmt.Printf("\n🔧 SVG Remediation Results\n")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Original: %s\n", r.OriginalPath)
	if r.FixedPath != "" {
		fmt.Printf("Fixed:    %s\n", r.FixedPath)
	}

	fmt.Printf("\nSize: %d bytes → %d bytes (%.1f%% reduction)\n",
		r.OriginalSize, r.FixedSize, r.SizeReduction)

	if r.FixedSize > 32768 {
		fmt.Printf("  ⚠️  WARNING: Still exceeds 32KB limit!\n")
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
		fmt.Println("✅ SVG is now BIMI compliant!")
	} else {
		fmt.Println("⚠️  Some issues require manual fixing")
	}
}

// PrintBatchResults outputs batch processing summary
func PrintBatchResults(results []FixResult) {
	success, failed := 0, 0
	var totalOrig, totalFixed int64

	for _, r := range results {
		if r.Success {
			success++
		} else {
			failed++
		}
		totalOrig += r.OriginalSize
		totalFixed += r.FixedSize
	}

	fmt.Printf("\n📊 Batch Summary\n")
	fmt.Println(strings.Repeat("═", 50))
	fmt.Printf("Files:     %d total (%d success, %d failed)\n", len(results), success, failed)
	fmt.Printf("Size:      %.1f KB → %.1f KB (%.1f%% reduction)\n",
		float64(totalOrig)/1024, float64(totalFixed)/1024,
		float64(totalOrig-totalFixed)/float64(totalOrig)*100)
}
