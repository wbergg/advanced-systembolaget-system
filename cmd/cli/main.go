package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"advanced-systembolaget-system/internal/systembolaget"
)

const configFile = "config.json"

func main() {
	getKey := flag.Bool("get-key", false, "Fetch a fresh API key and save to config.json")
	outFile := flag.String("o", "", "Output file (default: stdout)")

	flagValues := systembolaget.RegisterFlags(flag.CommandLine)

	flag.Parse()

	if *getKey {
		key, err := systembolaget.FetchAPIKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := systembolaget.SaveConfig(configFile, systembolaget.Config{APIKey: key}); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "API key saved to %s\n", configFile)
		return
	}

	cfg, err := systembolaget.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nRun with --get-key to fetch an API key\n", err)
		os.Exit(1)
	}

	query := systembolaget.BuildQueryFromFlags(flagValues)
	products, err := systembolaget.FetchAll(cfg.APIKey, query, func(page, totalPages, totalProducts int) {
		fmt.Fprintf(os.Stderr, "page %d/%d (%d products)\n", page, totalPages, totalProducts)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var w io.Writer = os.Stdout
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}

	enc := json.NewEncoder(w)
	for _, p := range products {
		enc.Encode(p)
	}
}
