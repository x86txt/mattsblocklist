# TAE - UniFi Region Blocking Automation Tool

Automated country blocklist aggregation and UniFi CyberSecure Region Blocking configuration.

## Overview

This toolkit provides three main commands:

1. **discover** - Probe UniFi API to find the Region Blocking endpoint
2. **aggregate** - Collect and normalize country blocklists from authoritative sources
3. **configure** - Apply the aggregated blocklist to UniFi with idempotent verification

## Installation

```bash
go install github.com/mattsblocklist/cmd/discover@latest
go install github.com/mattsblocklist/cmd/aggregate@latest
go install github.com/mattsblocklist/cmd/configure@latest
```

Or build from source:

```bash
git clone https://github.com/x86txt/mattsblocklist.git
cd mattsblocklist
go build -o bin/discover ./cmd/discover
go build -o bin/aggregate ./cmd/aggregate
go build -o bin/configure ./cmd/configure
```

## Quick Start

### 1. Aggregate Country Blocklist

```bash
# Run the aggregator to collect data from all sources
./bin/aggregate --verbose

# Output files:
#   data/blocked_countries.txt  - Simple list (one code per line)
#   data/blocked_countries.json - Full data with provenance
```

### 2. Discover UniFi API Endpoints

```bash
# Set credentials via environment
export UNIFI_HOST="https://10.5.22.1"
export UNIFI_USERNAME="programmatic"
export UNIFI_PASSWORD="your-password"

# Discover endpoints (focus on region blocking)
./bin/discover --insecure --region-only --output discovered.json
```

### 3. Apply to UniFi

```bash
# Dry run first
./bin/configure --insecure --dry-run

# Apply changes
./bin/configure --insecure --verbose
```

## Data Sources

All sources are scraped and normalized to ISO 3166-1 alpha-2 country codes.

### Censorship and Freedom Indices

| Source | Description | URL |
|--------|-------------|-----|
| Freedom House | Freedom on the Net report - countries rated "Not Free" | https://freedomhouse.org/countries/freedom-net/scores |
| OONI | Open Observatory of Network Interference - censorship data | https://ooni.org/countries/ |
| RSF | Reporters Without Borders Press Freedom Index | https://rsf.org/en/index |

### Government Sanctions Lists

| Source | Description | URL |
|--------|-------------|-----|
| EU Sanctions | European Union restrictive measures | https://www.sanctionsmap.eu/ |
| FATF Grey List | Financial Action Task Force high-risk jurisdictions | https://www.fatf-gafi.org/en/countries/black-and-grey-lists.html |
| UK Sanctions | UK financial sanctions consolidated list | https://www.gov.uk/government/collections/financial-sanctions-regime-specific-consolidated-lists-and-releases |
| UN Sanctions | United Nations Security Council sanctions | https://www.un.org/securitycouncil/sanctions/information |
| US OFAC | US Treasury Office of Foreign Assets Control | https://home.treasury.gov/policy-issues/financial-sanctions/sanctions-programs-and-country-information |

### Verification/Fallback

| Source | Description | URL |
|--------|-------------|-----|
| Blockpass Reference | Consolidated sanctions reference | https://help.blockpass.org/hc/en-us/articles/11881237145241-Which-countries-should-I-block-Sanctions-list-countries |

## Command Reference

### discover

```bash
./bin/discover [options]

Options:
  -host string        UniFi controller URL (or UNIFI_HOST env)
  -username string    UniFi username (or UNIFI_USERNAME env)
  -password string    UniFi password (or UNIFI_PASSWORD env)
  -site string        UniFi site name (default "default")
  -insecure          Skip TLS certificate verification
  -output string      Output file path (JSON format)
  -verbose           Enable verbose output
  -workers int       Number of concurrent workers (default 5)
  -region-only       Only test region blocking candidate endpoints
```

### aggregate

```bash
./bin/aggregate [options]

Options:
  -output-txt string   Output text file (default "data/blocked_countries.txt")
  -output-json string  Output JSON file (default "data/blocked_countries.json")
  -sources string      Comma-separated list of sources (empty = all)
  -verbose            Enable verbose output
  -timeout duration   HTTP request timeout (default 60s)
  -workers int        Number of concurrent workers (default 4)
```

### configure

```bash
./bin/configure [options]

Options:
  -host string       UniFi controller URL (or UNIFI_HOST env)
  -username string   UniFi username (or UNIFI_USERNAME env)
  -password string   UniFi password (or UNIFI_PASSWORD env)
  -site string       UniFi site name (default "default")
  -insecure         Skip TLS certificate verification
  -input string      Input file with country codes (default "data/blocked_countries.txt")
  -input-url string  URL to fetch country codes from (overrides -input)
  -dry-run          Show what would change without applying
  -verbose          Enable verbose output
  -output string     Write result to JSON file
  -endpoint string   Override the region blocking endpoint path
  -enable           Enable region blocking (default true)
```

## Configuration

### Environment Variables

```bash
export UNIFI_HOST="https://10.5.22.1"
export UNIFI_USERNAME="programmatic"
export UNIFI_PASSWORD="your-secure-password"
export UNIFI_SITE="default"
export UNIFI_SKIP_TLS_VERIFY="true"
export GITHUB_TOKEN="ghp_..."  # For GitHub integration
```

### Config File

Copy `config.yaml.example` to `config.yaml`:

```yaml
unifi:
  host: "https://10.5.22.1"
  username: "programmatic"
  password: "${UNIFI_PASSWORD}"
  site: "default"
  skip_tls_verify: true

github:
  repo: "mattsblocklist/tae"
  token: "${GITHUB_TOKEN}"
```

## Automated Updates with Cron

You can set up automated updates using cron to periodically refresh the blocklist and apply it to your UniFi controller.

### Basic Setup

1. **Create a wrapper script** (`update-blocklist.sh`):

```bash
#!/bin/bash
# Update UniFi region blocking blocklist

# Change to the directory containing the tools
cd /path/to/tae

# Set environment variables (or source from a file)
export UNIFI_HOST="https://10.5.22.1"
export UNIFI_USERNAME="programmatic"
export UNIFI_PASSWORD="your-password-here"

# Aggregate latest blocklist
./bin/aggregate --verbose 2>&1 | tee /var/log/unifi-blocklist-update.log

# Apply to UniFi (only updates if changes detected)
./bin/configure --insecure --verbose 2>&1 | tee -a /var/log/unifi-blocklist-update.log
```

2. **Make the script executable**:

```bash
chmod +x update-blocklist.sh
```

3. **Add to crontab**:

```bash
# Edit crontab
crontab -e

# Add entry to run daily at 3 AM
0 3 * * * /path/to/tae/update-blocklist.sh

# Or run twice daily (3 AM and 3 PM)
0 3,15 * * * /path/to/tae/update-blocklist.sh

# Or run weekly (every Monday at 3 AM)
0 3 * * 1 /path/to/tae/update-blocklist.sh
```

### Advanced Setup with Error Handling

For production use, create a more robust script:

```bash
#!/bin/bash
# update-blocklist.sh - Automated UniFi blocklist updater with error handling

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/unifi-blocklist-update.log"
LOCK_FILE="/tmp/unifi-blocklist-update.lock"

# Prevent concurrent runs
if [ -f "$LOCK_FILE" ]; then
    echo "$(date): Update already in progress, skipping..." >> "$LOG_FILE"
    exit 0
fi

trap "rm -f $LOCK_FILE" EXIT
touch "$LOCK_FILE"

cd "$SCRIPT_DIR"

# Source environment variables from a secure file
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
fi

echo "$(date): Starting blocklist update" >> "$LOG_FILE"

# Aggregate latest blocklist
if ./bin/aggregate --verbose >> "$LOG_FILE" 2>&1; then
    echo "$(date): Aggregation successful" >> "$LOG_FILE"
else
    echo "$(date): ERROR: Aggregation failed" >> "$LOG_FILE"
    exit 1
fi

# Apply to UniFi
if ./bin/configure --insecure --verbose >> "$LOG_FILE" 2>&1; then
    echo "$(date): Configuration update successful" >> "$LOG_FILE"
else
    echo "$(date): ERROR: Configuration update failed" >> "$LOG_FILE"
    exit 1
fi

echo "$(date): Update completed successfully" >> "$LOG_FILE"
```

### Secure Credential Storage

Store credentials in a `.env` file (add to `.gitignore`):

```bash
# .env file (chmod 600)
UNIFI_HOST="https://10.5.22.1"
UNIFI_USERNAME="programmatic"
UNIFI_PASSWORD="your-secure-password"
```

```bash
chmod 600 .env
```

### Log Rotation

Add log rotation to prevent logs from growing too large:

```bash
# /etc/logrotate.d/unifi-blocklist
/var/log/unifi-blocklist-update.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    missingok
}
```

### Testing Your Cron Job

Test the cron job manually first:

```bash
# Test the script
./update-blocklist.sh

# Check logs
tail -f /var/log/unifi-blocklist-update.log

# Test cron syntax (runs the job in 1 minute)
# Add this to crontab temporarily:
* * * * * /path/to/tae/update-blocklist.sh
```

### Recommended Schedule

- **Daily**: Good balance between freshness and resource usage
- **Twice daily**: If you need more frequent updates
- **Weekly**: If updates are infrequent and you want to minimize API calls

Note: The `configure` command is idempotent - it only makes changes when the blocklist actually differs from what's configured in UniFi, so running it frequently is safe.

## Output Format

### blocked_countries.txt

Simple text format, one ISO alpha-2 code per line:

```
AF
BY
CN
CU
IR
...
```

### blocked_countries.json

Full data with source provenance:

```json
{
  "timestamp": "2024-12-26T12:00:00Z",
  "total_codes": 42,
  "countries": [
    {
      "alpha2": "AF",
      "name": "Afghanistan",
      "sources": ["FATF Grey List", "Freedom House"],
      "raw_tokens": ["Afghanistan"]
    }
  ],
  "source_stats": {
    "EU Sanctions List": {
      "url": "https://www.sanctionsmap.eu/api/v1/sanctions",
      "fetched_at": "2024-12-26T12:00:00Z",
      "parse_status": "success",
      "raw_count": 19,
      "matched_count": 19
    }
  }
}
```

## Security Notes

- **Never commit credentials** - Use environment variables or gitignored config files
- **Use a dedicated local admin account** - Avoids MFA issues with UI.com accounts
- **TLS verification** - Only use `--insecure` flag with self-signed certificates you control
- **Rate limiting** - The tools include reasonable delays to avoid overloading APIs

## UniFi API Discovery

The Region Blocking feature in UniFi is located at:
**Settings → CyberSecure → Region Blocking**

For reliable endpoint discovery, use browser DevTools:

1. Log into UniFi Network UI
2. Open DevTools → Network tab
3. Navigate to Region Blocking
4. Toggle settings and observe API calls
5. Note the endpoint path and request format

Common endpoints to check:
- `v2/api/site/{site}/trafficrules`
- `api/s/{site}/rest/setting`

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Run tests: `go test ./...`
4. Submit a pull request

## Acknowledgments

- UniFi Community Wiki for API documentation
- Freedom House, RSF, OONI for open data on internet freedom
- Various government agencies for public sanctions data

