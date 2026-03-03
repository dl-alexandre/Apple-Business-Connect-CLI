package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ListLocations retrieves a list of locations for the company
func (c *Client) ListLocations(ctx context.Context, companyID string, limit int, pageToken string) (*LocationsResponse, error) {
	path := "/locations"
	params := url.Values{}
	if companyID != "" {
		params.Set("companyId", companyID)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}
	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}

	var result LocationsResponse
	if err := c.doRequest(ctx, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to list locations: %w", err)
	}

	return &result, nil
}

// GetLocation retrieves a single location by ID
func (c *Client) GetLocation(ctx context.Context, locationID string) (*Location, error) {
	path := fmt.Sprintf("/locations/%s", url.PathEscape(locationID))

	var result Location
	if err := c.doRequest(ctx, "GET", path, nil, &result); err != nil {
		if apiErr, ok := err.(*APIErrorResponse); ok && apiErr.StatusCode == http.StatusNotFound {
			return nil, &NotFoundError{Resource: locationID}
		}
		return nil, fmt.Errorf("failed to get location: %w", err)
	}

	return &result, nil
}

// CreateLocation creates a new location
func (c *Client) CreateLocation(ctx context.Context, location *Location) (*Location, error) {
	path := "/locations"

	var result Location
	if err := c.doRequest(ctx, "POST", path, location, &result); err != nil {
		return nil, fmt.Errorf("failed to create location: %w", err)
	}

	return &result, nil
}

// UpdateLocation updates an existing location
func (c *Client) UpdateLocation(ctx context.Context, locationID string, location *Location) (*Location, error) {
	path := fmt.Sprintf("/locations/%s", url.PathEscape(locationID))

	var result Location
	if err := c.doRequest(ctx, "PATCH", path, location, &result); err != nil {
		return nil, fmt.Errorf("failed to update location: %w", err)
	}

	return &result, nil
}

// DeleteLocation deletes a location
func (c *Client) DeleteLocation(ctx context.Context, locationID string) error {
	path := fmt.Sprintf("/locations/%s", url.PathEscape(locationID))

	if err := c.doRequest(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete location: %w", err)
	}

	return nil
}

// ListShowcases retrieves showcases for a location
func (c *Client) ListShowcases(ctx context.Context, locationID string, limit int, pageToken string) (*ShowcasesResponse, error) {
	path := fmt.Sprintf("/locations/%s/showcases", url.PathEscape(locationID))
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}
	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}

	var result ShowcasesResponse
	if err := c.doRequest(ctx, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to list showcases: %w", err)
	}

	return &result, nil
}

// GetShowcase retrieves a single showcase by ID
func (c *Client) GetShowcase(ctx context.Context, locationID, showcaseID string) (*Showcase, error) {
	path := fmt.Sprintf("/locations/%s/showcases/%s", url.PathEscape(locationID), url.PathEscape(showcaseID))

	var result Showcase
	if err := c.doRequest(ctx, "GET", path, nil, &result); err != nil {
		if apiErr, ok := err.(*APIErrorResponse); ok && apiErr.StatusCode == http.StatusNotFound {
			return nil, &NotFoundError{Resource: showcaseID}
		}
		return nil, fmt.Errorf("failed to get showcase: %w", err)
	}

	return &result, nil
}

// CreateShowcase creates a new showcase for a location
func (c *Client) CreateShowcase(ctx context.Context, locationID string, showcase *Showcase) (*Showcase, error) {
	path := fmt.Sprintf("/locations/%s/showcases", url.PathEscape(locationID))

	var result Showcase
	if err := c.doRequest(ctx, "POST", path, showcase, &result); err != nil {
		return nil, fmt.Errorf("failed to create showcase: %w", err)
	}

	return &result, nil
}

// UpdateShowcase updates an existing showcase
func (c *Client) UpdateShowcase(ctx context.Context, locationID, showcaseID string, showcase *Showcase) (*Showcase, error) {
	path := fmt.Sprintf("/locations/%s/showcases/%s", url.PathEscape(locationID), url.PathEscape(showcaseID))

	var result Showcase
	if err := c.doRequest(ctx, "PATCH", path, showcase, &result); err != nil {
		return nil, fmt.Errorf("failed to update showcase: %w", err)
	}

	return &result, nil
}

// DeleteShowcase deletes a showcase
func (c *Client) DeleteShowcase(ctx context.Context, locationID, showcaseID string) error {
	path := fmt.Sprintf("/locations/%s/showcases/%s", url.PathEscape(locationID), url.PathEscape(showcaseID))

	if err := c.doRequest(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete showcase: %w", err)
	}

	return nil
}

// GetInsights retrieves insights for a location
func (c *Client) GetInsights(ctx context.Context, locationID, period string, startDate, endDate string) (*InsightsResponse, error) {
	path := fmt.Sprintf("/locations/%s/insights", url.PathEscape(locationID))
	params := url.Values{}
	if period != "" {
		params.Set("period", period)
	}
	if startDate != "" {
		params.Set("startDate", startDate)
	}
	if endDate != "" {
		params.Set("endDate", endDate)
	}
	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}

	var result InsightsResponse
	if err := c.doRequest(ctx, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get insights: %w", err)
	}

	return &result, nil
}
