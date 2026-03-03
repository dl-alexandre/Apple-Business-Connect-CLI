package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	// Apple Business Connect API v3.0 base URL
	DefaultBaseURL = "https://api.businessconnect.apple.com/v3.0"
	// OAuth2 token endpoint
	TokenEndpoint = "https://businessconnect.apple.com/oauth2/v1/token"
)

// ClientOptions contains configuration for the API client
type ClientOptions struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Timeout      int
	Verbose      bool
	Debug        bool
}

// Client is the API client for Apple Business Connect
type Client struct {
	client       *resty.Client
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
	verbose      bool
	debug        bool
}

// TokenResponse represents the OAuth2 token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// NewClient creates a new Apple Business Connect API client with OAuth2 authentication
func NewClient(opts ClientOptions) (*Client, error) {
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	client := resty.New()
	client.SetBaseURL(baseURL)
	client.SetTimeout(time.Duration(opts.Timeout) * time.Second)
	client.SetHeader("Accept", "application/json")
	client.SetHeader("User-Agent", "abc/1.0.0")

	if opts.Debug {
		client.SetDebug(true)
	}

	return &Client{
		client:       client,
		clientID:     opts.ClientID,
		clientSecret: opts.ClientSecret,
		verbose:      opts.Verbose,
		debug:        opts.Debug,
	}, nil
}

// ensureToken ensures we have a valid access token
func (c *Client) ensureToken(ctx context.Context) error {
	// Check if we have a valid token (with 60 second buffer)
	if c.accessToken != "" && time.Now().Add(60*time.Second).Before(c.tokenExpiry) {
		return nil
	}

	// Request new token
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody(data.Encode()).
		Post(TokenEndpoint)

	if err != nil {
		return fmt.Errorf("failed to obtain access token: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return c.handleError(resp)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(resp.Body(), &tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Set the authorization header for future requests
	c.client.SetAuthToken(c.accessToken)

	return nil
}

// doRequest performs an authenticated request
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	if err := c.ensureToken(ctx); err != nil {
		return err
	}

	req := c.client.R().
		SetContext(ctx)

	if body != nil {
		req.SetBody(body)
	}

	if result != nil {
		req.SetResult(result)
	}

	var resp *resty.Response
	var err error

	switch strings.ToUpper(method) {
	case "GET":
		resp, err = req.Get(path)
	case "POST":
		resp, err = req.Post(path)
	case "PUT":
		resp, err = req.Put(path)
	case "PATCH":
		resp, err = req.Patch(path)
	case "DELETE":
		resp, err = req.Delete(path)
	default:
		return fmt.Errorf("unsupported HTTP method: %s", method)
	}

	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode() >= 400 {
		return c.handleError(resp)
	}

	if result != nil && resp.StatusCode() == http.StatusOK {
		if err := json.Unmarshal(resp.Body(), result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// handleError processes error responses
func (c *Client) handleError(resp *resty.Response) error {
	var apiErr APIErrorResponse
	if err := json.Unmarshal(resp.Body(), &apiErr); err == nil && apiErr.ErrorDetails.Message != "" {
		apiErr.StatusCode = resp.StatusCode()
		return &apiErr
	}

	return &APIErrorResponse{
		StatusCode: resp.StatusCode(),
		ErrorDetails: ErrorDetails{
			Message: resp.String(),
		},
	}
}

// APIErrorResponse represents an API error response
type APIErrorResponse struct {
	StatusCode   int          `json:"statusCode"`
	ErrorDetails ErrorDetails `json:"error"`
}

type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIErrorResponse) Error() string {
	if e.ErrorDetails.Code != "" {
		return fmt.Sprintf("API error %d: %s - %s", e.StatusCode, e.ErrorDetails.Code, e.ErrorDetails.Message)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.ErrorDetails.Message)
}

// NotFoundError represents a resource not found error
type NotFoundError struct {
	Resource string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("resource not found: %s", e.Resource)
}
