// Command configure applies the aggregated country blocklist to a UniFi controller's
// Region Blocking / CyberSecure settings.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mattsblocklist/tae/internal/unifi"
)

// ConfigResult contains the result of a configuration operation.
type ConfigResult struct {
	Timestamp    time.Time `json:"timestamp"`
	DryRun       bool      `json:"dry_run"`
	Changed      bool      `json:"changed"`
	PreviousCodes []string `json:"previous_codes,omitempty"`
	DesiredCodes  []string `json:"desired_codes"`
	AddedCodes    []string `json:"added_codes,omitempty"`
	RemovedCodes  []string `json:"removed_codes,omitempty"`
	Verified      bool     `json:"verified"`
	Error         string   `json:"error,omitempty"`
}

func main() {
	// Command line flags
	host := flag.String("host", "", "UniFi controller URL")
	username := flag.String("username", "", "UniFi username")
	password := flag.String("password", "", "UniFi password")
	site := flag.String("site", "default", "UniFi site name")
	insecure := flag.Bool("insecure", false, "Skip TLS certificate verification")
	inputFile := flag.String("input", "data/blocked_countries.txt", "Input file with country codes")
	inputURL := flag.String("input-url", "", "URL to fetch country codes from (overrides -input)")
	dryRun := flag.Bool("dry-run", false, "Show what would change without applying")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	outputJSON := flag.String("output", "", "Write result to JSON file")
	endpoint := flag.String("endpoint", "", "Override the region blocking endpoint path")
	enable := flag.Bool("enable", true, "Enable region blocking (set to false to disable)")

	flag.Parse()

	// Load from environment if not provided
	if *host == "" {
		*host = os.Getenv("UNIFI_HOST")
	}
	if *username == "" {
		*username = os.Getenv("UNIFI_USERNAME")
	}
	if *password == "" {
		*password = os.Getenv("UNIFI_PASSWORD")
	}

	if *host == "" || *username == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Error: host, username, and password are required")
		fmt.Fprintln(os.Stderr, "Use flags or environment variables: UNIFI_HOST, UNIFI_USERNAME, UNIFI_PASSWORD")
		flag.Usage()
		os.Exit(1)
	}

	// Load desired country codes
	codes, err := loadCodes(*inputFile, *inputURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading country codes: %v\n", err)
		os.Exit(1)
	}

	if len(codes) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no country codes loaded")
		os.Exit(1)
	}

	fmt.Printf("Loaded %d country codes to apply\n", len(codes))
	if *verbose {
		fmt.Printf("Codes: %s\n", strings.Join(codes, ", "))
	}

	if *dryRun {
		fmt.Println("\n[DRY RUN MODE - No changes will be applied]")
	}

	// Connect to UniFi
	fmt.Printf("\nConnecting to %s...\n", *host)

	client, err := unifi.NewClient(unifi.ClientConfig{
		Host:          *host,
		Username:      *username,
		Password:      *password,
		Site:          *site,
		SkipTLSVerify: *insecure,
		Verbose:       *verbose,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer client.Logout()

	fmt.Println("Connected successfully")

	// Run the configuration
	result := configureRegionBlocking(client, codes, *endpoint, *enable, *dryRun, *verbose)

	// Print result
	printResult(result)

	// Save result if requested
	if *outputJSON != "" {
		if err := saveResult(*outputJSON, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving result: %v\n", err)
		} else {
			fmt.Printf("\nResult saved to %s\n", *outputJSON)
		}
	}

	if result.Error != "" {
		os.Exit(1)
	}
}

func loadCodes(filePath, url string) ([]string, error) {
	var content []byte
	var err error

	if url != "" {
		// Fetch from URL
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		content, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
	} else {
		// Read from file
		content, err = os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
	}

	// Parse codes (one per line, skip comments and blank lines)
	var codes []string
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments (lines starting with #) and blank lines
		if line != "" && !strings.HasPrefix(line, "#") {
			// Validate it looks like a country code (2 uppercase letters)
			if len(line) == 2 {
				codes = append(codes, strings.ToUpper(line))
			}
		}
	}

	return codes, scanner.Err()
}

func configureRegionBlocking(client *unifi.Client, desiredCodes []string, endpointOverride string, enable, dryRun, verbose bool) *ConfigResult {
	result := &ConfigResult{
		Timestamp:    time.Now(),
		DryRun:       dryRun,
		DesiredCodes: desiredCodes,
	}

	// Use the discovered endpoint for region blocking (usg setting)
	// endpointOverride is ignored since we now use the specific API methods

	// Fetch current configuration using the new API
	currentCodes, err := client.GetBlockedCountries()
	if err != nil {
		result.Error = fmt.Sprintf("failed to get current config: %v", err)
		return result
	}

	if verbose {
		fmt.Printf("Current blocked countries: %v\n", currentCodes)
	}

	result.PreviousCodes = currentCodes

	// Check current enabled state
	setting, err := client.GetRegionBlockingSettings()
	currentEnabled := false
	if err == nil {
		if enabledVal, ok := setting["geo_ip_filtering_enabled"].(bool); ok {
			currentEnabled = enabledVal
		}
	}

	// Calculate diff
	added, removed := diffCodes(currentCodes, desiredCodes)
	result.AddedCodes = added
	result.RemovedCodes = removed
	result.Changed = len(added) > 0 || len(removed) > 0 || currentEnabled != enable

	if !result.Changed {
		fmt.Println("\nNo changes needed - configuration already matches")
		result.Verified = true
		return result
	}

	fmt.Printf("\nChanges required:\n")
	if currentEnabled != enable {
		fmt.Printf("  Enable: %v -> %v\n", currentEnabled, enable)
	}
	if len(added) > 0 {
		fmt.Printf("  Adding: %s\n", strings.Join(added, ", "))
	}
	if len(removed) > 0 {
		fmt.Printf("  Removing: %s\n", strings.Join(removed, ", "))
	}

	if dryRun {
		fmt.Println("\n[DRY RUN] Changes not applied")
		return result
	}

	// Apply changes using the new API
	if err := client.UpdateRegionBlockingSettings(enable, desiredCodes, "block", "both"); err != nil {
		result.Error = fmt.Sprintf("failed to apply changes: %v", err)
		return result
	}

	fmt.Println("Configuration applied successfully")

	// Verify
	newCodes, err := client.GetBlockedCountries()
	if err != nil {
		result.Error = fmt.Sprintf("failed to verify: %v", err)
		return result
	}

	// Check if the new config matches desired
	sort.Strings(newCodes)
	sort.Strings(desiredCodes)

	result.Verified = len(newCodes) == len(desiredCodes)
	if result.Verified {
		for i := range newCodes {
			if newCodes[i] != desiredCodes[i] {
				result.Verified = false
				break
			}
		}
	}

	return result
}

// discoverRegionBlockingEndpoint and getCurrentBlockedCountries are no longer needed
// as we now use the specific API methods in the unifi client.

// extractCountryCodesFromData and isUpperAlpha removed - no longer needed
// as we use the specific API methods in the unifi client

// applyBlockedCountries is no longer needed as we use client.UpdateRegionBlockingSettings

func diffCodes(current, desired []string) (added, removed []string) {
	currentSet := make(map[string]bool)
	desiredSet := make(map[string]bool)

	for _, c := range current {
		currentSet[c] = true
	}
	for _, c := range desired {
		desiredSet[c] = true
	}

	for c := range desiredSet {
		if !currentSet[c] {
			added = append(added, c)
		}
	}

	for c := range currentSet {
		if !desiredSet[c] {
			removed = append(removed, c)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)

	return
}

func printResult(result *ConfigResult) {
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("CONFIGURATION RESULT")
	fmt.Println(strings.Repeat("=", 40))

	if result.DryRun {
		fmt.Println("Mode: DRY RUN (no changes applied)")
	} else {
		fmt.Println("Mode: APPLY")
	}

	fmt.Printf("Changed: %v\n", result.Changed)

	if result.Changed {
		fmt.Printf("Added: %d codes\n", len(result.AddedCodes))
		fmt.Printf("Removed: %d codes\n", len(result.RemovedCodes))
	}

	if !result.DryRun && result.Changed {
		fmt.Printf("Verified: %v\n", result.Verified)
	}

	if result.Error != "" {
		fmt.Printf("Error: %s\n", result.Error)
	}
}

func saveResult(path string, result *ConfigResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

