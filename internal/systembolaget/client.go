package systembolaget

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	APIBaseURL       = "https://api-extern.systembolaget.se/sb-api-ecommerce/v1/productsearch/search"
	SystembolagetURL = "https://www.systembolaget.se"
)

var (
	appBundlePathRegex = regexp.MustCompile(`<script src="([^"]+_app-[^"]+\.js)"`)
	apiTokenRegex      = regexp.MustCompile(`NEXT_PUBLIC_API_KEY_APIM:"([^"]+)"`)
)

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid %s: %w", path, err)
	}
	if cfg.APIKey == "" {
		return Config{}, fmt.Errorf("api_key is empty in %s", path)
	}
	return cfg, nil
}

// LoadPrinterConfig reads only the printer section from config.json, ignoring
// whether api_key is set. Returns a zero-value PrinterConfig if the file is
// missing or unparseable.
func LoadPrinterConfig(path string) PrinterConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return PrinterConfig{}
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return PrinterConfig{}
	}
	return cfg.Printer
}

func SaveConfig(path string, cfg Config) error {
	existing := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &existing)
	}
	existing["api_key"] = cfg.APIKey

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}

func FetchAPIKey() (string, error) {
	fmt.Fprintf(os.Stderr, "Fetching %s...\n", SystembolagetURL)
	resp, err := http.Get(SystembolagetURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch homepage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("homepage returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read homepage: %w", err)
	}

	match := appBundlePathRegex.FindSubmatch(body)
	if match == nil {
		return "", fmt.Errorf("could not find _app bundle path in homepage HTML")
	}

	bundlePath := string(match[1])
	if strings.HasPrefix(bundlePath, "/") {
		bundlePath = SystembolagetURL + bundlePath
	}

	fmt.Fprintf(os.Stderr, "Fetching app bundle %s...\n", bundlePath)
	resp2, err := http.Get(bundlePath)
	if err != nil {
		return "", fmt.Errorf("failed to fetch app bundle: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		return "", fmt.Errorf("app bundle returned status %d", resp2.StatusCode)
	}

	jsBody, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read app bundle: %w", err)
	}

	keyMatch := apiTokenRegex.FindSubmatch(jsBody)
	if keyMatch == nil {
		return "", fmt.Errorf("could not find NEXT_PUBLIC_API_KEY_APIM in app bundle")
	}

	return string(keyMatch[1]), nil
}

// ProgressFunc is called after each page is fetched.
type ProgressFunc func(page, totalPages, totalProducts int)

func FetchAll(apiKey string, query url.Values, progress ProgressFunc) ([]Product, error) {
	var all []Product
	page := 1

	for {
		query.Set("page", strconv.Itoa(page))

		u := APIBaseURL + "?" + query.Encode()

		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)
		req.Header.Set("Origin", SystembolagetURL)
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed (page %d): %w", page, err)
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return nil, fmt.Errorf("API key invalid or expired")
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("API returned status %d on page %d", resp.StatusCode, page)
		}

		var sr SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response (page %d): %w", page, err)
		}
		resp.Body.Close()

		for _, rp := range sr.Products {
			if rp.IsRegionalRestricted || rp.IsDiscontinued || rp.IsCompletelyOutOfStock || rp.IsTemporaryOutOfStock || rp.RestrictedParcelQuantity > 0 {
				continue
			}
			p := rp.Product
			if len(rp.Images) > 0 {
				img := rp.Images[0]
				p.ImageURL = img.ImageURL + "_400." + img.FileType
			}
			all = append(all, p)
		}

		if progress != nil {
			progress(page, sr.Metadata.TotalPages, len(all))
		}

		if sr.Metadata.NextPage == -1 || sr.Metadata.NextPage <= page {
			break
		}
		page = sr.Metadata.NextPage
	}

	// Deduplicate by name (productNameBold + productNameThin), keeping newest launch date
	seen := make(map[string]int) // name key -> index in deduped
	var deduped []Product
	for _, p := range all {
		thin := ""
		if p.ProductNameThin != nil {
			thin = *p.ProductNameThin
		}
		key := p.ProductNameBold + "\x00" + thin
		if idx, ok := seen[key]; ok {
			// Keep the one with the newer launch date
			if p.ProductLaunchDate > deduped[idx].ProductLaunchDate {
				deduped[idx] = p
			}
		} else {
			seen[key] = len(deduped)
			deduped = append(deduped, p)
		}
	}

	return deduped, nil
}
