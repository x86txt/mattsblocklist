package scrapers

// DefaultRegistry creates a registry with all available scrapers.
func DefaultRegistry(client HTTPClient) *Registry {
	r := NewRegistry()

	// Censorship/Freedom indices
	r.Register(NewFreedomHouseScraper(client))
	r.Register(NewRSFScraper(client))
	r.Register(NewOONIScraper(client))

	// Government sanctions lists
	r.Register(NewEUSanctionsScraper(client))
	r.Register(NewUSOFACScraper(client))
	r.Register(NewUKSanctionsScraper(client))
	r.Register(NewUNSanctionsScraper(client))
	r.Register(NewFATFScraper(client))

	return r
}

