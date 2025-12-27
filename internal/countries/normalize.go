// Package countries provides country name normalization to ISO 3166-1 alpha-2 codes.
package countries

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Country represents a country with its code and names.
type Country struct {
	Alpha2 string `json:"alpha2"`
	Name   string `json:"name"`
}

// CountryList contains the result of aggregation.
type CountryList struct {
	Countries []CountryEntry `json:"countries"`
}

// CountryEntry represents a country with source provenance.
type CountryEntry struct {
	Alpha2    string   `json:"alpha2"`
	Name      string   `json:"name"`
	Sources   []string `json:"sources"`
	RawTokens []string `json:"raw_tokens,omitempty"`
}

// Normalizer handles country name normalization.
type Normalizer struct {
	nameToCode map[string]string
	codeToName map[string]string
}

// NewNormalizer creates a new country normalizer.
func NewNormalizer() *Normalizer {
	n := &Normalizer{
		nameToCode: make(map[string]string),
		codeToName: make(map[string]string),
	}

	// Build lookup maps
	for code, names := range countryNames {
		n.codeToName[code] = names[0] // First name is the primary name
		for _, name := range names {
			normalized := normalizeString(name)
			n.nameToCode[normalized] = code
		}
	}

	// Also add codes as self-referencing
	for code := range countryNames {
		n.nameToCode[strings.ToLower(code)] = code
	}

	return n
}

// Normalize converts a country name or code to ISO 3166-1 alpha-2.
func (n *Normalizer) Normalize(input string) (string, bool) {
	normalized := normalizeString(input)
	if code, ok := n.nameToCode[normalized]; ok {
		return code, true
	}

	// Try uppercase as-is (might be a code already)
	upper := strings.ToUpper(strings.TrimSpace(input))
	if len(upper) == 2 {
		if _, ok := n.codeToName[upper]; ok {
			return upper, true
		}
	}

	return "", false
}

// GetName returns the display name for a country code.
func (n *Normalizer) GetName(code string) string {
	if name, ok := n.codeToName[strings.ToUpper(code)]; ok {
		return name
	}
	return code
}

// IsValidCode checks if a code is a valid ISO 3166-1 alpha-2 code.
func (n *Normalizer) IsValidCode(code string) bool {
	_, ok := n.codeToName[strings.ToUpper(code)]
	return ok
}

// AllCodes returns all valid country codes.
func (n *Normalizer) AllCodes() []string {
	codes := make([]string, 0, len(n.codeToName))
	for code := range n.codeToName {
		codes = append(codes, code)
	}
	return codes
}

// normalizeString normalizes a string for comparison.
func normalizeString(s string) string {
	// Normalize unicode
	s = norm.NFKD.String(s)

	// Remove diacritics and convert to lowercase
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			if unicode.IsLetter(r) {
				result.WriteRune(unicode.ToLower(r))
			} else {
				result.WriteRune(r)
			}
		}
	}

	// Collapse whitespace
	return strings.Join(strings.Fields(result.String()), " ")
}

// countryNames maps ISO 3166-1 alpha-2 codes to common names and variations.
var countryNames = map[string][]string{
	"AF": {"Afghanistan"},
	"AL": {"Albania"},
	"DZ": {"Algeria"},
	"AS": {"American Samoa"},
	"AD": {"Andorra"},
	"AO": {"Angola"},
	"AI": {"Anguilla"},
	"AQ": {"Antarctica"},
	"AG": {"Antigua and Barbuda", "Antigua"},
	"AR": {"Argentina"},
	"AM": {"Armenia"},
	"AW": {"Aruba"},
	"AU": {"Australia"},
	"AT": {"Austria"},
	"AZ": {"Azerbaijan"},
	"BS": {"Bahamas", "The Bahamas"},
	"BH": {"Bahrain"},
	"BD": {"Bangladesh"},
	"BB": {"Barbados"},
	"BY": {"Belarus"},
	"BE": {"Belgium"},
	"BZ": {"Belize"},
	"BJ": {"Benin"},
	"BM": {"Bermuda"},
	"BT": {"Bhutan"},
	"BO": {"Bolivia", "Bolivia, Plurinational State of"},
	"BA": {"Bosnia and Herzegovina", "Bosnia"},
	"BW": {"Botswana"},
	"BR": {"Brazil"},
	"BN": {"Brunei", "Brunei Darussalam"},
	"BG": {"Bulgaria"},
	"BF": {"Burkina Faso"},
	"BI": {"Burundi"},
	"CV": {"Cabo Verde", "Cape Verde"},
	"KH": {"Cambodia"},
	"CM": {"Cameroon"},
	"CA": {"Canada"},
	"KY": {"Cayman Islands"},
	"CF": {"Central African Republic", "CAR"},
	"TD": {"Chad"},
	"CL": {"Chile"},
	"CN": {"China", "People's Republic of China", "PRC"},
	"CO": {"Colombia"},
	"KM": {"Comoros"},
	"CG": {"Congo", "Republic of the Congo", "Congo-Brazzaville"},
	"CD": {"Democratic Republic of the Congo", "DRC", "Congo-Kinshasa", "DR Congo"},
	"CR": {"Costa Rica"},
	"CI": {"Côte d'Ivoire", "Ivory Coast", "Cote d'Ivoire"},
	"HR": {"Croatia"},
	"CU": {"Cuba"},
	"CY": {"Cyprus"},
	"CZ": {"Czechia", "Czech Republic"},
	"DK": {"Denmark"},
	"DJ": {"Djibouti"},
	"DM": {"Dominica"},
	"DO": {"Dominican Republic"},
	"EC": {"Ecuador"},
	"EG": {"Egypt"},
	"SV": {"El Salvador"},
	"GQ": {"Equatorial Guinea"},
	"ER": {"Eritrea"},
	"EE": {"Estonia"},
	"SZ": {"Eswatini", "Swaziland"},
	"ET": {"Ethiopia"},
	"FJ": {"Fiji"},
	"FI": {"Finland"},
	"FR": {"France"},
	"GA": {"Gabon"},
	"GM": {"Gambia", "The Gambia"},
	"GE": {"Georgia"},
	"DE": {"Germany"},
	"GH": {"Ghana"},
	"GR": {"Greece"},
	"GD": {"Grenada"},
	"GT": {"Guatemala"},
	"GN": {"Guinea"},
	"GW": {"Guinea-Bissau"},
	"GY": {"Guyana"},
	"HT": {"Haiti"},
	"HN": {"Honduras"},
	"HK": {"Hong Kong", "Hong Kong SAR"},
	"HU": {"Hungary"},
	"IS": {"Iceland"},
	"IN": {"India"},
	"ID": {"Indonesia"},
	"IR": {"Iran", "Islamic Republic of Iran"},
	"IQ": {"Iraq"},
	"IE": {"Ireland"},
	"IL": {"Israel"},
	"IT": {"Italy"},
	"JM": {"Jamaica"},
	"JP": {"Japan"},
	"JO": {"Jordan"},
	"KZ": {"Kazakhstan"},
	"KE": {"Kenya"},
	"KI": {"Kiribati"},
	"KP": {"North Korea", "DPRK", "Democratic People's Republic of Korea"},
	"KR": {"South Korea", "Republic of Korea", "Korea"},
	"KW": {"Kuwait"},
	"KG": {"Kyrgyzstan"},
	"LA": {"Laos", "Lao People's Democratic Republic"},
	"LV": {"Latvia"},
	"LB": {"Lebanon"},
	"LS": {"Lesotho"},
	"LR": {"Liberia"},
	"LY": {"Libya"},
	"LI": {"Liechtenstein"},
	"LT": {"Lithuania"},
	"LU": {"Luxembourg"},
	"MO": {"Macao", "Macau"},
	"MG": {"Madagascar"},
	"MW": {"Malawi"},
	"MY": {"Malaysia"},
	"MV": {"Maldives"},
	"ML": {"Mali"},
	"MT": {"Malta"},
	"MH": {"Marshall Islands"},
	"MR": {"Mauritania"},
	"MU": {"Mauritius"},
	"MX": {"Mexico"},
	"FM": {"Micronesia", "Federated States of Micronesia"},
	"MD": {"Moldova", "Republic of Moldova"},
	"MC": {"Monaco"},
	"MN": {"Mongolia"},
	"ME": {"Montenegro"},
	"MA": {"Morocco"},
	"MZ": {"Mozambique"},
	"MM": {"Myanmar", "Burma"},
	"NA": {"Namibia"},
	"NR": {"Nauru"},
	"NP": {"Nepal"},
	"NL": {"Netherlands", "Holland"},
	"NZ": {"New Zealand"},
	"NI": {"Nicaragua"},
	"NE": {"Niger"},
	"NG": {"Nigeria"},
	"MK": {"North Macedonia", "Macedonia", "FYROM"},
	"NO": {"Norway"},
	"OM": {"Oman"},
	"PK": {"Pakistan"},
	"PW": {"Palau"},
	"PS": {"Palestine", "Palestinian Territories", "State of Palestine"},
	"PA": {"Panama"},
	"PG": {"Papua New Guinea"},
	"PY": {"Paraguay"},
	"PE": {"Peru"},
	"PH": {"Philippines"},
	"PL": {"Poland"},
	"PT": {"Portugal"},
	"PR": {"Puerto Rico"},
	"QA": {"Qatar"},
	"RO": {"Romania"},
	"RU": {"Russia", "Russian Federation"},
	"RW": {"Rwanda"},
	"KN": {"Saint Kitts and Nevis", "St. Kitts and Nevis"},
	"LC": {"Saint Lucia", "St. Lucia"},
	"VC": {"Saint Vincent and the Grenadines", "St. Vincent"},
	"WS": {"Samoa"},
	"SM": {"San Marino"},
	"ST": {"Sao Tome and Principe", "São Tomé and Príncipe"},
	"SA": {"Saudi Arabia"},
	"SN": {"Senegal"},
	"RS": {"Serbia"},
	"SC": {"Seychelles"},
	"SL": {"Sierra Leone"},
	"SG": {"Singapore"},
	"SK": {"Slovakia"},
	"SI": {"Slovenia"},
	"SB": {"Solomon Islands"},
	"SO": {"Somalia"},
	"ZA": {"South Africa"},
	"SS": {"South Sudan"},
	"ES": {"Spain"},
	"LK": {"Sri Lanka"},
	"SD": {"Sudan"},
	"SR": {"Suriname"},
	"SE": {"Sweden"},
	"CH": {"Switzerland"},
	"SY": {"Syria", "Syrian Arab Republic"},
	"TW": {"Taiwan", "Republic of China", "Chinese Taipei"},
	"TJ": {"Tajikistan"},
	"TZ": {"Tanzania", "United Republic of Tanzania"},
	"TH": {"Thailand"},
	"TL": {"Timor-Leste", "East Timor"},
	"TG": {"Togo"},
	"TO": {"Tonga"},
	"TT": {"Trinidad and Tobago"},
	"TN": {"Tunisia"},
	"TR": {"Turkey", "Türkiye"},
	"TM": {"Turkmenistan"},
	"TV": {"Tuvalu"},
	"UG": {"Uganda"},
	"UA": {"Ukraine"},
	"AE": {"United Arab Emirates", "UAE"},
	"GB": {"United Kingdom", "UK", "Great Britain", "Britain"},
	"US": {"United States", "USA", "United States of America", "America"},
	"UY": {"Uruguay"},
	"UZ": {"Uzbekistan"},
	"VU": {"Vanuatu"},
	"VA": {"Vatican City", "Holy See"},
	"VE": {"Venezuela", "Bolivarian Republic of Venezuela"},
	"VN": {"Vietnam", "Viet Nam"},
	"YE": {"Yemen"},
	"ZM": {"Zambia"},
	"ZW": {"Zimbabwe"},
}

