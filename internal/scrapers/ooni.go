package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// OONIScraper scrapes OONI (Open Observatory of Network Interference) data.
type OONIScraper struct {
	*BaseScraper
	// Minimum confirmed blocks to include a country
	minBlocks int
}

// NewOONIScraper creates a new OONI scraper.
func NewOONIScraper(client HTTPClient) *OONIScraper {
	return &OONIScraper{
		BaseScraper: NewBaseScraper(
			"OONI (Open Observatory of Network Interference)",
			"https://ooni.org/countries/",
			client,
		),
		minBlocks: 100, // Minimum confirmed blocks to include
	}
}

// Scrape fetches and parses OONI data.
func (s *OONIScraper) Scrape(ctx context.Context) (*ScrapeResult, error) {
	result := s.NewResult()

	// OONI has an API for country-level stats
	apiURLs := []string{
		"https://api.ooni.io/api/v1/aggregation?probe_cc=*&since=2023-01-01",
		"https://api.ooni.io/api/v1/countries",
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
		// Fallback to scraping the countries page
		content, err = s.Fetch(ctx, s.url)
		if err != nil {
			result.Error = fmt.Sprintf("failed to fetch: %v", err)
			result.ParseStatus = "error"
			return result, nil
		}
		return s.parseHTML(content, result)
	}

	result.ContentHash = HashContent(content)

	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return s.parseHTML(content, result)
	}

	return s.parseJSON(data, result)
}

// parseJSON extracts countries with significant censorship from OONI data.
func (s *OONIScraper) parseJSON(data interface{}, result *ScrapeResult) (*ScrapeResult, error) {
	var countries []string

	switch v := data.(type) {
	case map[string]interface{}:
		// Look for countries/results array
		for _, key := range []string{"countries", "results", "data"} {
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
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				country := s.extractCountryFromMap(m)
				if country != "" {
					countries = append(countries, country)
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

// extractCountryFromMap extracts a country if it shows significant censorship.
func (s *OONIScraper) extractCountryFromMap(m map[string]interface{}) string {
	// Look for confirmed/anomaly counts
	confirmed := 0
	anomaly := 0

	if v, ok := m["confirmed_count"].(float64); ok {
		confirmed = int(v)
	}
	if v, ok := m["anomaly_count"].(float64); ok {
		anomaly = int(v)
	}

	// Get country code or name
	country := ""
	for _, key := range []string{"probe_cc", "country_code", "alpha_2", "country"} {
		if v, ok := m[key].(string); ok {
			country = v
			break
		}
	}

	// Include if significant blocking detected
	if confirmed >= s.minBlocks || anomaly >= s.minBlocks*2 {
		return country
	}

	return ""
}

// parseHTML extracts country codes from OONI countries page.
func (s *OONIScraper) parseHTML(content []byte, result *ScrapeResult) (*ScrapeResult, error) {
	result.ContentHash = HashContent(content)

	html := string(content)
	var countries []string

	// Look for country links in the page
	// Pattern: /country/XX where XX is the country code
	patterns := []string{
		`/country/([A-Z]{2})`,
		`data-country="([A-Z]{2})"`,
		`probe_cc=([A-Z]{2})`,
	}

	seen := make(map[string]bool)

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, m := range matches {
			if len(m) > 1 {
				code := strings.ToUpper(m[1])
				if !seen[code] {
					seen[code] = true
					countries = append(countries, code)
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

