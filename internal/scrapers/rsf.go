package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// RSFScraper scrapes Reporters Without Borders (RSF) Press Freedom Index.
type RSFScraper struct {
	*BaseScraper
	// Threshold score for "very serious" situation (higher = worse in RSF scale)
	threshold float64
}

// NewRSFScraper creates a new RSF scraper.
func NewRSFScraper(client HTTPClient) *RSFScraper {
	return &RSFScraper{
		BaseScraper: NewBaseScraper(
			"Reporters Without Borders (RSF)",
			"https://rsf.org/en/index",
			client,
		),
		threshold: 55.0, // Countries with score > 55 are in "very serious" situation
	}
}

// Scrape fetches and parses RSF data.
func (s *RSFScraper) Scrape(ctx context.Context) (*ScrapeResult, error) {
	result := s.NewResult()

	// Try the RSF JSON API first
	apiURLs := []string{
		"https://rsf.org/api/v1/index",
		"https://rsf.org/sites/default/files/index_data.json",
	}

	var content []byte
	var err error

	for _, url := range apiURLs {
		content, err = s.Fetch(ctx, url)
		if err == nil {
			break
		}
	}

	if err != nil {
		// Fallback to HTML
		content, err = s.Fetch(ctx, s.url)
		if err != nil {
			result.Error = fmt.Sprintf("failed to fetch: %v", err)
			result.ParseStatus = "error"
			return result, nil
		}
		return s.parseHTML(content, result)
	}

	result.ContentHash = HashContent(content)

	// Try to parse as JSON
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return s.parseHTML(content, result)
	}

	return s.parseJSON(data, result)
}

// parseJSON extracts countries with poor press freedom scores.
func (s *RSFScraper) parseJSON(data interface{}, result *ScrapeResult) (*ScrapeResult, error) {
	var countries []string

	switch v := data.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				country := s.extractCountryFromMap(m)
				if country != "" {
					countries = append(countries, country)
				}
			}
		}
	case map[string]interface{}:
		// Look for countries or data array
		for _, key := range []string{"countries", "data", "rankings"} {
			if arr, ok := v[key].([]interface{}); ok {
				for _, item := range arr {
					if m, ok := item.(map[string]interface{}); ok {
						country := s.extractCountryFromMap(m)
						if country != "" {
							countries = append(countries, country)
						}
					}
				}
			}
		}
	}

	result.RawCountries = countries
	if len(countries) > 0 {
		result.ParseStatus = "success"
	} else {
		result.ParseStatus = "no_data"
	}

	return result, nil
}

// extractCountryFromMap extracts a country if it has poor press freedom.
func (s *RSFScraper) extractCountryFromMap(m map[string]interface{}) string {
	// Look for score
	score := 0.0
	for _, key := range []string{"score", "global_score", "index"} {
		if v, ok := m[key].(float64); ok {
			score = v
			break
		}
	}

	// Look for zone/category
	zone := ""
	for _, key := range []string{"zone", "category", "status", "situation"} {
		if v, ok := m[key].(string); ok {
			zone = strings.ToLower(v)
			break
		}
	}

	// Get country name
	country := ""
	for _, key := range []string{"country", "name", "country_name", "en_country"} {
		if v, ok := m[key].(string); ok {
			country = v
			break
		}
	}

	// Include if in "very serious" or "difficult" situation, or score above threshold
	badZones := []string{"very serious", "difficult", "black", "red"}
	for _, bad := range badZones {
		if strings.Contains(zone, bad) {
			return country
		}
	}

	if score >= s.threshold {
		return country
	}

	return ""
}

// parseHTML extracts countries from RSF HTML page.
func (s *RSFScraper) parseHTML(content []byte, result *ScrapeResult) (*ScrapeResult, error) {
	result.ContentHash = HashContent(content)

	html := string(content)
	var countries []string

	// Look for countries in "very serious" or "difficult" situation
	// RSF uses color coding: black/red = very serious, orange = difficult
	patterns := []string{
		`class="[^"]*(?:black|very-serious)[^"]*"[^>]*>([^<]+)<`,
		`data-situation="(?:very-serious|difficult)"[^>]*>([^<]+)<`,
		`<span[^>]*class="country-name"[^>]*>([^<]+)</span>`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, m := range matches {
			if len(m) > 1 {
				country := strings.TrimSpace(m[1])
				if country != "" && len(country) < 50 {
					countries = append(countries, country)
				}
			}
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, c := range countries {
		if !seen[c] {
			seen[c] = true
			unique = append(unique, c)
		}
	}

	result.RawCountries = unique
	if len(unique) > 0 {
		result.ParseStatus = "success"
	} else {
		result.ParseStatus = "no_data"
	}

	return result, nil
}

