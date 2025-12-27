# UniFi Region Blocking API Discovery Guide

This guide will help you capture the exact API calls made when interacting with the Region Blocking feature in UniFi.

## Prerequisites

- Access to UniFi controller at `https://10.5.22.1`
- Credentials:
  - Username: `programmatic`
  - Password: `TMjlbL8Xb*WhwJhsN55CHW7I!eHzCOI#C$I@jb@iFTcIz8vNWqLEj5sEb*FPB2E1`
- Modern web browser (Chrome, Firefox, Edge, Safari)

## Step-by-Step Instructions

### 1. Open Browser and Navigate to UniFi

1. Open your web browser
2. Navigate to `https://10.5.22.1`
3. **Accept the security warning** (self-signed certificate)
4. You should see the UniFi login page

### 2. Open Developer Tools

**Chrome/Edge:**
- Press `F12` or `Ctrl+Shift+I` (Windows/Linux) or `Cmd+Option+I` (Mac)
- Or: Right-click → Inspect

**Firefox:**
- Press `F12` or `Ctrl+Shift+I` (Windows/Linux) or `Cmd+Option+I` (Mac)

**Safari:**
- Enable Developer menu: Safari → Preferences → Advanced → "Show Develop menu"
- Press `Cmd+Option+I`

### 3. Open Network Tab

1. In Developer Tools, click on the **"Network"** tab
2. Make sure **"Preserve log"** is checked (important!)
3. Filter by **"Fetch/XHR"** or **"All"** to see API requests

### 4. Log In

1. Enter username: `programmatic`
2. Enter password: `TMjlbL8Xb*WhwJhsN55CHW7I!eHzCOI#C$I@jb@iFTcIz8vNWqLEj5sEb*FPB2E1`
3. Click "Sign In"
4. **Watch the Network tab** - you should see API calls appear

### 5. Navigate to Region Blocking Settings

Once logged in, navigate to:
1. Click on **"Settings"** (usually in the left sidebar or top navigation)
2. Look for **"CyberSecure"** or **"Security"** section
3. Click on **"CyberSecure"** or **"Security"**
4. Find and click on **"Region Blocking"**

**Watch the Network tab** as the page loads - there should be API calls to fetch the current settings.

### 6. Capture API Calls

Now perform these actions and observe the API calls:

#### A. Initial Page Load
- When the Region Blocking page loads, look for:
  - GET requests that might fetch current settings
  - Look for URLs containing: `setting`, `geo`, `region`, `country`, `block`, `cybersecure`

#### B. Toggle Enable/Disable
1. Toggle the Region Blocking switch (on/off)
2. Look for a PUT or POST request
3. Click on the request in the Network tab
4. Note:
   - **Request URL** (full path)
   - **Request Method** (GET, POST, PUT, DELETE)
   - **Request Headers** (especially `X-Csrf-Token` if present)
   - **Request Payload/Body** (JSON content)
   - **Response** (JSON response)

#### C. Add/Remove a Country
1. Try to add a country to the blocklist (or remove one)
2. Watch for API calls in the Network tab
3. Capture the same information as above

#### D. Save Changes
1. If there's a "Save" or "Apply" button, click it
2. Capture any API calls that are made

### 7. Document the API Calls

For each relevant API call, record:

```markdown
## API Call: [Description]

**URL:** `/proxy/network/api/s/default/...`
**Method:** `PUT`
**Headers:**
- `Content-Type: application/json`
- `X-Csrf-Token: [token]` (if present)
- `Cookie: [session cookie]`

**Request Body:**
```json
{
  "key": "geo_ip_filtering",
  "enabled": true,
  "countries": ["CN", "RU", "KP"]
}
```

**Response:**
```json
{
  "meta": {"rc": "ok"},
  "data": [...]
}
```
```

### 8. Key Information to Capture

Focus on finding:

1. **Endpoint Path**
   - Full URL path (e.g., `/proxy/network/api/s/default/rest/setting/geo_ip_filtering`)
   - Site identifier (usually `default`)

2. **HTTP Method**
   - GET (fetch current settings)
   - PUT (update settings)
   - POST (create/apply settings)

3. **Request Format**
   - JSON structure
   - Required fields
   - Country code format (ISO alpha-2? array? object?)

4. **Authentication**
   - How cookies/session is maintained
   - CSRF token usage

5. **Response Format**
   - JSON structure
   - Where country codes are stored
   - Enabled/disabled flag location

## Expected Endpoints

Based on UniFi API patterns, you might see:

- `/proxy/network/api/s/default/rest/setting` - General settings
- `/proxy/network/api/s/default/rest/setting/[key]/[id]` - Specific setting
- `/proxy/network/v2/api/site/default/trafficrules` - Traffic rules (v2 API)
- `/proxy/network/api/s/default/stat/ccode` - Country codes list

Look for settings with keys like:
- `geo_ip_filtering`
- `geoip_filtering`
- `country_restriction`
- `region_blocking`
- `threat_management`

## Creating Documentation File

After capturing the information, create a file called `unifi-api-discovery.md` in this directory with your findings.

**Template:**

```markdown
# UniFi Region Blocking API Discovery

**Date:** [Date]
**UniFi Version:** [if visible]
**Device:** UCG Fiber

## Endpoint for Reading Current Settings

**URL:** 
**Method:** GET
**Headers:**
**Response Example:**

## Endpoint for Updating Settings

**URL:**
**Method:** PUT/POST
**Headers:**
**Request Format:**
**Response Format:**

## Country Code Format

[How are country codes represented? Array? Object? ISO format?]

## Notes

[Any additional observations]
```

## Next Steps

Once you have documented the API calls:

1. Update `cmd/configure/main.go` with the discovered endpoint
2. Update the request/response parsing logic
3. Test with `--dry-run` flag
4. Apply to your UniFi device

## Troubleshooting

**If you don't see API calls:**
- Make sure "Preserve log" is enabled
- Clear the filter (show "All" requests)
- Look for requests to your IP (10.5.22.1)
- Try refreshing the page

**If login doesn't work:**
- Check credentials are correct
- Try a different browser
- Clear browser cache/cookies

**If you can't find Region Blocking:**
- Check if the feature is available on your device
- Look in different menu locations (Security, Firewall, etc.)
- Check UniFi documentation for your device model

