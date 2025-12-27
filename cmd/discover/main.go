// Command discover probes a UniFi controller to find API endpoints,
// with a focus on discovering the Region Blocking / CyberSecure endpoint.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mattsblocklist/tae/internal/unifi"
)

type DiscoveryResult struct {
	Timestamp        time.Time               `json:"timestamp"`
	ControllerURL    string                  `json:"controller_url"`
	Site             string                  `json:"site"`
	TotalTested      int                     `json:"total_tested"`
	FoundEndpoints   int                     `json:"found_endpoints"`
	Endpoints        []*unifi.EndpointResult `json:"endpoints"`
	RegionBlocking   *RegionBlockingInfo     `json:"region_blocking,omitempty"`
	SettingsAnalysis *SettingsAnalysis       `json:"settings_analysis,omitempty"`
}

type RegionBlockingInfo struct {
	EndpointFound bool   `json:"endpoint_found"`
	Endpoint      string `json:"endpoint,omitempty"`
	Method        string `json:"method,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

type SettingsAnalysis struct {
	Keys          []string `json:"keys"`
	GeoRelated    []string `json:"geo_related,omitempty"`
	SecurityKeys  []string `json:"security_keys,omitempty"`
	ThreatKeys    []string `json:"threat_keys,omitempty"`
}

func main() {
	// Command line flags
	host := flag.String("host", "", "UniFi controller URL (e.g., https://10.5.22.1)")
	username := flag.String("username", "", "UniFi username")
	password := flag.String("password", "", "UniFi password")
	site := flag.String("site", "default", "UniFi site name")
	insecure := flag.Bool("insecure", false, "Skip TLS certificate verification")
	output := flag.String("output", "", "Output file path (JSON format)")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	workers := flag.Int("workers", 5, "Number of concurrent workers")
	regionOnly := flag.Bool("region-only", false, "Only test region blocking candidate endpoints")

	flag.Parse()

	// Validate required flags or try environment variables
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

	fmt.Printf("Connecting to UniFi controller at %s...\n", *host)

	// Create client
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

	fmt.Println("Authentication successful!")

	// Build list of endpoints to test
	var endpoints []string
	if *regionOnly {
		endpoints = buildRegionBlockingEndpoints(*site)
		fmt.Printf("Testing %d region blocking candidate endpoints...\n", len(endpoints))
	} else {
		endpoints = buildAllEndpoints(*site)
		fmt.Printf("Testing %d endpoints...\n", len(endpoints))
	}

	// Test endpoints concurrently
	results := testEndpoints(client, endpoints, *workers, *verbose)

	// Analyze results
	discoveryResult := analyzeResults(client, results, *site)

	// Analyze settings endpoint for geo-related keys
	analyzeSettings(client, discoveryResult, *verbose)

	// Output results
	printSummary(discoveryResult)

	if *output != "" {
		if err := saveResults(*output, discoveryResult); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save results: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nResults saved to %s\n", *output)
	}
}

func buildRegionBlockingEndpoints(site string) []string {
	var endpoints []string

	for _, ep := range unifi.RegionBlockingCandidates {
		ep = strings.ReplaceAll(ep, "{site}", site)
		endpoints = append(endpoints, ep)
	}

	return endpoints
}

func buildAllEndpoints(site string) []string {
	seen := make(map[string]bool)
	var endpoints []string

	addEndpoint := func(ep string) {
		ep = strings.ReplaceAll(ep, "{site}", site)
		if !seen[ep] {
			seen[ep] = true
			endpoints = append(endpoints, ep)
		}
	}

	// Add known endpoints
	for _, ep := range unifi.KnownEndpoints {
		addEndpoint(ep)
	}

	// Add v2 endpoints
	for _, ep := range unifi.V2Endpoints {
		addEndpoint(ep)
	}

	// Add region blocking candidates
	for _, ep := range unifi.RegionBlockingCandidates {
		addEndpoint(ep)
	}

	return endpoints
}

func testEndpoints(client *unifi.Client, endpoints []string, workerCount int, verbose bool) []*unifi.EndpointResult {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []*unifi.EndpointResult
	)

	// Create work channel
	work := make(chan string, len(endpoints))
	for _, ep := range endpoints {
		work <- ep
	}
	close(work)

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ep := range work {
				result, err := client.TestEndpoint(ep)
				if err != nil {
					if verbose {
						fmt.Printf("  [ERROR] %s: %v\n", ep, err)
					}
					continue
				}

				mu.Lock()
				results = append(results, result)
				mu.Unlock()

				if verbose {
					if result.Exists {
						fmt.Printf("  [FOUND] %s (status: %d, size: %d)\n", ep, result.StatusCode, result.ResponseSize)
					} else {
						fmt.Printf("  [MISS]  %s (status: %d)\n", ep, result.StatusCode)
					}
				} else if result.Exists {
					fmt.Printf("  Found: %s\n", ep)
				}
			}
		}()
	}

	wg.Wait()

	// Sort by path
	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results
}

func analyzeResults(client *unifi.Client, results []*unifi.EndpointResult, site string) *DiscoveryResult {
	dr := &DiscoveryResult{
		Timestamp:     time.Now(),
		ControllerURL: client.BaseURL(),
		Site:          site,
		TotalTested:   len(results),
		RegionBlocking: &RegionBlockingInfo{
			EndpointFound: false,
		},
	}

	var foundEndpoints []*unifi.EndpointResult
	for _, r := range results {
		if r.Exists {
			foundEndpoints = append(foundEndpoints, r)
		}
	}
	dr.FoundEndpoints = len(foundEndpoints)
	dr.Endpoints = foundEndpoints

	// Look for region blocking indicators in found endpoints
	geoKeywords := []string{"geo", "region", "country", "block", "restrict", "cybersecure", "threat"}
	for _, ep := range foundEndpoints {
		pathLower := strings.ToLower(ep.Path)
		for _, kw := range geoKeywords {
			if strings.Contains(pathLower, kw) {
				// Check if response contains country/region data
				if strings.Contains(ep.ResponseSample, "country") ||
					strings.Contains(ep.ResponseSample, "geo") ||
					strings.Contains(ep.ResponseSample, "region") ||
					strings.Contains(ep.ResponseSample, "block") {
					dr.RegionBlocking.EndpointFound = true
					dr.RegionBlocking.Endpoint = ep.Path
					dr.RegionBlocking.Notes = "Found endpoint with geo-related response data"
					break
				}
			}
		}
	}

	return dr
}

func analyzeSettings(client *unifi.Client, dr *DiscoveryResult, verbose bool) {
	// Fetch the settings endpoint to look for geo-related configuration
	body, status, err := client.Get("rest/setting")
	if err != nil || status != 200 {
		if verbose {
			fmt.Printf("Could not analyze settings endpoint: %v (status: %d)\n", err, status)
		}
		return
	}

	var settings []map[string]interface{}
	if err := json.Unmarshal(body, &settings); err != nil {
		// Try alternate format
		var wrapper struct {
			Data []map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(body, &wrapper); err != nil {
			if verbose {
				fmt.Printf("Could not parse settings: %v\n", err)
			}
			return
		}
		settings = wrapper.Data
	}

	analysis := &SettingsAnalysis{}

	geoKeywords := []string{"geo", "region", "country", "block"}
	securityKeywords := []string{"security", "firewall", "threat", "cybersecure"}
	threatKeywords := []string{"threat", "ips", "ids", "malware"}

	for _, s := range settings {
		if key, ok := s["key"].(string); ok {
			analysis.Keys = append(analysis.Keys, key)

			keyLower := strings.ToLower(key)
			for _, kw := range geoKeywords {
				if strings.Contains(keyLower, kw) {
					analysis.GeoRelated = append(analysis.GeoRelated, key)
					break
				}
			}
			for _, kw := range securityKeywords {
				if strings.Contains(keyLower, kw) {
					analysis.SecurityKeys = append(analysis.SecurityKeys, key)
					break
				}
			}
			for _, kw := range threatKeywords {
				if strings.Contains(keyLower, kw) {
					analysis.ThreatKeys = append(analysis.ThreatKeys, key)
					break
				}
			}
		}
	}

	dr.SettingsAnalysis = analysis

	if len(analysis.GeoRelated) > 0 {
		dr.RegionBlocking.Notes = fmt.Sprintf("Found geo-related settings keys: %v", analysis.GeoRelated)
		if !dr.RegionBlocking.EndpointFound {
			dr.RegionBlocking.Endpoint = "rest/setting (check geo-related keys)"
		}
	}
}

func printSummary(dr *DiscoveryResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("DISCOVERY SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("Controller: %s\n", dr.ControllerURL)
	fmt.Printf("Site: %s\n", dr.Site)
	fmt.Printf("Endpoints tested: %d\n", dr.TotalTested)
	fmt.Printf("Endpoints found: %d\n", dr.FoundEndpoints)

	if dr.FoundEndpoints > 0 {
		fmt.Println("\nFound endpoints:")
		for _, ep := range dr.Endpoints {
			fmt.Printf("  - %s (size: %d bytes)\n", ep.Path, ep.ResponseSize)
		}
	}

	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("REGION BLOCKING ANALYSIS")
	fmt.Println(strings.Repeat("-", 60))

	if dr.RegionBlocking.EndpointFound {
		fmt.Printf("Status: FOUND\n")
		fmt.Printf("Endpoint: %s\n", dr.RegionBlocking.Endpoint)
		if dr.RegionBlocking.Notes != "" {
			fmt.Printf("Notes: %s\n", dr.RegionBlocking.Notes)
		}
	} else {
		fmt.Println("Status: NOT FOUND (may require UI capture)")
		fmt.Println("Recommendation: Use browser DevTools to capture the API call")
		fmt.Println("when toggling Region Blocking in Settings -> CyberSecure")
	}

	if dr.SettingsAnalysis != nil {
		fmt.Println("\n" + strings.Repeat("-", 60))
		fmt.Println("SETTINGS ANALYSIS")
		fmt.Println(strings.Repeat("-", 60))

		fmt.Printf("Total setting keys: %d\n", len(dr.SettingsAnalysis.Keys))

		if len(dr.SettingsAnalysis.GeoRelated) > 0 {
			fmt.Printf("Geo-related keys: %v\n", dr.SettingsAnalysis.GeoRelated)
		}
		if len(dr.SettingsAnalysis.SecurityKeys) > 0 {
			fmt.Printf("Security keys: %v\n", dr.SettingsAnalysis.SecurityKeys)
		}
		if len(dr.SettingsAnalysis.ThreatKeys) > 0 {
			fmt.Printf("Threat keys: %v\n", dr.SettingsAnalysis.ThreatKeys)
		}

		// Print all keys if verbose
		if len(dr.SettingsAnalysis.Keys) > 0 {
			fmt.Println("\nAll setting keys:")
			for _, k := range dr.SettingsAnalysis.Keys {
				fmt.Printf("  - %s\n", k)
			}
		}
	}
}

func saveResults(path string, dr *DiscoveryResult) error {
	data, err := json.MarshalIndent(dr, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

