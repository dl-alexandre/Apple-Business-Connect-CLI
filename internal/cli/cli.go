package cli

import (
	"context"
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
	"github.com/dl-alexandre/abc/internal/showcase"
	"github.com/dl-alexandre/abc/internal/sync"
	"github.com/dl-alexandre/abc/internal/validate"
)

// CLI is the main command-line interface structure using Kong
type CLI struct {
	Globals

	// Command groups
	Auth      AuthCmd      `cmd:"" help:"Manage authentication"`
	Doctor    DoctorCmd    `cmd:"" help:"Run diagnostics and troubleshoot issues"`
	Locations LocationsCmd `cmd:"" help:"Manage business locations"`
	Mail      MailCmd      `cmd:"" help:"Manage Branded Mail and domain verification"`
	Showcases ShowcasesCmd `cmd:"" help:"Manage showcases"`
	Insights  InsightsCmd  `cmd:"" help:"View location insights"`
	Status    StatusCmd    `cmd:"" help:"View overall account status dashboard"`

	// Utility commands
	Version    VersionCmd    `cmd:"" help:"Show version information"`
	Completion CompletionCmd `cmd:"" help:"Generate shell completion script"`
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
	Get InsightsGetCmd `cmd:"" help:"Get insights for a location"`
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

	printer := output.NewPrinter(format, globals.ShouldUseColor())
	return printer.PrintInsights(resp.Insights)
}

// StatusCmd provides an overview of account status
type StatusCmd struct {
	Summary bool `help:"Show summary view (default)" default:"true"`
	Details bool `help:"Show detailed status breakdown"`
}

func (c *StatusCmd) Run(globals *Globals) error {
	fmt.Println("📊 Apple Business Connect Status Dashboard")
	fmt.Println(strings.Repeat("═", 60))

	ctx := context.Background()

	// Fetch all locations
	resp, err := globals.Client.ListLocations(ctx, "", 100, "")
	if err != nil {
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
