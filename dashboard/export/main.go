package main

import (
	"encoding/json"
	"fmt"
	"os"

	carbonmonitoring "github.com/grafana/sustainability-o11y/dashboard"
)

func main() {
	dashboard, err := carbonmonitoring.GetBuilder().Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build dashboard: %v\n", err)
		os.Exit(1)
	}

	b, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal dashboard: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile("carbon_emissions_report.json", b, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("wrote carbon_emissions_report.json")
}
