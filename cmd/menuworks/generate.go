package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/benworks/menuworks/discover"
	discoverwin "github.com/benworks/menuworks/discover/windows"
)

// runGenerate handles the "menuworks generate" subcommand.
// It is completely isolated from the TUI code path.
func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	output := fs.String("output", "config.yaml", "Output file path")
	sources := fs.String("sources", "", "Comma-separated list of sources (default: all available)")
	listSources := fs.Bool("list-sources", false, "List available sources and exit")
	dryRun := fs.Bool("dry-run", false, "Print config to stdout instead of writing a file")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: menuworks generate [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Discover installed applications and generate a config.yaml file.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	// Build registry with platform sources
	registry := discover.NewRegistry()
	discoverwin.RegisterAll(registry)

	// List sources mode
	if *listSources {
		allSources := registry.Sources()
		if len(allSources) == 0 {
			fmt.Println("No discovery sources available on this platform.")
			return
		}
		fmt.Println("Available discovery sources:")
		for _, s := range allSources {
			avail := "available"
			if !s.Available() {
				avail = "not found"
			}
			fmt.Printf("  %-20s [%s] (%s)\n", s.Name(), s.Category(), avail)
		}
		return
	}

	// Parse source filter
	var sourceNames []string
	if *sources != "" {
		for _, s := range strings.Split(*sources, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				sourceNames = append(sourceNames, s)
			}
		}
	}

	// Run discovery
	fmt.Fprintf(os.Stderr, "Discovering applications...\n")
	results, err := registry.DiscoverAll(sourceNames)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Report per-source results
	totalApps := 0
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: %s: %v\n", r.Source, r.Err)
		} else {
			fmt.Fprintf(os.Stderr, "  %s: found %d applications\n", r.Source, len(r.Apps))
			totalApps += len(r.Apps)
		}
	}

	if totalApps == 0 {
		fmt.Fprintf(os.Stderr, "No applications discovered.\n")
		return
	}

	// Collect, deduplicate, and generate
	apps := discover.CollectApps(results)
	apps = discover.DeduplicateApps(apps)
	fmt.Fprintf(os.Stderr, "Total: %d unique applications\n", len(apps))

	if *dryRun {
		if err := discover.WriteConfigStdout(apps); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating config: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Write to file
	if err := discover.WriteConfig(apps, *output); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Config written to: %s\n", *output)
}
