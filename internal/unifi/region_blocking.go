package unifi

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GetRegionBlockingSettings fetches the current USG setting containing region blocking configuration.
// Returns the full setting as a map to preserve all fields when updating.
func (c *Client) GetRegionBlockingSettings() (map[string]interface{}, error) {
	// Try to get the usg setting - it's usually an array with one element
	path := fmt.Sprintf("api/s/%s/rest/setting/usg", c.site)
	body, status, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get usg settings: %w", err)
	}

	if status != 200 {
		return nil, fmt.Errorf("unexpected status %d when getting usg settings", status)
	}

	// Parse response - could be array or single object
	var settings []map[string]interface{}
	var singleSetting map[string]interface{}

	// Try array first
	if err := json.Unmarshal(body, &settings); err == nil && len(settings) > 0 {
		// Use the first setting (usually there's only one)
		return settings[0], nil
	}

	// Try single object
	if err := json.Unmarshal(body, &singleSetting); err == nil {
		if id, ok := singleSetting["_id"].(string); ok && id != "" {
			return singleSetting, nil
		}
	}

	// Try wrapped format { "data": [...] }
	var wrapper struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil && len(wrapper.Data) > 0 {
		return wrapper.Data[0], nil
	}

	return nil, fmt.Errorf("could not parse usg settings response")
}

// UpdateRegionBlockingSettings updates the region blocking configuration.
// This requires sending the complete USG setting object, so we need to GET it first,
// modify the geo-ip fields, then POST it back.
func (c *Client) UpdateRegionBlockingSettings(
	enabled bool,
	countryCodes []string, // ISO 3166-1 alpha-2 codes
	block string,          // Usually "block"
	trafficDirection string, // "both", "inbound", or "outbound"
) error {
	// First, get the current setting (as a map to preserve all fields)
	current, err := c.GetRegionBlockingSettings()
	if err != nil {
		return fmt.Errorf("failed to get current settings: %w", err)
	}

	// Update the geo-ip filtering fields
	current["geo_ip_filtering_enabled"] = enabled
	current["geo_ip_filtering_countries"] = strings.Join(countryCodes, ",")
	if block != "" {
		current["geo_ip_filtering_block"] = block
	} else {
		current["geo_ip_filtering_block"] = "block" // Default
	}
	if trafficDirection != "" {
		current["geo_ip_filtering_traffic_direction"] = trafficDirection
	} else {
		current["geo_ip_filtering_traffic_direction"] = "both" // Default
	}

	// Ensure required fields exist
	if current["key"] == nil {
		current["key"] = "usg"
	}

	// Post the updated setting
	path := fmt.Sprintf("api/s/%s/set/setting/usg", c.site)
	body, status, err := c.Post(path, current)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	if status != 200 {
		return fmt.Errorf("unexpected status %d when updating settings: %s", status, string(body))
	}

	return nil
}

// GetBlockedCountries returns the current list of blocked country codes.
func (c *Client) GetBlockedCountries() ([]string, error) {
	setting, err := c.GetRegionBlockingSettings()
	if err != nil {
		return nil, err
	}

	enabled, _ := setting["geo_ip_filtering_enabled"].(bool)
	countriesStr, _ := setting["geo_ip_filtering_countries"].(string)

	if !enabled || countriesStr == "" {
		return []string{}, nil
	}

	// Parse comma-separated string
	codes := strings.Split(countriesStr, ",")
	var result []string
	for _, code := range codes {
		code = strings.TrimSpace(strings.ToUpper(code))
		if code != "" {
			result = append(result, code)
		}
	}

	return result, nil
}
