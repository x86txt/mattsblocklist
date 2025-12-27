#!/bin/bash
# Analyze HAR file for ALL API endpoints, especially PUT/POST

echo "=== All PUT/POST Requests ==="
cat r.json | jq -r '.relevant_apis[] | select(.method == "PUT" or .method == "POST") | "\(.method) \(.url)\nBody: \(.request_body // "none")\n---"'

echo -e "\n=== All v2/api/site endpoints ==="
cat r.json | jq -r '.relevant_apis[] | select(.url | contains("v2/api/site")) | "\(.method) \(.url)"'

echo -e "\n=== Settings endpoints ==="
cat r.json | jq -r '.relevant_apis[] | select(.url | contains("settings")) | "\(.method) \(.url)\nResponse: \(.response_body[:300] // "none")\n---"'

echo -e "\n=== Full details of first few endpoints ==="
cat r.json | jq '.relevant_apis[0:3]'

