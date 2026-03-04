// Package dns provides DNS record validation for Apple Business Connect
// Branded Mail requirements (DMARC, DKIM, SPF checking)
//
// Future Roadmap:
//   - BIMI (Brand Indicators for Message Identification) validation
//   - SVG logo compliance checking (Tiny-PS profile)
//   - VMC (Verified Mark Certificate) validation
//
// As Apple continues to align with BIMI standards for Branded Mail,
// this package is positioned to expand into complete brand identity
// validation including logo format and certificate verification.
package dns

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Checker performs DNS record validation
type Checker struct {
	errors   []ValidationError
	warnings []ValidationWarning
}

// ValidationError represents a critical DNS validation error
type ValidationError struct {
	Record  string
	Message string
	Code    string
}

// ValidationWarning represents a non-blocking DNS issue
type ValidationWarning struct {
	Record  string
	Message string
}

// CheckResult holds DNS validation results
type CheckResult struct {
	Domain        string
	DMARC         DMARCRecord
	DKIM          []DKIMRecord
	SPF           SPFRecord
	BIMI          BIMIRecord
	Errors        []ValidationError
	Warnings      []ValidationWarning
	Valid         bool
	ReadyForApple bool
}

// DMARCRecord represents parsed DMARC record
type DMARCRecord struct {
	Raw             string
	Policy          string
	SubdomainPolicy string
	Percentage      int
	Present         bool
	Valid           bool
}

// DKIMRecord represents a DKIM record
type DKIMRecord struct {
	Selector string
	Raw      string
	Present  bool
	Valid    bool
}

// SPFRecord represents parsed SPF record
type SPFRecord struct {
	Raw        string
	Present    bool
	Valid      bool
	Mechanisms []string
}

// BIMIRecord represents parsed BIMI record and logo URL status
type BIMIRecord struct {
	Present       bool
	Raw           string
	LogoURL       string
	VMCURL        string
	URLAccessible bool
	StatusCode    int
	ContentType   string
	Error         string
}

// NewChecker creates a new DNS checker
func NewChecker() *Checker {
	return &Checker{
		errors:   make([]ValidationError, 0),
		warnings: make([]ValidationWarning, 0),
	}
}

// CheckDomain validates all DNS records for a domain
func (c *Checker) CheckDomain(domain string) CheckResult {
	result := CheckResult{
		Domain: domain,
		Valid:  true,
	}

	// Check DMARC
	result.DMARC = c.checkDMARC(domain)

	// Check common DKIM selectors
	result.DKIM = c.checkDKIM(domain)

	// Check SPF
	result.SPF = c.checkSPF(domain)

	// Check BIMI
	result.BIMI = c.checkBIMI(domain)

	// Aggregate results
	result.Errors = c.errors
	result.Warnings = c.warnings
	result.Valid = len(c.errors) == 0

	// Check if ready for Apple Branded Mail
	result.ReadyForApple = result.DMARC.Valid &&
		(result.DMARC.Policy == "quarantine" || result.DMARC.Policy == "reject") &&
		result.DMARC.Percentage == 100 &&
		len(result.DKIM) > 0

	return result
}

// checkDMARC validates DMARC record
func (c *Checker) checkDMARC(domain string) DMARCRecord {
	record := DMARCRecord{Present: false, Valid: false}

	// Query DMARC record (_dmarc.domain)
	txtRecords, err := net.LookupTXT("_dmarc." + domain)
	if err != nil {
		c.addError("DMARC", fmt.Sprintf("No DMARC record found for %s", domain), "MISSING_DMARC")
		return record
	}

	record.Present = true

	// Parse DMARC record
	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "v=DMARC1") {
			record.Raw = txt
			record.Valid = true

			// Extract policy (p=)
			if matches := regexp.MustCompile(`p=(\w+)`).FindStringSubmatch(txt); len(matches) > 1 {
				record.Policy = matches[1]
			}

			// Extract subdomain policy (sp=)
			if matches := regexp.MustCompile(`sp=(\w+)`).FindStringSubmatch(txt); len(matches) > 1 {
				record.SubdomainPolicy = matches[1]
			}

			// Extract percentage (pct=)
			record.Percentage = 100 // Default
			if matches := regexp.MustCompile(`pct=(\d+)`).FindStringSubmatch(txt); len(matches) > 1 {
				fmt.Sscanf(matches[1], "%d", &record.Percentage)
			}

			break
		}
	}

	if !record.Valid {
		c.addError("DMARC", "DMARC record found but invalid format", "INVALID_DMARC")
		return record
	}

	// Apple requires p=quarantine or p=reject
	if record.Policy != "quarantine" && record.Policy != "reject" {
		c.addError("DMARC",
			fmt.Sprintf("DMARC policy is '%s' but Apple requires 'quarantine' or 'reject'", record.Policy),
			"DMARC_POLICY_TOO_WEAK")
	}

	// Apple requires pct=100
	if record.Percentage != 100 {
		c.addWarning("DMARC",
			fmt.Sprintf("DMARC percentage is %d%%, Apple recommends 100%%", record.Percentage))
	}

	return record
}

// checkDKIM checks for DKIM selectors
func (c *Checker) checkDKIM(domain string) []DKIMRecord {
	// Common DKIM selectors to check
	selectors := []string{"default", "dkim", "mail", "google", "selector1", "selector2", "k1", "smtp"}
	var records []DKIMRecord

	for _, selector := range selectors {
		txtRecords, err := net.LookupTXT(selector + "._domainkey." + domain)
		if err != nil {
			continue
		}

		for _, txt := range txtRecords {
			if strings.HasPrefix(txt, "v=DKIM1") || strings.Contains(txt, "k=rsa") {
				records = append(records, DKIMRecord{
					Selector: selector,
					Raw:      txt,
					Present:  true,
					Valid:    true,
				})
				break
			}
		}
	}

	if len(records) == 0 {
		c.addError("DKIM", "No DKIM records found. Apple requires DKIM for Branded Mail.", "MISSING_DKIM")
	}

	return records
}

// checkSPF validates SPF record
func (c *Checker) checkSPF(domain string) SPFRecord {
	record := SPFRecord{Present: false, Valid: false}

	txtRecords, err := net.LookupTXT(domain)
	if err != nil {
		c.addWarning("SPF", fmt.Sprintf("Could not query SPF records: %v", err))
		return record
	}

	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "v=spf1") {
			record.Present = true
			record.Raw = txt
			record.Valid = true

			// Parse mechanisms
			parts := strings.Fields(txt)
			for _, part := range parts {
				if part != "v=spf1" && !strings.HasPrefix(part, "+") {
					record.Mechanisms = append(record.Mechanisms, part)
				}
			}

			break
		}
	}

	if !record.Present {
		c.addWarning("SPF", "No SPF record found. While not strictly required by Apple, it's recommended.")
	}

	return record
}

// checkBIMI checks for BIMI record and validates logo URL
func (c *Checker) checkBIMI(domain string) BIMIRecord {
	record := BIMIRecord{Present: false, URLAccessible: false}

	// Query BIMI record (default._bimi.domain)
	txtRecords, err := net.LookupTXT("default._bimi." + domain)
	if err != nil {
		// BIMI is optional, so no error - just return empty
		return record
	}

	record.Present = true

	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "v=BIMI1") {
			record.Raw = txt

			// Extract logo URL (l=)
			if matches := regexp.MustCompile(`l=(https?://[^;\s]+)`).FindStringSubmatch(txt); len(matches) > 1 {
				record.LogoURL = matches[1]
			}

			// Extract VMC URL (a=) - optional
			if matches := regexp.MustCompile(`a=(https?://[^;\s]+)`).FindStringSubmatch(txt); len(matches) > 1 {
				record.VMCURL = matches[1]
			}

			break
		}
	}

	// If we have a logo URL, validate it's accessible
	if record.LogoURL != "" {
		record = c.validateLogoURL(record)
	}

	return record
}

// validateLogoURL performs HTTP check to verify logo accessibility
func (c *Checker) validateLogoURL(record BIMIRecord) BIMIRecord {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow redirects but limit to prevent loops
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// Perform HEAD request first (lighter than GET)
	resp, err := client.Head(record.LogoURL)
	if err != nil {
		// Try GET if HEAD fails (some servers don't support HEAD)
		resp, err = client.Get(record.LogoURL)
		if err != nil {
			record.Error = fmt.Sprintf("Cannot access logo URL: %v", err)
			c.addWarning("BIMI", record.Error)
			return record
		}
		defer resp.Body.Close()
	}

	record.StatusCode = resp.StatusCode

	// Check status code
	if resp.StatusCode != http.StatusOK {
		record.Error = fmt.Sprintf("Logo URL returned HTTP %d (expected 200)", resp.StatusCode)

		// Special handling for common errors
		switch resp.StatusCode {
		case http.StatusNotFound:
			record.Error = "Logo URL returned 404 Not Found - CDN URL may have changed"
			c.addError("BIMI", record.Error, "BIMI_LOGO_404")
		case http.StatusForbidden:
			record.Error = "Logo URL returned 403 Forbidden - access may be restricted"
			c.addWarning("BIMI", record.Error)
		default:
			c.addWarning("BIMI", record.Error)
		}
		return record
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	record.ContentType = contentType

	if contentType != "" && !strings.Contains(contentType, "svg") {
		c.addWarning("BIMI",
			fmt.Sprintf("Logo URL content-type is '%s' (expected 'image/svg+xml')", contentType))
	}

	// Check content length (BIMI limit is typically 32KB)
	if resp.ContentLength > 0 && resp.ContentLength > 32768 {
		c.addWarning("BIMI",
			fmt.Sprintf("Logo file size is %d bytes (exceeds 32KB BIMI limit)", resp.ContentLength))
	}

	record.URLAccessible = true
	return record
}

func (c *Checker) addError(record, message, code string) {
	c.errors = append(c.errors, ValidationError{
		Record:  record,
		Message: message,
		Code:    code,
	})
}

func (c *Checker) addWarning(record, message string) {
	c.warnings = append(c.warnings, ValidationWarning{
		Record:  record,
		Message: message,
	})
}

// PrintResults outputs DNS validation results
func (r CheckResult) PrintResults() {
	fmt.Printf("\n📧 DNS Trust Stack Check for %s\n", r.Domain)
	fmt.Println(strings.Repeat("─", 50))

	// DMARC Status
	fmt.Println("\n🔒 DMARC (Domain-based Message Authentication)")
	if r.DMARC.Present {
		fmt.Printf("  Status: %s\n", getStatusString(r.DMARC.Valid))
		fmt.Printf("  Policy: %s\n", r.DMARC.Policy)
		fmt.Printf("  Percentage: %d%%\n", r.DMARC.Percentage)
		if r.DMARC.SubdomainPolicy != "" {
			fmt.Printf("  Subdomain Policy: %s\n", r.DMARC.SubdomainPolicy)
		}
		if !r.DMARC.Valid || (r.DMARC.Policy != "quarantine" && r.DMARC.Policy != "reject") {
			fmt.Printf("  ⚠️  Apple Requirement: Policy must be 'quarantine' or 'reject'\n")
		}
	} else {
		fmt.Printf("  Status: ❌ Not Found\n")
		fmt.Printf("  ⚠️  Apple Requirement: DMARC is mandatory for Branded Mail\n")
	}

	// DKIM Status
	fmt.Println("\n🔑 DKIM (DomainKeys Identified Mail)")
	if len(r.DKIM) > 0 {
		fmt.Printf("  Status: ✅ Found (%d selector(s))\n", len(r.DKIM))
		for _, dkim := range r.DKIM {
			fmt.Printf("  - Selector: %s\n", dkim.Selector)
		}
	} else {
		fmt.Printf("  Status: ❌ Not Found\n")
		fmt.Printf("  ⚠️  Apple Requirement: DKIM is mandatory for Branded Mail\n")
	}

	// SPF Status
	fmt.Println("\n📨 SPF (Sender Policy Framework)")
	if r.SPF.Present {
		fmt.Printf("  Status: %s\n", getStatusString(r.SPF.Valid))
		if len(r.SPF.Mechanisms) > 0 {
			fmt.Printf("  Mechanisms: %s\n", strings.Join(r.SPF.Mechanisms, ", "))
		}
	} else {
		fmt.Printf("  Status: ⚠️  Not Found (Recommended but not required)\n")
	}

	// BIMI Status (if present)
	if r.BIMI.Present {
		fmt.Println("\n🎨 BIMI (Brand Indicators for Message Identification)")
		fmt.Printf("  Status: ✅ Found\n")

		if r.BIMI.LogoURL != "" {
			fmt.Printf("  Logo URL: %s\n", r.BIMI.LogoURL)

			if r.BIMI.URLAccessible {
				fmt.Printf("  Logo Access: ✅ HTTP %d (Accessible)\n", r.BIMI.StatusCode)
				if r.BIMI.ContentType != "" {
					fmt.Printf("  Content-Type: %s\n", r.BIMI.ContentType)
				}
			} else {
				fmt.Printf("  Logo Access: ❌ %s\n", r.BIMI.Error)
			}
		}

		if r.BIMI.VMCURL != "" {
			fmt.Printf("  VMC URL: %s\n", r.BIMI.VMCURL)
		}
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("─", 50))
	if r.ReadyForApple {
		fmt.Println("✅ Domain is READY for Apple Branded Mail!")
	} else {
		fmt.Println("❌ Domain is NOT ready for Apple Branded Mail")
		fmt.Println("   Fix the errors above before submitting to Apple")
	}

	if len(r.Errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for _, err := range r.Errors {
			fmt.Printf("   [%s] %s\n", err.Code, err.Message)
		}
	}

	if len(r.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, warn := range r.Warnings {
			fmt.Printf("   [%s] %s\n", warn.Record, warn.Message)
		}
	}
}

func getStatusString(valid bool) string {
	if valid {
		return "✅ Valid"
	}
	return "❌ Invalid"
}

// IsValidDomain checks if a string is a valid domain format
func IsValidDomain(domain string) bool {
	// Simple domain validation regex
	pattern := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	return pattern.MatchString(domain)
}

// GetAppleVerificationRecord generates Apple verification TXT record format
func GetAppleVerificationRecord(verificationID string) string {
	return fmt.Sprintf("apple-domain-verification=%s", verificationID)
}

// TODO: Future BIMI (Brand Indicators for Message Identification) Support
// As Apple aligns with BIMI standards, implement the following:
//
// 1. BIMI Record Validation
//    - Check for default._bimi.domain TXT record
//    - Parse version, logo URL, and optional VMC URL
//
// 2. SVG Logo Validation (Tiny-PS Profile)
//    - Verify SVG is Tiny Portable/Secure profile compliant
//    - Check for forbidden elements (scripts, external references)
//    - Validate base64 encoding if embedded
//    - Check dimensions (square aspect ratio required)
//
// 3. VMC (Verified Mark Certificate) Support
//    - Validate certificate chain
//    - Check for mark-validation entity certificate
//    - Verify logo hash matches certificate
//
// 4. DNS Record Structure
//    - v=BIMI1; l=https://example.com/logo.svg; a=https://example.com/vmc.pem
//
// Reference: https://bimigroup.org/implementation-guide/
// This positions the CLI as the complete brand identity validator for Apple's ecosystem.
