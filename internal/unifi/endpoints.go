package unifi

// KnownEndpoints contains documented UniFi API endpoints from ubntwiki.com.
var KnownEndpoints = []string{
	// Controller endpoints
	"api/self",
	"api/self/sites",
	"api/stat/sites",
	"api/stat/admin",

	// Site-scoped endpoints (will be prefixed with api/s/{site}/)
	"self",
	"stat/ccode",
	"stat/current-channel",
	"stat/health",
	"stat/sta",
	"stat/user",
	"stat/device-basic",
	"stat/device",
	"stat/sysinfo",
	"stat/event",
	"stat/alarm",
	"stat/session",
	"stat/stream",
	"stat/portforward",
	"stat/alluser",

	// REST endpoints
	"rest/setting",
	"rest/firewallrule",
	"rest/firewallgroup",
	"rest/routing",
	"rest/alarm",
	"rest/usergroup",
	"rest/wlangroup",
	"rest/wlanconf",
	"rest/tag",
	"rest/networkconf",
	"rest/portconf",
	"rest/user",
}

// V2Endpoints contains v2 API endpoints (newer features).
var V2Endpoints = []string{
	"v2/api/site/{site}/trafficrules",
	"v2/api/site/{site}/trafficroutes",
	"v2/api/site/{site}/security",
	"v2/api/site/{site}/threat-management",
	"v2/api/site/{site}/settings",
}

// RegionBlockingCandidates contains endpoints likely related to region blocking.
var RegionBlockingCandidates = []string{
	// v2 API candidates
	"v2/api/site/{site}/trafficrules",
	"v2/api/site/{site}/trafficroutes",
	"v2/api/site/{site}/security",
	"v2/api/site/{site}/security/region-blocking",
	"v2/api/site/{site}/security/geo-blocking",
	"v2/api/site/{site}/security/country-restriction",
	"v2/api/site/{site}/threat-management",
	"v2/api/site/{site}/threat-management/region-blocking",
	"v2/api/site/{site}/threat-management/geo-ip",
	"v2/api/site/{site}/cybersecure",
	"v2/api/site/{site}/cybersecure/region-blocking",
	"v2/api/site/{site}/geo-ip",
	"v2/api/site/{site}/geo-ip-filtering",
	"v2/api/site/{site}/country-blocking",
	"v2/api/site/{site}/country-restriction",

	// Site-scoped REST endpoints
	"rest/setting",
	"rest/setting/geo_ip",
	"rest/setting/geoip",
	"rest/setting/region_blocking",
	"rest/setting/country_restriction",
	"rest/setting/cybersecure",
	"rest/setting/threat_management",
	"rest/setting/security",
	"rest/firewallrule",
	"rest/firewallgroup",
	"rest/geoip",
	"rest/geo-ip",
	"rest/region-blocking",
	"rest/country-restriction",
	"rest/threatmanagement",
	"rest/threat-management",

	// Stat endpoints
	"stat/ccode",
	"stat/setting",
	"stat/security",
}

// DiscoveryWordlist contains words to combine for endpoint discovery.
var DiscoveryWordlist = []string{
	// Actions
	"get", "set", "list", "stat", "rest", "cmd", "config", "settings",

	// Security related
	"security", "firewall", "threat", "block", "restrict", "filter",
	"cybersecure", "cyber-secure", "cyber_secure",

	// Geographic
	"geo", "geoip", "geo-ip", "geo_ip",
	"region", "country", "countries", "territory",
	"blocking", "restriction", "filter",

	// UniFi specific
	"trafficrules", "traffic-rules", "traffic_rules",
	"trafficroutes", "traffic-routes", "traffic_routes",
	"threat-management", "threatmanagement", "threat_management",
	"region-blocking", "regionblocking", "region_blocking",
	"country-restriction", "countryrestriction", "country_restriction",
	"geo-ip-filtering", "geoipfiltering", "geo_ip_filtering",
}

