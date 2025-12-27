// Command aggregate collects country blocklists from multiple sources,
// normalizes them to ISO 3166-1 alpha-2 codes, and outputs the combined list.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mattsblocklist/tae/internal/countries"
	"github.com/mattsblocklist/tae/internal/scrapers"
)

// AggregationResult contains the final output.
type AggregationResult struct {
	// Metadata header
	Name         string                   `json:"name"`
	Version      string                   `json:"version"`
	Description  string                   `json:"description"`
	LastModified time.Time                `json:"last_modified"`
	
	// Data
	Timestamp    time.Time                `json:"timestamp"`
	TotalCodes   int                      `json:"total_codes"`
	Countries    []CountryWithProvenance  `json:"countries"`
	SourceStats  map[string]SourceStats   `json:"source_stats"`
	Errors       []string                 `json:"errors,omitempty"`
}

// CountryWithProvenance includes source information.
type CountryWithProvenance struct {
	Alpha2    string   `json:"alpha2"`
	Name      string   `json:"name"`
	Sources   []string `json:"sources"`
	RawTokens []string `json:"raw_tokens,omitempty"`
}

// SourceStats contains statistics for each source.
type SourceStats struct {
	URL          string    `json:"url"`
	FetchedAt    time.Time `json:"fetched_at"`
	ParseStatus  string    `json:"parse_status"`
	RawCount     int       `json:"raw_count"`
	MatchedCount int       `json:"matched_count"`
	Error        string    `json:"error,omitempty"`
}

func main() {
	// Command line flags
	outputTxt := flag.String("output-txt", "data/blocked_countries.txt", "Output text file (one code per line)")
	outputJSON := flag.String("output-json", "data/blocked_countries.json", "Output JSON file with provenance")
	sources := flag.String("sources", "", "Comma-separated list of sources to use (empty = all)")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	timeout := flag.Duration("timeout", 60*time.Second, "HTTP request timeout")
	workers := flag.Int("workers", 4, "Number of concurrent workers")

	flag.Parse()

	fmt.Println("Country Blocklist Aggregator")
	fmt.Println(strings.Repeat("=", 40))

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: *timeout,
	}

	// Create scraper registry
	registry := scrapers.DefaultRegistry(httpClient)

	// Determine which sources to use
	var selectedSources []string
	if *sources != "" {
		selectedSources = strings.Split(*sources, ",")
		for i := range selectedSources {
			selectedSources[i] = strings.TrimSpace(selectedSources[i])
		}
	} else {
		selectedSources = registry.Names()
	}

	fmt.Printf("Using %d sources\n\n", len(selectedSources))

	// Run scrapers concurrently
	ctx := context.Background()
	results := runScrapers(ctx, registry, selectedSources, *workers, *verbose)

	// Create normalizer
	normalizer := countries.NewNormalizer()

	// Aggregate results
	aggregated := aggregate(results, normalizer, *verbose)
	
	// Set metadata
	aggregated.Name = "UniFi Region Blocking Country List"
	aggregated.Version = "1.0.0"
	aggregated.Description = "Aggregated list of countries subject to sanctions, export controls, or other restrictions from multiple authoritative sources. This list is intended for use with UniFi Network's Region Blocking (GeoIP Filtering) feature to block traffic from these countries."
	aggregated.LastModified = time.Now()

	// Print summary
	printSummary(aggregated)

	// Write output files
	if err := writeOutputs(aggregated, *outputTxt, *outputJSON); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing outputs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nOutput written to:\n")
	fmt.Printf("  - %s\n", *outputTxt)
	fmt.Printf("  - %s\n", *outputJSON)
}

func runScrapers(ctx context.Context, registry *scrapers.Registry, sources []string, workers int, verbose bool) []*scrapers.ScrapeResult {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []*scrapers.ScrapeResult
	)

	work := make(chan scrapers.Scraper, len(sources))

	// Queue work
	for _, name := range sources {
		if s, ok := registry.Get(name); ok {
			work <- s
		} else if verbose {
			fmt.Printf("  [WARN] Unknown source: %s\n", name)
		}
	}
	close(work)

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range work {
				fmt.Printf("  Fetching: %s...\n", s.Name())

				result, err := s.Scrape(ctx)
				if err != nil {
					fmt.Printf("    [ERROR] %s: %v\n", s.Name(), err)
					continue
				}

				if verbose {
					fmt.Printf("    Status: %s, Raw countries: %d\n", result.ParseStatus, len(result.RawCountries))
				}

				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return results
}

func aggregate(results []*scrapers.ScrapeResult, normalizer *countries.Normalizer, verbose bool) *AggregationResult {
	agg := &AggregationResult{
		Timestamp:   time.Now(),
		SourceStats: make(map[string]SourceStats),
	}

	// Map from country code to provenance
	countryMap := make(map[string]*CountryWithProvenance)

	for _, result := range results {
		stats := SourceStats{
			URL:         result.URL,
			FetchedAt:   result.FetchedAt,
			ParseStatus: result.ParseStatus,
			RawCount:    len(result.RawCountries),
			Error:       result.Error,
		}

		matched := 0
		for _, raw := range result.RawCountries {
			code, ok := normalizer.Normalize(raw)
			if !ok {
				if verbose {
					fmt.Printf("    [SKIP] Could not normalize: %q\n", raw)
				}
				continue
			}

			matched++

			if existing, ok := countryMap[code]; ok {
				// Add source if not already present
				hasSource := false
				for _, s := range existing.Sources {
					if s == result.Source {
						hasSource = true
						break
					}
				}
				if !hasSource {
					existing.Sources = append(existing.Sources, result.Source)
				}
				existing.RawTokens = append(existing.RawTokens, raw)
			} else {
				countryMap[code] = &CountryWithProvenance{
					Alpha2:    code,
					Name:      normalizer.GetName(code),
					Sources:   []string{result.Source},
					RawTokens: []string{raw},
				}
			}
		}

		stats.MatchedCount = matched
		agg.SourceStats[result.Source] = stats

		if result.Error != "" {
			agg.Errors = append(agg.Errors, fmt.Sprintf("%s: %s", result.Source, result.Error))
		}
	}

	// Convert map to sorted slice
	for _, c := range countryMap {
		agg.Countries = append(agg.Countries, *c)
	}

	sort.Slice(agg.Countries, func(i, j int) bool {
		return agg.Countries[i].Alpha2 < agg.Countries[j].Alpha2
	})

	agg.TotalCodes = len(agg.Countries)

	return agg
}

func printSummary(agg *AggregationResult) {
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("AGGREGATION SUMMARY")
	fmt.Println(strings.Repeat("=", 40))

	fmt.Printf("Total unique country codes: %d\n\n", agg.TotalCodes)

	fmt.Println("Source statistics:")
	for name, stats := range agg.SourceStats {
		status := stats.ParseStatus
		if stats.Error != "" {
			status = "error"
		}
		fmt.Printf("  - %s: %d raw -> %d matched (%s)\n", name, stats.RawCount, stats.MatchedCount, status)
	}

	fmt.Println("\nCountries by source count:")
	sourceCounts := make(map[int][]string)
	for _, c := range agg.Countries {
		n := len(c.Sources)
		sourceCounts[n] = append(sourceCounts[n], c.Alpha2)
	}

	var counts []int
	for n := range sourceCounts {
		counts = append(counts, n)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(counts)))

	for _, n := range counts {
		codes := sourceCounts[n]
		sort.Strings(codes)
		fmt.Printf("  %d sources: %s\n", n, strings.Join(codes, ", "))
	}

	if len(agg.Errors) > 0 {
		fmt.Println("\nWarnings/Errors:")
		for _, e := range agg.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}
}

func writeOutputs(agg *AggregationResult, txtPath, jsonPath string) error {
	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Write text file with header
	var txtBuilder strings.Builder
	txtBuilder.WriteString("# " + agg.Name + "\n")
	txtBuilder.WriteString("# Version: " + agg.Version + "\n")
	txtBuilder.WriteString("# Last Modified: " + agg.LastModified.Format("2006-01-02 15:04:05 MST") + "\n")
	txtBuilder.WriteString("#\n")
	txtBuilder.WriteString("# " + strings.ReplaceAll(agg.Description, "\n", "\n# ") + "\n")
	txtBuilder.WriteString("#\n")
	txtBuilder.WriteString("# Country codes (ISO 3166-1 alpha-2)\n")
	txtBuilder.WriteString("#\n")
	
	var codes []string
	for _, c := range agg.Countries {
		codes = append(codes, c.Alpha2)
	}
	txtBuilder.WriteString(strings.Join(codes, "\n"))
	txtBuilder.WriteString("\n")

	if err := os.WriteFile(txtPath, []byte(txtBuilder.String()), 0644); err != nil {
		return fmt.Errorf("failed to write txt file: %w", err)
	}

	// Write JSON file
	jsonContent, err := json.MarshalIndent(agg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(jsonPath, jsonContent, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

