#!/bin/bash
# Find all PUT/POST requests in the parsed JSON

echo "=== Searching for PUT/POST in parsed JSON ==="
cat r.json | jq -r '.relevant_apis[] | select(.method == "PUT" or .method == "POST") | "\(.method) \(.url)"'

echo -e "\n=== Checking if we need to look at original HAR ==="
echo "The parsed JSON only shows GET requests."
echo "This suggests either:"
echo "1. The HAR file was captured before making changes"
echo "2. The region blocking endpoint uses different keywords"
echo ""
echo "Please re-capture the HAR file while:"
echo "- Toggling region blocking on/off"
echo "- Adding a country to the blocklist"
echo "- Removing a country from the blocklist"
echo "- Clicking Save/Apply button"
echo ""
echo "Or, if you have the original .har file, we can parse it directly."

