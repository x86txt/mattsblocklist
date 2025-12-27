// Command parse-har extracts UniFi API endpoint information from a browser HAR file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

type HAR struct {
	Log Log `json:"log"`
}

type Log struct {
	Entries []Entry `json:"entries"`
}

type Entry struct {
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

type Request struct {
	Method  string   `json:"method"`
	URL     string   `json:"url"`
	Headers []Header `json:"headers"`
	PostData *PostData `json:"postData,omitempty"`
}

type Response struct {
	Status  int      `json:"status"`
	Headers []Header `json:"headers"`
	Content Content  `json:"content"`
}

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type Content struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
}

type APIEndpoint struct {
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Status       int               `json:"status"`
	Headers      map[string]string `json:"headers"`
	RequestBody  string            `json:"request_body,omitempty"`
	ResponseBody string            `json:"response_body,omitempty"`
	IsRelevant   bool              `json:"is_relevant"`
}

type AnalysisResult struct {
	TotalEntries   int                     `json:"total_entries"`
	RelevantAPIs   []APIEndpoint           `json:"relevant_apis"`
	AuthInfo       map[string]string       `json:"auth_info,omitempty"`
	CSRFToken      string                  `json:"csrf_token,omitempty"`
	RegionBlocking map[string]interface{}  `json:"region_blocking,omitempty"`
}

func main() {
	harFile := flag.String("har", "", "Path to HAR file")
	output := flag.String("output", "api-endpoints.json", "Output file")
	verbose := flag.Bool("verbose", false, "Verbose output")
	flag.Parse()

	if *harFile == "" {
		fmt.Fprintln(os.Stderr, "Usage: parse-har -har <file.har> [-output <out.json>]")
		os.Exit(1)
	}

	data, err := os.ReadFile(*harFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var har HAR
	if err := json.Unmarshal(data, &har); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing HAR: %v\n", err)
		os.Exit(1)
	}

	result := analyzeHAR(har, *verbose)
	
	outputData, _ := json.MarshalIndent(result, "", "  ")
	os.WriteFile(*output, outputData, 0644)
	
	fmt.Printf("Analyzed %d entries, found %d relevant APIs\n", result.TotalEntries, len(result.RelevantAPIs))
	fmt.Printf("Results saved to: %s\n", *output)
	printSummary(result)
}

func analyzeHAR(har HAR, verbose bool) *AnalysisResult {
	result := &AnalysisResult{
		TotalEntries: len(har.Log.Entries),
		AuthInfo:     make(map[string]string),
		RegionBlocking: make(map[string]interface{}),
	}

	for _, entry := range har.Log.Entries {
		ep := parseEntry(entry)
		if isRelevantAPI(ep) {
			ep.IsRelevant = true
			result.RelevantAPIs = append(result.RelevantAPIs, ep)
		}
		if token := ep.Headers["x-csrf-token"]; token != "" {
			result.CSRFToken = token
		}
	}

	sort.Slice(result.RelevantAPIs, func(i, j int) bool {
		return result.RelevantAPIs[i].URL < result.RelevantAPIs[j].URL
	})

	return result
}

func parseEntry(entry Entry) APIEndpoint {
	ep := APIEndpoint{
		URL:     entry.Request.URL,
		Method:  entry.Request.Method,
		Status:  entry.Response.Status,
		Headers: make(map[string]string),
	}
	for _, h := range entry.Request.Headers {
		ep.Headers[strings.ToLower(h.Name)] = h.Value
	}
	if entry.Request.PostData != nil {
		ep.RequestBody = entry.Request.PostData.Text
	}
	if entry.Response.Content.Text != "" {
		ep.ResponseBody = entry.Response.Content.Text
	}
	return ep
}

func isRelevantAPI(ep APIEndpoint) bool {
	url := strings.ToLower(ep.URL)
	if !strings.Contains(url, "/proxy/network/") && !strings.Contains(url, "/api/") {
		return false
	}
	keywords := []string{"setting", "geo", "region", "country", "block", "cybersecure", "threat"}
	for _, kw := range keywords {
		if strings.Contains(url, kw) {
			return true
		}
	}
	return (ep.Method == "PUT" || ep.Method == "POST") && strings.Contains(url, "/api/")
}

func printSummary(result *AnalysisResult) {
	fmt.Println("\nRelevant API Endpoints:")
	for i, ep := range result.RelevantAPIs {
		fmt.Printf("\n%d. %s %s (Status: %d)\n", i+1, ep.Method, ep.URL, ep.Status)
		if ep.RequestBody != "" {
			fmt.Printf("   Request: %s\n", truncate(ep.RequestBody, 150))
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
