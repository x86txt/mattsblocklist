// Package scrapers provides interfaces and implementations for scraping
// country blocklists from various authoritative sources.
package scrapers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Scraper is the interface for all country list scrapers.
type Scraper interface {
	// Name returns the name of this data source.
	Name() string

	// URL returns the source URL.
	URL() string

	// Scrape fetches and parses country data from the source.
	Scrape(ctx context.Context) (*ScrapeResult, error)
}

// ScrapeResult contains the output of a scrape operation.
type ScrapeResult struct {
	Source       string    `json:"source"`
	URL          string    `json:"url"`
	FetchedAt    time.Time `json:"fetched_at"`
	ContentHash  string    `json:"content_hash"`
	RawCountries []string  `json:"raw_countries"`
	ParseStatus  string    `json:"parse_status"`
	Error        string    `json:"error,omitempty"`
}

// HTTPClient is an interface for making HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// BaseScraper provides common functionality for scrapers.
type BaseScraper struct {
	name       string
	url        string
	httpClient HTTPClient
}

// NewBaseScraper creates a new base scraper.
func NewBaseScraper(name, url string, client HTTPClient) *BaseScraper {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &BaseScraper{
		name:       name,
		url:        url,
		httpClient: client,
	}
}

// Name returns the scraper name.
func (b *BaseScraper) Name() string {
	return b.name
}

// URL returns the source URL.
func (b *BaseScraper) URL() string {
	return b.url
}

// Fetch retrieves content from a URL.
func (b *BaseScraper) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; tae-blocklist-aggregator/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// HashContent returns a SHA256 hash of the content.
func HashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// NewResult creates a new ScrapeResult with common fields populated.
func (b *BaseScraper) NewResult() *ScrapeResult {
	return &ScrapeResult{
		Source:    b.name,
		URL:       b.url,
		FetchedAt: time.Now(),
	}
}

// Registry holds all available scrapers.
type Registry struct {
	scrapers map[string]Scraper
}

// NewRegistry creates a new scraper registry.
func NewRegistry() *Registry {
	return &Registry{
		scrapers: make(map[string]Scraper),
	}
}

// Register adds a scraper to the registry.
func (r *Registry) Register(s Scraper) {
	r.scrapers[s.Name()] = s
}

// Get retrieves a scraper by name.
func (r *Registry) Get(name string) (Scraper, bool) {
	s, ok := r.scrapers[name]
	return s, ok
}

// All returns all registered scrapers.
func (r *Registry) All() []Scraper {
	scrapers := make([]Scraper, 0, len(r.scrapers))
	for _, s := range r.scrapers {
		scrapers = append(scrapers, s)
	}
	return scrapers
}

// Names returns the names of all registered scrapers.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.scrapers))
	for name := range r.scrapers {
		names = append(names, name)
	}
	return names
}

