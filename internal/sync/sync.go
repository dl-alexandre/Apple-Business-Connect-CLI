// Package sync provides bulk import/sync capabilities for locations.
// It supports CSV and JSON file formats and provides dry-run capabilities.
package sync

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/dl-alexandre/abc/internal/api"
	"github.com/dl-alexandre/abc/internal/validate"
	"github.com/gocarina/gocsv"
)

// FileFormat represents the supported file formats
type FileFormat string

const (
	FormatCSV  FileFormat = "csv"
	FormatJSON FileFormat = "json"
)

// LocationRecord represents a location as defined in import files
type LocationRecord struct {
	PartnerID      string `csv:"partner_id" json:"partner_id"`
	Name           string `csv:"name" json:"name"`
	Street         string `csv:"street" json:"street"`
	City           string `csv:"city" json:"city"`
	Region         string `csv:"region" json:"region"`
	PostalCode     string `csv:"postal_code" json:"postal_code"`
	Country        string `csv:"country" json:"country"`
	Phone          string `csv:"phone" json:"phone"`
	Category       string `csv:"category" json:"category"`
	Latitude       string `csv:"latitude" json:"latitude"`
	Longitude      string `csv:"longitude" json:"longitude"`
	HoursMonday    string `csv:"hours_monday" json:"hours_monday"`
	HoursTuesday   string `csv:"hours_tuesday" json:"hours_tuesday"`
	HoursWednesday string `csv:"hours_wednesday" json:"hours_wednesday"`
	HoursThursday  string `csv:"hours_thursday" json:"hours_thursday"`
	HoursFriday    string `csv:"hours_friday" json:"hours_friday"`
	HoursSaturday  string `csv:"hours_saturday" json:"hours_saturday"`
	HoursSunday    string `csv:"hours_sunday" json:"hours_sunday"`
}

// ToValidateRecord converts to a validation record
func (r LocationRecord) ToValidateRecord() validate.LocationRecord {
	return validate.LocationRecord{
		PartnerID:  r.PartnerID,
		Name:       r.Name,
		Street:     r.Street,
		City:       r.City,
		Region:     r.Region,
		PostalCode: r.PostalCode,
		Country:    r.Country,
		Phone:      r.Phone,
		Category:   r.Category,
		Latitude:   r.Latitude,
		Longitude:  r.Longitude,
	}
}

// RequiredFields are mandatory for import
type RequiredFields struct {
	Name       bool
	Street     bool
	City       bool
	Region     bool
	PostalCode bool
	Country    bool
}

// ChangeType represents the type of change detected
type ChangeType string

const (
	ChangeCreate ChangeType = "CREATE"
	ChangeUpdate ChangeType = "UPDATE"
	ChangeDelete ChangeType = "DELETE"
	ChangeNoOp   ChangeType = "NO_CHANGE"
)

// LocationChange represents a single change operation
type LocationChange struct {
	Type           ChangeType
	PartnerID      string
	LocalLocation  *LocationRecord
	RemoteLocation *api.Location
	Differences    []string
}

// SyncResult holds the complete sync operation results
type SyncResult struct {
	ToCreate int
	ToUpdate int
	ToDelete int
	NoChange int
	Changes  []LocationChange
	Errors   []error
}

// Parser handles file parsing
type Parser struct {
	format FileFormat
}

// NewParser creates a new parser for the given file
func NewParser(filename string) (*Parser, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		return &Parser{format: FormatCSV}, nil
	case ".json":
		return &Parser{format: FormatJSON}, nil
	default:
		return nil, fmt.Errorf("unsupported file format: %s (supported: .csv, .json)", ext)
	}
}

// Parse reads and parses the file into location records
func (p *Parser) Parse(filename string) ([]LocationRecord, error) {
	// Sanitize filename to prevent path traversal (gosec G304)
	cleanPath := filepath.Clean(filename)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid filename: path traversal detected")
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close file: %w", closeErr)
		}
	}()

	switch p.format {
	case FormatCSV:
		return p.parseCSV(file)
	case FormatJSON:
		return p.parseJSON(file)
	default:
		return nil, fmt.Errorf("unknown format: %s", p.format)
	}
}

// parseCSV parses CSV format
func (p *Parser) parseCSV(file *os.File) ([]LocationRecord, error) {
	// Read first line to check headers
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Validate required headers
	if err := validateHeaders(headers); err != nil {
		return nil, err
	}

	// Reset file pointer
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file: %w", err)
	}

	var records []LocationRecord
	if err := gocsv.UnmarshalFile(file, &records); err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	return records, nil
}

// parseJSON parses JSON format
func (p *Parser) parseJSON(file *os.File) ([]LocationRecord, error) {
	var records []LocationRecord
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&records); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return records, nil
}

// validateHeaders checks if required headers are present
func validateHeaders(headers []string) error {
	required := []string{"name", "street", "city", "region", "postal_code", "country"}
	headerMap := make(map[string]bool)
	for _, h := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = true
	}

	var missing []string
	for _, req := range required {
		if !headerMap[req] {
			missing = append(missing, req)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required headers: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ValidateRecords checks all records for required fields
func ValidateRecords(records []LocationRecord) []error {
	var errs []error
	for i, record := range records {
		if record.Name == "" {
			errs = append(errs, fmt.Errorf("record %d: name is required", i+1))
		}
		if record.Street == "" {
			errs = append(errs, fmt.Errorf("record %d: street is required", i+1))
		}
		if record.City == "" {
			errs = append(errs, fmt.Errorf("record %d: city is required", i+1))
		}
		if record.Region == "" {
			errs = append(errs, fmt.Errorf("record %d: region is required", i+1))
		}
		if record.PostalCode == "" {
			errs = append(errs, fmt.Errorf("record %d: postal_code is required", i+1))
		}
		if record.Country == "" {
			errs = append(errs, fmt.Errorf("record %d: country is required", i+1))
		}
	}
	return errs
}

// ToAPILocation converts a LocationRecord to api.Location
func (r *LocationRecord) ToAPILocation() *api.Location {
	loc := &api.Location{
		LocationName: api.LocalizedString{Default: r.Name},
		PrimaryAddress: api.Address{
			StreetAddress: r.Street,
			Locality:      r.City,
			Region:        r.Region,
			PostalCode:    r.PostalCode,
			Country:       r.Country,
		},
	}

	if r.Phone != "" {
		loc.PhoneNumber = r.Phone
	}

	if r.Category != "" {
		loc.Categories = []string{r.Category}
	}

	return loc
}

// DiffEngine compares local records with remote locations
type DiffEngine struct {
	localRecords    []LocationRecord
	remoteLocations []api.Location
}

// NewDiffEngine creates a new diff engine
func NewDiffEngine(local []LocationRecord, remote []api.Location) *DiffEngine {
	return &DiffEngine{
		localRecords:    local,
		remoteLocations: remote,
	}
}

// Compare performs the diff operation
func (d *DiffEngine) Compare() *SyncResult {
	result := &SyncResult{
		Changes: make([]LocationChange, 0),
	}

	// Build map of remote locations by partner_id
	remoteMap := make(map[string]api.Location)
	for _, loc := range d.remoteLocations {
		// Use a unique identifier - either partner_id from metadata or location ID
		key := getLocationKey(&loc)
		remoteMap[key] = loc
	}

	// Build map of local records by partner_id
	localMap := make(map[string]LocationRecord)
	for _, rec := range d.localRecords {
		if rec.PartnerID != "" {
			localMap[rec.PartnerID] = rec
		}
	}

	// Check for changes
	for _, local := range d.localRecords {
		key := local.PartnerID
		if key == "" {
			key = generateKeyFromAddress(&local)
		}

		if remote, exists := remoteMap[key]; exists {
			// Location exists - check for changes
			diffs := compareLocations(&local, &remote)
			if len(diffs) > 0 {
				result.ToUpdate++
				result.Changes = append(result.Changes, LocationChange{
					Type:           ChangeUpdate,
					PartnerID:      key,
					LocalLocation:  &local,
					RemoteLocation: &remote,
					Differences:    diffs,
				})
			} else {
				result.NoChange++
				result.Changes = append(result.Changes, LocationChange{
					Type:           ChangeNoOp,
					PartnerID:      key,
					LocalLocation:  &local,
					RemoteLocation: &remote,
				})
			}
		} else {
			// New location
			result.ToCreate++
			result.Changes = append(result.Changes, LocationChange{
				Type:          ChangeCreate,
				PartnerID:     key,
				LocalLocation: &local,
			})
		}
	}

	return result
}

// getLocationKey extracts a unique key from an API location
func getLocationKey(loc *api.Location) string {
	// If there's metadata or external ID, use that
	// Otherwise, generate from address
	return fmt.Sprintf("%s-%s-%s-%s",
		loc.PrimaryAddress.StreetAddress,
		loc.PrimaryAddress.Locality,
		loc.PrimaryAddress.Region,
		loc.PrimaryAddress.PostalCode,
	)
}

// generateKeyFromAddress generates a key from address fields
func generateKeyFromAddress(rec *LocationRecord) string {
	return fmt.Sprintf("%s-%s-%s-%s",
		rec.Street,
		rec.City,
		rec.Region,
		rec.PostalCode,
	)
}

// compareLocations compares local and remote locations and returns differences
func compareLocations(local *LocationRecord, remote *api.Location) []string {
	var diffs []string

	if local.Name != remote.LocationName.Default {
		diffs = append(diffs, fmt.Sprintf("name: '%s' -> '%s'", remote.LocationName.Default, local.Name))
	}

	if local.Phone != remote.PhoneNumber {
		diffs = append(diffs, fmt.Sprintf("phone: '%s' -> '%s'", remote.PhoneNumber, local.Phone))
	}

	// Compare address fields
	if local.Street != remote.PrimaryAddress.StreetAddress {
		diffs = append(diffs, fmt.Sprintf("street: '%s' -> '%s'", remote.PrimaryAddress.StreetAddress, local.Street))
	}
	if local.City != remote.PrimaryAddress.Locality {
		diffs = append(diffs, fmt.Sprintf("city: '%s' -> '%s'", remote.PrimaryAddress.Locality, local.City))
	}
	if local.Region != remote.PrimaryAddress.Region {
		diffs = append(diffs, fmt.Sprintf("region: '%s' -> '%s'", remote.PrimaryAddress.Region, local.Region))
	}
	if local.PostalCode != remote.PrimaryAddress.PostalCode {
		diffs = append(diffs, fmt.Sprintf("postal_code: '%s' -> '%s'", remote.PrimaryAddress.PostalCode, local.PostalCode))
	}
	if local.Country != remote.PrimaryAddress.Country {
		diffs = append(diffs, fmt.Sprintf("country: '%s' -> '%s'", remote.PrimaryAddress.Country, local.Country))
	}

	return diffs
}

// HasErrors returns true if there are validation errors
func (r *SyncResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// Summary returns a human-readable summary of the sync operation
func (r *SyncResult) Summary() string {
	return fmt.Sprintf("Summary: %d to create, %d to update, %d unchanged, %d errors",
		r.ToCreate, r.ToUpdate, r.NoChange, len(r.Errors))
}

// PrintChanges outputs the changes in a formatted way
func (r *SyncResult) PrintChanges() {
	if r.ToCreate > 0 {
		fmt.Printf("\n[NEW] %d location(s) to be created:\n", r.ToCreate)
		for _, change := range r.Changes {
			if change.Type == ChangeCreate {
				fmt.Printf("  + %s (%s, %s)\n",
					change.LocalLocation.Name,
					change.LocalLocation.City,
					change.LocalLocation.Region)
			}
		}
	}

	if r.ToUpdate > 0 {
		fmt.Printf("\n[UPDATE] %d location(s) to be updated:\n", r.ToUpdate)
		for _, change := range r.Changes {
			if change.Type == ChangeUpdate {
				fmt.Printf("  ~ %s:\n", change.LocalLocation.Name)
				for _, diff := range change.Differences {
					fmt.Printf("    - %s\n", diff)
				}
			}
		}
	}

	if r.NoChange > 0 {
		fmt.Printf("\n[UNCHANGED] %d location(s) unchanged\n", r.NoChange)
	}

	if len(r.Errors) > 0 {
		fmt.Printf("\n[ERRORS] %d error(s):\n", len(r.Errors))
		for _, err := range r.Errors {
			fmt.Printf("  ! %v\n", err)
		}
	}

	fmt.Printf("\n%s\n", r.Summary())
}

// IsEmpty checks if two locations are functionally identical
func IsEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case []string:
		return len(val) == 0
	case api.Address:
		return reflect.DeepEqual(val, api.Address{})
	default:
		return false
	}
}

// WorkerPool manages concurrent API operations with rate limiting
type WorkerPool struct {
	workers   int
	semaphore chan struct{}
	results   chan WorkerResult
	wg        sync.WaitGroup
}

// WorkerResult represents the result of a single worker operation
type WorkerResult struct {
	Change LocationChange
	Error  error
}

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = 5 // Default: 5 concurrent workers
	}
	return &WorkerPool{
		workers:   workers,
		semaphore: make(chan struct{}, workers),
		results:   make(chan WorkerResult, workers),
	}
}

// SyncOptions contains options for sync operations
type SyncOptions struct {
	Workers     int
	RateLimitMs int // Milliseconds between requests (0 = no delay)
}

// DefaultSyncOptions returns default sync options
func DefaultSyncOptions() SyncOptions {
	return SyncOptions{
		Workers:     5,   // 5 concurrent workers
		RateLimitMs: 100, // 100ms between requests (10 req/sec max)
	}
}

// ExecuteSync performs the sync with rate limiting and concurrency control
// This is called from the CLI and handles the actual API operations
func ExecuteSync(ctx context.Context, client *api.Client, changes []LocationChange, opts SyncOptions, dryRun bool) (*SyncExecutionResult, error) {
	if dryRun {
		return &SyncExecutionResult{
			Applied: 0,
			Skipped: len(changes),
			Failed:  0,
			DryRun:  true,
		}, nil
	}

	pool := NewWorkerPool(opts.Workers)
	result := &SyncExecutionResult{
		Results: make([]WorkerResult, 0, len(changes)),
	}

	// Submit all work
	pool.wg.Add(len(changes))
	for _, change := range changes {
		go pool.executeChange(ctx, client, change, opts.RateLimitMs)
	}

	// Collect results
	go func() {
		pool.wg.Wait()
		close(pool.results)
	}()

	for res := range pool.results {
		result.Results = append(result.Results, res)
		if res.Error != nil {
			result.Failed++
		} else {
			result.Applied++
		}
	}

	return result, nil
}

func (p *WorkerPool) executeChange(ctx context.Context, client *api.Client, change LocationChange, rateLimitMs int) {
	defer p.wg.Done()

	// Acquire semaphore (rate limiting)
	p.semaphore <- struct{}{}
	defer func() { <-p.semaphore }()

	// Apply rate limit delay
	if rateLimitMs > 0 {
		time.Sleep(time.Duration(rateLimitMs) * time.Millisecond)
	}

	var err error
	switch change.Type {
	case ChangeCreate:
		location := change.LocalLocation.ToAPILocation()
		_, err = client.CreateLocation(ctx, location)
	case ChangeUpdate:
		location := change.LocalLocation.ToAPILocation()
		_, err = client.UpdateLocation(ctx, change.RemoteLocation.ID, location)
	case ChangeDelete:
		err = client.DeleteLocation(ctx, change.RemoteLocation.ID)
	case ChangeNoOp:
		// No operation needed
	}

	p.results <- WorkerResult{
		Change: change,
		Error:  err,
	}
}

// SyncExecutionResult holds the results of executing a sync
type SyncExecutionResult struct {
	Applied int
	Skipped int
	Failed  int
	DryRun  bool
	Results []WorkerResult
}

// Summary returns a human-readable summary
func (r *SyncExecutionResult) Summary() string {
	if r.DryRun {
		return fmt.Sprintf("Dry-run: %d changes would be applied (no modifications made)", r.Skipped)
	}
	return fmt.Sprintf("Sync complete: %d applied, %d failed", r.Applied, r.Failed)
}
