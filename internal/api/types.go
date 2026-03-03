package api

import (
	"time"
)

// Location represents a business location in Apple Business Connect
type Location struct {
	ID                 string          `json:"id"`
	CompanyID          string          `json:"companyId"`
	Status             string          `json:"status"`
	LocationName       LocalizedString `json:"locationName"`
	LocationURL        string          `json:"locationUrl,omitempty"`
	PrimaryAddress     Address         `json:"primaryAddress"`
	GeoPoint           GeoPoint        `json:"geoPoint,omitempty"`
	PhoneNumber        string          `json:"phoneNumber,omitempty"`
	Hours              BusinessHours   `json:"hours,omitempty"`
	Categories         []string        `json:"categories,omitempty"`
	CoverPhotoID       string          `json:"coverPhotoId,omitempty"`
	LogoID             string          `json:"logoId,omitempty"`
	VerificationStatus string          `json:"verificationStatus"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
}

// LocalizedString represents a string with localization
type LocalizedString struct {
	Default       string            `json:"default"`
	Localizations map[string]string `json:"localizations,omitempty"`
}

// Address represents a physical address
type Address struct {
	StreetAddress string `json:"streetAddress"`
	Locality      string `json:"locality"`
	Region        string `json:"region"`
	PostalCode    string `json:"postalCode"`
	Country       string `json:"country"`
}

// GeoPoint represents geographic coordinates
type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// BusinessHours represents operating hours
type BusinessHours struct {
	Monday    []Hours `json:"monday,omitempty"`
	Tuesday   []Hours `json:"tuesday,omitempty"`
	Wednesday []Hours `json:"wednesday,omitempty"`
	Thursday  []Hours `json:"thursday,omitempty"`
	Friday    []Hours `json:"friday,omitempty"`
	Saturday  []Hours `json:"saturday,omitempty"`
	Sunday    []Hours `json:"sunday,omitempty"`
}

// Hours represents a single time range
type Hours struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// LocationsResponse represents the response from listing locations
type LocationsResponse struct {
	Locations     []Location `json:"locations"`
	NextPageToken string     `json:"nextPageToken,omitempty"`
}

// Showcase represents a promotional showcase
type Showcase struct {
	ID          string          `json:"id"`
	LocationID  string          `json:"locationId"`
	Status      string          `json:"status"`
	Title       LocalizedString `json:"title"`
	Description LocalizedString `json:"description,omitempty"`
	Type        string          `json:"type"`
	StartDate   time.Time       `json:"startDate,omitempty"`
	EndDate     time.Time       `json:"endDate,omitempty"`
	Media       []MediaItem     `json:"media,omitempty"`
	ActionLink  *ActionLink     `json:"actionLink,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// MediaItem represents a media asset in a showcase
type MediaItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	URL     string `json:"url,omitempty"`
	AltText string `json:"altText,omitempty"`
}

// ActionLink represents a call-to-action link
type ActionLink struct {
	Title     LocalizedString `json:"title"`
	URL       string          `json:"url"`
	AppLinkID string          `json:"appLinkId,omitempty"`
}

// ShowcasesResponse represents the response from listing showcases
type ShowcasesResponse struct {
	Showcases     []Showcase `json:"showcases"`
	NextPageToken string     `json:"nextPageToken,omitempty"`
}

// Insight represents analytics data for a location
type Insight struct {
	LocationID string    `json:"locationId"`
	Period     string    `json:"period"`
	StartDate  time.Time `json:"startDate"`
	EndDate    time.Time `json:"endDate"`
	Metrics    Metrics   `json:"metrics"`
}

// Metrics represents insight metrics
type Metrics struct {
	Views             int64 `json:"views"`
	Searches          int64 `json:"searches"`
	Calls             int64 `json:"calls"`
	WebsiteClicks     int64 `json:"websiteClicks"`
	DirectionRequests int64 `json:"directionRequests"`
}

// InsightsResponse represents the response from listing insights
type InsightsResponse struct {
	Insights []Insight `json:"insights"`
}
