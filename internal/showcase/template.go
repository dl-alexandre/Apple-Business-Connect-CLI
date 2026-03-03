// Package showcase provides templating and bulk management for Apple Business Connect showcases.
// It supports Go templates for dynamic content generation across multiple locations.
package showcase

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/dl-alexandre/abc/internal/api"
)

// TemplateConfig represents a showcase template configuration
type TemplateConfig struct {
	Name        string            `yaml:"name" json:"name"`
	Type        string            `yaml:"type" json:"type"`
	Title       string            `yaml:"title" json:"title"`
	Description string            `yaml:"description" json:"description"`
	StartDate   string            `yaml:"start_date" json:"start_date"`
	EndDate     string            `yaml:"end_date" json:"end_date"`
	ActionLink  ActionLinkConfig  `yaml:"action_link" json:"action_link"`
	Media       []MediaConfig     `yaml:"media" json:"media"`
	Variables   map[string]string `yaml:"variables" json:"variables"`
}

// ActionLinkConfig represents a call-to-action link configuration
type ActionLinkConfig struct {
	Title   string `yaml:"title" json:"title"`
	URL     string `yaml:"url" json:"url"`
	AppLink string `yaml:"app_link" json:"app_link"`
}

// MediaConfig represents media asset configuration
type MediaConfig struct {
	Type    string `yaml:"type" json:"type"`
	URL     string `yaml:"url" json:"url"`
	AltText string `yaml:"alt_text" json:"alt_text"`
}

// LocationData represents data available for template substitution
type LocationData struct {
	PartnerID  string
	Name       string
	City       string
	Region     string
	Country    string
	PostalCode string
	Phone      string
	Category   string
	LocationID string // For updates
}

// TemplateEngine handles showcase template processing
type TemplateEngine struct {
	template *template.Template
	config   TemplateConfig
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(config TemplateConfig) (*TemplateEngine, error) {
	// Create template with custom functions
	funcMap := template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"now":   time.Now,
		"date": func(format string) string {
			return time.Now().Format(format)
		},
	}

	tmpl, err := template.New("showcase").Funcs(funcMap).Parse(config.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to parse title template: %w", err)
	}

	if config.Description != "" {
		tmpl, err = tmpl.Parse(config.Description)
		if err != nil {
			return nil, fmt.Errorf("failed to parse description template: %w", err)
		}
	}

	return &TemplateEngine{
		template: tmpl,
		config:   config,
	}, nil
}

// Generate creates a showcase for a specific location
func (e *TemplateEngine) Generate(data LocationData) (*api.Showcase, error) {
	showcase := &api.Showcase{
		Type: e.config.Type,
	}

	// Generate title
	var titleBuf bytes.Buffer
	if err := e.template.ExecuteTemplate(&titleBuf, "showcase", data); err != nil {
		return nil, fmt.Errorf("failed to render title: %w", err)
	}
	showcase.Title = api.LocalizedString{Default: titleBuf.String()}

	// Generate description if present
	if e.config.Description != "" {
		descTmpl, err := template.New("desc").Parse(e.config.Description)
		if err != nil {
			return nil, fmt.Errorf("failed to parse description template: %w", err)
		}
		var descBuf bytes.Buffer
		if err := descTmpl.Execute(&descBuf, data); err != nil {
			return nil, fmt.Errorf("failed to render description: %w", err)
		}
		showcase.Description = api.LocalizedString{Default: descBuf.String()}
	}

	// Parse dates
	if e.config.StartDate != "" {
		start, err := time.Parse("2006-01-02", e.config.StartDate)
		if err == nil {
			showcase.StartDate = start
		}
	}
	if e.config.EndDate != "" {
		end, err := time.Parse("2006-01-02", e.config.EndDate)
		if err == nil {
			showcase.EndDate = end
		}
	}

	// Generate action link
	if e.config.ActionLink.URL != "" {
		actionURL, err := e.renderTemplate(e.config.ActionLink.URL, data)
		if err != nil {
			return nil, fmt.Errorf("failed to render action URL: %w", err)
		}

		actionTitle := e.config.ActionLink.Title
		if actionTitle == "" {
			actionTitle = "Learn More"
		}

		showcase.ActionLink = &api.ActionLink{
			Title: api.LocalizedString{Default: actionTitle},
			URL:   actionURL,
		}

		if e.config.ActionLink.AppLink != "" {
			showcase.ActionLink.AppLinkID = e.config.ActionLink.AppLink
		}
	}

	return showcase, nil
}

// renderTemplate renders a template string with location data
func (e *TemplateEngine) renderTemplate(tmplStr string, data LocationData) (string, error) {
	tmpl, err := template.New("dynamic").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ActionLinkValidator validates Apple Business Connect action links
type ActionLinkValidator struct {
	errors []ValidationError
}

// ValidationError represents an action link validation error
type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// NewActionLinkValidator creates a new validator
func NewActionLinkValidator() *ActionLinkValidator {
	return &ActionLinkValidator{
		errors: make([]ValidationError, 0),
	}
}

// Validate checks an action link for compliance
func (v *ActionLinkValidator) Validate(link ActionLinkConfig) []ValidationError {
	v.errors = make([]ValidationError, 0)

	// Check URL is present
	if link.URL == "" {
		v.addError("url", "action link URL is required", "MISSING_URL")
		return v.errors
	}

	// Check HTTPS protocol
	if !strings.HasPrefix(link.URL, "https://") {
		v.addError("url", "action link must use HTTPS protocol", "INSECURE_URL")
	}

	// Validate URL format
	parsed, err := url.Parse(link.URL)
	if err != nil {
		v.addError("url", fmt.Sprintf("invalid URL format: %v", err), "INVALID_URL")
		return v.errors
	}

	// Check for forbidden parameters
	forbiddenParams := []string{"affiliate", "partner", "ref", "clickid", "subid"}
	for _, param := range forbiddenParams {
		if parsed.Query().Get(param) != "" {
			v.addError("url",
				fmt.Sprintf("URL contains forbidden parameter '%s' (Apple rejects tracking parameters)", param),
				"FORBIDDEN_PARAM")
		}
	}

	// Check URL length (Apple has limits)
	if len(link.URL) > 2048 {
		v.addError("url", "URL exceeds maximum length of 2048 characters", "URL_TOO_LONG")
	}

	// Check for common issues
	if strings.Contains(link.URL, " ") {
		v.addError("url", "URL contains spaces (must be URL-encoded)", "URL_HAS_SPACES")
	}

	// Check for legacy URI schemes (not recommended by Apple in 2026)
	if IsLegacyURIScheme(link.URL) {
		v.addError("url",
			"URL uses legacy custom URI scheme (e.g., myapp://). Apple recommends Universal Links (https://) for better compatibility",
			"LEGACY_URI_SCHEME")
	}

	// Check if it might be a Universal Link (https with app association)
	if strings.HasPrefix(link.URL, "https://") {
		v.addWarning("url", "Ensure 'apple-app-site-association' file is properly configured for Universal Links")
	}

	// Validate title
	if link.Title == "" {
		v.addWarning("title", "action link title is empty (will use default)")
	} else if len(link.Title) > 50 {
		v.addError("title", "action link title exceeds 50 characters", "TITLE_TOO_LONG")
	}

	return v.errors
}

func (v *ActionLinkValidator) addError(field, message, code string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

func (v *ActionLinkValidator) addWarning(field, message string) {
	// Warnings don't block validation
}

// BatchGenerator creates showcases for multiple locations
type BatchGenerator struct {
	engine *TemplateEngine
}

// NewBatchGenerator creates a batch generator
func NewBatchGenerator(config TemplateConfig) (*BatchGenerator, error) {
	engine, err := NewTemplateEngine(config)
	if err != nil {
		return nil, err
	}
	return &BatchGenerator{engine: engine}, nil
}

// GenerateAll creates showcases for all locations
func (b *BatchGenerator) GenerateAll(locations []LocationData) ([]GeneratedShowcase, error) {
	var showcases []GeneratedShowcase

	for _, loc := range locations {
		showcase, err := b.engine.Generate(loc)
		if err != nil {
			return nil, fmt.Errorf("failed to generate showcase for %s: %w", loc.Name, err)
		}

		showcases = append(showcases, GeneratedShowcase{
			LocationID: loc.LocationID,
			PartnerID:  loc.PartnerID,
			Showcase:   showcase,
		})
	}

	return showcases, nil
}

// GeneratedShowcase represents a showcase ready to be created/updated
type GeneratedShowcase struct {
	LocationID string
	PartnerID  string
	Showcase   *api.Showcase
}

// ValidateTemplate checks a template configuration for errors
func ValidateTemplate(config TemplateConfig) []string {
	var errors []string

	// Check required fields
	if config.Name == "" {
		errors = append(errors, "template name is required")
	}
	if config.Type == "" {
		errors = append(errors, "showcase type is required (EVENT or OFFER)")
	} else if config.Type != "EVENT" && config.Type != "OFFER" {
		errors = append(errors, "showcase type must be EVENT or OFFER")
	}
	if config.Title == "" {
		errors = append(errors, "title template is required")
	}

	// Validate action link if present
	if config.ActionLink.URL != "" {
		validator := NewActionLinkValidator()
		valErrors := validator.Validate(config.ActionLink)
		for _, err := range valErrors {
			errors = append(errors, fmt.Sprintf("action link: %s", err.Message))
		}
	}

	// Validate dates
	if config.StartDate != "" {
		if _, err := time.Parse("2006-01-02", config.StartDate); err != nil {
			errors = append(errors, fmt.Sprintf("invalid start_date format: %v", err))
		}
	}
	if config.EndDate != "" {
		if _, err := time.Parse("2006-01-02", config.EndDate); err != nil {
			errors = append(errors, fmt.Sprintf("invalid end_date format: %v", err))
		}
	}

	return errors
}

// IsForbiddenURL checks if a URL contains forbidden patterns
func IsForbiddenURL(urlStr string) bool {
	forbiddenPatterns := []string{
		`(?i)affiliate`,
		`(?i)partner.*id`,
		`(?i)click.*id`,
		`(?i)utm_.*`,
		`(?i)fbclid`,
		`(?i)gclid`,
		`(?i)msclkid`,
	}

	for _, pattern := range forbiddenPatterns {
		if matched, _ := regexp.MatchString(pattern, urlStr); matched {
			return true
		}
	}
	return false
}

// IsLegacyURIScheme checks if URL uses a custom URI scheme instead of HTTPS
func IsLegacyURIScheme(urlStr string) bool {
	// Check if URL starts with a custom scheme (not http:// or https://)
	legacySchemes := []string{
		"myapp://",
		"app://",
		"brand://",
		"shop://",
		"order://",
		"book://",
		"reserve://",
		"custom://",
	}

	for _, scheme := range legacySchemes {
		if strings.HasPrefix(urlStr, scheme) {
			return true
		}
	}

	// Check for any custom scheme pattern (word://)
	if matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9+.-]*://`, urlStr); matched {
		// It's a custom scheme, but check if it's http/https
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			return true
		}
	}

	return false
}
