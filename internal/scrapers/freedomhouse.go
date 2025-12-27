package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// FreedomHouseScraper scrapes Freedom House's Freedom on the Net report.
type FreedomHouseScraper struct {
	*BaseScraper
	// Threshold below which a country is considered "Not Free" (0-100 scale)
	threshold int
}

// NewFreedomHouseScraper creates a new Freedom House scraper.
func NewFreedomHouseScraper(client HTTPClient) *FreedomHouseScraper {
	return &FreedomHouseScraper{
		BaseScraper: NewBaseScraper(
			"Freedom House",
			"https://freedomhouse.org/countries/freedom-net/scores",
			client,
		),
		threshold: 40, // Countries with score < 40 are "Not Free"
	}
}

// Scrape fetches and parses Freedom House data.
func (s *FreedomHouseScraper) Scrape(ctx context.Context) (*ScrapeResult, error) {
	result := s.NewResult()

	// Freedom House has a JSON API endpoint for their data
	apiURL := "https://freedomhouse.org/api/fotn-scores"
	content, err := s.Fetch(ctx, apiURL)
	if err != nil {
		// Fallback: try to scrape the HTML page
		content, err = s.Fetch(ctx, s.url)
		if err != nil {
			result.Error = fmt.Sprintf("failed to fetch: %v", err)
			result.ParseStatus = "error"
			return result, nil
		}
		// Parse HTML fallback
		return s.parseHTML(content, result)
	}

	result.ContentHash = HashContent(content)

	// Try to parse as JSON
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		// Try HTML fallback
		return s.parseHTML(content, result)
	}

	return s.parseJSON(data, result)
}

// parseJSON extracts countries from Freedom House JSON data.
func (s *FreedomHouseScraper) parseJSON(data interface{}, result *ScrapeResult) (*ScrapeResult, error) {
	// Freedom House JSON structure varies, try to extract countries with low scores
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
		// Look for a data array
		if dataArr, ok := v["data"].([]interface{}); ok {
			for _, item := range dataArr {
				if m, ok := item.(map[string]interface{}); ok {
					country := s.extractCountryFromMap(m)
					if country != "" {
						countries = append(countries, country)
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

// extractCountryFromMap extracts a country name if it meets the threshold.
func (s *FreedomHouseScraper) extractCountryFromMap(m map[string]interface{}) string {
	// Look for score field
	score := 0.0
	if v, ok := m["score"].(float64); ok {
		score = v
	} else if v, ok := m["total"].(float64); ok {
		score = v
	}

	// Look for status field
	status := ""
	if v, ok := m["status"].(string); ok {
		status = strings.ToLower(v)
	}

	// Get country name
	country := ""
	if v, ok := m["country"].(string); ok {
		country = v
	} else if v, ok := m["name"].(string); ok {
		country = v
	}

	// Include if "Not Free" or score below threshold
	if status == "not free" || status == "nf" || score < float64(s.threshold) {
		return country
	}

	return ""
}

// parseHTML extracts countries from Freedom House HTML page.
func (s *FreedomHouseScraper) parseHTML(content []byte, result *ScrapeResult) (*ScrapeResult, error) {
	result.ContentHash = HashContent(content)

	html := string(content)

	// Look for countries marked as "Not Free"
	// Pattern: country name followed by "Not Free" status
	var countries []string

	// Try multiple patterns
	patterns := []string{
		`<td[^>]*>([^<]+)</td>\s*<td[^>]*>Not Free</td>`,
		`"country":\s*"([^"]+)"[^}]*"status":\s*"Not Free"`,
		`data-status="not-free"[^>]*>([^<]+)<`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, m := range matches {
			if len(m) > 1 {
				country := strings.TrimSpace(m[1])
				if country != "" {
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
		result.Error = "could not parse country data from HTML"
	}

	return result, nil
}

// SetThreshold sets the score threshold for "Not Free" classification.
func (s *FreedomHouseScraper) SetThreshold(threshold int) {
	s.threshold = threshold
}

