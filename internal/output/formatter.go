package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rodaine/table"

	"github.com/dl-alexandre/abc/internal/api"
)

// Printer handles output formatting
type Printer struct {
	format   string
	useColor bool
}

// NewPrinter creates a new output printer
func NewPrinter(format string, useColor bool) *Printer {
	return &Printer{
		format:   format,
		useColor: useColor,
	}
}

// PrintLocations prints a list of locations in the specified format
func (p *Printer) PrintLocations(locations []api.Location) error {
	switch p.format {
	case "json":
		return p.printJSON(locations)
	case "markdown":
		return p.printLocationsMarkdown(locations)
	case "table":
		return p.printLocationsTable(locations)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintLocation prints a single location in the specified format
func (p *Printer) PrintLocation(location *api.Location) error {
	switch p.format {
	case "json":
		return p.printJSON(location)
	case "markdown":
		return p.printLocationMarkdown(location)
	case "table":
		return p.printLocationTable(location)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintShowcases prints a list of showcases in the specified format
func (p *Printer) PrintShowcases(showcases []api.Showcase) error {
	switch p.format {
	case "json":
		return p.printJSON(showcases)
	case "markdown":
		return p.printShowcasesMarkdown(showcases)
	case "table":
		return p.printShowcasesTable(showcases)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintShowcase prints a single showcase in the specified format
func (p *Printer) PrintShowcase(showcase *api.Showcase) error {
	switch p.format {
	case "json":
		return p.printJSON(showcase)
	case "markdown":
		return p.printShowcaseMarkdown(showcase)
	case "table":
		return p.printShowcaseTable(showcase)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintInsights prints a list of insights in the specified format
func (p *Printer) PrintInsights(insights []api.Insight) error {
	switch p.format {
	case "json":
		return p.printJSON(insights)
	case "markdown":
		return p.printInsightsMarkdown(insights)
	case "table":
		return p.printInsightsTable(insights)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// printJSON outputs data as formatted JSON
func (p *Printer) printJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// printLocationsTable outputs locations as a formatted table
func (p *Printer) printLocationsTable(locations []api.Location) error {
	if len(locations) == 0 {
		fmt.Println("No locations found.")
		return nil
	}

	tbl := table.New("ID", "Name", "Status", "Address", "Phone", "Verification").
		WithWriter(os.Stdout)

	if p.useColor {
		tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
			return fmt.Sprintf("\033[1m%s\033[0m", fmt.Sprintf(format, vals...))
		})
	}

	for _, loc := range locations {
		address := formatAddress(loc.PrimaryAddress)
		statusColor := getStatusColor(loc.Status)
		verifyColor := getVerificationColor(loc.VerificationStatus)

		tbl.AddRow(
			truncate(loc.ID, 20),
			truncate(loc.LocationName.Default, 30),
			statusColor+truncate(loc.Status, 12)+"\033[0m",
			truncate(address, 35),
			truncate(loc.PhoneNumber, 15),
			verifyColor+truncate(loc.VerificationStatus, 12)+"\033[0m",
		)
	}

	tbl.Print()
	fmt.Printf("\nShowing %d locations\n", len(locations))

	return nil
}

// printLocationsMarkdown outputs locations as markdown
func (p *Printer) printLocationsMarkdown(locations []api.Location) error {
	if len(locations) == 0 {
		fmt.Println("No locations found.")
		return nil
	}

	fmt.Println("# Locations")
	fmt.Println()

	for _, loc := range locations {
		fmt.Printf("## %s\n\n", loc.LocationName.Default)
		fmt.Printf("**ID:** %s\n\n", loc.ID)
		fmt.Printf("**Status:** %s\n\n", loc.Status)
		fmt.Printf("**Verification:** %s\n\n", loc.VerificationStatus)

		if loc.LocationURL != "" {
			fmt.Printf("**URL:** %s\n\n", loc.LocationURL)
		}

		fmt.Println("**Address:**")
		fmt.Printf("```\n%s\n```\n\n", formatAddressFull(loc.PrimaryAddress))

		if loc.PhoneNumber != "" {
			fmt.Printf("**Phone:** %s\n\n", loc.PhoneNumber)
		}

		if len(loc.Categories) > 0 {
			fmt.Printf("**Categories:** %s\n\n", strings.Join(loc.Categories, ", "))
		}

		fmt.Printf("**Created:** %s\n\n", loc.CreatedAt.Format(time.RFC3339))
		fmt.Printf("**Updated:** %s\n\n", loc.UpdatedAt.Format(time.RFC3339))
		fmt.Println("---")
		fmt.Println()
	}

	return nil
}

// printLocationTable prints a single location as a table
func (p *Printer) printLocationTable(location *api.Location) error {
	tbl := table.New("Property", "Value").WithWriter(os.Stdout)

	if p.useColor {
		tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
			return fmt.Sprintf("\033[1m%s\033[0m", fmt.Sprintf(format, vals...))
		})
	}

	tbl.AddRow("ID", location.ID)
	tbl.AddRow("Company ID", location.CompanyID)
	tbl.AddRow("Name", location.LocationName.Default)
	tbl.AddRow("Status", location.Status)
	tbl.AddRow("Verification", location.VerificationStatus)

	if location.LocationURL != "" {
		tbl.AddRow("URL", location.LocationURL)
	}

	tbl.AddRow("Address", formatAddress(location.PrimaryAddress))

	if location.GeoPoint.Latitude != 0 || location.GeoPoint.Longitude != 0 {
		tbl.AddRow("Coordinates", fmt.Sprintf("%.6f, %.6f", location.GeoPoint.Latitude, location.GeoPoint.Longitude))
	}

	if location.PhoneNumber != "" {
		tbl.AddRow("Phone", location.PhoneNumber)
	}

	if len(location.Categories) > 0 {
		tbl.AddRow("Categories", strings.Join(location.Categories, ", "))
	}

	tbl.AddRow("Created", formatTime(location.CreatedAt))
	tbl.AddRow("Updated", formatTime(location.UpdatedAt))

	tbl.Print()

	return nil
}

// printLocationMarkdown prints a single location as markdown
func (p *Printer) printLocationMarkdown(location *api.Location) error {
	fmt.Printf("# %s\n\n", location.LocationName.Default)
	fmt.Printf("**ID:** %s\n\n", location.ID)
	fmt.Printf("**Company ID:** %s\n\n", location.CompanyID)
	fmt.Printf("**Status:** %s\n\n", location.Status)
	fmt.Printf("**Verification:** %s\n\n", location.VerificationStatus)

	if location.LocationURL != "" {
		fmt.Printf("**URL:** %s\n\n", location.LocationURL)
	}

	fmt.Println("**Address:**")
	fmt.Printf("```\n%s\n```\n\n", formatAddressFull(location.PrimaryAddress))

	if location.GeoPoint.Latitude != 0 || location.GeoPoint.Longitude != 0 {
		fmt.Printf("**Coordinates:** %.6f, %.6f\n\n", location.GeoPoint.Latitude, location.GeoPoint.Longitude)
	}

	if location.PhoneNumber != "" {
		fmt.Printf("**Phone:** %s\n\n", location.PhoneNumber)
	}

	if len(location.Categories) > 0 {
		fmt.Printf("**Categories:** %s\n\n", strings.Join(location.Categories, ", "))
	}

	fmt.Printf("**Created:** %s\n\n", location.CreatedAt.Format(time.RFC3339))
	fmt.Printf("**Updated:** %s\n\n", location.UpdatedAt.Format(time.RFC3339))

	return nil
}

// printShowcasesTable outputs showcases as a formatted table
func (p *Printer) printShowcasesTable(showcases []api.Showcase) error {
	if len(showcases) == 0 {
		fmt.Println("No showcases found.")
		return nil
	}

	tbl := table.New("ID", "Title", "Type", "Status", "Start Date", "End Date").
		WithWriter(os.Stdout)

	if p.useColor {
		tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
			return fmt.Sprintf("\033[1m%s\033[0m", fmt.Sprintf(format, vals...))
		})
	}

	for _, sc := range showcases {
		startDate := "-"
		if !sc.StartDate.IsZero() {
			startDate = formatTime(sc.StartDate)
		}
		endDate := "-"
		if !sc.EndDate.IsZero() {
			endDate = formatTime(sc.EndDate)
		}

		tbl.AddRow(
			truncate(sc.ID, 20),
			truncate(sc.Title.Default, 30),
			sc.Type,
			sc.Status,
			startDate,
			endDate,
		)
	}

	tbl.Print()
	fmt.Printf("\nShowing %d showcases\n", len(showcases))

	return nil
}

// printShowcasesMarkdown outputs showcases as markdown
func (p *Printer) printShowcasesMarkdown(showcases []api.Showcase) error {
	if len(showcases) == 0 {
		fmt.Println("No showcases found.")
		return nil
	}

	fmt.Println("# Showcases")
	fmt.Println()

	for _, sc := range showcases {
		fmt.Printf("## %s\n\n", sc.Title.Default)
		fmt.Printf("**ID:** %s\n\n", sc.ID)
		fmt.Printf("**Location ID:** %s\n\n", sc.LocationID)
		fmt.Printf("**Type:** %s\n\n", sc.Type)
		fmt.Printf("**Status:** %s\n\n", sc.Status)

		if sc.Description.Default != "" {
			fmt.Printf("**Description:** %s\n\n", sc.Description.Default)
		}

		if !sc.StartDate.IsZero() {
			fmt.Printf("**Start Date:** %s\n\n", sc.StartDate.Format(time.RFC3339))
		}
		if !sc.EndDate.IsZero() {
			fmt.Printf("**End Date:** %s\n\n", sc.EndDate.Format(time.RFC3339))
		}

		if sc.ActionLink != nil {
			fmt.Printf("**Action Link:** [%s](%s)\n\n", sc.ActionLink.Title.Default, sc.ActionLink.URL)
		}

		fmt.Printf("**Created:** %s\n\n", sc.CreatedAt.Format(time.RFC3339))
		fmt.Printf("**Updated:** %s\n\n", sc.UpdatedAt.Format(time.RFC3339))
		fmt.Println("---")
		fmt.Println()
	}

	return nil
}

// printShowcaseTable prints a single showcase as a table
func (p *Printer) printShowcaseTable(showcase *api.Showcase) error {
	tbl := table.New("Property", "Value").WithWriter(os.Stdout)

	if p.useColor {
		tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
			return fmt.Sprintf("\033[1m%s\033[0m", fmt.Sprintf(format, vals...))
		})
	}

	tbl.AddRow("ID", showcase.ID)
	tbl.AddRow("Location ID", showcase.LocationID)
	tbl.AddRow("Title", showcase.Title.Default)
	tbl.AddRow("Type", showcase.Type)
	tbl.AddRow("Status", showcase.Status)

	if showcase.Description.Default != "" {
		tbl.AddRow("Description", showcase.Description.Default)
	}

	if !showcase.StartDate.IsZero() {
		tbl.AddRow("Start Date", formatTime(showcase.StartDate))
	}
	if !showcase.EndDate.IsZero() {
		tbl.AddRow("End Date", formatTime(showcase.EndDate))
	}

	if showcase.ActionLink != nil {
		tbl.AddRow("Action Link", showcase.ActionLink.URL)
		tbl.AddRow("Action Title", showcase.ActionLink.Title.Default)
	}

	tbl.AddRow("Created", formatTime(showcase.CreatedAt))
	tbl.AddRow("Updated", formatTime(showcase.UpdatedAt))

	tbl.Print()

	return nil
}

// printShowcaseMarkdown prints a single showcase as markdown
func (p *Printer) printShowcaseMarkdown(showcase *api.Showcase) error {
	fmt.Printf("# %s\n\n", showcase.Title.Default)
	fmt.Printf("**ID:** %s\n\n", showcase.ID)
	fmt.Printf("**Location ID:** %s\n\n", showcase.LocationID)
	fmt.Printf("**Type:** %s\n\n", showcase.Type)
	fmt.Printf("**Status:** %s\n\n", showcase.Status)

	if showcase.Description.Default != "" {
		fmt.Printf("**Description:** %s\n\n", showcase.Description.Default)
	}

	if !showcase.StartDate.IsZero() {
		fmt.Printf("**Start Date:** %s\n\n", showcase.StartDate.Format(time.RFC3339))
	}
	if !showcase.EndDate.IsZero() {
		fmt.Printf("**End Date:** %s\n\n", showcase.EndDate.Format(time.RFC3339))
	}

	if showcase.ActionLink != nil {
		fmt.Printf("**Action Link:** [%s](%s)\n\n", showcase.ActionLink.Title.Default, showcase.ActionLink.URL)
	}

	fmt.Printf("**Created:** %s\n\n", showcase.CreatedAt.Format(time.RFC3339))
	fmt.Printf("**Updated:** %s\n\n", showcase.UpdatedAt.Format(time.RFC3339))

	return nil
}

// printInsightsTable outputs insights as a formatted table
func (p *Printer) printInsightsTable(insights []api.Insight) error {
	if len(insights) == 0 {
		fmt.Println("No insights found.")
		return nil
	}

	tbl := table.New("Location ID", "Period", "Views", "Searches", "Calls", "Website Clicks", "Directions").
		WithWriter(os.Stdout)

	if p.useColor {
		tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
			return fmt.Sprintf("\033[1m%s\033[0m", fmt.Sprintf(format, vals...))
		})
	}

	for _, in := range insights {
		tbl.AddRow(
			truncate(in.LocationID, 20),
			in.Period,
			formatNumber(in.Metrics.Views),
			formatNumber(in.Metrics.Searches),
			formatNumber(in.Metrics.Calls),
			formatNumber(in.Metrics.WebsiteClicks),
			formatNumber(in.Metrics.DirectionRequests),
		)
	}

	tbl.Print()
	fmt.Printf("\nShowing %d insight periods\n", len(insights))

	return nil
}

// printInsightsMarkdown outputs insights as markdown
func (p *Printer) printInsightsMarkdown(insights []api.Insight) error {
	if len(insights) == 0 {
		fmt.Println("No insights found.")
		return nil
	}

	fmt.Println("# Location Insights")
	fmt.Println()

	for _, in := range insights {
		fmt.Printf("## %s (%s)\n\n", in.LocationID, in.Period)
		fmt.Printf("**Period:** %s to %s\n\n", in.StartDate.Format("2006-01-02"), in.EndDate.Format("2006-01-02"))

		fmt.Println("### Metrics")
		fmt.Printf("- **Views:** %s\n", formatNumber(in.Metrics.Views))
		fmt.Printf("- **Searches:** %s\n", formatNumber(in.Metrics.Searches))
		fmt.Printf("- **Calls:** %s\n", formatNumber(in.Metrics.Calls))
		fmt.Printf("- **Website Clicks:** %s\n", formatNumber(in.Metrics.WebsiteClicks))
		fmt.Printf("- **Direction Requests:** %s\n\n", formatNumber(in.Metrics.DirectionRequests))

		fmt.Println("---")
		fmt.Println()
	}

	return nil
}

// Helper functions

func formatAddress(addr api.Address) string {
	return fmt.Sprintf("%s, %s, %s %s", addr.StreetAddress, addr.Locality, addr.Region, addr.PostalCode)
}

func formatAddressFull(addr api.Address) string {
	return fmt.Sprintf("%s\n%s, %s %s\n%s", addr.StreetAddress, addr.Locality, addr.Region, addr.PostalCode, addr.Country)
}

func getStatusColor(status string) string {
	switch status {
	case "ACTIVE", "VERIFIED":
		return "\033[32m" // Green
	case "PENDING":
		return "\033[33m" // Yellow
	case "REJECTED", "DELETED":
		return "\033[31m" // Red
	default:
		return ""
	}
}

func getVerificationColor(status string) string {
	switch status {
	case "VERIFIED":
		return "\033[32m" // Green
	case "PENDING":
		return "\033[33m" // Yellow
	case "REJECTED":
		return "\033[31m" // Red
	default:
		return ""
	}
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// truncate shortens a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// ValidateFormat checks if a format is supported
func ValidateFormat(format string, allowed []string) error {
	for _, f := range allowed {
		if f == format {
			return nil
		}
	}
	return fmt.Errorf("invalid format '%s', must be one of: %v", format, allowed)
}

// ParseBool parses a boolean string
func ParseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}
