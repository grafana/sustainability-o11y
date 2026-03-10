# Carbon Emissions Report Dashboard Template

Grafana dashboard template for the Carbon Emissions Report formatted to share in the [Grafana dashboard library](https://grafana.com/grafana/dashboards/).

| File | Description |
|---|---|
| `carbon_emissions_report.json` | Generated dashboard template |
| `main.go` | Tool that generates the template |

## How to generate the template

From the `dashboard/` directory:

```bash
go run ./export
```

This tool:
* builds the dashboard from the Go source code
* writes the template to `carbon_emissions_report.json`

It applies the following transformations:

1. Adds `__inputs` declarations so Grafana prompts for datasources on import
2. Adds `__requires` declaring panel and datasource plugin dependencies
3. Strips the dashboard `uid` (Grafana assigns a fresh one on import)
4. Sets `editable` to `true`
