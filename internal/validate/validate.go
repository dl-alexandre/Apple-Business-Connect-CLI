// Package validate provides pre-flight validation for Apple Business Connect data.
// It ensures data integrity before API submission to prevent manual review delays.
package validate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Validator performs pre-flight checks on location data
type Validator struct {
	errors   []ValidationError
	warnings []ValidationWarning
}

// ValidationError represents a critical error that will block sync
type ValidationError struct {
	Field   string
	Record  string
	Message string
	Code    string
}

// ValidationWarning represents a non-blocking issue
type ValidationWarning struct {
	Field   string
	Record  string
	Message string
}

// Result holds all validation findings
type Result struct {
	Errors      []ValidationError
	Warnings    []ValidationWarning
	Valid       bool
	RecordCount int
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		errors:   make([]ValidationError, 0),
		warnings: make([]ValidationWarning, 0),
	}
}

// ValidateRecord performs full validation on a single location record
func (v *Validator) ValidateRecord(record interface{}, identifier string) {
	switch r := record.(type) {
	case LocationRecord:
		v.validateLocationRecord(r, identifier)
	default:
		v.addError("unknown", identifier, "unsupported record type", "INVALID_TYPE")
	}
}

// LocationRecord represents a location to be validated
type LocationRecord struct {
	PartnerID  string
	Name       string
	Street     string
	City       string
	Region     string
	PostalCode string
	Country    string
	Phone      string
	Category   string
	Latitude   string
	Longitude  string
}

func (v *Validator) validateLocationRecord(r LocationRecord, identifier string) {
	// Required fields
	if strings.TrimSpace(r.Name) == "" {
		v.addError("name", identifier, "location name is required", "MISSING_NAME")
	}
	if strings.TrimSpace(r.Street) == "" {
		v.addError("street", identifier, "street address is required", "MISSING_STREET")
	}
	if strings.TrimSpace(r.City) == "" {
		v.addError("city", identifier, "city is required", "MISSING_CITY")
	}
	if strings.TrimSpace(r.Region) == "" {
		v.addError("region", identifier, "state/region is required", "MISSING_REGION")
	}
	if strings.TrimSpace(r.PostalCode) == "" {
		v.addError("postal_code", identifier, "postal code is required", "MISSING_POSTAL")
	}
	if strings.TrimSpace(r.Country) == "" {
		v.addError("country", identifier, "country is required", "MISSING_COUNTRY")
	}

	// Validate country code
	if r.Country != "" && !isValidCountryCode(r.Country) {
		v.addError("country", identifier,
			fmt.Sprintf("invalid country code '%s' (expected 2-letter ISO code like 'US', 'CA', 'GB')", r.Country),
			"INVALID_COUNTRY")
	}

	// Validate postal code format
	if r.PostalCode != "" && r.Country != "" {
		if !isValidPostalCode(r.PostalCode, r.Country) {
			v.addWarning("postal_code", identifier,
				fmt.Sprintf("postal code '%s' may be invalid for %s", r.PostalCode, r.Country))
		}
	}

	// Validate coordinates if provided
	if r.Latitude != "" || r.Longitude != "" {
		v.validateCoordinates(r.Latitude, r.Longitude, identifier)
	}

	// Validate phone number format
	if r.Phone != "" && !isValidPhone(r.Phone) {
		v.addWarning("phone", identifier,
			fmt.Sprintf("phone number '%s' format may be invalid (expected E.164 format like +1-415-555-0100)", r.Phone))
	}

	// Validate category
	if r.Category != "" && !isValidCategory(r.Category) {
		v.addError("category", identifier,
			fmt.Sprintf("unknown category '%s' (see valid categories with 'abc categories list')", r.Category),
			"INVALID_CATEGORY")
	}

	// Check for suspicious data patterns
	if len(r.Name) < 2 {
		v.addWarning("name", identifier, "location name seems very short")
	}
	if len(r.Street) < 5 {
		v.addWarning("street", identifier, "street address seems very short")
	}
}

// validateCoordinates checks latitude and longitude values
func (v *Validator) validateCoordinates(latStr, lonStr, identifier string) {
	if latStr == "" || lonStr == "" {
		v.addError("coordinates", identifier, "both latitude and longitude must be provided together", "INCOMPLETE_COORDS")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		v.addError("latitude", identifier, fmt.Sprintf("latitude '%s' is not a valid number", latStr), "INVALID_LATITUDE")
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		v.addError("longitude", identifier, fmt.Sprintf("longitude '%s' is not a valid number", lonStr), "INVALID_LONGITUDE")
		return
	}

	// Latitude must be between -90 and 90
	if lat < -90 || lat > 90 {
		v.addError("latitude", identifier,
			fmt.Sprintf("latitude %f is out of valid range (-90 to 90)", lat),
			"LATITUDE_OUT_OF_RANGE")
	}

	// Longitude must be between -180 and 180
	if lon < -180 || lon > 180 {
		v.addError("longitude", identifier,
			fmt.Sprintf("longitude %f is out of valid range (-180 to 180)", lon),
			"LONGITUDE_OUT_OF_RANGE")
	}

	// Check for suspicious coordinates (0,0 is often a default/error)
	if lat == 0 && lon == 0 {
		v.addWarning("coordinates", identifier, "coordinates are 0,0 (Null Island) - this is likely an error")
	}
}

// addError adds a validation error
func (v *Validator) addError(field, record, message, code string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Record:  record,
		Message: message,
		Code:    code,
	})
}

// addWarning adds a validation warning
func (v *Validator) addWarning(field, record, message string) {
	v.warnings = append(v.warnings, ValidationWarning{
		Field:   field,
		Record:  record,
		Message: message,
	})
}

// GetResult returns the validation result
func (v *Validator) GetResult(recordCount int) Result {
	return Result{
		Errors:      v.errors,
		Warnings:    v.warnings,
		Valid:       len(v.errors) == 0,
		RecordCount: recordCount,
	}
}

// isValidCountryCode checks if a country code is valid (ISO 3166-1 alpha-2)
func isValidCountryCode(code string) bool {
	code = strings.ToUpper(code)
	validCodes := map[string]bool{
		"US": true, "CA": true, "GB": true, "AU": true, "DE": true,
		"FR": true, "IT": true, "ES": true, "JP": true, "CN": true,
		"MX": true, "BR": true, "IN": true, "RU": true, "KR": true,
		"NL": true, "BE": true, "CH": true, "AT": true, "SE": true,
		"DK": true, "NO": true, "FI": true, "PL": true, "IE": true,
		"PT": true, "GR": true, "CZ": true, "HU": true, "RO": true,
		"BG": true, "HR": true, "SI": true, "SK": true, "LT": true,
		"LV": true, "EE": true, "LU": true, "MT": true, "CY": true,
		"NZ": true, "ZA": true, "SG": true, "HK": true, "TW": true,
		"MY": true, "TH": true, "ID": true, "PH": true, "VN": true,
		"AE": true, "SA": true, "IL": true, "TR": true, "UA": true,
	}
	return validCodes[code]
}

// isValidPostalCode checks postal code format by country
func isValidPostalCode(postalCode, country string) bool {
	country = strings.ToUpper(country)
	patterns := map[string]*regexp.Regexp{
		"US": regexp.MustCompile(`^\d{5}(-\d{4})?$`),
		"CA": regexp.MustCompile(`^[A-Z]\d[A-Z]\s?\d[A-Z]\d$`),
		"GB": regexp.MustCompile(`^[A-Z]{1,2}\d[A-Z\d]?\s?\d[A-Z]{2}$`),
		"AU": regexp.MustCompile(`^\d{4}$`),
		"DE": regexp.MustCompile(`^\d{5}$`),
		"FR": regexp.MustCompile(`^\d{5}$`),
		"IT": regexp.MustCompile(`^\d{5}$`),
		"ES": regexp.MustCompile(`^\d{5}$`),
		"JP": regexp.MustCompile(`^\d{3}-?\d{4}$`),
		"CN": regexp.MustCompile(`^\d{6}$`),
		"MX": regexp.MustCompile(`^\d{5}$`),
		"BR": regexp.MustCompile(`^\d{5}-?\d{3}$`),
	}

	pattern, exists := patterns[country]
	if !exists {
		// Unknown country, accept any non-empty postal code
		return strings.TrimSpace(postalCode) != ""
	}

	return pattern.MatchString(strings.ToUpper(postalCode))
}

// isValidPhone checks if phone number is in valid E.164 format
func isValidPhone(phone string) bool {
	// E.164 format: +[country code][national number]
	// Examples: +1-415-555-0100, +44-20-7946-0958
	pattern := regexp.MustCompile(`^\+[1-9]\d{1,3}[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}$`)
	return pattern.MatchString(phone)
}

// isValidCategory checks if the category is in the allowed list
func isValidCategory(category string) bool {
	validCategories := map[string]bool{
		"RESTAURANT":             true,
		"RETAIL":                 true,
		"RETAIL_COFFEE_SHOP":     true,
		"RETAIL_GROCERY":         true,
		"RETAIL_PHARMACY":        true,
		"RETAIL_BOOKS":           true,
		"RETAIL_ELECTRONICS":     true,
		"RETAIL_HOME_GOODS":      true,
		"RETAIL_CLOTHING":        true,
		"RETAIL_JEWELRY":         true,
		"RETAIL_FURNITURE":       true,
		"RETAIL_SPORTING_GOODS":  true,
		"RETAIL_TOYS":            true,
		"RETAIL_BEAUTY":          true,
		"RETAIL_PET_STORE":       true,
		"RETAIL_AUTOMOTIVE":      true,
		"RETAIL_GARDEN_CENTER":   true,
		"RETAIL_LIQUOR":          true,
		"HEALTH_MEDICAL":         true,
		"HEALTH_DENTIST":         true,
		"HEALTH_HOSPITAL":        true,
		"HEALTH_GYM":             true,
		"HEALTH_VETERINARIAN":    true,
		"FINANCE_BANK":           true,
		"FINANCE_ATM":            true,
		"FINANCE_INSURANCE":      true,
		"AUTOMOTIVE_DEALER":      true,
		"AUTOMOTIVE_REPAIR":      true,
		"AUTOMOTIVE_CAR_WASH":    true,
		"AUTOMOTIVE_GAS_STATION": true,
		"AUTOMOTIVE_PARKING":     true,
		"TRAVEL_HOTEL":           true,
		"TRAVEL_CAR_RENTAL":      true,
		"TRAVEL_AIRPORT":         true,
		"TRAVEL_TRAIN_STATION":   true,
		"TRAVEL_BUS_STATION":     true,
		"ENTERTAINMENT_CINEMA":   true,
		"ENTERTAINMENT_THEATER":  true,
		"ENTERTAINMENT_MUSEUM":   true,
		"ENTERTAINMENT_CASINO":   true,
		"ENTERTAINMENT_BOWLING":  true,
		"ENTERTAINMENT_GOLF":     true,
		"SERVICE_DRY_CLEANING":   true,
		"SERVICE_LAUNDRY":        true,
		"SERVICE_REPAIR":         true,
		"SERVICE_STORAGE":        true,
		"SERVICE_PHOTOGRAPHY":    true,
		"SERVICE_POST_OFFICE":    true,
		"SERVICE_LIBRARY":        true,
		"SERVICE_GOV":            true,
		"SERVICE_POLICE":         true,
		"SERVICE_FIRE":           true,
		"EDUCATION_SCHOOL":       true,
		"EDUCATION_UNIVERSITY":   true,
		"PLACE_PLACE_OF_WORSHIP": true,
		"PLACE_CEMETERY":         true,
		"FOOD_BAR":               true,
		"FOOD_NIGHT_CLUB":        true,
		"FOOD_BREWERY":           true,
		"FOOD_WINERY":            true,
		"FOOD_DISTILLERY":        true,
	}

	upperCat := strings.ToUpper(category)
	return validCategories[upperCat]
}

// PrintResults outputs validation findings
func (r Result) PrintResults() {
	if r.Valid && len(r.Warnings) == 0 {
		fmt.Printf("✅ All %d records passed validation\n", r.RecordCount)
		return
	}

	if len(r.Errors) > 0 {
		fmt.Printf("❌ Validation failed with %d error(s):\n", len(r.Errors))
		for _, err := range r.Errors {
			fmt.Printf("   [%s] %s: %s\n", err.Code, err.Record, err.Message)
		}
	}

	if len(r.Warnings) > 0 {
		fmt.Printf("⚠️  %d warning(s) (non-blocking):\n", len(r.Warnings))
		for _, warn := range r.Warnings {
			fmt.Printf("   [%s] %s: %s\n", warn.Field, warn.Record, warn.Message)
		}
	}

	if r.Valid {
		fmt.Printf("\n✅ Validation passed with warnings - %d records ready to sync\n", r.RecordCount)
	}
}

// Summary returns a one-line summary
func (r Result) Summary() string {
	if !r.Valid {
		return fmt.Sprintf("Validation failed: %d error(s), %d warning(s)", len(r.Errors), len(r.Warnings))
	}
	if len(r.Warnings) > 0 {
		return fmt.Sprintf("Validation passed with %d warning(s)", len(r.Warnings))
	}
	return fmt.Sprintf("All %d records valid", r.RecordCount)
}
