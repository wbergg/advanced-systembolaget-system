package systembolaget

import (
	"flag"
	"net/url"
)

type ParamMapping struct {
	Flag     string
	APIParam string
	Help     string
}

var ParamMappings = []ParamMapping{
	{"pris-fran", "price.min", "Minimum price (SEK)"},
	{"pris-till", "price.max", "Maximum price (SEK)"},
	{"alkoholhalt-fran", "alcoholPercentage.min", "Minimum ABV%"},
	{"alkoholhalt-till", "alcoholPercentage.max", "Maximum ABV%"},
	{"kategori", "categoryLevel1", "Category (e.g. Öl, Vin, Sprit)"},
	{"typ", "categoryLevel2", "Type (e.g. Ljus lager, Röda)"},
	{"stil", "categoryLevel3", "Style"},
	{"forpackning", "packagingLevel1", "Packaging (e.g. Burk, Flaska)"},
	{"land", "country", "Country of origin"},
	{"volym-fran", "volume.min", "Minimum volume (ml)"},
	{"volym-till", "volume.max", "Maximum volume (ml)"},
	{"sortera-pa", "sortBy", "Sort by (Price, Name, Volume, Score, ProductLaunchDate, Vintage)"},
	{"i-riktning", "sortDirection", "Sort direction (Ascending, Descending)"},
	{"q", "textQuery", "Free text search"},
	{"producent", "producerName", "Producer name"},
	{"argang", "vintage", "Vintage year"},
	{"sortiment", "assortmentText", "Assortment (e.g. Fast sortiment)"},
	{"nyhet", "isNews", "Only new products (true/false)"},
}

// RegisterFlags registers CLI flags from the param mappings table.
func RegisterFlags(fs *flag.FlagSet) map[string]*string {
	flags := make(map[string]*string, len(ParamMappings))
	for _, m := range ParamMappings {
		flags[m.APIParam] = fs.String(m.Flag, "", m.Help)
	}
	return flags
}

// BuildQueryFromFlags builds API query params from CLI flag values.
func BuildQueryFromFlags(flagValues map[string]*string) url.Values {
	q := url.Values{}
	q.Set("size", "30")
	for apiParam, val := range flagValues {
		if *val != "" {
			q.Set(apiParam, *val)
		}
	}
	return q
}

// BuildQueryFromMap builds API query params from a Swedish-name → value map.
// Used by the API server where filters come as a JSON object.
func BuildQueryFromMap(filters map[string]string) url.Values {
	lookup := make(map[string]string, len(ParamMappings))
	for _, m := range ParamMappings {
		lookup[m.Flag] = m.APIParam
	}

	q := url.Values{}
	q.Set("size", "30")
	for flag, val := range filters {
		if apiParam, ok := lookup[flag]; ok && val != "" {
			q.Set(apiParam, val)
		}
	}
	return q
}
