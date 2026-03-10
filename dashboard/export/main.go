// Converts the generated carbon monitoring dashboard JSON into a
// Grafana dashboard library template suitable for sharing externally.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	carbonmonitoring "github.com/grafana/sustainability-o11y/dashboard"
)

type datasourceInput struct {
	VarName    string `json:"-"`
	Name       string `json:"name"`
	Label      string `json:"label"`
	Type       string `json:"type"`
	PluginID   string `json:"pluginId"`
	PluginName string `json:"pluginName"`
}

type requirement struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Ordered list of datasource variable declarations.
// Order here determines the order of entries in __inputs.
var datasourceInputs = []datasourceInput{
	{VarName: "bigquery_ds", Label: "BigQuery Datasource", PluginID: "grafana-bigquery-datasource", PluginName: "Google BigQuery", Type: "datasource"},
	{VarName: "athena_ds", Label: "Athena Datasource", PluginID: "grafana-athena-datasource", PluginName: "Amazon Athena", Type: "datasource"},
	{VarName: "infinity_ds", Label: "Infinity Datasource", PluginID: "yesoreyeram-infinity-datasource", PluginName: "Infinity", Type: "datasource"},
}

// buildInputs returns __inputs entries only for datasource variables
// that were actually found in the dashboard JSON.
func buildInputs(usedVars map[string]bool) []datasourceInput {
	var inputs []datasourceInput
	for _, d := range datasourceInputs {
		if !usedVars[d.VarName] {
			continue
		}
		entry := d
		entry.Name = d.VarName
		inputs = append(inputs, entry)
	}
	return inputs
}

// panelNameOverrides maps panel type IDs to display names that can't be
// derived by simple title-casing.
var panelNameOverrides = map[string]string{
	"barchart": "Bar chart",
	"piechart": "Pie chart",
}

func panelDisplayName(panelType string) string {
	if name, ok := panelNameOverrides[panelType]; ok {
		return name
	}
	if len(panelType) == 0 {
		return panelType
	}
	return strings.ToUpper(panelType[:1]) + panelType[1:]
}

// buildRequires scans the dashboard JSON for panel types and datasource types
// and returns a sorted __requires list.
func buildRequires(dashboard map[string]any) []requirement {
	panelTypes := map[string]bool{}
	dsTypes := map[string]bool{}
	collectTypes(dashboard, panelTypes, dsTypes)

	var reqs []requirement
	for pt := range panelTypes {
		if pt == "row" {
			continue
		}
		reqs = append(reqs, requirement{Type: "panel", ID: pt, Name: panelDisplayName(pt)})
	}

	reqs = append(reqs, requirement{Type: "grafana", ID: "grafana", Name: "Grafana"})

	seen := map[string]bool{}
	for _, d := range datasourceInputs {
		if dsTypes[d.PluginID] && !seen[d.PluginID] {
			reqs = append(reqs, requirement{
				Type: "datasource",
				ID:   d.PluginID,
				Name: d.PluginName,
			})
			seen[d.PluginID] = true
		}
	}

	sort.Slice(reqs, func(i, j int) bool {
		if reqs[i].Type != reqs[j].Type {
			return reqs[i].Type < reqs[j].Type
		}
		return reqs[i].ID < reqs[j].ID
	})
	return reqs
}

func collectTypes(obj any, panelTypes, dsTypes map[string]bool) {
	switch v := obj.(type) {
	case map[string]any:
		if t, ok := v["type"].(string); ok {
			if _, hasGridPos := v["gridPos"]; hasGridPos {
				panelTypes[t] = true
			}
		}
		if ds, ok := v["datasource"].(map[string]any); ok {
			if t, ok := ds["type"].(string); ok && t != "datasource" {
				dsTypes[t] = true
			}
		}
		for _, val := range v {
			collectTypes(val, panelTypes, dsTypes)
		}
	case []any:
		for _, item := range v {
			collectTypes(item, panelTypes, dsTypes)
		}
	}
}

// collectUsedVars walks the JSON tree and finds which datasource template
// variables (e.g. "${bigquery_ds}") are referenced in datasource uid fields.
func collectUsedVars(obj any, usedVars map[string]bool) {
	switch v := obj.(type) {
	case map[string]any:
		if uid, ok := v["uid"].(string); ok {
			if strings.HasPrefix(uid, "${") && strings.HasSuffix(uid, "}") {
				varName := uid[2 : len(uid)-1]
				usedVars[varName] = true
			}
		}
		for _, val := range v {
			collectUsedVars(val, usedVars)
		}
	case []any:
		for _, item := range v {
			collectUsedVars(item, usedVars)
		}
	}
}

// marshalOrderedJSON produces JSON with __inputs, __elements, and __requires as
// the first keys, followed by all remaining keys in sorted order.
func marshalOrderedJSON(dashboard map[string]any) ([]byte, error) {
	priorityKeys := []string{"__inputs", "__elements", "__requires"}

	skip := map[string]bool{"__inputs": true, "__elements": true, "__requires": true}
	remaining := make([]string, 0, len(dashboard))
	for k := range dashboard {
		if !skip[k] {
			remaining = append(remaining, k)
		}
	}
	sort.Strings(remaining)

	allKeys := append(priorityKeys, remaining...)

	buf := &bytes.Buffer{}
	buf.WriteString("{\n")
	for i, key := range allKeys {
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, fmt.Errorf("marshaling key %q: %w", key, err)
		}
		valJSON, err := json.MarshalIndent(dashboard[key], "  ", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshaling value for key %q: %w", key, err)
		}
		fmt.Fprintf(buf, "  %s: %s", keyJSON, valJSON)
		if i < len(allKeys)-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.WriteString("}\n")
	return buf.Bytes(), nil
}

func main() {
	built, err := carbonmonitoring.GetBuilder().Build()
	if err != nil {
		log.Fatalf("building dashboard: %v", err)
	}

	data, err := json.Marshal(built)
	if err != nil {
		log.Fatalf("marshaling dashboard: %v", err)
	}

	var dashboard map[string]any
	if err := json.Unmarshal(data, &dashboard); err != nil {
		log.Fatalf("parsing JSON: %v", err)
	}

	dashboard["__requires"] = buildRequires(dashboard)

	usedVars := make(map[string]bool)
	collectUsedVars(dashboard, usedVars)

	dashboard["__inputs"] = buildInputs(usedVars)
	dashboard["__elements"] = map[string]any{}

	delete(dashboard, "uid")
	delete(dashboard, "id")
	delete(dashboard, "version")
	dashboard["editable"] = true

	output, err := marshalOrderedJSON(dashboard)
	if err != nil {
		log.Fatalf("marshaling JSON: %v", err)
	}

	outputPath := "carbon_emissions_report.json"
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		log.Fatalf("writing %s: %v", outputPath, err)
	}

	log.Printf("Success generating carbon emissions report dashboard template:\n   - JSON: %s", outputPath)
}
