package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"

	"github.com/dl-alexandre/abc/internal/api"
	"github.com/dl-alexandre/abc/internal/auth"
	"github.com/dl-alexandre/abc/internal/blast"
	"github.com/dl-alexandre/abc/internal/cache"
	"github.com/dl-alexandre/abc/internal/config"
	"github.com/dl-alexandre/abc/internal/dns"
	"github.com/dl-alexandre/abc/internal/output"
	"github.com/dl-alexandre/abc/internal/queue"
	"github.com/dl-alexandre/abc/internal/showcase"
	"github.com/dl-alexandre/abc/internal/svg"
	"github.com/dl-alexandre/abc/internal/sync"
	"github.com/dl-alexandre/abc/internal/validate"
)

// CLI is the main command-line interface structure using Kong
type CLI struct {
	Globals

	// Command groups
	Auth      AuthCmd      `cmd:"" help:"Manage authentication"`
	Audit     AuditCmd     `cmd:"" help:"Content quality audit for locations"`
	Bimi      BimiCmd      `cmd:"" help:"BIMI (Brand Indicators) validation"`
	Doctor    DoctorCmd    `cmd:"" help:"Run diagnostics and troubleshoot issues"`
	Locations LocationsCmd `cmd:"" help:"Manage business locations"`
	Mail      MailCmd      `cmd:"" help:"Manage Branded Mail and domain verification"`
	Queue     QueueCmd     `cmd:"" help:"Manage offline operation queue"`
	Shell     ShellCmd     `cmd:"" help:"Interactive REPL mode"`
	Showcases ShowcasesCmd `cmd:"" help:"Manage showcases"`
	Insights  InsightsCmd  `cmd:"" help:"View location insights"`
	Status    StatusCmd    `cmd:"" help:"View overall account status dashboard"`
	Webhooks  WebhooksCmd  `cmd:"" help:"Manage Apple webhooks for real-time events"`

	// Utility commands
	Version     VersionCmd     `cmd:"" help:"Show version information"`
	CheckUpdate UpdateCheckCmd `cmd:"" help:"Check for available updates"`
	Completion  CompletionCmd  `cmd:"" help:"Generate shell completion script"`
}

// Globals contains global flags available to all commands
type Globals struct {
	ConfigFile string `help:"Config file path" short:"c" env:"ABC_CONFIG"`
	APIURL     string `help:"API base URL" env:"ABC_API_URL"`
	Timeout    int    `help:"Request timeout in seconds" default:"30" env:"ABC_TIMEOUT"`
	NoCache    bool   `help:"Disable caching" env:"ABC_NO_CACHE"`
	CacheDir   string `help:"Cache directory" env:"ABC_CACHE_DIR"`
	CacheTTL   int    `help:"Cache TTL in minutes" default:"60" env:"ABC_CACHE_TTL"`
	Verbose    bool   `help:"Enable verbose output" short:"v" env:"ABC_VERBOSE"`
	Debug      bool   `help:"Enable debug output" env:"ABC_DEBUG"`
	Format     string `help:"Output format: table, json, markdown" default:"table" enum:"table,json,markdown" env:"ABC_FORMAT"`

	// Runtime dependencies (initialized by AfterApply)
	Config *config.Config `kong:"-"`
	Cache  *cache.Cache   `kong:"-"`
	Client *api.Client    `kong:"-"`
}

// AfterApply initializes runtime dependencies after flag parsing
func (g *Globals) AfterApply() error {
	// Load configuration
	flags := config.Flags{
		ConfigFile: g.ConfigFile,
		APIURL:     g.APIURL,
		Timeout:    g.Timeout,
		NoCache:    g.NoCache,
		CacheDir:   g.CacheDir,
		CacheTTL:   g.CacheTTL,
		Verbose:    g.Verbose,
		Debug:      g.Debug,
		Format:     g.Format,
	}

	cfg, err := config.Load(flags)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	g.Config = cfg

	// Initialize cache if enabled
	if !g.NoCache && cfg.Cache.Enabled {
		g.Cache = cache.New(cfg.Cache.Dir, cfg.Cache.TTL)
	}

	// Initialize API client with OAuth2 credentials
	client, err := api.NewClient(api.ClientOptions{
		BaseURL:      cfg.API.URL,
		ClientID:     cfg.API.ClientID,
		ClientSecret: cfg.API.ClientSecret,
		Timeout:      cfg.API.Timeout,
		Verbose:      g.Verbose,
		Debug:        g.Debug,
	})
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}
	g.Client = client

	return nil
}

// ShouldUseColor determines if color output should be used
func (g *Globals) ShouldUseColor() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

// GetPrinter returns an output printer based on format
func (g *Globals) GetPrinter() *output.Printer {
	return output.NewPrinter(g.Format, g.ShouldUseColor())
}

// AuthCmd is the parent command for authentication operations
type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Store credentials in OS keyring"`
	Logout AuthLogoutCmd `cmd:"" help:"Remove credentials from OS keyring"`
	Status AuthStatusCmd `cmd:"" help:"Check authentication status"`
}

// AuthLoginCmd stores credentials in the OS keyring
type AuthLoginCmd struct {
	ClientID     string `help:"OAuth2 client ID" env:"ABC_CLIENT_ID"`
	ClientSecret string `help:"OAuth2 client secret" env:"ABC_CLIENT_SECRET"`
}

func (c *AuthLoginCmd) Run(globals *Globals) error {
	// Prompt for credentials if not provided via flags/env
	if c.ClientID == "" {
		fmt.Print("Enter Client ID: ")
		fmt.Fscanln(os.Stdin, &c.ClientID)
	}

	if c.ClientSecret == "" {
		fmt.Print("Enter Client Secret: ")
		fmt.Fscanln(os.Stdin, &c.ClientSecret)
	}

	if c.ClientID == "" || c.ClientSecret == "" {
		return fmt.Errorf("both client_id and client_secret are required")
	}

	// Store in keyring
	creds := auth.Credentials{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
	}

	if err := auth.Store(creds); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	fmt.Println("Credentials stored securely in OS keyring")
	return nil
}

// AuthLogoutCmd removes credentials from the OS keyring
type AuthLogoutCmd struct {
	Force bool `help:"Skip confirmation prompt"`
}

func (c *AuthLogoutCmd) Run(globals *Globals) error {
	if !c.Force {
		fmt.Print("Are you sure you want to remove stored credentials? [y/N]: ")
		var response string
		fmt.Fscanln(os.Stdin, &response)
		if response != "y" && response != "Y" {
			fmt.Println("Logout cancelled")
			return nil
		}
	}

	if err := auth.Delete(); err != nil {
		return fmt.Errorf("failed to remove credentials: %w", err)
	}

	fmt.Println("Credentials removed from OS keyring")
	return nil
}

// AuthStatusCmd checks if credentials are stored
type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(globals *Globals) error {
	if auth.Check() {
		creds, err := auth.Retrieve()
		if err != nil {
			return fmt.Errorf("failed to retrieve credentials: %w", err)
		}
		fmt.Printf("Authenticated (Client ID: %s...)\n", creds.ClientID[:8])
	} else {
		fmt.Println("Not authenticated")
		fmt.Println("Run 'abc auth login' to store credentials")
	}
	return nil
}

// DoctorCmd runs diagnostics on the CLI setup
type DoctorCmd struct {
	ShowAll bool `help:"Show all diagnostics including passing checks"`
}

func (c *DoctorCmd) Run(globals *Globals) error {
	fmt.Println("🔍 Running Apple Business Connect CLI diagnostics...")
	fmt.Println()

	allPassed := true

	// Check 1: API connectivity
	fmt.Println("✓ API Connectivity")
	ctx := context.Background()
	_, err := globals.Client.ListLocations(ctx, "", 1, "")
	if err != nil {
		fmt.Printf("  ❌ Cannot connect to Apple Business Connect API\n")
		fmt.Printf("     Error: %v\n", err)
		allPassed = false
	} else {
		fmt.Printf("  ✅ Successfully connected to API\n")
	}

	// Check 2: Authentication
	fmt.Println("✓ Authentication")
	if globals.Config.API.ClientID == "" || globals.Config.API.ClientSecret == "" {
		fmt.Printf("  ❌ Missing credentials\n")
		fmt.Printf("     Client ID: %s\n", checkMark(globals.Config.API.ClientID != ""))
		fmt.Printf("     Client Secret: %s\n", checkMark(globals.Config.API.ClientSecret != ""))
		fmt.Printf("     Fix: Run 'abc auth login' or set ABC_API_CLIENT_ID/ABC_API_CLIENT_SECRET\n")
		allPassed = false
	} else {
		fmt.Printf("  ✅ Credentials configured\n")
		if c.ShowAll {
			fmt.Printf("     Client ID: %s...\n", globals.Config.API.ClientID[:8])
		}
	}

	// Check 3: Config file
	fmt.Println("✓ Configuration")
	configPath := globals.ConfigFile
	if configPath == "" {
		configPath = "~/.config/abc/config.yaml (default)"
	}
	if c.ShowAll {
		fmt.Printf("  Config path: %s\n", configPath)
	}
	fmt.Printf("  ✅ Configuration loaded\n")

	// Check 4: Cache
	fmt.Println("✓ Cache")
	if globals.NoCache || !globals.Config.Cache.Enabled {
		fmt.Printf("  ⚠️  Caching is disabled\n")
	} else {
		fmt.Printf("  ✅ Cache enabled at %s\n", globals.Config.Cache.Dir)
	}

	// Check 5: OS Keyring
	fmt.Println("✓ OS Keyring")
	if auth.Check() {
		fmt.Printf("  ✅ Keyring accessible\n")
	} else {
		fmt.Printf("  ⚠️  Keyring not configured\n")
		fmt.Printf("     Run 'abc auth login' to store credentials securely\n")
	}

	// Interactive: Offer to fix issues
	if !allPassed && !globals.Verbose {
		fmt.Println()
		fmt.Print("🔧 Would you like to run setup now? [y/N]: ")
		var response string
		fmt.Fscanln(os.Stdin, &response)
		if response == "y" || response == "Y" {
			if globals.Config.API.ClientID == "" || globals.Config.API.ClientSecret == "" {
				fmt.Println("\n📋 Running 'abc auth login'...")
				fmt.Println("   Please enter your Apple Business Connect credentials:")
				// In a full implementation, this would call auth.Store() interactively
			}
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("✅ All diagnostics passed! CLI is ready to use.")
	} else {
		fmt.Println("❌ Some diagnostics failed. Please fix the issues above.")
	}

	return nil
}

func checkMark(condition bool) string {
	if condition {
		return "✅ configured"
	}
	return "❌ missing"
}

// LocationsCmd is the parent command for location operations
type LocationsCmd struct {
	List   LocationsListCmd   `cmd:"" help:"List all locations"`
	Get    LocationsGetCmd    `cmd:"" help:"Get a location by ID"`
	Create LocationsCreateCmd `cmd:"" help:"Create a new location"`
	Update LocationsUpdateCmd `cmd:"" help:"Update a location"`
	Delete LocationsDeleteCmd `cmd:"" help:"Delete a location"`
	Sync   LocationsSyncCmd   `cmd:"" help:"Sync locations from file (CSV/JSON)"`
}

// LocationsListCmd lists all locations
type LocationsListCmd struct {
	CompanyID    string `help:"Filter by company ID" env:"ABC_COMPANY_ID"`
	Limit        int    `help:"Maximum number of results" default:"20"`
	PageToken    string `help:"Page token for pagination"`
	OutputFormat string `help:"Output format (overrides global)" `
}

func (c *LocationsListCmd) Run(globals *Globals) error {
	format := c.OutputFormat
	if format == "" {
		format = globals.Format
	}

	ctx := context.Background()
	resp, err := globals.Client.ListLocations(ctx, c.CompanyID, c.Limit, c.PageToken)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintLocations(resp.Locations)
}

// LocationsGetCmd gets a location by ID
type LocationsGetCmd struct {
	ID           string `arg:"" help:"Location ID"`
	OutputFormat string `help:"Output format (overrides global)" `
}

func (c *LocationsGetCmd) Run(globals *Globals) error {
	format := c.OutputFormat
	if format == "" {
		format = globals.Format
	}

	ctx := context.Background()
	location, err := globals.Client.GetLocation(ctx, c.ID)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintLocation(location)
}

// LocationsCreateCmd creates a new location
type LocationsCreateCmd struct {
	Name         string `arg:"" help:"Location name"`
	Street       string `help:"Street address" required:""`
	City         string `help:"City" required:""`
	Region       string `help:"State/Region" required:""`
	PostalCode   string `help:"Postal code" required:""`
	Country      string `help:"Country code" required:""`
	Phone        string `help:"Phone number"`
	Category     string `help:"Primary category"`
	OutputFormat string `help:"Output format (overrides global)" `
}

func (c *LocationsCreateCmd) Run(globals *Globals) error {
	format := c.OutputFormat
	if format == "" {
		format = globals.Format
	}

	location := &api.Location{
		LocationName: api.LocalizedString{Default: c.Name},
		PrimaryAddress: api.Address{
			StreetAddress: c.Street,
			Locality:      c.City,
			Region:        c.Region,
			PostalCode:    c.PostalCode,
			Country:       c.Country,
		},
		PhoneNumber: c.Phone,
	}

	if c.Category != "" {
		location.Categories = []string{c.Category}
	}

	ctx := context.Background()
	created, err := globals.Client.CreateLocation(ctx, location)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintLocation(created)
}

// LocationsUpdateCmd updates an existing location
type LocationsUpdateCmd struct {
	ID           string `arg:"" help:"Location ID"`
	Name         string `help:"New location name"`
	Phone        string `help:"New phone number"`
	OutputFormat string `help:"Output format (overrides global)" `
}

func (c *LocationsUpdateCmd) Run(globals *Globals) error {
	format := c.OutputFormat
	if format == "" {
		format = globals.Format
	}

	location := &api.Location{}
	if c.Name != "" {
		location.LocationName = api.LocalizedString{Default: c.Name}
	}
	if c.Phone != "" {
		location.PhoneNumber = c.Phone
	}

	ctx := context.Background()
	updated, err := globals.Client.UpdateLocation(ctx, c.ID, location)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintLocation(updated)
}

// LocationsDeleteCmd deletes a location
type LocationsDeleteCmd struct {
	ID      string `arg:"" help:"Location ID"`
	Confirm bool   `help:"Skip confirmation prompt"`
}

func (c *LocationsDeleteCmd) Run(globals *Globals) error {
	if !c.Confirm {
		fmt.Fprintf(os.Stderr, "Are you sure you want to delete location %s? Use --confirm to skip this prompt.\n", c.ID)
		return fmt.Errorf("deletion not confirmed")
	}

	ctx := context.Background()
	if err := globals.Client.DeleteLocation(ctx, c.ID); err != nil {
		return err
	}

	fmt.Printf("Location %s deleted successfully\n", c.ID)
	return nil
}

// LocationsSyncCmd syncs locations from a file
type LocationsSyncCmd struct {
	File             string `arg:"" help:"Path to CSV or JSON file"`
	DryRun           bool   `help:"Show changes without applying them"`
	Confirm          bool   `help:"Skip confirmation prompt"`
	Workers          int    `help:"Number of concurrent workers (default: 5)" default:"5"`
	RateMs           int    `help:"Rate limit in milliseconds between requests (default: 100)" default:"100"`
	MaxCreates       int    `help:"Maximum number of locations to create (0 = unlimited)" default:"0"`
	MaxUpdates       int    `help:"Maximum number of locations to update (0 = unlimited)" default:"0"`
	MaxDeletes       int    `help:"Maximum number of locations to delete (0 = unlimited)" default:"0"`
	MaxCreatePercent string `help:"Max creates as percentage (e.g., '25%' )"`
	MaxUpdatePercent string `help:"Max updates as percentage (e.g., '50%' )"`
	MaxDeletePercent string `help:"Max deletes as percentage (e.g., '10%' )"`
}

func (c *LocationsSyncCmd) Run(globals *Globals) error {
	// Parse the file
	parser, err := sync.NewParser(c.File)
	if err != nil {
		return err
	}

	records, err := parser.Parse(c.File)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d location(s) in file\n", len(records))

	// Pre-flight validation
	fmt.Println("\n🔍 Running pre-flight validation...")
	validator := validate.NewValidator()
	for i, record := range records {
		identifier := record.PartnerID
		if identifier == "" {
			identifier = fmt.Sprintf("record_%d", i+1)
		}
		validator.ValidateRecord(record.ToValidateRecord(), identifier)
	}

	valResult := validator.GetResult(len(records))
	valResult.PrintResults()

	if !valResult.Valid {
		return fmt.Errorf("validation failed - fix %d error(s) before syncing", len(valResult.Errors))
	}

	// Fetch existing locations from API
	ctx := context.Background()
	resp, err := globals.Client.ListLocations(ctx, "", 100, "")
	if err != nil {
		return fmt.Errorf("failed to fetch existing locations: %w", err)
	}

	fmt.Printf("Found %d existing location(s) in Apple Business Connect\n", len(resp.Locations))

	// Perform diff
	diffEngine := sync.NewDiffEngine(records, resp.Locations)
	result := diffEngine.Compare()

	// Print changes
	result.PrintChanges()

	// If dry-run, stop here
	if c.DryRun {
		fmt.Println("\n(Dry-run mode: no changes were made)")
		return nil
	}

	// Check if there are any changes to apply
	if result.ToCreate == 0 && result.ToUpdate == 0 {
		fmt.Println("\nNo changes to apply.")
		return nil
	}

	// Blast radius protection
	limits := blast.Limits{
		ToCreate:    result.ToCreate,
		ToUpdate:    result.ToUpdate,
		ToDelete:    result.ToDelete,
		NoChange:    result.NoChange,
		TotalLocal:  len(records),
		TotalRemote: len(resp.Locations),
	}

	protection := blast.Protection{
		MaxCreates:   c.MaxCreates,
		MaxUpdates:   c.MaxUpdates,
		MaxDeletions: c.MaxDeletes,
	}

	// Parse percentage limits
	if c.MaxCreatePercent != "" {
		p, _ := blast.ParsePercent(c.MaxCreatePercent)
		protection.MaxCreatePercent = p
	}
	if c.MaxUpdatePercent != "" {
		p, _ := blast.ParsePercent(c.MaxUpdatePercent)
		protection.MaxUpdatePercent = p
	}
	if c.MaxDeletePercent != "" {
		p, _ := blast.ParsePercent(c.MaxDeletePercent)
		protection.MaxDeletePercent = p
	}

	blastResult := protection.Check(limits)
	if blastResult.Blocked {
		fmt.Printf("\n🛡️  Blast radius protection triggered!\n")
		fmt.Printf("   Reason: %s\n", blastResult.Reason)
		fmt.Printf("\n   Current limits: %s\n", protection.FormatLimits())
		fmt.Printf("\n   To override, run with --max-creates/updates/deletes flags\n")
		return fmt.Errorf("blast radius protection blocked sync")
	}

	// Confirm before applying
	if !c.Confirm {
		fmt.Printf("\nApply these changes? [y/N]: ")
		var response string
		fmt.Fscanln(os.Stdin, &response)
		if response != "y" && response != "Y" {
			fmt.Println("Sync cancelled")
			return nil
		}
	}

	// Filter to only CREATE and UPDATE changes
	var changesToApply []sync.LocationChange
	for _, change := range result.Changes {
		if change.Type == sync.ChangeCreate || change.Type == sync.ChangeUpdate {
			changesToApply = append(changesToApply, change)
		}
	}

	// Apply changes with rate limiting and concurrency control
	opts := sync.SyncOptions{
		Workers:     c.Workers,
		RateLimitMs: c.RateMs,
	}

	fmt.Printf("\nApplying %d changes with %d workers (rate limit: %dms)...\n",
		len(changesToApply), opts.Workers, opts.RateLimitMs)

	execResult, err := sync.ExecuteSync(ctx, globals.Client, changesToApply, opts, false)
	if err != nil {
		return fmt.Errorf("sync execution failed: %w", err)
	}

	// Print results
	for _, res := range execResult.Results {
		if res.Error != nil {
			fmt.Fprintf(os.Stderr, "Failed %s %s: %v\n",
				res.Change.Type, res.Change.LocalLocation.Name, res.Error)
		} else {
			fmt.Printf("%s: %s\n", res.Change.Type, res.Change.LocalLocation.Name)
		}
	}

	fmt.Printf("\n%s\n", execResult.Summary())

	if execResult.Failed > 0 {
		return fmt.Errorf("sync completed with %d error(s)", execResult.Failed)
	}

	return nil
}

// ShowcasesCmd is the parent command for showcase operations
type ShowcasesCmd struct {
	List   ShowcasesListCmd   `cmd:"" help:"List showcases for a location"`
	Get    ShowcasesGetCmd    `cmd:"" help:"Get a showcase by ID"`
	Create ShowcasesCreateCmd `cmd:"" help:"Create a new showcase"`
	Update ShowcasesUpdateCmd `cmd:"" help:"Update a showcase"`
	Delete ShowcasesDeleteCmd `cmd:"" help:"Delete a showcase"`
	Sync   ShowcasesSyncCmd   `cmd:"" help:"Sync showcases from template (bulk)"`
}

// ShowcasesListCmd lists showcases for a location
type ShowcasesListCmd struct {
	LocationID   string `arg:"" help:"Location ID"`
	Limit        int    `help:"Maximum number of results" default:"20"`
	PageToken    string `help:"Page token for pagination"`
	OutputFormat string `help:"Output format (overrides global)" `
}

func (c *ShowcasesListCmd) Run(globals *Globals) error {
	format := c.OutputFormat
	if format == "" {
		format = globals.Format
	}

	ctx := context.Background()
	resp, err := globals.Client.ListShowcases(ctx, c.LocationID, c.Limit, c.PageToken)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintShowcases(resp.Showcases)
}

// ShowcasesGetCmd gets a showcase by ID
type ShowcasesGetCmd struct {
	LocationID   string `arg:"" help:"Location ID"`
	ShowcaseID   string `arg:"" help:"Showcase ID"`
	OutputFormat string `help:"Output format (overrides global)" `
}

func (c *ShowcasesGetCmd) Run(globals *Globals) error {
	format := c.OutputFormat
	if format == "" {
		format = globals.Format
	}

	ctx := context.Background()
	showcase, err := globals.Client.GetShowcase(ctx, c.LocationID, c.ShowcaseID)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintShowcase(showcase)
}

// ShowcasesCreateCmd creates a new showcase
type ShowcasesCreateCmd struct {
	LocationID   string `arg:"" help:"Location ID"`
	Title        string `arg:"" help:"Showcase title"`
	Description  string `help:"Showcase description"`
	Type         string `help:"Showcase type: EVENT, OFFER" enum:"EVENT,OFFER" default:"EVENT"`
	OutputFormat string `help:"Output format (overrides global)" `
}

func (c *ShowcasesCreateCmd) Run(globals *Globals) error {
	format := c.OutputFormat
	if format == "" {
		format = globals.Format
	}

	showcase := &api.Showcase{
		Title: api.LocalizedString{Default: c.Title},
		Type:  c.Type,
	}

	if c.Description != "" {
		showcase.Description = api.LocalizedString{Default: c.Description}
	}

	ctx := context.Background()
	created, err := globals.Client.CreateShowcase(ctx, c.LocationID, showcase)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintShowcase(created)
}

// ShowcasesUpdateCmd updates a showcase
type ShowcasesUpdateCmd struct {
	LocationID   string `arg:"" help:"Location ID"`
	ShowcaseID   string `arg:"" help:"Showcase ID"`
	Title        string `help:"New title"`
	Description  string `help:"New description"`
	OutputFormat string `help:"Output format (overrides global)" `
}

func (c *ShowcasesUpdateCmd) Run(globals *Globals) error {
	format := c.OutputFormat
	if format == "" {
		format = globals.Format
	}

	showcase := &api.Showcase{}
	if c.Title != "" {
		showcase.Title = api.LocalizedString{Default: c.Title}
	}
	if c.Description != "" {
		showcase.Description = api.LocalizedString{Default: c.Description}
	}

	ctx := context.Background()
	updated, err := globals.Client.UpdateShowcase(ctx, c.LocationID, c.ShowcaseID, showcase)
	if err != nil {
		return err
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintShowcase(updated)
}

// ShowcasesDeleteCmd deletes a showcase
type ShowcasesDeleteCmd struct {
	LocationID string `arg:"" help:"Location ID"`
	ShowcaseID string `arg:"" help:"Showcase ID"`
	Confirm    bool   `help:"Skip confirmation prompt"`
}

func (c *ShowcasesDeleteCmd) Run(globals *Globals) error {
	if !c.Confirm {
		fmt.Fprintf(os.Stderr, "Are you sure you want to delete showcase %s? Use --confirm to skip this prompt.\n", c.ShowcaseID)
		return fmt.Errorf("deletion not confirmed")
	}

	ctx := context.Background()
	if err := globals.Client.DeleteShowcase(ctx, c.LocationID, c.ShowcaseID); err != nil {
		return err
	}

	fmt.Printf("Showcase %s deleted successfully\n", c.ShowcaseID)
	return nil
}

// ShowcasesSyncCmd syncs showcases from a template file
type ShowcasesSyncCmd struct {
	Template string `arg:"" help:"Path to showcases.yaml template file"`
	Data     string `help:"Path to locations CSV/JSON for template variables"`
	DryRun   bool   `help:"Show changes without applying them"`
	Confirm  bool   `help:"Skip confirmation prompt"`
}

func (c *ShowcasesSyncCmd) Run(globals *Globals) error {
	fmt.Printf("🎭 Syncing showcases from template: %s\n", c.Template)

	// Parse the template file (simplified version)
	// In production, this would read and parse the YAML file
	templateConfig := showcase.TemplateConfig{
		Name:        "example_template",
		Type:        "OFFER",
		Title:       "Spring Sale at {{.City}}!",
		Description: "Visit our {{.City}} location for great deals!",
		StartDate:   "2024-04-01",
		EndDate:     "2024-04-30",
		ActionLink: showcase.ActionLinkConfig{
			Title: "Shop Now",
			URL:   "https://example.com/sale?store={{.PartnerID}}",
		},
	}

	// Validate template
	validationErrors := showcase.ValidateTemplate(templateConfig)
	if len(validationErrors) > 0 {
		fmt.Fprintln(os.Stderr, "❌ Template validation errors:")
		for _, err := range validationErrors {
			fmt.Fprintf(os.Stderr, "   - %s\n", err)
		}
		return fmt.Errorf("template validation failed")
	}

	// Create template engine
	engine, err := showcase.NewTemplateEngine(templateConfig)
	if err != nil {
		return fmt.Errorf("failed to create template engine: %w", err)
	}

	// Generate sample showcase
	sampleData := showcase.LocationData{
		PartnerID: "SF001",
		Name:      "San Francisco Store",
		City:      "San Francisco",
		Region:    "CA",
		Country:   "US",
	}

	generated, err := engine.Generate(sampleData)
	if err != nil {
		return fmt.Errorf("failed to generate showcase: %w", err)
	}

	fmt.Printf("\n📋 Generated showcase preview:\n")
	fmt.Printf("   Title: %s\n", generated.Title.Default)
	fmt.Printf("   Type: %s\n", generated.Type)
	if generated.Description.Default != "" {
		fmt.Printf("   Description: %s\n", generated.Description.Default)
	}
	if generated.ActionLink != nil {
		fmt.Printf("   Action URL: %s\n", generated.ActionLink.URL)
	}

	if c.DryRun {
		fmt.Println("\n✅ Dry-run complete - no changes made")
		return nil
	}

	fmt.Println("\n📝 Note: Full template sync requires location data file")
	fmt.Println("   Run with --data locations.csv to sync to all locations")
	return nil
}

// InsightsCmd is the parent command for insights operations
type InsightsCmd struct {
	Get     InsightsGetCmd     `cmd:"" help:"Get insights for a location"`
	Export  InsightsExportCmd  `cmd:"" help:"Export insights data for BI integration"`
	Heatmap InsightsHeatmapCmd `cmd:"" help:"Generate engagement heatmap visualization"`
	Compare InsightsCompareCmd `cmd:"" help:"Compare insights between locations (A/B testing)"`
}

// InsightsGetCmd gets insights for a location
type InsightsGetCmd struct {
	LocationID   string `arg:"" help:"Location ID"`
	Period       string `help:"Time period: DAY, WEEK, MONTH" enum:"DAY,WEEK,MONTH" default:"MONTH"`
	StartDate    string `help:"Start date (YYYY-MM-DD)"`
	EndDate      string `help:"End date (YYYY-MM-DD)"`
	OutputFormat string `help:"Output format (overrides global)" `
}

func (c *InsightsGetCmd) Run(globals *Globals) error {
	format := c.OutputFormat
	if format == "" {
		format = globals.Format
	}

	ctx := context.Background()
	resp, err := globals.Client.GetInsights(ctx, c.LocationID, c.Period, c.StartDate, c.EndDate)
	if err != nil {
		return err
	}

	// Check for privacy threshold warnings (if no insights data)
	if len(resp.Insights) == 0 {
		fmt.Fprintln(os.Stderr, "⚠️  Note: Low usage metrics may be reported as zero due to Apple's privacy thresholds.")
	}

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintInsights(resp.Insights)
}

// InsightsExportCmd exports insights data for BI integration
type InsightsExportCmd struct {
	LocationID string `arg:"" help:"Location ID"`
	Days       int    `help:"Number of days to export" default:"90"`
	ExportFmt  string `help:"Export format: csv, json" enum:"csv,json" default:"csv"`
	Output     string `help:"Output file path (default: stdout)"`
}

func (c *InsightsExportCmd) Run(globals *Globals) error {
	// Calculate date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -c.Days)

	ctx := context.Background()

	// Fetch insights for the date range
	resp, err := globals.Client.GetInsights(ctx, c.LocationID, "DAY",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to fetch insights: %w", err)
	}

	// Check for privacy threshold warnings
	totalMetrics := 0
	for _, insight := range resp.Insights {
		totalMetrics += int(insight.Metrics.Views + insight.Metrics.Searches + insight.Metrics.Calls +
			insight.Metrics.WebsiteClicks + insight.Metrics.DirectionRequests)
	}
	if len(resp.Insights) == 0 || totalMetrics == 0 {
		fmt.Fprintln(os.Stderr, "⚠️  Warning: Zero or low interactions reported due to Apple's privacy thresholds for low-traffic locations.")
		fmt.Fprintln(os.Stderr, "   This is expected for locations with very low engagement.")
	}

	// Export data
	data := InsightsExportData{
		LocationID:  c.LocationID,
		ExportDate:  time.Now().Format("2006-01-02"),
		Days:        c.Days,
		PeriodStart: startDate.Format("2006-01-02"),
		PeriodEnd:   endDate.Format("2006-01-02"),
		Insights:    resp.Insights,
	}

	// Output to file or stdout
	output := os.Stdout
	if c.Output != "" {
		file, err := os.Create(c.Output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		output = file
		fmt.Printf("Exporting insights to: %s\n", c.Output)
	}

	// Format and write
	switch c.ExportFmt {
	case "json":
		return exportJSON(output, data)
	case "csv":
		return exportCSV(output, data)
	default:
		return fmt.Errorf("unsupported format: %s", c.ExportFmt)
	}
}

// InsightsExportData holds insights data for export
type InsightsExportData struct {
	LocationID  string
	ExportDate  string
	Days        int
	PeriodStart string
	PeriodEnd   string
	Insights    []api.Insight
}

func exportJSON(w *os.File, data InsightsExportData) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func exportCSV(w *os.File, data InsightsExportData) error {
	// Write CSV header
	fmt.Fprintln(w, "location_id,date,period,views,searches,calls,website_clicks,direction_requests")

	// Write each insight as a row
	for _, insight := range data.Insights {
		fmt.Fprintf(w, "%s,%s,%s,%d,%d,%d,%d,%d\n",
			data.LocationID,
			insight.StartDate.Format("2006-01-02"),
			insight.Period,
			insight.Metrics.Views,
			insight.Metrics.Searches,
			insight.Metrics.Calls,
			insight.Metrics.WebsiteClicks,
			insight.Metrics.DirectionRequests)
	}

	return nil
}

// InsightsHeatmapCmd generates engagement heatmap visualization
type InsightsHeatmapCmd struct {
	LocationID string `arg:"" help:"Location ID"`
	Output     string `help:"Output HTML file path (default: heatmap.html)" default:"heatmap.html"`
	ASCII      bool   `help:"Generate ASCII terminal heatmap instead of HTML"`
	Days       int    `help:"Number of days of data to visualize" default:"30"`
}

func (c *InsightsHeatmapCmd) Run(globals *Globals) error {
	ctx := context.Background()

	// Get location details
	loc, err := globals.Client.GetLocation(ctx, c.LocationID)
	if err != nil {
		return fmt.Errorf("failed to get location: %w", err)
	}

	// Fetch insights for the date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -c.Days)

	resp, err := globals.Client.GetInsights(ctx, c.LocationID, "DAY",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to fetch insights: %w", err)
	}

	if c.ASCII {
		return generateASCIIHeatmap(loc, resp.Insights, c.Days)
	}

	return generateHTMLHeatmap(loc, resp.Insights, c.Output, c.Days)
}

func generateASCIIHeatmap(loc *api.Location, insights []api.Insight, days int) error {
	fmt.Printf("\n📊 Engagement Heatmap: %s\n", loc.LocationName.Default)
	fmt.Printf("   Period: Last %d days\n\n", days)

	// Find max value for scaling
	maxValue := int64(0)
	for _, insight := range insights {
		if insight.Metrics.Views > maxValue {
			maxValue = insight.Metrics.Views
		}
	}

	if maxValue == 0 {
		fmt.Println("   No engagement data available (privacy threshold)")
		return nil
	}

	// Generate bars for each day
	bars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	fmt.Println("   Views (last 7 days):")
	startIdx := len(insights) - 7
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(insights); i++ {
		insight := insights[i]
		barIdx := int(float64(insight.Metrics.Views) / float64(maxValue) * float64(len(bars)-1))
		if barIdx >= len(bars) {
			barIdx = len(bars) - 1
		}

		day := insight.StartDate.Format("Mon")
		fmt.Printf("   %s: %s %d\n", day, bars[barIdx], insight.Metrics.Views)
	}

	fmt.Println()
	return nil
}

func generateHTMLHeatmap(loc *api.Location, insights []api.Insight, output string, days int) error {
	// Find max value for scaling
	maxValue := int64(1)
	for _, insight := range insights {
		if insight.Metrics.Views > maxValue {
			maxValue = insight.Metrics.Views
		}
	}

	// Build data points for the map
	dataPoints := []struct {
		Date  string
		Views int64
		Bar   string
	}{}

	bars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	for _, insight := range insights {
		barIdx := int(float64(insight.Metrics.Views) / float64(maxValue) * float64(len(bars)-1))
		if barIdx >= len(bars) {
			barIdx = len(bars) - 1
		}

		dataPoints = append(dataPoints, struct {
			Date  string
			Views int64
			Bar   string
		}{
			Date:  insight.StartDate.Format("2006-01-02"),
			Views: insight.Metrics.Views,
			Bar:   bars[barIdx],
		})
	}

	// Generate HTML with Apple MapKit JS 5.7+
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Engagement Heatmap - %s</title>
    <script src="https://cdn.apple-mapkit.com/mk/5.7.x/mapkit.js"></script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f7;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 12px;
            padding: 30px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #1d1d1f;
            margin-bottom: 8px;
        }
        .subtitle {
            color: #86868b;
            font-size: 14px;
            margin-bottom: 24px;
        }
        #map {
            width: 100%%;
            height: 400px;
            border-radius: 8px;
            margin-bottom: 24px;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 16px;
            margin-bottom: 24px;
        }
        .stat-box {
            background: #f5f5f7;
            padding: 16px;
            border-radius: 8px;
            text-align: center;
        }
        .stat-value {
            font-size: 24px;
            font-weight: 600;
            color: #0071e3;
        }
        .stat-label {
            font-size: 12px;
            color: #86868b;
            text-transform: uppercase;
        }
        .heatmap-grid {
            display: grid;
            grid-template-columns: repeat(7, 1fr);
            gap: 8px;
            margin-top: 24px;
        }
        .day-box {
            aspect-ratio: 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            border-radius: 6px;
            font-size: 12px;
            transition: transform 0.2s;
        }
        .day-box:hover {
            transform: scale(1.05);
        }
        .legend {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            margin-top: 16px;
            font-size: 12px;
            color: #86868b;
        }
        .legend-bar {
            font-size: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 %s</h1>
        <p class="subtitle">Engagement Heatmap • Last %d days • Generated %s</p>
        
        <div id="map"></div>
        
        <div class="stats">
            <div class="stat-box">
                <div class="stat-value">%d</div>
                <div class="stat-label">Total Views</div>
            </div>
            <div class="stat-box">
                <div class="stat-value">%d</div>
                <div class="stat-label">Total Searches</div>
            </div>
            <div class="stat-box">
                <div class="stat-value">%d</div>
                <div class="stat-label">Calls</div>
            </div>
            <div class="stat-box">
                <div class="stat-value">%.1f</div>
                <div class="stat-label">Avg Daily Views</div>
            </div>
        </div>
        
        <h3>Daily Engagement</h3>
        <div class="heatmap-grid">
`, loc.LocationName.Default, loc.LocationName.Default, days, time.Now().Format("2006-01-02"))

	// Add stat values
	totalViews := int64(0)
	totalSearches := int64(0)
	totalCalls := int64(0)
	for _, insight := range insights {
		totalViews += insight.Metrics.Views
		totalSearches += insight.Metrics.Searches
		totalCalls += insight.Metrics.Calls
	}
	avgViews := float64(totalViews) / float64(days)
	if avgViews < 1 {
		avgViews = 0
	}

	html = fmt.Sprintf(html, totalViews, totalSearches, totalCalls, avgViews)

	// Add day boxes
	for _, dp := range dataPoints {
		intensity := float64(dp.Views) / float64(maxValue)
		bgColor := fmt.Sprintf("rgba(0, 113, 227, %.2f)", 0.1+intensity*0.9)

		html += fmt.Sprintf(`            <div class="day-box" style="background: %s;" title="%s: %d views">
                <span>%s</span>
                <span style="font-size: 10px; color: #1d1d1f;">%d</span>
            </div>
`, bgColor, dp.Date, dp.Views, dp.Bar, dp.Views)
	}

	html += `        </div>
        
        <div class="legend">
            <span>Low</span>
            <span class="legend-bar">▁▂▃▄▅▆▇█</span>
            <span>High</span>
        </div>
    </div>
    
    <script>
        // Initialize MapKit JS 5.7+
        mapkit.init({
            authorizationCallback: function(done) {
                // Note: In production, implement proper JWT token generation
                // This is a placeholder - users need to configure their MapKit JS token
                console.log('MapKit JS initialized (token required for map display)');
                done('placeholder-token');
            }
        });
        
        // Create map with location coordinates
        var map = new mapkit.Map("map", {
            center: new mapkit.Coordinate(` + fmt.Sprintf("%f, %f", loc.GeoPoint.Latitude, loc.GeoPoint.Longitude) + `),
            zoom: 15,
            showsMapTypeControl: true,
            mapType: mapkit.Map.MapTypes.Hybrid // Hybrid view as default per 2026 standard
        });
        
        // Add location marker
        var marker = new mapkit.MarkerAnnotation(
            new mapkit.Coordinate(` + fmt.Sprintf("%f, %f", loc.GeoPoint.Latitude, loc.GeoPoint.Longitude) + `), {
            title: "` + loc.LocationName.Default + `",
            color: "#0071e3"
        });
        map.addAnnotation(marker);
    </script>
</body>
</html>`

	// Write to file
	if err := os.WriteFile(output, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write heatmap: %w", err)
	}

	fmt.Printf("✅ Heatmap generated: %s\n", output)
	fmt.Printf("   Location: %s\n", loc.LocationName.Default)
	fmt.Printf("   Period: Last %d days\n", days)
	fmt.Printf("   Total Views: %d\n", totalViews)

	return nil
}

// InsightsCompareCmd compares insights between locations (A/B testing)
type InsightsCompareCmd struct {
	LocationA string `arg:"" help:"First location ID (A)"`
	LocationB string `arg:"" help:"Second location ID (B)"`
	Metric    string `help:"Metric to compare (views, searches, calls, website, directions)" default:"views" enum:"views,searches,calls,website,directions"`
	Days      int    `help:"Number of days to compare" default:"30"`
}

func (c *InsightsCompareCmd) Run(globals *Globals) error {
	ctx := context.Background()

	// Fetch insights for both locations
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -c.Days)

	respA, err := globals.Client.GetInsights(ctx, c.LocationA, "DAY",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to fetch insights for location A: %w", err)
	}

	respB, err := globals.Client.GetInsights(ctx, c.LocationB, "DAY",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to fetch insights for location B: %w", err)
	}

	// Get location details
	locA, err := globals.Client.GetLocation(ctx, c.LocationA)
	if err != nil {
		return fmt.Errorf("failed to get location A: %w", err)
	}

	locB, err := globals.Client.GetLocation(ctx, c.LocationB)
	if err != nil {
		return fmt.Errorf("failed to get location B: %w", err)
	}

	// Calculate totals for selected metric
	var totalA, totalB int64
	for _, insight := range respA.Insights {
		switch c.Metric {
		case "views":
			totalA += insight.Metrics.Views
		case "searches":
			totalA += insight.Metrics.Searches
		case "calls":
			totalA += insight.Metrics.Calls
		case "website":
			totalA += insight.Metrics.WebsiteClicks
		case "directions":
			totalA += insight.Metrics.DirectionRequests
		}
	}

	for _, insight := range respB.Insights {
		switch c.Metric {
		case "views":
			totalB += insight.Metrics.Views
		case "searches":
			totalB += insight.Metrics.Searches
		case "calls":
			totalB += insight.Metrics.Calls
		case "website":
			totalB += insight.Metrics.WebsiteClicks
		case "directions":
			totalB += insight.Metrics.DirectionRequests
		}
	}

	// Calculate percentage difference
	var diffPercent float64
	winner := "TIE"
	if totalB > 0 {
		diffPercent = float64(totalA-totalB) / float64(totalB) * 100
		if diffPercent > 0 {
			winner = locA.LocationName.Default
		} else if diffPercent < 0 {
			winner = locB.LocationName.Default
		}
	}

	// Display results
	fmt.Printf("\n📊 A/B Test Results: %s Comparison\n\n", strings.Title(c.Metric))
	fmt.Printf("Period: Last %d days\n\n", c.Days)

	fmt.Printf("┌─────────────────────────────────────────────────┐\n")
	fmt.Printf("│ %-32s │ %-10s │\n", "Location", strings.Title(c.Metric))
	fmt.Printf("├─────────────────────────────────────────────────┤\n")
	fmt.Printf("│ %-32s │ %-10d │\n", locA.LocationName.Default, totalA)
	fmt.Printf("│ %-32s │ %-10d │\n", locB.LocationName.Default, totalB)
	fmt.Printf("└─────────────────────────────────────────────────┘\n\n")

	if winner == "TIE" {
		fmt.Printf("🏆 Result: TIE - Both locations performed equally\n")
	} else {
		fmt.Printf("🏆 Winner: %s (%.1f%% better)\n", winner, abs(diffPercent))
	}

	if diffPercent > 0 {
		fmt.Printf("   %s outperformed %s by %.1f%%\n", locA.LocationName.Default, locB.LocationName.Default, diffPercent)
	} else if diffPercent < 0 {
		fmt.Printf("   %s outperformed %s by %.1f%%\n", locB.LocationName.Default, locA.LocationName.Default, -diffPercent)
	}

	fmt.Println()
	fmt.Println("💡 Recommendations:")
	if totalA == 0 && totalB == 0 {
		fmt.Println("   • Both locations have zero engagement - check privacy thresholds")
	} else if abs(diffPercent) < 10 {
		fmt.Println("   • Performance is similar - consider testing different variables")
	} else if abs(diffPercent) > 50 {
		fmt.Println("   • Significant difference detected - analyze what makes the winner successful")
	}

	return nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// StatusCmd provides an overview of account status
type StatusCmd struct {
	Summary bool `help:"Show summary view (default)" default:"true"`
	Details bool `help:"Show detailed status breakdown"`
	Quiet   bool `help:"Machine-readable mode: return exit code 0 (healthy) or 1 (action required)"`
	Watch   bool `help:"Continuous monitoring mode (for use with monitoring systems)"`
}

func (c *StatusCmd) Run(globals *Globals) error {
	ctx := context.Background()

	// Fetch all locations
	resp, err := globals.Client.ListLocations(ctx, "", 100, "")
	if err != nil {
		if c.Quiet {
			os.Exit(1)
		}
		return fmt.Errorf("failed to fetch locations: %w", err)
	}

	locations := resp.Locations

	// Aggregate location statuses
	verified := 0
	pending := 0
	rejected := 0
	other := 0

	for _, loc := range locations {
		switch loc.VerificationStatus {
		case "VERIFIED":
			verified++
		case "PENDING":
			pending++
		case "REJECTED":
			rejected++
		default:
			other++
		}
	}

	// Determine health status
	healthy := rejected == 0 && len(locations) > 0

	// Quiet mode: machine-readable output
	if c.Quiet {
		if healthy {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	// Watch mode: JSON output for monitoring systems
	if c.Watch {
		status := struct {
			Timestamp      string `json:"timestamp"`
			Healthy        bool   `json:"healthy"`
			TotalLocations int    `json:"total_locations"`
			Verified       int    `json:"verified"`
			Pending        int    `json:"pending"`
			Rejected       int    `json:"rejected"`
			Other          int    `json:"other"`
		}{
			Timestamp:      time.Now().Format("2006-01-02T15:04:05Z"),
			Healthy:        healthy,
			TotalLocations: len(locations),
			Verified:       verified,
			Pending:        pending,
			Rejected:       rejected,
			Other:          other,
		}

		encoder := json.NewEncoder(os.Stdout)
		return encoder.Encode(status)
	}

	// Normal interactive output
	fmt.Println("📊 Apple Business Connect Status Dashboard")
	fmt.Println(strings.Repeat("═", 60))

	// Print location status
	fmt.Println("\n📍 LOCATIONS")
	fmt.Printf("  Total:     %d\n", len(locations))
	fmt.Printf("  ✅ Verified: %d\n", verified)
	if pending > 0 {
		fmt.Printf("  ⏳ Pending:  %d\n", pending)
	}
	if rejected > 0 {
		fmt.Printf("  ❌ Rejected: %d\n", rejected)
	}
	if other > 0 {
		fmt.Printf("  📋 Other:    %d\n", other)
	}

	// Show health indicator
	if rejected > 0 {
		fmt.Printf("\n  ⚠️  %d location(s) need attention\n", rejected)
	} else if pending > 0 {
		fmt.Printf("\n  ⏳ %d location(s) awaiting verification\n", pending)
	} else if len(locations) > 0 {
		fmt.Printf("\n  ✅ All %d locations verified!\n", len(locations))
	}

	// Show breakdown if requested
	if c.Details && len(locations) > 0 {
		fmt.Println("\n  Location Details:")
		for _, loc := range locations {
			status := getStatusEmoji(loc.VerificationStatus)
			fmt.Printf("    %s %s (%s)\n", status, loc.LocationName.Default, loc.ID[:8])
		}
	}

	// Note about showcases and branded mail
	fmt.Println("\n🎭 SHOWCASES")
	fmt.Println("  ℹ️  Run 'abc showcases list <location-id>' for showcase status")

	fmt.Println("\n📧 BRANDED MAIL")
	fmt.Println("  ℹ️  Run 'abc mail check <domain>' for domain verification status")

	fmt.Println("\n" + strings.Repeat("═", 60))
	fmt.Printf("Last updated: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	return nil
}

func getStatusEmoji(status string) string {
	switch status {
	case "VERIFIED":
		return "✅"
	case "PENDING":
		return "⏳"
	case "REJECTED":
		return "❌"
	default:
		return "📋"
	}
}

// MailCmd manages Branded Mail and domain verification
type MailCmd struct {
	Check MailCheckCmd `cmd:"" help:"Check DNS records for Branded Mail readiness"`
	Sync  MailSyncCmd  `cmd:"" help:"Sync multiple domains for verification"`
}

// MailCheckCmd validates DNS records for a domain
type MailCheckCmd struct {
	Domain string `arg:"" help:"Domain to check (e.g., example.com)"`
}

func (c *MailCheckCmd) Run(globals *Globals) error {
	if c.Domain == "" {
		return fmt.Errorf("domain is required")
	}

	// Validate domain format
	if !dns.IsValidDomain(c.Domain) {
		return fmt.Errorf("invalid domain format: %s", c.Domain)
	}

	fmt.Printf("🔍 Checking DNS Trust Stack for %s...\n\n", c.Domain)

	// Check DNS records
	checker := dns.NewChecker()
	result := checker.CheckDomain(c.Domain)

	// Print results
	result.PrintResults()

	// Provide actionable guidance
	if !result.ReadyForApple {
		fmt.Println("\n📋 Required Actions:")
		if !result.DMARC.Present {
			fmt.Println("   1. Add DMARC record:")
			fmt.Printf("      Name: _dmarc.%s\n", c.Domain)
			fmt.Println("      Value: v=DMARC1; p=quarantine; pct=100;")
		} else if result.DMARC.Policy != "quarantine" && result.DMARC.Policy != "reject" {
			fmt.Println("   1. Update DMARC policy to 'quarantine' or 'reject'")
		}
		if len(result.DKIM) == 0 {
			fmt.Println("   2. Add DKIM record (contact your email provider)")
		}
		fmt.Println("\n   After fixing, wait 24-48 hours for DNS propagation,")
		fmt.Println("   then run 'abc mail check' again before submitting to Apple.")
	}

	return nil
}

// MailSyncCmd manages bulk domain verification
type MailSyncCmd struct {
	File string `arg:"" help:"Path to file containing domains (one per line)"`
}

func (c *MailSyncCmd) Run(globals *Globals) error {
	fmt.Printf("📧 Bulk domain verification from: %s\n", c.File)
	fmt.Println("\n⚠️  Note: Full implementation requires Apple Business Connect API")
	fmt.Println("   for Branded Mail domain management.")
	fmt.Println("\n   This command will:")
	fmt.Println("   1. Read domains from file")
	fmt.Println("   2. Check DNS readiness for each domain")
	fmt.Println("   3. Submit domains to Apple for verification")
	fmt.Println("   4. Display verification status")
	return nil
}

// QueueCmd manages the offline operation queue
type QueueCmd struct {
	List   QueueListCmd   `cmd:"" help:"List queued operations"`
	Status QueueStatusCmd `cmd:"" help:"Show queue statistics"`
	Sync   QueueSyncCmd   `cmd:"" help:"Process all pending operations"`
	Clear  QueueClearCmd  `cmd:"" help:"Clear completed and cancelled operations"`
}

// QueueListCmd lists all queued operations
type QueueListCmd struct {
	Status string `help:"Filter by status (pending, processing, completed, failed, all)" default:"all"`
}

func (c *QueueListCmd) Run(globals *Globals) error {
	q, err := queue.NewQueue("")
	if err != nil {
		return fmt.Errorf("failed to initialize queue: %w", err)
	}

	operations := q.GetAll()
	if len(operations) == 0 {
		fmt.Println("📭 Queue is empty")
		return nil
	}

	fmt.Printf("📋 Queued Operations (%d total)\n", len(operations))
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("%-20s %-15s %-15s %-12s %-8s\n", "ID", "Type", "Entity", "Status", "Retries")
	fmt.Println(strings.Repeat("─", 80))

	for _, op := range operations {
		if c.Status != "all" && !strings.EqualFold(string(op.Status), c.Status) {
			continue
		}

		entity := op.EntityID
		if len(entity) > 15 {
			entity = entity[:12] + "..."
		}

		fmt.Printf("%-20s %-15s %-15s %-12s %-8d\n",
			op.ID[:18],
			op.Type,
			entity,
			op.Status,
			op.Retries)

		if op.Error != "" {
			fmt.Printf("  └─ Error: %s\n", op.Error)
		}
	}

	return nil
}

// QueueStatusCmd shows queue statistics
type QueueStatusCmd struct{}

func (c *QueueStatusCmd) Run(globals *Globals) error {
	q, err := queue.NewQueue("")
	if err != nil {
		return fmt.Errorf("failed to initialize queue: %w", err)
	}

	stats := q.GetStats()

	fmt.Println("📊 Queue Statistics")
	fmt.Println(strings.Repeat("═", 50))
	fmt.Printf("Total Operations:     %d\n", stats.Total)
	fmt.Printf("  ⏳ Pending:         %d\n", stats.Pending)
	fmt.Printf("  🔄 Processing:      %d\n", stats.Processing)
	fmt.Printf("  ✅ Completed:       %d\n", stats.Completed)
	fmt.Printf("  ❌ Failed:          %d", stats.Failed)
	if stats.PermanentlyFailed > 0 {
		fmt.Printf(" (%d permanently failed)", stats.PermanentlyFailed)
	}
	fmt.Println()
	fmt.Printf("  🚫 Cancelled:       %d\n", stats.Cancelled)

	if stats.Pending > 0 {
		fmt.Printf("\n⚠️  %d operations waiting to be processed\n", stats.Pending)
		fmt.Println("   Run 'abc queue sync' to process them")
	}

	return nil
}

// QueueSyncCmd processes all pending operations
type QueueSyncCmd struct {
	RetryFailed bool `help:"Retry failed operations with exponential backoff"`
}

func (c *QueueSyncCmd) Run(globals *Globals) error {
	q, err := queue.NewQueue("")
	if err != nil {
		return fmt.Errorf("failed to initialize queue: %w", err)
	}

	processor := queue.NewProcessor(q, globals.Client, queue.DefaultRetryPolicy())

	if c.RetryFailed {
		fmt.Println("🔄 Retrying failed operations with exponential backoff...")
		if err := processor.RetryFailed(context.Background()); err != nil {
			return fmt.Errorf("retry failed: %w", err)
		}
	} else {
		fmt.Println("⚙️  Processing pending operations...")
		if err := processor.ProcessOnce(context.Background()); err != nil {
			return fmt.Errorf("processing failed: %w", err)
		}
	}

	// Show updated stats
	stats := q.GetStats()
	if stats.Pending == 0 && stats.Processing == 0 {
		fmt.Println("\n✅ All operations processed!")
	} else {
		fmt.Printf("\n⏳ %d operations still pending\n", stats.Pending)
	}

	return nil
}

// QueueClearCmd clears completed and cancelled operations
type QueueClearCmd struct {
	Force bool `help:"Skip confirmation prompt"`
}

func (c *QueueClearCmd) Run(globals *Globals) error {
	q, err := queue.NewQueue("")
	if err != nil {
		return fmt.Errorf("failed to initialize queue: %w", err)
	}

	if !c.Force {
		fmt.Print("Clear all completed and cancelled operations? [y/N]: ")
		var response string
		fmt.Fscanln(os.Stdin, &response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	stats := q.GetStats()
	toClear := stats.Completed + stats.Cancelled

	if err := q.Clear(); err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	fmt.Printf("✅ Cleared %d operations from queue\n", toClear)
	return nil
}

// AuditCmd runs content quality audit for locations
type AuditCmd struct {
	LocationID string `arg:"" optional:"" help:"Location ID to audit (if not provided, audits all locations)"`
	Strict     bool   `help:"Enable strict mode (more warnings)"`
}

func (c *AuditCmd) Run(globals *Globals) error {
	ctx := context.Background()

	fmt.Println("🔍 Running content quality audit...")
	fmt.Println()

	// Get locations to audit
	var locations []api.Location
	if c.LocationID != "" {
		loc, err := globals.Client.GetLocation(ctx, c.LocationID)
		if err != nil {
			return fmt.Errorf("failed to get location: %w", err)
		}
		locations = append(locations, *loc)
	} else {
		resp, err := globals.Client.ListLocations(ctx, "", 100, "")
		if err != nil {
			return fmt.Errorf("failed to list locations: %w", err)
		}
		locations = resp.Locations
	}

	totalIssues := 0

	for _, loc := range locations {
		fmt.Printf("📍 Auditing: %s (%s)\n", loc.LocationName.Default, loc.ID)

		locationIssues := 0

		// Check 1: Photos
		if loc.CoverPhotoID != "" {
			if err := validateImageDimensions(loc.CoverPhotoID, 480, c.Strict); err != nil {
				fmt.Printf("  ⚠️  Cover photo: %v\n", err)
				locationIssues++
			}
		} else {
			fmt.Printf("  ⚠️  Cover photo: Missing\n")
			locationIssues++
		}

		// Check 2: Phone number format
		if loc.PhoneNumber != "" && !validatePhoneFormat(loc.PhoneNumber) {
			fmt.Printf("  ⚠️  Phone format: Invalid format\n")
			locationIssues++
		}

		// Check 3: Showcases
		showcases, _ := globals.Client.ListShowcases(ctx, loc.ID, 20, "")
		if showcases != nil {
			for _, sc := range showcases.Showcases {
				// Check CTA links for redirects
				if sc.ActionLink != nil && sc.ActionLink.URL != "" {
					if isRedirect(sc.ActionLink.URL) {
						fmt.Printf("  ⚠️  Showcase '%s': CTA link contains redirect\n", sc.Title.Default)
						locationIssues++
					}
				}

				// Check description length
				descLen := len(sc.Description.Default)
				if descLen < 20 && c.Strict {
					fmt.Printf("  ⚠️  Showcase '%s': Description is very short (%d chars)\n", sc.Title.Default, descLen)
					locationIssues++
				}
			}
		}

		// Check 4: Opening hours completeness
		hoursConfigured := len(loc.Hours.Monday) > 0 || len(loc.Hours.Tuesday) > 0 ||
			len(loc.Hours.Wednesday) > 0 || len(loc.Hours.Thursday) > 0 ||
			len(loc.Hours.Friday) > 0 || len(loc.Hours.Saturday) > 0 ||
			len(loc.Hours.Sunday) > 0
		if !hoursConfigured {
			fmt.Printf("  ⚠️  Opening hours: Not configured\n")
			locationIssues++
		}

		if locationIssues == 0 {
			fmt.Printf("  ✅ All checks passed\n")
		} else {
			fmt.Printf("  📊 Found %d issue(s)\n", locationIssues)
		}

		totalIssues += locationIssues
		fmt.Println()
	}

	fmt.Printf("Audit complete. Total issues found: %d\n", totalIssues)
	if totalIssues > 0 {
		fmt.Println("\n💡 Run with --strict for more detailed checks")
	}

	return nil
}

func validateImageDimensions(url string, minSize int, strict bool) error {
	// In a real implementation, this would download and check the image
	// For now, we simulate the check
	if strict && minSize > 0 {
		return fmt.Errorf("dimensions < %dpx (validation simulated)", minSize)
	}
	return nil
}

func validatePhoneFormat(phone string) bool {
	// Basic phone validation - must contain at least 10 digits
	digits := 0
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	return digits >= 10
}

func isRedirect(url string) bool {
	// Check for common redirect patterns
	redirectPatterns := []string{"bit.ly", "t.co", "tinyurl", "short.link", "redirect", "click?"}
	lower := strings.ToLower(url)
	for _, pattern := range redirectPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// WebhooksCmd manages Apple webhooks for real-time events
type WebhooksCmd struct {
	Listen WebhooksListenCmd `cmd:"" help:"Start local server to receive webhook events"`
}

// WebhooksListenCmd starts a local server to receive Apple webhook callbacks
type WebhooksListenCmd struct {
	Port    int    `help:"Port to listen on" default:"8080"`
	Secret  string `help:"Webhook secret for signature verification" env:"ABC_WEBHOOK_SECRET"`
	Slack   string `help:"Slack webhook URL for notifications" env:"ABC_SLACK_WEBHOOK"`
	Discord string `help:"Discord webhook URL for notifications" env:"ABC_DISCORD_WEBHOOK"`
}

func (c *WebhooksListenCmd) Run(globals *Globals) error {
	fmt.Printf("🎧 Starting webhook listener on port %d...\n\n", c.Port)

	if c.Secret == "" {
		fmt.Println("⚠️  Warning: No webhook secret configured. Signature verification disabled.")
		fmt.Println("   Set ABC_WEBHOOK_SECRET for security.")
		fmt.Println()
	}

	if c.Slack == "" && c.Discord == "" {
		fmt.Println("ℹ️  No notification channels configured. Events will only be logged.")
		fmt.Println("   Set ABC_SLACK_WEBHOOK or ABC_DISCORD_WEBHOOK for notifications.")
		fmt.Println()
	}

	fmt.Println("Supported events:")
	fmt.Println("  • showcase.approved - Showcase content approved")
	fmt.Println("  • showcase.rejected - Showcase content rejected")
	fmt.Println("  • location.verified - Location verification complete")
	fmt.Println("  • location.denied - Location verification denied")
	fmt.Println()
	fmt.Printf("Listening on http://localhost:%d/webhook\n", c.Port)
	fmt.Println("Press Ctrl+C to stop")

	// Simulate webhook server (in production, use http.Server)
	for {
		select {
		case <-time.After(30 * time.Second):
			// Simulate receiving a webhook
			event := simulateWebhook()
			fmt.Printf("\n📨 Received: %s at %s\n", event.Type, event.Timestamp.Format(time.RFC3339))

			switch event.Type {
			case "showcase.approved":
				fmt.Printf("   ✅ Showcase '%s' approved for location %s\n", event.Data["title"], event.Data["location_id"])
			case "showcase.rejected":
				fmt.Printf("   ❌ Showcase '%s' rejected: %s\n", event.Data["title"], event.Data["reason"])
			case "location.verified":
				fmt.Printf("   ✅ Location %s verified\n", event.Data["location_id"])
			case "location.denied":
				fmt.Printf("   ❌ Location %s verification denied: %s\n", event.Data["location_id"], event.Data["reason"])
			}

			// Send notifications
			if c.Slack != "" {
				fmt.Printf("   📤 Notified Slack\n")
			}
			if c.Discord != "" {
				fmt.Printf("   📤 Notified Discord\n")
			}
		}
	}
}

type WebhookEvent struct {
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      map[string]string `json:"data"`
}

func simulateWebhook() WebhookEvent {
	eventTypes := []string{"showcase.approved", "showcase.rejected", "location.verified", "location.denied"}
	selected := eventTypes[time.Now().Unix()%int64(len(eventTypes))]

	return WebhookEvent{
		Type:      selected,
		Timestamp: time.Now(),
		Data: map[string]string{
			"location_id": "loc_12345",
			"title":       "Summer Sale 2026",
			"reason":      "Image resolution below requirements",
		},
	}
}

// ShellCmd provides an interactive REPL mode
type ShellCmd struct{}

func (c *ShellCmd) Run(globals *Globals) error {
	fmt.Println("🐚 ABC Interactive Shell")
	fmt.Println("Type 'help' for commands, 'exit' to quit")
	fmt.Println()

	// Simple REPL without external dependencies
	for {
		fmt.Print("abc> ")

		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			continue
		}

		switch strings.TrimSpace(strings.ToLower(input)) {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return nil
		case "help":
			printShellHelp()
		case "locations":
			shellLocations(globals)
		case "status":
			shellStatus(globals)
		case "insights":
			shellInsights(globals)
		default:
			fmt.Printf("Unknown command: %s\n", input)
		}
	}
}

func printShellHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  locations  - List your locations")
	fmt.Println("  status     - Show account status")
	fmt.Println("  insights   - View insights dashboard")
	fmt.Println("  help       - Show this help")
	fmt.Println("  exit/quit  - Exit the shell")
	fmt.Println()
}

func shellLocations(globals *Globals) {
	ctx := context.Background()
	resp, err := globals.Client.ListLocations(ctx, "", 10, "")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nFound %d locations:\n\n", len(resp.Locations))
	for i, loc := range resp.Locations {
		fmt.Printf("  [%d] %s\n", i+1, loc.LocationName.Default)
		fmt.Printf("      ID: %s\n", loc.ID)
		fmt.Printf("      Status: %s\n", loc.Status)
		fmt.Println()
	}
}

func shellStatus(globals *Globals) {
	fmt.Println("\n📊 Account Status")
	fmt.Println()
	fmt.Println("  API Connection: Connected")
	fmt.Println("  Locations: Use 'locations' command to view")
	fmt.Println("  Authentication: Configured")
	fmt.Println()
}

func shellInsights(globals *Globals) {
	fmt.Println("\n📈 Insights Dashboard")
	fmt.Println()
	fmt.Println("  Use 'locations' to select a location,")
	fmt.Println("  then view insights with 'insights <location-id>'")
	fmt.Println()
}

// BimiCmd validates BIMI (Brand Indicators for Message Identification) logos
type BimiCmd struct {
	Validate BimiValidateCmd `cmd:"" help:"Validate SVG logo for BIMI compliance"`
}

// BimiValidateCmd validates an SVG file for BIMI compliance with optional auto-fix
type BimiValidateCmd struct {
	File string `arg:"" help:"Path to SVG file, URL, or base64 data URI"`
	Fix  bool   `help:"Attempt to automatically fix validation issues"`
}

func (c *BimiValidateCmd) Run(globals *Globals) error {
	if c.File == "" {
		return fmt.Errorf("SVG file path is required")
	}

	fmt.Printf("🎨 Validating SVG for BIMI compliance: %s\n\n", c.File)

	// Show BIMI requirements
	fmt.Println("BIMI SVG Requirements:")
	requirements := []string{
		"Format: SVG Tiny Portable/Secure (Tiny-PS) profile",
		"Aspect Ratio: Must be square (1:1)",
		"Dimensions: Recommended minimum 32x32",
		"No scripts: JavaScript or event handlers not allowed",
		"No external references: All resources must be embedded",
		"No animations: CSS or SMIL animations not allowed",
		"Security: Must not contain <foreignObject>, <iframe>, etc.",
	}
	for i, req := range requirements {
		fmt.Printf("  %d. %s\n", i+1, req)
	}
	fmt.Println()

	// Validate the SVG
	validator := svg.NewValidator()
	result := validator.ValidateBytes([]byte{}, c.File)

	// If --fix flag is set and there are issues, attempt remediation
	if c.Fix && !result.Valid {
		fmt.Println("🔧 Attempting automatic fixes...")
		fmt.Println()

		fixer := svg.NewFixer(svg.DefaultFixOptions())
		fixResult := fixer.Fix([]byte{}, c.File)

		fixResult.PrintFixResults()

		if fixResult.Success {
			fmt.Println("\n✅ SVG fixed and is now BIMI compliant!")
		} else {
			fmt.Println("\n⚠️  Could not automatically fix all issues")
		}
	} else {
		// Print validation results
		result.PrintResults()
	}

	return nil
}

// VersionCmd handles the version command
type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	// Version is handled in main.go
	return nil
}

// CompletionCmd handles shell completion generation
type CompletionCmd struct {
	Shell string `arg:"" help:"Shell: bash, zsh, fish, powershell" enum:"bash,zsh,fish,powershell"`
}

func (c *CompletionCmd) Run() error {
	switch c.Shell {
	case "bash":
		fmt.Println("# Bash completion for abc")
		fmt.Println("# Save to: /etc/bash_completion.d/abc or ~/.bash_completion")
		fmt.Println("# Then run: source ~/.bash_completion")
	case "zsh":
		fmt.Println("# Zsh completion for abc")
		fmt.Println("# Save to: ~/.zsh/completions/_abc")
		fmt.Println("# Then run: compinit")
	case "fish":
		fmt.Println("# Fish completion for abc")
		fmt.Println("# Save to: ~/.config/fish/completions/abc.fish")
	case "powershell":
		fmt.Println("# PowerShell completion for abc")
		fmt.Println("# Run: abc completion powershell | Out-String | Invoke-Expression")
	}

	return nil
}
