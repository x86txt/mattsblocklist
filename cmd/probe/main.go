// Command probe uses the UniFi client to directly probe for region blocking endpoints
// and capture the exact API structure.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mattsblocklist/tae/internal/unifi"
)

func main() {
	host := flag.String("host", "", "UniFi controller URL")
	username := flag.String("username", "", "UniFi username")
	password := flag.String("password", "", "UniFi password")
	site := flag.String("site", "default", "UniFi site name")
	insecure := flag.Bool("insecure", false, "Skip TLS certificate verification")
	output := flag.String("output", "api-discovery.json", "Output file for discovered API structure")

	flag.Parse()

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
		os.Exit(1)
	}

	fmt.Printf("Connecting to %s...\n", *host)

	client, err := unifi.NewClient(unifi.ClientConfig{
		Host:          *host,
		Username:      *username,
		Password:      *password,
		Site:          *site,
		SkipTLSVerify: *insecure,
		Verbose:       true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer client.Logout()

	fmt.Println("Connected! Probing endpoints...\n")

	results := make(map[string]interface{})

	// 1. Get all settings to find geo-related keys
	fmt.Println("1. Fetching all settings...")
	body, status, err := client.Get("rest/setting")
	if err == nil && status == 200 {
		var settings []map[string]interface{}
		if err := json.Unmarshal(body, &settings); err != nil {
			var wrapper struct {
				Data []map[string]interface{} `json:"data"`
			}
			if err := json.Unmarshal(body, &wrapper); err == nil {
				settings = wrapper.Data
			}
		}

		// Find geo-related settings
		var geoSettings []map[string]interface{}
		for _, s := range settings {
			if key, ok := s["key"].(string); ok {
				keyLower := strings.ToLower(key)
				if strings.Contains(keyLower, "geo") ||
					strings.Contains(keyLower, "region") ||
					strings.Contains(keyLower, "country") ||
					strings.Contains(keyLower, "block") ||
					strings.Contains(keyLower, "cybersecure") ||
					strings.Contains(keyLower, "threat") {
					geoSettings = append(geoSettings, s)
				}
			}
		}

		results["all_settings"] = settings
		results["geo_related_settings"] = geoSettings

		fmt.Printf("   Found %d total settings, %d geo-related\n", len(settings), len(geoSettings))

		// Try to find the region blocking setting specifically
		for _, s := range geoSettings {
			fmt.Printf("\n   Setting: %v\n", s["key"])
			pretty, _ := json.MarshalIndent(s, "     ", "  ")
			fmt.Printf("     %s\n", string(pretty))
		}
	}

	// 2. Get country codes
	fmt.Println("\n2. Fetching country codes...")
	body, status, err = client.Get("stat/ccode")
	if err == nil && status == 200 {
		var ccodeData interface{}
		json.Unmarshal(body, &ccodeData)
		results["country_codes"] = ccodeData
		fmt.Printf("   Status: %d\n", status)
		fmt.Printf("   Response size: %d bytes\n", len(body))
		if len(body) < 1000 {
			fmt.Printf("   Response: %s\n", string(body))
		}
	}

	// 3. Try v2 API endpoints
	fmt.Println("\n3. Trying v2 API endpoints...")
	v2Endpoints := []string{
		"v2/api/site/" + *site + "/trafficrules",
		"v2/api/site/" + *site + "/security",
		"v2/api/site/" + *site + "/threat-management",
	}

	for _, ep := range v2Endpoints {
		fmt.Printf("\n   Trying: %s\n", ep)
		body, status, err := client.Get(ep)
		if err == nil {
			fmt.Printf("     Status: %d\n", status)
			if status == 200 {
				var data interface{}
				if err := json.Unmarshal(body, &data); err == nil {
					results[ep] = data
					pretty, _ := json.MarshalIndent(data, "     ", "  ")
					if len(pretty) < 2000 {
						fmt.Printf("     Response:\n%s\n", string(pretty))
					} else {
						fmt.Printf("     Response: %d bytes (truncated)\n", len(body))
					}
				} else {
					results[ep+"_raw"] = string(body)
					fmt.Printf("     Response: %s\n", string(body[:min(200, len(body))]))
				}
			}
		} else {
			fmt.Printf("     Error: %v\n", err)
		}
	}

	// 4. Try to GET specific setting keys if we found them
	if geoSettingsRaw, ok := results["geo_related_settings"].([]map[string]interface{}); ok {
		fmt.Println("\n4. Fetching detailed setting data...")
		for _, s := range geoSettingsRaw {
			if key, ok := s["key"].(string); ok {
				if id, ok := s["_id"].(string); ok {
					settingPath := fmt.Sprintf("rest/setting/%s/%s", key, id)
					fmt.Printf("\n   Fetching: %s\n", settingPath)
					body, status, err := client.Get(settingPath)
					if err == nil && status == 200 {
						var data interface{}
						if err := json.Unmarshal(body, &data); err == nil {
							results["setting_"+key] = data
							pretty, _ := json.MarshalIndent(data, "     ", "  ")
							if len(pretty) < 2000 {
								fmt.Printf("     %s\n", string(pretty))
							} else {
								fmt.Printf("     Response: %d bytes\n", len(body))
							}
						}
					}
				}
			}
		}
	}

	// Save results
	outputData := map[string]interface{}{
		"controller_url": *host,
		"site":          *site,
		"discovered":    results,
	}

	jsonData, err := json.MarshalIndent(outputData, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling results: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*output, jsonData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n\nResults saved to %s\n", *output)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Review the discovered settings in the output file")
	fmt.Println("2. Look for setting keys containing 'geo', 'region', 'country', or 'block'")
	fmt.Println("3. Note the structure of the setting data (especially country code format)")
	fmt.Println("4. Use this information to update cmd/configure/main.go")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
