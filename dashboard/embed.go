package carbonmonitoring

import (
	_ "embed"
)

// Total / aggregate queries

//go:embed queries/total/scope1_query.sql
var scope1Query string

//go:embed queries/total/scope2_query.sql
var scope2Query string

//go:embed queries/total/scope3_query.sql
var scope3Query string

//go:embed queries/total/gcp_total.sql
var gcpTotalQuery string

//go:embed queries/total/gcp_monthly_region_query.sql
var gcpMonthlyRegionQuery string

//go:embed queries/total/aws_total.sql
var awsTotalQuery string

//go:embed queries/total/aws_monthly_total.sql
var awsMonthlyTotalQuery string

//go:embed queries/total/aws_scope1_total.sql
var awsScope1TotalQuery string

//go:embed queries/total/aws_scope2_total.sql
var awsScope2TotalQuery string

//go:embed queries/total/aws_scope3_total.sql
var awsScope3TotalQuery string

//go:embed queries/total/azure_total.sql
var azureTotalQuery string

//go:embed queries/total/azure_monthly_total.sql
var azureMonthlyTotalQuery string

//go:embed queries/total/azure_scope1_total.sql
var azureScope1TotalQuery string

//go:embed queries/total/azure_scope2_total.sql
var azureScope2TotalQuery string

//go:embed queries/total/azure_scope3_total.sql
var azureScope3TotalQuery string

// GCP queries

//go:embed queries/gcp/gcp_carbon_region_query.sql
var gcpCarbonRegionQuery string

//go:embed queries/gcp/gcp_carbon_service_query.sql
var gcpCarbonServiceQuery string

//go:embed queries/gcp/scope1_query_with_usage.sql
var scope1QueryWithUsage string

//go:embed queries/gcp/scope2_query_with_usage.sql
var scope2QueryWithUsage string

//go:embed queries/gcp/scope3_query_with_usage.sql
var scope3QueryWithUsage string

// AWS queries

//go:embed queries/aws/aws_scope_emissions.sql
var awsScopeEmissionsQuery string

//go:embed queries/aws/aws_emissions_by_region.sql
var awsEmissionsByRegionQuery string

//go:embed queries/aws/aws_monthly_scope_emissions.sql
var awsMonthlyScopeEmissionsQuery string

//go:embed queries/aws/aws_region_table.sql
var awsRegionTableQuery string

//go:embed queries/aws/aws_market_vs_location.sql
var awsMarketVsLocationQuery string

//go:embed queries/aws/aws_geomap.sql
var awsGeomapQuery string

// Azure queries

//go:embed queries/azure/azure_scope_emissions.sql
var azureScopeEmissionsQuery string

//go:embed queries/azure/azure_monthly_scope_emissions.sql
var azureMonthlyScopeEmissionsQuery string

//go:embed queries/azure/azure_region_table.sql
var azureRegionTableQuery string

//go:embed queries/azure/azure_resource_type.sql
var azureResourceTypeQuery string

// Infinity coordinate data

//go:embed queries/gcp/target_infinityDS_region_coordinates.json
var targetRegionCoordinatesJSON []byte

//go:embed queries/aws/target_infinityDS_aws_region_coordinates.json
var awsRegionCoordinatesJSON []byte
