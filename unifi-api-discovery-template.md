# UniFi Region Blocking API Discovery Results

**Date:** _________________  
**UniFi Version:** _________________  
**Device:** UCG Fiber  
**Controller URL:** `https://10.5.22.1`

---

## 1. Endpoint for Reading Current Region Blocking Settings

**Full URL:** `https://10.5.22.1/proxy/network/...`  
**Relative Path:** `/proxy/network/...`  
**HTTP Method:** `GET` / `POST` / `PUT` / `DELETE`

**Request Headers:**
```
Content-Type: application/json
X-Csrf-Token: [if present]
Cookie: [session cookie]
```

**Request Body (if any):**
```json

```

**Response Example:**
```json

```

**Notes:**

---

## 2. Endpoint for Updating Region Blocking Settings

**Full URL:** `https://10.5.22.1/proxy/network/...`  
**Relative Path:** `/proxy/network/...`  
**HTTP Method:** `GET` / `POST` / `PUT` / `DELETE`

**Request Headers:**
```
Content-Type: application/json
X-Csrf-Token: [if present]
Cookie: [session cookie]
```

**Request Body Example (Enable with countries):**
```json

```

**Request Body Example (Disable):**
```json

```

**Response Example:**
```json

```

**Notes:**

---

## 3. Country Code Format

**Format:** Array / Object / String / Other

**Example:**
```json

```

**Country codes are:**
- [ ] ISO 3166-1 alpha-2 (2 letters: "US", "CN", etc.)
- [ ] ISO 3166-1 alpha-3 (3 letters: "USA", "CHN", etc.)
- [ ] Numeric codes
- [ ] Other: _________________

---

## 4. Setting Key/Identifier

If using `/rest/setting` endpoint, what is the setting key?

**Key Name:** `_________________`

**How to find:** Look for a `key` field in the GET response, or check the URL path if it's `/rest/setting/[key]/[id]`

---

## 5. Enable/Disable Flag

**Field Name:** `_________________`

**Format:**
- [ ] Boolean (`true`/`false`)
- [ ] String (`"enabled"`/`"disabled"`)
- [ ] Number (`1`/`0`)
- [ ] Other: _________________

---

## 6. Additional Observations

- How is the site identified? (usually "default" in the URL)
- Is a CSRF token required?
- Any special authentication headers?
- Are there any other related endpoints (e.g., getting country list)?
- Any rate limiting or special considerations?

---

## 7. Screenshots or Network Tab Export

[Optional: Attach screenshots or export Network tab data]

---

## Next Steps

Once this template is filled out:
1. Save this file (rename from `-template.md`)
2. Update `cmd/configure/main.go` with the discovered endpoint
3. Test with `--dry-run` flag
4. Apply to your UniFi device

