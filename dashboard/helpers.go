package carbonmonitoring

import (
	"encoding/json"

	"github.com/grafana/grafana-foundation-sdk/go/athena"
	"github.com/grafana/grafana-foundation-sdk/go/bigquery"
	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/cog/variants"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/datasource"
)

// bigqueryRaw builds a raw-SQL BigQuery query.
func bigqueryRaw(rawSQL string) *bigquery.DataqueryBuilder {
	return bigquery.NewDataqueryBuilder().
		RawSql(rawSQL).
		RawQuery(true).
		Format(bigquery.QueryFormatTimeseries).
		EditorMode(bigquery.EditorModeCode)
}

// athenaRaw builds a raw-SQL Athena query.
func athenaRaw(rawSQL string) *athena.DataqueryBuilder {
	return athena.NewDataqueryBuilder().
		RawSQL(rawSQL)
}

// dashboardSource returns a query that references another panel in the same dashboard.
func dashboardSource() *datasource.DataqueryBuilder {
	return datasource.NewDataqueryBuilder().
		Datasource(dashboard.DataSourceRef{
			Type: cog.ToPtr("datasource"),
			Uid:  cog.ToPtr("-- Dashboard --"),
		})
}

// mixedDSRef is the datasource ref for a mixed-datasource panel.
var mixedDSRef = dashboard.DataSourceRef{
	Type: cog.ToPtr("datasource"),
	Uid:  cog.ToPtr("-- Mixed --"),
}

// newCustomQueryBuilder wraps a raw JSON query object.
func newCustomQueryBuilder(query json.RawMessage) cog.Builder[variants.Dataquery] {
	var queryObj map[string]interface{}
	if err := json.Unmarshal(query, &queryObj); err != nil {
		queryObj = map[string]interface{}{
			"query": query,
		}
	}
	return variants.NewUnknownDataqueryBuilderFromObject(variants.UnknownDataquery(queryObj))
}
