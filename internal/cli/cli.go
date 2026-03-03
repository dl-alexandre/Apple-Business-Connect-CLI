package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"

	"github.com/dl-alexandre/abc/internal/api"
	"github.com/dl-alexandre/abc/internal/cache"
	"github.com/dl-alexandre/abc/internal/config"
	"github.com/dl-alexandre/abc/internal/output"
)

// CLI is the main command-line interface structure using Kong
type CLI struct {
	Globals

	// Command groups
	Locations LocationsCmd `cmd:"" help:"Manage business locations"`
	Showcases ShowcasesCmd `cmd:"" help:"Manage showcases"`
	Insights  InsightsCmd  `cmd:"" help:"View location insights"`

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

// LocationsCmd is the parent command for location operations
type LocationsCmd struct {
	List   LocationsListCmd   `cmd:"" help:"List all locations"`
	Get    LocationsGetCmd    `cmd:"" help:"Get a location by ID"`
	Create LocationsCreateCmd `cmd:"" help:"Create a new location"`
	Update LocationsUpdateCmd `cmd:"" help:"Update a location"`
	Delete LocationsDeleteCmd `cmd:"" help:"Delete a location"`
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

// ShowcasesCmd is the parent command for showcase operations
type ShowcasesCmd struct {
	List   ShowcasesListCmd   `cmd:"" help:"List showcases for a location"`
	Get    ShowcasesGetCmd    `cmd:"" help:"Get a showcase by ID"`
	Create ShowcasesCreateCmd `cmd:"" help:"Create a new showcase"`
	Update ShowcasesUpdateCmd `cmd:"" help:"Update a showcase"`
	Delete ShowcasesDeleteCmd `cmd:"" help:"Delete a showcase"`
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
