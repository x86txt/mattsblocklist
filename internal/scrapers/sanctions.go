package scrapers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// EUSanctionsScraper scrapes the EU sanctions map.
type EUSanctionsScraper struct {
	*BaseScraper
}

// NewEUSanctionsScraper creates a new EU sanctions scraper.
func NewEUSanctionsScraper(client HTTPClient) *EUSanctionsScraper {
	return &EUSanctionsScraper{
		BaseScraper: NewBaseScraper(
			"EU Sanctions List",
			"https://www.sanctionsmap.eu/api/v1/sanctions",
			client,
		),
	}
}

// Scrape fetches EU sanctions data.
func (s *EUSanctionsScraper) Scrape(ctx context.Context) (*ScrapeResult, error) {
	result := s.NewResult()

	// Try the API first
	content, err := s.Fetch(ctx, s.url)
	if err != nil {
		// Fallback to known sanctioned countries
		result.RawCountries = euSanctionedCountries
		result.ParseStatus = "fallback"
		return result, nil
	}

	result.ContentHash = HashContent(content)

	// Parse the response for country names
	countries := extractCountriesFromText(string(content))

	if len(countries) > 0 {
		result.RawCountries = countries
		result.ParseStatus = "success"
	} else {
		result.RawCountries = euSanctionedCountries
		result.ParseStatus = "fallback"
	}

	return result, nil
}

// US OFAC Sanctions
type USOFACScraper struct {
	*BaseScraper
}

func NewUSOFACScraper(client HTTPClient) *USOFACScraper {
	return &USOFACScraper{
		BaseScraper: NewBaseScraper(
			"US OFAC Sanctions List",
			"https://home.treasury.gov/policy-issues/financial-sanctions/sanctions-programs-and-country-information",
			client,
		),
	}
}

func (s *USOFACScraper) Scrape(ctx context.Context) (*ScrapeResult, error) {
	result := s.NewResult()

	content, err := s.Fetch(ctx, s.url)
	if err != nil {
		result.RawCountries = usOFACSanctionedCountries
		result.ParseStatus = "fallback"
		return result, nil
	}

	result.ContentHash = HashContent(content)

	// Parse for country names from the page
	countries := extractCountriesFromText(string(content))

	if len(countries) > 0 {
		result.RawCountries = countries
		result.ParseStatus = "success"
	} else {
		result.RawCountries = usOFACSanctionedCountries
		result.ParseStatus = "fallback"
	}

	return result, nil
}

// UK Sanctions
type UKSanctionsScraper struct {
	*BaseScraper
}

func NewUKSanctionsScraper(client HTTPClient) *UKSanctionsScraper {
	return &UKSanctionsScraper{
		BaseScraper: NewBaseScraper(
			"UK Sanctions List",
			"https://www.gov.uk/government/collections/financial-sanctions-regime-specific-consolidated-lists-and-releases",
			client,
		),
	}
}

func (s *UKSanctionsScraper) Scrape(ctx context.Context) (*ScrapeResult, error) {
	result := s.NewResult()

	content, err := s.Fetch(ctx, s.url)
	if err != nil {
		result.RawCountries = ukSanctionedCountries
		result.ParseStatus = "fallback"
		return result, nil
	}

	result.ContentHash = HashContent(content)

	countries := extractCountriesFromText(string(content))

	if len(countries) > 0 {
		result.RawCountries = countries
		result.ParseStatus = "success"
	} else {
		result.RawCountries = ukSanctionedCountries
		result.ParseStatus = "fallback"
	}

	return result, nil
}

// UN Sanctions
type UNSanctionsScraper struct {
	*BaseScraper
}

func NewUNSanctionsScraper(client HTTPClient) *UNSanctionsScraper {
	return &UNSanctionsScraper{
		BaseScraper: NewBaseScraper(
			"UN Sanctions List",
			"https://www.un.org/securitycouncil/sanctions/information",
			client,
		),
	}
}

func (s *UNSanctionsScraper) Scrape(ctx context.Context) (*ScrapeResult, error) {
	result := s.NewResult()

	content, err := s.Fetch(ctx, s.url)
	if err != nil {
		result.RawCountries = unSanctionedCountries
		result.ParseStatus = "fallback"
		return result, nil
	}

	result.ContentHash = HashContent(content)

	countries := extractCountriesFromText(string(content))

	if len(countries) > 0 {
		result.RawCountries = countries
		result.ParseStatus = "success"
	} else {
		result.RawCountries = unSanctionedCountries
		result.ParseStatus = "fallback"
	}

	return result, nil
}

// FATF Grey List
type FATFScraper struct {
	*BaseScraper
}

func NewFATFScraper(client HTTPClient) *FATFScraper {
	return &FATFScraper{
		BaseScraper: NewBaseScraper(
			"FATF Grey List",
			"https://www.fatf-gafi.org/en/countries/black-and-grey-lists.html",
			client,
		),
	}
}

func (s *FATFScraper) Scrape(ctx context.Context) (*ScrapeResult, error) {
	result := s.NewResult()

	content, err := s.Fetch(ctx, s.url)
	if err != nil {
		result.RawCountries = fatfGreyListCountries
		result.ParseStatus = "fallback"
		return result, nil
	}

	result.ContentHash = HashContent(content)

	countries := extractCountriesFromText(string(content))

	if len(countries) > 0 {
		result.RawCountries = countries
		result.ParseStatus = "success"
	} else {
		result.RawCountries = fatfGreyListCountries
		result.ParseStatus = "fallback"
	}

	return result, nil
}

// extractCountriesFromText extracts country names from text using regex patterns.
func extractCountriesFromText(text string) []string {
	var countries []string
	seen := make(map[string]bool)

	// Look for country names in the text
	for _, country := range knownCountryNames {
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(country))
		re := regexp.MustCompile("(?i)" + pattern)
		if re.MatchString(text) {
			lower := strings.ToLower(country)
			if !seen[lower] {
				seen[lower] = true
				countries = append(countries, country)
			}
		}
	}

	return countries
}

// Fallback country lists (as of 2024)
var euSanctionedCountries = []string{
	"Russia", "Belarus", "Iran", "Syria", "North Korea", "Myanmar",
	"Venezuela", "Nicaragua", "Mali", "Guinea", "Sudan", "South Sudan",
	"Central African Republic", "Democratic Republic of the Congo",
	"Somalia", "Eritrea", "Libya", "Yemen", "Zimbabwe",
}

var usOFACSanctionedCountries = []string{
	"Cuba", "Iran", "North Korea", "Syria", "Russia", "Belarus",
	"Venezuela", "Myanmar", "Nicaragua", "Central African Republic",
	"Democratic Republic of the Congo", "Ethiopia", "Iraq", "Lebanon",
	"Libya", "Mali", "Somalia", "South Sudan", "Sudan", "Yemen", "Zimbabwe",
}

var ukSanctionedCountries = []string{
	"Russia", "Belarus", "Iran", "Syria", "North Korea", "Myanmar",
	"Venezuela", "Nicaragua", "Libya", "Mali", "Somalia", "South Sudan",
	"Sudan", "Yemen", "Zimbabwe", "Guinea", "Central African Republic",
}

var unSanctionedCountries = []string{
	"North Korea", "Iran", "Libya", "Mali", "Somalia", "South Sudan",
	"Sudan", "Yemen", "Central African Republic", "Democratic Republic of the Congo",
	"Iraq", "Lebanon",
}

var fatfGreyListCountries = []string{
	"Bulgaria", "Burkina Faso", "Cameroon", "Croatia", "Democratic Republic of the Congo",
	"Haiti", "Kenya", "Mali", "Monaco", "Mozambique", "Namibia", "Nigeria",
	"Philippines", "Senegal", "South Africa", "South Sudan", "Syria",
	"Tanzania", "Venezuela", "Vietnam", "Yemen",
}

// knownCountryNames is a list of country names for text matching.
var knownCountryNames = []string{
	"Afghanistan", "Albania", "Algeria", "Angola", "Argentina", "Armenia",
	"Azerbaijan", "Bahrain", "Bangladesh", "Belarus", "Benin", "Bolivia",
	"Bosnia", "Botswana", "Brazil", "Bulgaria", "Burkina Faso", "Burundi",
	"Cambodia", "Cameroon", "Central African Republic", "Chad", "China",
	"Colombia", "Comoros", "Congo", "Croatia", "Cuba", "Cyprus",
	"Democratic Republic of the Congo", "Djibouti", "Dominican Republic",
	"Ecuador", "Egypt", "El Salvador", "Equatorial Guinea", "Eritrea",
	"Eswatini", "Ethiopia", "Gabon", "Gambia", "Georgia", "Ghana",
	"Guatemala", "Guinea", "Guinea-Bissau", "Haiti", "Honduras", "Hungary",
	"India", "Indonesia", "Iran", "Iraq", "Israel", "Ivory Coast",
	"Jamaica", "Jordan", "Kazakhstan", "Kenya", "Kosovo", "Kuwait",
	"Kyrgyzstan", "Laos", "Lebanon", "Lesotho", "Liberia", "Libya",
	"Madagascar", "Malawi", "Malaysia", "Maldives", "Mali", "Mauritania",
	"Mexico", "Moldova", "Monaco", "Mongolia", "Montenegro", "Morocco",
	"Mozambique", "Myanmar", "Burma", "Namibia", "Nepal", "Nicaragua",
	"Niger", "Nigeria", "North Korea", "DPRK", "North Macedonia", "Oman",
	"Pakistan", "Palestine", "Panama", "Papua New Guinea", "Paraguay",
	"Peru", "Philippines", "Poland", "Qatar", "Romania", "Russia",
	"Rwanda", "Saudi Arabia", "Senegal", "Serbia", "Sierra Leone",
	"Somalia", "South Africa", "South Sudan", "Sri Lanka", "Sudan",
	"Swaziland", "Syria", "Tajikistan", "Tanzania", "Thailand", "Togo",
	"Trinidad and Tobago", "Tunisia", "Turkey", "Turkmenistan", "Uganda",
	"Ukraine", "United Arab Emirates", "UAE", "Uzbekistan", "Venezuela",
	"Vietnam", "Yemen", "Zambia", "Zimbabwe",
}

