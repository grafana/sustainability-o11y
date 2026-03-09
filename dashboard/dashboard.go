package carbonmonitoring

import (
	"encoding/json"

	"github.com/grafana/grafana-foundation-sdk/go/athena"
	"github.com/grafana/grafana-foundation-sdk/go/barchart"
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/geomap"
	"github.com/grafana/grafana-foundation-sdk/go/piechart"
	"github.com/grafana/grafana-foundation-sdk/go/stat"
	"github.com/grafana/grafana-foundation-sdk/go/table"
	"github.com/grafana/grafana-foundation-sdk/go/text"
)

var (
	title = "Carbon Emissions Report"

	bigQueryDSRef = dashboard.DataSourceRef{Type: cog.ToPtr("grafana-bigquery-datasource"), Uid: cog.ToPtr("${bigquery_ds}")}
	athenaDSRef   = dashboard.DataSourceRef{Type: cog.ToPtr("grafana-athena-datasource"), Uid: cog.ToPtr("${athena_ds}")}

	targetRegionCoordinates = json.RawMessage(targetRegionCoordinatesJSON)
	awsRegionCoordinates    = json.RawMessage(awsRegionCoordinatesJSON)
)

// GetBuilder returns the open-source Carbon Emissions Report dashboard.
// Datasource UIDs and BigQuery project ID are template variables, allowing
// users to supply their own connections.
func GetBuilder() *dashboard.DashboardBuilder {
	return dashboard.NewDashboardBuilder(title).
		Title(title).
		Tooltip(0).
		Readonly().
		Time("now-365d", "now").
		Timezone("utc").
		Variables([]cog.Builder[dashboard.VariableModel]{
			dashboard.NewTextBoxVariableBuilder("bigquery_project").
				Label("BigQuery Project ID").
				DefaultValue(dashboard.StringOrMap{String: cog.ToPtr("")}),
			dashboard.NewTextBoxVariableBuilder("gcp_dataset").
				Label("GCP BigQuery Dataset").
				DefaultValue(dashboard.StringOrMap{String: cog.ToPtr("")}),
			dashboard.NewTextBoxVariableBuilder("azure_dataset").
				Label("Azure BigQuery Dataset").
				DefaultValue(dashboard.StringOrMap{String: cog.ToPtr("")}),
			dashboard.NewTextBoxVariableBuilder("athena_database").
				Label("Athena Database").
				DefaultValue(dashboard.StringOrMap{String: cog.ToPtr("")}),
			dashboard.NewTextBoxVariableBuilder("athena_table").
				Label("Athena Table").
				DefaultValue(dashboard.StringOrMap{String: cog.ToPtr("")}),
			dashboard.NewDatasourceVariableBuilder("bigquery_ds").
				Name("bigquery_ds").
				Label("BigQuery Datasource").
				Type("grafana-bigquery-datasource"),
			dashboard.NewDatasourceVariableBuilder("athena_ds").
				Name("athena_ds").
				Label("Athena Datasource").
				Type("grafana-athena-datasource"),
			dashboard.NewDatasourceVariableBuilder("infinity_ds").
				Name("infinity_ds").
				Label("Infinity Datasource").
				Type("yesoreyeram-infinity-datasource"),
		}).
		WithPanel(text.NewPanelBuilder().
			Height(3).
			Span(24).
			Mode("markdown").
			Transparent(true).
			Code(text.NewCodeOptionsBuilder().
				Language("plaintext")).
			Content(`
# Carbon Emissions Report

This dashboard tracks carbon emissions across various sources of business activity. The data sources surface carbon emissions from Cloud Service Providers (GCP, AWS, Azure). [Learn more](https://github.com/grafana/sustainability-o11y).
`)).
		WithPanel(buildTotalCarbonPanel()).
		WithPanel(buildEmissionsBySourcePanel()).
		WithPanel(buildTripsAroundWorldPanel()).
		// Section: Total Emissions Breakdown by Scope
		WithPanel(text.NewPanelBuilder().
			Height(2).
			Span(24).
			Mode("markdown").
			Transparent(true).
			Code(text.NewCodeOptionsBuilder().
				Language("plaintext")).
			Content(`
This dashboard displays carbon emissions in [metric tons of carbon dioxide equivalent (MtCO₂e)](https://www.ipcc.ch/sr15/chapter/glossary), an industry-standard measurement for all greenhouse gases.
`)).
		WithPanel(text.NewPanelBuilder().
			Height(5).
			Span(24).
			Mode("markdown").
			Transparent(true).
			Code(text.NewCodeOptionsBuilder().
				Language("plaintext")).
			Content(`
#### Carbon Emissions Breakdown

Carbon emissions are tracked according to the [Greenhouse Gas Protocol](https://ghgprotocol.org/corporate-standard), the most widely used and recognised greenhouse gas accounting standard, which categorizes emissions into three scopes:
- **Scope 1:** All direct emissions from owned or controlled sources.
- **Scope 2:** Indirect emissions from purchased energy generation. For example, these can either use the [Location-Based Method or Market-Based Method](https://github.com/grafana/sustainability-o11y#location-based-method-lbm-vs-market-based-method-mbm).
- **Scope 3:** All other indirect emissions in the value chain (upstream and downstream), including emissions from workloads running on CSPs and business travel.
`)).
		WithPanel(piechart.NewPanelBuilder().
			Title("Total Emissions by Scope").
			Datasource(mixedDSRef).
			WithTarget(dashboardSource().
				RefId("Scope 1").
				PanelId(15).
				WithTransforms(true),
			).
			WithTarget(dashboardSource().
				RefId("Scope 2").
				PanelId(14).
				WithTransforms(true),
			).
			WithTarget(dashboardSource().
				RefId("Scope 3").
				PanelId(13).
				WithTransforms(true),
			).
			Transparent(true).
			Height(12).
			Span(6).
			Unit(metricTonsCO2e).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("palette-classic").
				FixedColor("text")).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "scope1"}, []dashboard.DynamicConfigValue{{Id: "displayName", Value: "Scope 1"}}).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "scope2"}, []dashboard.DynamicConfigValue{{Id: "displayName", Value: "Scope 2"}}).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "scope3"}, []dashboard.DynamicConfigValue{{Id: "displayName", Value: "Scope 3"}}).
			PieType("pie").
			DisplayLabels([]piechart.PieChartLabels{"percent", "name"}).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			ReduceOptions(common.NewReduceDataOptionsBuilder().
				Values(false).
				Calcs([]string{"sum"})).
			Legend(piechart.NewPieChartLegendOptionsBuilder().
				Values([]piechart.PieChartLegendValues{"percent", "value"}).
				DisplayMode("table").
				Placement("bottom").
				ShowLegend(true).
				SortBy("Value").
				SortDesc(true)).
			Orientation("").
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false))).
		WithPanel(buildScope3Panel()).
		WithPanel(barchart.NewPanelBuilder().
			Id(14).
			Datasource(mixedDSRef).
			WithTarget(bigqueryRaw(scope2Query).
				Project("${bigquery_project}").
				Format(0).
				RefId("Scope 2").
				Datasource(bigQueryDSRef)).
			WithTarget(athena.NewDataqueryBuilder().
				Format(1).
				ConnectionArgs(athena.NewConnectionArgsBuilder()).
				RefId("AWS Scope 2").
				RawSQL(awsScope2TotalQuery).
				Datasource(athenaDSRef)).
			WithTarget(bigqueryRaw(azureScope2TotalQuery).
				Project("${bigquery_project}").
				Format(0).
				RefId("Azure Scope 2").
				Datasource(bigQueryDSRef)).
			Title("Scope 2").
			Transparent(true).
			Height(12).
			Span(6).
			Transformations([]dashboard.DataTransformerConfig{
				{Id: "convertFieldType", Options: map[string]any{"conversions": []any{map[string]any{"destinationType": "number", "targetField": "Scope 2 scope2"}, map[string]any{"destinationType": "number", "targetField": "AWS Scope 2 scope2"}, map[string]any{"destinationType": "number", "targetField": "Azure Scope 2 scope2"}}}},
				{Id: "merge"},
			}).
			Unit(metricTonsCO2e).
			Max(6000).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"},
					{Value: cog.ToPtr[float64](80), Color: "red"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("fixed").
				FixedColor("light-blue")).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "scope2"}, []dashboard.DynamicConfigValue{{Id: "displayName", Value: "Scope 2"}}).
			Orientation("vertical").
			XField("source").
			XTickLabelRotation(0).
			XTickLabelSpacing(0).
			XTickLabelMaxLength(100).
			Stacking("none").
			ShowValue("auto").
			GroupWidth(0.7).
			BarWidth(0.97).
			BarRadius(0).
			FullHighlight(false).
			Legend(common.NewVizLegendOptionsBuilder().
				DisplayMode("list").
				Placement("bottom").
				ShowLegend(true)).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			GradientMode("none").
			AxisPlacement("auto").
			AxisColorMode("text").
			ScaleDistribution(common.NewScaleDistributionConfigBuilder().
				Type("linear")).
			AxisCenteredZero(false).
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false)).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
				Mode("off")).
			AxisBorderShow(false)).
		WithPanel(barchart.NewPanelBuilder().
			Id(15).
			Datasource(mixedDSRef).
			Height(12).
			Span(6).
			WithTarget(bigqueryRaw(scope1Query).
				Project("${bigquery_project}").
				Format(0).
				RefId("Scope 1").
				Datasource(bigQueryDSRef)).
			WithTarget(athena.NewDataqueryBuilder().
				Format(1).
				ConnectionArgs(athena.NewConnectionArgsBuilder()).
				RefId("AWS Scope 1").
				RawSQL(awsScope1TotalQuery).
				Datasource(athenaDSRef)).
			WithTarget(bigqueryRaw(azureScope1TotalQuery).
				Project("${bigquery_project}").
				Format(0).
				RefId("Azure Scope 1").
				Datasource(bigQueryDSRef)).
			Title("Scope 1").
			Transparent(true).
			Transformations([]dashboard.DataTransformerConfig{
				{Id: "convertFieldType", Options: map[string]any{"conversions": []any{map[string]any{"destinationType": "number", "targetField": "Scope 1 scope1"}, map[string]any{"destinationType": "number", "targetField": "AWS Scope 1 scope1"}, map[string]any{"destinationType": "number", "targetField": "Azure Scope 1 scope1"}}}},
				{Id: "merge"},
			}).
			Unit(metricTonsCO2e).
			Max(6000).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("fixed").
				FixedColor("light-blue")).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "scope1"}, []dashboard.DynamicConfigValue{{Id: "displayName", Value: "Scope 1"}}).
			Orientation("vertical").
			XField("source").
			XTickLabelRotation(0).
			XTickLabelSpacing(0).
			XTickLabelMaxLength(100).
			Stacking("none").
			ShowValue("auto").
			GroupWidth(0.7).
			BarWidth(0.97).
			BarRadius(0).
			FullHighlight(false).
			Legend(common.NewVizLegendOptionsBuilder().
				DisplayMode("list").
				Placement("bottom").
				ShowLegend(true)).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			GradientMode("none").
			AxisPlacement("auto").
			AxisColorMode("text").
			ScaleDistribution(common.NewScaleDistributionConfigBuilder().
				Type("linear")).
			AxisCenteredZero(false).
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false)).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
				Mode("off")).
			AxisBorderShow(false)).
		// Google Cloud section
		WithRow(dashboard.NewRowBuilder("Google Cloud").
			Title("Google Cloud")).
		WithPanel(piechart.NewPanelBuilder().
			Datasource(bigQueryDSRef).
			WithTarget(bigqueryRaw(scope1QueryWithUsage).
				Project("${bigquery_project}").
				Format(0).
				EditorMode("code").
				RefId("Scope 1").
				Datasource(bigQueryDSRef),
			).
			WithTarget(bigqueryRaw(scope2QueryWithUsage).
				Project("${bigquery_project}").
				Format(0).
				EditorMode("code").
				RefId("Scope 2").
				Hide(false).
				Datasource(bigQueryDSRef),
			).
			WithTarget(bigqueryRaw(scope3QueryWithUsage).
				Project("${bigquery_project}").
				Format(0).
				EditorMode("code").
				RefId("Scope 3").
				Hide(false).
				Datasource(bigQueryDSRef),
			).
			Title("Google Cloud | Total Emissions by Scope").
			Transparent(true).
			Height(11).
			Transformations([]dashboard.DataTransformerConfig{{Id: "convertFieldType", Options: map[string]any{"conversions": []any{map[string]any{"dateFormat": "YYYY-MM", "destinationType": "time", "targetField": "usage_month"}}}}}).
			Unit(metricTonsCO2e).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("palette-classic").
				FixedColor("yellow")).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "Scope 1 carbon_footprint"}, []dashboard.DynamicConfigValue{{Id: "displayName", Value: "Scope 1"}}).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "Scope 2 carbon_footprint"}, []dashboard.DynamicConfigValue{{Id: "displayName", Value: "Scope 2"}}).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "Scope 3 carbon_footprint"}, []dashboard.DynamicConfigValue{{Id: "displayName", Value: "Scope 3"}}).
			PieType("pie").
			DisplayLabels([]piechart.PieChartLabels{"percent", "name", "value"}).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			ReduceOptions(common.NewReduceDataOptionsBuilder().
				Values(false).
				Calcs([]string{"sum"})).
			Legend(piechart.NewPieChartLegendOptionsBuilder().
				DisplayMode("hidden").
				Placement("right").
				ShowLegend(false)).
			Orientation("").
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false))).
		WithPanel(barchart.NewPanelBuilder().
			Datasource(bigQueryDSRef).
			WithTarget(bigqueryRaw(scope1QueryWithUsage).
				Project("${bigquery_project}").
				Format(0).
				EditorMode("code").
				RefId("Scope 1").
				Datasource(bigQueryDSRef),
			).
			WithTarget(bigqueryRaw(scope2QueryWithUsage).
				Project("${bigquery_project}").
				Format(0).
				EditorMode("code").
				RefId("Scope 2").
				Hide(false).
				Datasource(bigQueryDSRef),
			).
			WithTarget(bigqueryRaw(scope3QueryWithUsage).
				Project("${bigquery_project}").
				Format(0).
				EditorMode("code").
				RefId("Scope 3").
				Hide(false).
				Datasource(bigQueryDSRef),
			).
			Title("Google Cloud | Emissions by Scope Over Time").
			Transparent(true).
			Height(11).
			Transformations([]dashboard.DataTransformerConfig{{Id: "convertFieldType", Options: map[string]any{"conversions": []any{map[string]any{"dateFormat": "YYYY-MM", "destinationType": "time", "targetField": "usage_month"}}}}}).
			Unit(metricTonsCO2e).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("palette-classic")).
			XField("usage_month").
			Orientation("auto").
			XTickLabelMaxLength(0).
			XTickLabelSpacing(100).
			Stacking("normal").
			ShowValue("never").
			Legend(common.NewVizLegendOptionsBuilder().
				DisplayMode("table").
				Placement("right").
				ShowLegend(true).
				SortBy("Name").
				SortDesc(false)).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			LineWidth(0).
			GradientMode("none").
			AxisPlacement("auto").
			AxisColorMode("text").
			ScaleDistribution(common.NewScaleDistributionConfigBuilder().
				Type("linear")).
			AxisCenteredZero(false).
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false)).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
				Mode("off")).
			AxisBorderShow(false)).
		WithPanel(barchart.NewPanelBuilder().
			Datasource(bigQueryDSRef).
			WithTarget(bigqueryRaw(gcpCarbonServiceQuery).
				Project("${bigquery_project}").
				Format(1).
				EditorMode("code").
				RefId("A").
				Datasource(bigQueryDSRef),
			).
			Title("Google Cloud | Emissions by GCP Service").
			Transparent(true).
			Height(12).
			Unit(metricTonsCO2e).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("palette-classic")).
			Orientation("horizontal").
			XTickLabelMaxLength(0).
			Stacking("none").
			ShowValue("auto").
			Legend(common.NewVizLegendOptionsBuilder().
				DisplayMode("list").
				Placement("bottom").
				ShowLegend(true).
				Calcs([]string{"sum"})).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			GradientMode("none").
			AxisPlacement("auto").
			AxisColorMode("text").
			ScaleDistribution(common.NewScaleDistributionConfigBuilder().
				Type("linear")).
			AxisCenteredZero(false).
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false)).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
				Mode("off")).
			AxisBorderShow(false)).
		WithPanel(geomap.NewPanelBuilder().
			Datasource(mixedDSRef).
			WithTarget(bigqueryRaw(gcpCarbonRegionQuery).
				Project("${bigquery_project}").
				Format(1).
				EditorMode("code").
				RefId("A").
				Datasource(bigQueryDSRef),
			).
			WithTarget(newCustomQueryBuilder(targetRegionCoordinates)).
			Title("Google Cloud | Emissions by GCP Region").
			Transparent(true).
			Height(12).
			Transformations([]dashboard.DataTransformerConfig{{Id: "joinByField", Options: map[string]any{"byField": "region", "mode": "outerTabular"}}}).
			Unit(metricTonsCO2e).
			ColorScheme(dashboard.NewFieldColorBuilder().Mode("fixed").FixedColor("dark-blue")).
			WithOverride(dashboard.MatcherConfig{Id: "byNames", Options: map[string]any{"names": []any{"total_carbon_footprint"}, "prefix": "All except:", "readOnly": true, "mode": "exclude"}},
				[]dashboard.DynamicConfigValue{{Id: "custom.hideFrom", Value: map[string]any{"legend": false, "tooltip": true, "viz": true}}}).
			View(geomap.NewMapViewConfigBuilder()).
			Controls(geomap.NewControlsOptionsBuilder().
				ShowZoom(true).
				MouseWheelZoom(true).
				ShowAttribution(true).
				ShowScale(false).
				ShowDebug(false).
				ShowMeasure(false)).
			Basemap(common.NewMapLayerOptionsBuilder().
				Type("osm-standard").
				Name("Layer 0").
				Config(map[string]any{"server": "streets", "showLabels": true, "theme": "auto"})).
			Layers([]cog.Builder[common.MapLayerOptions]{common.NewMapLayerOptionsBuilder().
				Type("markers").
				Name(metricTonsCO2e).
				Config(map[string]any{"style": map[string]any{"rotation": map[string]any{"min": -360, "mode": "mod", "fixed": 0, "max": 360}, "size": map[string]any{"field": "total_carbon_footprint", "fixed": 5, "max": 100, "min": 5}, "symbol": map[string]any{"fixed": "img/icons/marker/circle.svg", "mode": "fixed"}, "symbolAlign": map[string]any{"horizontal": "center", "vertical": "center"}, "text": map[string]any{"field": "total_carbon_footprint", "mode": "field"}, "textConfig": map[string]any{"textAlign": "center", "textBaseline": "middle", "fontSize": 20, "offsetX": 0, "offsetY": 0}, "color": map[string]any{"field": "total_carbon_footprint", "fixed": "dark-green"}, "opacity": 0.8}, "showLegend": false}).
				Location(common.NewFrameGeometrySourceBuilder().
					Mode("auto")).
				Tooltip(true)}).
			Tooltip(geomap.NewTooltipOptionsBuilder().
				Mode("details"))).
		// AWS section
		WithRow(dashboard.NewRowBuilder("AWS").
			Title("AWS")).
		WithPanel(piechart.NewPanelBuilder().
			Datasource(athenaDSRef).
			WithTarget(athena.NewDataqueryBuilder().
				Format(1).
				ConnectionArgs(athena.NewConnectionArgsBuilder()).
				RefId("A").
				RawSQL(awsScopeEmissionsQuery).
				Datasource(athenaDSRef)).
			Title("AWS | Total Emissions by Scope (Location-Based)").
			Description("This pie chart shows the proportion of location-based (LBM) carbon emissions contributed by Scope 1, Scope 2 LBM, and Scope 3 LBM for the selected time range.").
			Transparent(true).
			Height(11).
			Span(12).
			Unit(metricTonsCO2e).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("palette-classic").
				FixedColor("super-light-blue")).
			PieType("pie").
			DisplayLabels([]piechart.PieChartLabels{"value"}).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			ReduceOptions(common.NewReduceDataOptionsBuilder().
				Values(false)).
			Legend(piechart.NewPieChartLegendOptionsBuilder().
				Values([]piechart.PieChartLegendValues{"percent"}).
				DisplayMode("list").
				Placement("right").
				ShowLegend(true)).
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false))).
		WithPanel(barchart.NewPanelBuilder().
			Datasource(athenaDSRef).
			WithTarget(athena.NewDataqueryBuilder().
				Format(1).
				ConnectionArgs(athena.NewConnectionArgsBuilder()).
				RefId("A").
				RawSQL(awsEmissionsByRegionQuery).
				Datasource(athenaDSRef)).
			Title("AWS | Carbon Emissions by Region").
			Transparent(true).
			Height(11).
			Span(12).
			Transformations([]dashboard.DataTransformerConfig{
				{Id: "prepareTimeSeries", Options: map[string]any{"format": "multi"}},
				{Id: "renameByRegex", Options: map[string]any{"regex": "value (.+)", "renamePattern": "$1"}}}).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}, {Value: cog.ToPtr[float64](80), Color: "red"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("palette-classic")).
			XField("time").
			Orientation("vertical").
			BarRadius(0.1).
			Stacking("normal").
			ShowValue("never").
			Legend(common.NewVizLegendOptionsBuilder().
				DisplayMode("list").
				Placement("right").
				ShowLegend(true)).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			GradientMode("none").
			AxisPlacement("auto").
			AxisColorMode("text").
			ScaleDistribution(common.NewScaleDistributionConfigBuilder().
				Type("linear")).
			AxisCenteredZero(false).
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false)).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
				Mode("off")).
			AxisBorderShow(false)).
		WithPanel(barchart.NewPanelBuilder().
			Datasource(athenaDSRef).
			WithTarget(athena.NewDataqueryBuilder().
				Format(1).
				ConnectionArgs(athena.NewConnectionArgsBuilder()).
				RefId("A").
				RawSQL(awsMonthlyScopeEmissionsQuery).
				Datasource(athenaDSRef)).
			Title("AWS | Monthly Carbon Emissions by Scope").
			Description("Displays monthly CO₂e emissions from AWS, broken down by Scope 1, Scope 2 LBM, and Scope 3 LBM.").
			Transparent(true).
			Height(12).
			Span(12).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}, {Value: cog.ToPtr[float64](80), Color: "red"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("continuous-BlPu")).
			Orientation("auto").
			BarRadius(0.1).
			XTickLabelRotation(-15).
			Stacking("normal").
			ShowValue("never").
			BarWidth(0.74).
			Legend(common.NewVizLegendOptionsBuilder().
				DisplayMode("list").
				Placement("right").
				ShowLegend(true)).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			GradientMode("none").
			AxisPlacement("auto").
			AxisColorMode("text").
			ScaleDistribution(common.NewScaleDistributionConfigBuilder().
				Type("linear")).
			AxisCenteredZero(false).
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false)).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
				Mode("off")).
			AxisBorderShow(false)).
		WithPanel(table.NewPanelBuilder().
			Datasource(athenaDSRef).
			WithTarget(athena.NewDataqueryBuilder().
				Format(1).
				ConnectionArgs(athena.NewConnectionArgsBuilder()).
				RefId("A").
				RawSQL(awsRegionTableQuery).
				Datasource(athenaDSRef)).
			Title("AWS | Carbon Emissions per AWS region (LBM Methods)").
			Transparent(true).
			Height(12).
			Span(12).
			Unit(metricTonsCO2e).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("thresholds")).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "scope1"}, []dashboard.DynamicConfigValue{{Id: "color", Value: map[string]any{"mode": "continuous-BlPu"}}}).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "scope2"}, []dashboard.DynamicConfigValue{{Id: "color", Value: map[string]any{"mode": "continuous-viridis"}}}).
			CellHeight("sm")).
		WithPanel(barchart.NewPanelBuilder().
			Datasource(athenaDSRef).
			WithTarget(athena.NewDataqueryBuilder().
				Format(1).
				ConnectionArgs(athena.NewConnectionArgsBuilder()).
				RefId("A").
				RawSQL(awsMarketVsLocationQuery).
				Datasource(athenaDSRef)).
			Title("AWS | Carbon Emissions Over Time (Market- vs Location-Based)").
			Description("This visualization shows monthly carbon emissions, comparing market-based (MB) and location-based (LB) totals over time.").
			Transparent(true).
			Height(12).
			Span(12).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}, {Value: cog.ToPtr[float64](80), Color: "red"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("continuous-BlPu")).
			XField("time").
			Orientation("auto").
			BarRadius(0.25).
			XTickLabelRotation(0).
			XTickLabelMaxLength(0).
			Stacking("none").
			ShowValue("never").
			BarWidth(0.9).
			Legend(common.NewVizLegendOptionsBuilder().
				DisplayMode("list").
				Placement("right").
				ShowLegend(true)).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			GradientMode("none").
			AxisPlacement("auto").
			AxisColorMode("text").
			ScaleDistribution(common.NewScaleDistributionConfigBuilder().
				Type("linear")).
			AxisCenteredZero(false).
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false)).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
				Mode("off")).
			AxisBorderShow(false)).
		WithPanel(geomap.NewPanelBuilder().
			Datasource(mixedDSRef).
			WithTarget(athena.NewDataqueryBuilder().
				Format(1).
				ConnectionArgs(athena.NewConnectionArgsBuilder()).
				RefId("A").
				RawSQL(awsGeomapQuery).
				Datasource(athenaDSRef)).
			WithTarget(newCustomQueryBuilder(awsRegionCoordinates)).
			Transformations([]dashboard.DataTransformerConfig{{Id: "joinByField", Options: map[string]any{"byField": "region", "mode": "outerTabular"}}}).
			Title("AWS | Emissions by AWS Region").
			Transparent(true).
			Height(12).
			Span(12).
			Unit(metricTonsCO2e).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("palette-classic-by-name")).
			View(geomap.NewMapViewConfigBuilder()).
			Controls(geomap.NewControlsOptionsBuilder().
				ShowZoom(true).
				MouseWheelZoom(true).
				ShowAttribution(true).
				ShowScale(false).
				ShowDebug(false).
				ShowMeasure(false)).
			Basemap(common.NewMapLayerOptionsBuilder().
				Type("osm-standard").
				Name("Layer 0").
				Config(map[string]any{"server": "streets", "showLabels": true, "theme": "auto"})).
			Layers([]cog.Builder[common.MapLayerOptions]{common.NewMapLayerOptionsBuilder().
				Type("markers").
				Config(map[string]any{
					"showLegend": false,
					"style": map[string]any{
						"color":       map[string]interface{}{"fixed": "dark-blue"},
						"opacity":     0.8,
						"rotation":    map[string]interface{}{"min": -360, "mode": "mod", "fixed": 0, "max": 360},
						"size":        map[string]interface{}{"field": "total_carbon_footprint", "fixed": 5, "max": 100, "min": 5},
						"symbol":      map[string]interface{}{"fixed": "img/icons/marker/circle.svg", "mode": "fixed"},
						"symbolAlign": map[string]interface{}{"horizontal": "center", "vertical": "center"},
						"text":        map[string]interface{}{"field": "total_carbon_footprint", "mode": "field"},
						"textConfig":  map[string]interface{}{"fontSize": 20, "offsetX": 0, "offsetY": 0, "textAlign": "center", "textBaseline": "middle"},
					}}).
				Location(common.NewFrameGeometrySourceBuilder().
					Mode("auto")).
				Tooltip(true)}).
			Tooltip(geomap.NewTooltipOptionsBuilder().
				Mode("details"))).
		// Azure section
		WithRow(dashboard.NewRowBuilder("Azure").
			Title("Azure")).
		WithPanel(piechart.NewPanelBuilder().
			Datasource(mixedDSRef).
			WithTarget(bigqueryRaw(azureScopeEmissionsQuery).
				Project("${bigquery_project}").
				Format(1).
				EditorMode("code").
				RefId("Prod").
				Datasource(bigQueryDSRef),
			).
			Title("Azure | Total Emissions by Scope").
			Transparent(true).
			Height(11).
			Unit(metricTonsCO2e).
			Transformations([]dashboard.DataTransformerConfig{
				{Id: "merge"},
				{Id: "groupBy", Options: map[string]interface{}{
					"fields": map[string]interface{}{
						"Scope 1": map[string]interface{}{"aggregations": []interface{}{"sum"}, "operation": "aggregate"},
						"Scope 2": map[string]interface{}{"aggregations": []interface{}{"sum"}, "operation": "aggregate"},
						"Scope 3": map[string]interface{}{"aggregations": []interface{}{"sum"}, "operation": "aggregate"},
					},
				}},
			}).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("palette-classic").
				FixedColor("blue")).
			PieType("pie").
			DisplayLabels([]piechart.PieChartLabels{"percent", "name", "value"}).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			ReduceOptions(common.NewReduceDataOptionsBuilder().
				Values(false)).
			Legend(piechart.NewPieChartLegendOptionsBuilder().
				DisplayMode("table").
				Placement("right").
				ShowLegend(true).
				Values([]piechart.PieChartLegendValues{"percent", "value"}).
				SortBy("Value").
				SortDesc(false)).
			Orientation("").
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false))).
		WithPanel(barchart.NewPanelBuilder().
			Datasource(mixedDSRef).
			WithTarget(bigqueryRaw(azureMonthlyScopeEmissionsQuery).
				Project("${bigquery_project}").
				Format(1).
				EditorMode("code").
				RefId("Prod").
				Datasource(bigQueryDSRef),
			).
			Title("Azure | Monthly CO₂ Emissions by Scope").
			Description("Displays monthly CO₂ emissions from Azure, broken down by Scope 1, Scope 2, and Scope 3.").
			Transparent(true).
			Height(11).
			Span(12).
			Transformations([]dashboard.DataTransformerConfig{
				{Id: "merge"},
				{Id: "groupBy", Options: map[string]interface{}{
					"fields": map[string]interface{}{
						"time":    map[string]interface{}{"aggregations": []interface{}{}, "operation": "groupby"},
						"Scope 1": map[string]interface{}{"aggregations": []interface{}{"sum"}, "operation": "aggregate"},
						"Scope 2": map[string]interface{}{"aggregations": []interface{}{"sum"}, "operation": "aggregate"},
						"Scope 3": map[string]interface{}{"aggregations": []interface{}{"sum"}, "operation": "aggregate"},
					},
				}},
				{Id: "convertFieldType", Options: map[string]interface{}{"conversions": []interface{}{map[string]interface{}{"dateFormat": "YYYY-MM-DD", "destinationType": "time", "targetField": "time"}}}},
			}).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}, {Value: cog.ToPtr[float64](80), Color: "red"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("continuous-BlPu")).
			XField("time").
			Orientation("auto").
			BarRadius(0.1).
			XTickLabelRotation(0).
			Stacking("normal").
			ShowValue("never").
			BarWidth(0.74).
			Legend(common.NewVizLegendOptionsBuilder().
				DisplayMode("list").
				Placement("right").
				ShowLegend(true)).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			GradientMode("none").
			AxisPlacement("auto").
			AxisColorMode("text").
			ScaleDistribution(common.NewScaleDistributionConfigBuilder().
				Type("linear")).
			AxisCenteredZero(false).
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false)).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
				Mode("off")).
			AxisBorderShow(false)).
		WithPanel(table.NewPanelBuilder().
			Datasource(mixedDSRef).
			WithTarget(bigqueryRaw(azureRegionTableQuery).
				Project("${bigquery_project}").
				Format(1).
				EditorMode("code").
				RefId("Prod").
				Datasource(bigQueryDSRef),
			).
			Title("Azure | Carbon Emissions by Azure region").
			Transparent(true).
			Height(12).
			Span(12).
			Transformations([]dashboard.DataTransformerConfig{
				{Id: "merge"},
				{Id: "groupBy", Options: map[string]interface{}{
					"fields": map[string]interface{}{
						"region": map[string]interface{}{"aggregations": []interface{}{}, "operation": "groupby"},
						"scope1": map[string]interface{}{"aggregations": []interface{}{"sum"}, "operation": "aggregate"},
						"scope2": map[string]interface{}{"aggregations": []interface{}{"sum"}, "operation": "aggregate"},
						"scope3": map[string]interface{}{"aggregations": []interface{}{"sum"}, "operation": "aggregate"},
					},
				}},
			}).
			Unit(metricTonsCO2e).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("thresholds")).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "scope1"}, []dashboard.DynamicConfigValue{{Id: "color", Value: map[string]interface{}{"mode": "continuous-BlPu"}}}).
			WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "scope2"}, []dashboard.DynamicConfigValue{{Id: "color", Value: map[string]any{"mode": "continuous-viridis"}}}).
			CellHeight("sm")).
		WithPanel(barchart.NewPanelBuilder().
			Datasource(mixedDSRef).
			WithTarget(bigqueryRaw(azureResourceTypeQuery).
				Project("${bigquery_project}").
				Format(1).
				EditorMode("code").
				RefId("Prod").
				Datasource(bigQueryDSRef),
			).
			Title("Azure | Emissions by Resource Type").
			Transparent(true).
			Height(12).
			Span(12).
			Transformations([]dashboard.DataTransformerConfig{
				{Id: "merge"},
				{Id: "groupBy", Options: map[string]interface{}{
					"fields": map[string]interface{}{
						"resource_type":          map[string]interface{}{"aggregations": []interface{}{}, "operation": "groupby"},
						"total_carbon_footprint": map[string]interface{}{"aggregations": []interface{}{"sum"}, "operation": "aggregate"},
					},
				}},
			}).
			Unit(metricTonsCO2e).
			Thresholds(dashboard.NewThresholdsConfigBuilder().
				Mode("absolute").
				Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}})).
			ColorScheme(dashboard.NewFieldColorBuilder().
				Mode("palette-classic")).
			Orientation("horizontal").
			XTickLabelMaxLength(0).
			Stacking("none").
			ShowValue("auto").
			Legend(common.NewVizLegendOptionsBuilder().
				DisplayMode("list").
				Placement("bottom").
				ShowLegend(true).
				Calcs([]string{"sum"})).
			Tooltip(common.NewVizTooltipOptionsBuilder().
				Mode("single").
				Sort("none").
				HideZeros(false)).
			GradientMode("none").
			AxisPlacement("auto").
			AxisColorMode("text").
			ScaleDistribution(common.NewScaleDistributionConfigBuilder().
				Type("linear")).
			AxisCenteredZero(false).
			HideFrom(common.NewHideSeriesConfigBuilder().
				Tooltip(false).
				Legend(false).
				Viz(false)).
			ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
				Mode("off")).
			AxisBorderShow(false))
}

func buildTotalCarbonPanel() *stat.PanelBuilder {
	return stat.NewPanelBuilder().
		Datasource(mixedDSRef).
		WithTarget(bigqueryRaw(gcpTotalQuery).
			Project("${bigquery_project}").
			RefId("GCP").
			Datasource(bigQueryDSRef),
		).
		WithTarget(athena.NewDataqueryBuilder().
			Format(1).
			ConnectionArgs(athena.NewConnectionArgsBuilder()).
			RefId("AWS").
			RawSQL(awsTotalQuery).
			Datasource(athenaDSRef)).
		WithTarget(bigqueryRaw(azureTotalQuery).
			Project("${bigquery_project}").
			RefId("Azure").
			Datasource(bigQueryDSRef),
		).
		Title("🌍 Total Carbon Emissions").
		Transparent(true).
		GridPos(dashboard.GridPos{H: 6, W: 5, X: 0, Y: 3}).
		Height(6).
		Span(5).
		Transformations([]dashboard.DataTransformerConfig{{Id: "reduce", Options: map[string]interface{}{"reducers": []interface{}{"sum"}}}}).
		Unit(metricTonsCO2e).
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}})).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("fixed").
			FixedColor("light-blue")).
		GraphMode("none").
		ColorMode("value").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Values(false).
			Calcs([]string{"sum"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func buildEmissionsBySourcePanel() *barchart.PanelBuilder {
	return barchart.NewPanelBuilder().
		Datasource(mixedDSRef).
		WithTarget(bigqueryRaw(gcpMonthlyRegionQuery).
			Project("${bigquery_project}").
			RefId("GCP").
			Hide(false).
			Datasource(bigQueryDSRef),
		).
		WithTarget(athena.NewDataqueryBuilder().
			Format(1).
			ConnectionArgs(athena.NewConnectionArgsBuilder()).
			RefId("AWS").
			RawSQL(awsMonthlyTotalQuery).
			Datasource(athenaDSRef)).
		WithTarget(bigqueryRaw(azureMonthlyTotalQuery).
			Project("${bigquery_project}").
			RefId("Azure").
			Hide(false).
			Datasource(bigQueryDSRef),
		).
		Title("Emissions by Source Over Time").
		Transparent(true).
		GridPos(dashboard.GridPos{H: 12, W: 19, X: 5, Y: 3}).
		Height(12).
		Span(19).
		Transformations([]dashboard.DataTransformerConfig{
			{Id: "organize", Options: map[string]interface{}{"renameByName": map[string]interface{}{"usage_month": "created_month"}}},
			{Id: "convertFieldType", Options: map[string]interface{}{"conversions": []interface{}{map[string]interface{}{"destinationType": "time", "targetField": "created_month"}, map[string]interface{}{"dateFormat": "YYYY-MM", "destinationType": "time", "targetField": "usage_month"}}}},
		}).
		Unit(metricTonsCO2e).
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"},
				{Value: cog.ToPtr[float64](80), Color: "red"}})).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("palette-classic")).
		WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "GCP metric_tons_CO2eq"}, []dashboard.DynamicConfigValue{{Id: "color", Value: map[string]interface{}{"fixedColor": "green", "mode": "fixed"}}}).
		WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "AWS metric_tons_CO2eq"}, []dashboard.DynamicConfigValue{{Id: "color", Value: map[string]interface{}{"fixedColor": "blue", "mode": "fixed"}}}).
		WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "Azure metric_tons_CO2eq"}, []dashboard.DynamicConfigValue{{Id: "color", Value: map[string]interface{}{"fixedColor": "yellow", "mode": "fixed"}}}).
		XField("created_month").
		Orientation("auto").
		XTickLabelMaxLength(0).
		XTickLabelSpacing(100).
		Stacking("normal").
		ShowValue("never").
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("list").
			Placement("right").
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("single").
			Sort("none").
			HideZeros(false)).
		GradientMode("none").
		AxisPlacement("auto").
		AxisColorMode("text").
		ScaleDistribution(common.NewScaleDistributionConfigBuilder().
			Type("linear")).
		AxisCenteredZero(false).
		HideFrom(common.NewHideSeriesConfigBuilder().
			Tooltip(false).
			Legend(false).
			Viz(false)).
		ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
			Mode("off")).
		AxisBorderShow(false)
}

func buildTripsAroundWorldPanel() *stat.PanelBuilder {
	return stat.NewPanelBuilder().
		Datasource(mixedDSRef).
		WithTarget(bigqueryRaw(gcpTotalQuery).
			Project("${bigquery_project}").
			RefId("GCP").
			Datasource(bigQueryDSRef),
		).
		WithTarget(athena.NewDataqueryBuilder().
			Format(1).
			ConnectionArgs(athena.NewConnectionArgsBuilder()).
			RefId("AWS").
			RawSQL(awsTotalQuery).
			Datasource(athenaDSRef)).
		WithTarget(bigqueryRaw(azureTotalQuery).
			Project("${bigquery_project}").
			RefId("Azure").
			Datasource(bigQueryDSRef),
		).
		Title("🚗 Equivalent Trips Around the World").
		Description("Equivalent trips around the world by an average gasoline-powered passenger vehicle, calculated using the EPA Greenhouse Gas Equivalencies Calculator (2,481 miles per metric ton CO₂e).").
		Transparent(true).
		GridPos(dashboard.GridPos{H: 6, W: 5, X: 0, Y: 9}).
		Height(6).
		Span(5).
		Transformations([]dashboard.DataTransformerConfig{
			{Id: "reduce", Options: map[string]any{"includeTimeField": false, "mode": "seriesToRows", "reducers": []any{"sum"}}},
			{Id: "calculateField", Options: map[string]any{"alias": "Miles Driven", "binary": map[string]any{"left": map[string]any{"matcher": map[string]any{"id": "byName", "options": "Total"}}, "operator": "*", "right": map[string]any{"fixed": 2481}}, "mode": "binary", "reduce": map[string]any{"reducer": "sum"}, "replaceFields": false}},
			{Id: "calculateField", Options: map[string]any{"alias": "Trips Around the World", "binary": map[string]any{"left": map[string]any{"matcher": map[string]any{"id": "byName", "options": "Miles Driven"}}, "operator": "/", "right": map[string]any{"fixed": 24901}}, "mode": "binary", "reduce": map[string]any{"reducer": "sum"}, "replaceFields": false}},
			{Id: "organize", Options: map[string]any{"excludeByName": map[string]any{"Total": true, "Miles Driven": true}}},
		}).
		Unit("trips").
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}})).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("fixed").
			FixedColor("light-blue")).
		GraphMode("none").
		ColorMode("value").
		JustifyMode("auto").
		TextMode("auto").
		ReduceOptions(common.NewReduceDataOptionsBuilder().
			Values(false).
			Calcs([]string{"sum"})).
		PercentChangeColorMode("standard").
		Orientation("auto")
}

func buildScope3Panel() *barchart.PanelBuilder {
	return barchart.NewPanelBuilder().
		Id(13).
		Datasource(mixedDSRef).
		WithTarget(bigqueryRaw(scope3Query).
			Project("${bigquery_project}").
			Format(0).
			RefId("Scope 3 (GCP)").
			Datasource(bigQueryDSRef)).
		WithTarget(bigqueryRaw(azureScope3TotalQuery).
			Project("${bigquery_project}").
			Format(0).
			RefId("Azure Scope 3").
			Datasource(bigQueryDSRef)).
		WithTarget(athena.NewDataqueryBuilder().
			Format(1).
			ConnectionArgs(athena.NewConnectionArgsBuilder()).
			RefId("AWS Scope 3").
			RawSQL(awsScope3TotalQuery).
			Datasource(athenaDSRef)).
		Title("Scope 3").
		Transparent(true).
		Height(12).
		Span(6).
		Transformations([]dashboard.DataTransformerConfig{
			{Id: "convertFieldType", Options: map[string]any{"conversions": []any{
				map[string]any{"destinationType": "number", "targetField": "Scope 3 (GCP) scope3"},
				map[string]any{"destinationType": "number", "targetField": "Azure Scope 3 scope3"},
				map[string]any{"destinationType": "number", "targetField": "AWS Scope 3 scope3"},
			}}},
			{Id: "merge"},
		}).
		Unit(metricTonsCO2e).
		Thresholds(dashboard.NewThresholdsConfigBuilder().
			Mode("absolute").
			Steps([]dashboard.Threshold{{Value: cog.ToPtr[float64](0), Color: "green"}})).
		ColorScheme(dashboard.NewFieldColorBuilder().
			Mode("fixed").
			FixedColor("light-blue")).
		WithOverride(dashboard.MatcherConfig{Id: "byName", Options: "scope3"}, []dashboard.DynamicConfigValue{{Id: "displayName", Value: "Scope 3"}}).
		Orientation("vertical").
		XField("source").
		XTickLabelRotation(0).
		XTickLabelSpacing(0).
		XTickLabelMaxLength(100).
		Stacking("none").
		ShowValue("auto").
		GroupWidth(0.7).
		BarWidth(0.97).
		BarRadius(0).
		FullHighlight(false).
		Legend(common.NewVizLegendOptionsBuilder().
			DisplayMode("list").
			Placement("bottom").
			ShowLegend(true)).
		Tooltip(common.NewVizTooltipOptionsBuilder().
			Mode("single").
			Sort("none").
			HideZeros(false)).
		GradientMode("none").
		AxisPlacement("auto").
		AxisColorMode("text").
		ScaleDistribution(common.NewScaleDistributionConfigBuilder().
			Type("linear")).
		AxisCenteredZero(false).
		HideFrom(common.NewHideSeriesConfigBuilder().
			Tooltip(false).
			Legend(false).
			Viz(false)).
		ThresholdsStyle(common.NewGraphThresholdsStyleConfigBuilder().
			Mode("off")).
		AxisBorderShow(false)
}
