# UniFi Region Blocking API Discovery Results

**Date:** 2024-12-26  
**Device:** UCG Fiber  
**Controller:** `https://10.5.22.1`

---

## Endpoint Discovered ✅

### POST Endpoint for Updating Region Blocking

**Full URL:** `https://10.5.22.1/proxy/network/api/s/default/set/setting/usg`  
**Relative Path:** `/proxy/network/api/s/default/set/setting/usg`  
**HTTP Method:** `POST`  
**Status Code:** `200` (success)

---

## Request Format

### Headers

```
Content-Type: application/json
X-Csrf-Token: [token from login]
Cookie: [session cookie from login]
```

### Request Body Structure

The request body contains the entire `usg` setting object, including region blocking fields:

```json
{
  "key": "usg",
  "_id": "5c653f4446b41307c37379fb",
  "site_id": "5c653f3646b41307c37379ed",
  "geo_ip_filtering_enabled": true,
  "geo_ip_filtering_countries": "AF,BD,PK,IN,RU,NP,CN,RO,IR,BR,AD",
  "geo_ip_filtering_block": "block",
  "geo_ip_filtering_traffic_direction": "both",
  // ... many other usg settings fields
}
```

### Key Fields for Region Blocking

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `geo_ip_filtering_enabled` | boolean | Enable/disable region blocking | `true` or `false` |
| `geo_ip_filtering_countries` | string | Comma-separated country codes (ISO 3166-1 alpha-2) | `"AF,BD,PK,IN,RU,NP,CN,RO,IR,BR,AD"` |
| `geo_ip_filtering_block` | string | Block action (usually `"block"`) | `"block"` |
| `geo_ip_filtering_traffic_direction` | string | Traffic direction to block | `"both"`, `"inbound"`, or `"outbound"` |
| `key` | string | Setting key (always `"usg"`) | `"usg"` |
| `_id` | string | Setting ID (required) | `"5c653f4446b41307c37379fb"` |

### Country Code Format

- **Format:** Comma-separated string (NOT an array)
- **Code Format:** ISO 3166-1 alpha-2 (2 letters)
- **Example:** `"AF,BD,PK,IN,RU"` (not `["AF","BD","PK"]`)

### Example Request Body (Minimal for Region Blocking)

When updating only region blocking, you still need to include the full setting object. The minimum required fields appear to be:

```json
{
  "key": "usg",
  "_id": "5c653f4446b41307c37379fb",
  "geo_ip_filtering_enabled": true,
  "geo_ip_filtering_countries": "AF,BD,PK,IN,RU,NP,CN,RO,IR,BR,AD",
  "geo_ip_filtering_block": "block",
  "geo_ip_filtering_traffic_direction": "both"
}
```

**Note:** In practice, UniFi may require the full setting object. Best practice: GET the current setting first, modify the geo-ip fields, then POST the complete object back.

---

## GET Endpoint for Reading Current Settings

To read the current region blocking configuration:

**Full URL:** `https://10.5.22.1/proxy/network/api/s/default/rest/setting/usg/[id]`  
**Or:** `https://10.5.22.1/proxy/network/api/s/default/get/setting/usg`  
**HTTP Method:** `GET`

Response contains the same structure as the POST request body.

---

## Implementation Notes

1. **Setting Key:** The setting is stored under the `usg` key, not a separate `geo-ip-filtering` key
2. **Full Object Required:** You must POST the complete setting object, not just the geo-ip fields
3. **Country Format:** Countries are a comma-separated string, not an array
4. **Setting ID Required:** The `_id` field is required in the POST request
5. **CSRF Token:** Required in headers for POST requests
6. **Site ID:** Included in the request body (usually `"default"` site)

---

## Next Steps

1. ✅ Update `cmd/configure/main.go` with this endpoint structure
2. ✅ Implement GET to fetch current settings
3. ✅ Implement POST to update settings with country list
4. ✅ Handle country code conversion (array → comma-separated string)
5. ✅ Test with dry-run mode
6. ✅ Apply to UniFi device

