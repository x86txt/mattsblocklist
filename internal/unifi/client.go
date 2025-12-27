// Package unifi provides a client for interacting with UniFi Network controllers.
package unifi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Client represents a UniFi API client.
type Client struct {
	baseURL       string
	site          string
	httpClient    *http.Client
	csrfToken     string
	authenticated bool
	verbose       bool
}

// ClientConfig holds configuration for creating a new client.
type ClientConfig struct {
	Host          string
	Username      string
	Password      string
	Site          string
	SkipTLSVerify bool
	Verbose       bool
	Timeout       time.Duration
}

// NewClient creates a new UniFi API client.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if cfg.Site == "" {
		cfg.Site = "default"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// Create cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Create HTTP client with TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.SkipTLSVerify,
		},
	}

	httpClient := &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	baseURL := strings.TrimSuffix(cfg.Host, "/")

	client := &Client{
		baseURL:    baseURL,
		site:       cfg.Site,
		httpClient: httpClient,
		verbose:    cfg.Verbose,
	}

	// Authenticate
	if err := client.login(cfg.Username, cfg.Password); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return client, nil
}

// login authenticates with the UniFi controller.
func (c *Client) login(username, password string) error {
	loginURL := c.baseURL + "/api/auth/login"

	payload := map[string]interface{}{
		"username": username,
		"password": password,
		"remember": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal login payload: %w", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Extract CSRF token from response header
	if token := resp.Header.Get("X-Csrf-Token"); token != "" {
		c.csrfToken = token
	}

	c.authenticated = true
	return nil
}

// Logout ends the current session.
func (c *Client) Logout() error {
	if !c.authenticated {
		return nil
	}

	logoutURL := c.baseURL + "/api/auth/logout"
	req, err := http.NewRequest("POST", logoutURL, nil)
	if err != nil {
		return err
	}

	c.addHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.authenticated = false
	return nil
}

// addHeaders adds required headers to a request.
func (c *Client) addHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.csrfToken != "" {
		req.Header.Set("X-Csrf-Token", c.csrfToken)
	}
}

// Get performs a GET request to the specified path.
// The path should not include the /proxy/network prefix - it will be added automatically.
func (c *Client) Get(path string) ([]byte, int, error) {
	return c.request("GET", path, nil)
}

// Post performs a POST request to the specified path.
func (c *Client) Post(path string, body interface{}) ([]byte, int, error) {
	return c.request("POST", path, body)
}

// Put performs a PUT request to the specified path.
func (c *Client) Put(path string, body interface{}) ([]byte, int, error) {
	return c.request("PUT", path, body)
}

// Delete performs a DELETE request to the specified path.
func (c *Client) Delete(path string) ([]byte, int, error) {
	return c.request("DELETE", path, nil)
}

// request performs an HTTP request to the UniFi API.
func (c *Client) request(method, path string, body interface{}) ([]byte, int, error) {
	if !c.authenticated {
		return nil, 0, fmt.Errorf("not authenticated")
	}

	// Build full URL with proxy prefix
	fullURL := c.buildURL(path)

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	c.addHeaders(req)

	if c.verbose {
		fmt.Printf("[DEBUG] %s %s\n", method, fullURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Update CSRF token if present in response
	if token := resp.Header.Get("X-Csrf-Token"); token != "" {
		c.csrfToken = token
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// buildURL constructs the full URL for an API path.
func (c *Client) buildURL(path string) string {
	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")

	// If path already has proxy/network prefix, use it as-is
	if strings.HasPrefix(path, "proxy/network/") {
		return c.baseURL + "/" + path
	}

	// For v2 API paths, they go directly under /proxy/network/
	if strings.HasPrefix(path, "v2/") {
		return c.baseURL + "/proxy/network/" + path
	}

	// For api/s/{site}/... paths, add proxy/network prefix
	if strings.HasPrefix(path, "api/s/") {
		return c.baseURL + "/proxy/network/" + path
	}

	// For api/... paths (without site), add proxy/network prefix
	if strings.HasPrefix(path, "api/") {
		return c.baseURL + "/proxy/network/" + path
	}

	// Default: assume it's a site-scoped path
	return c.baseURL + "/proxy/network/api/s/" + c.site + "/" + path
}

// GetSitePath returns the API path prefix for the current site.
func (c *Client) GetSitePath() string {
	return fmt.Sprintf("api/s/%s", c.site)
}

// BaseURL returns the base URL of the controller.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// IsAuthenticated returns whether the client is authenticated.
func (c *Client) IsAuthenticated() bool {
	return c.authenticated
}

// TestEndpoint tests if an endpoint exists and returns useful information.
func (c *Client) TestEndpoint(path string) (*EndpointResult, error) {
	startTime := time.Now()
	body, statusCode, err := c.Get(path)
	duration := time.Since(startTime)

	result := &EndpointResult{
		Path:       path,
		FullURL:    c.buildURL(path),
		StatusCode: statusCode,
		Duration:   duration,
	}

	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	result.Exists = statusCode == http.StatusOK
	result.ResponseSize = len(body)

	// Try to parse as JSON to check validity
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err == nil {
		result.IsJSON = true
		result.ResponseSample = truncateJSON(body, 500)
	}

	return result, nil
}

// EndpointResult contains information about an endpoint test.
type EndpointResult struct {
	Path           string        `json:"path"`
	FullURL        string        `json:"full_url"`
	Exists         bool          `json:"exists"`
	StatusCode     int           `json:"status_code"`
	ResponseSize   int           `json:"response_size"`
	IsJSON         bool          `json:"is_json"`
	ResponseSample string        `json:"response_sample,omitempty"`
	Duration       time.Duration `json:"duration"`
	Error          string        `json:"error,omitempty"`
}

// truncateJSON truncates a JSON response for display.
func truncateJSON(data []byte, maxLen int) string {
	if len(data) <= maxLen {
		return string(data)
	}
	return string(data[:maxLen]) + "..."
}

// RawRequest performs a raw HTTP request without the proxy/network prefix.
func (c *Client) RawRequest(method, fullPath string, body interface{}) ([]byte, int, error) {
	if !c.authenticated {
		return nil, 0, fmt.Errorf("not authenticated")
	}

	fullURL := c.baseURL + "/" + strings.TrimPrefix(fullPath, "/")

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	c.addHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// ParseURL parses a URL string.
func ParseURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}

