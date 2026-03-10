SELECT
  'AWS' as source,
  SUM(total_scope_1_emissions_value) AS scope1
FROM "AwsDataCatalog"."${athena_database}"."${athena_table}"
WHERE usage_period_start BETWEEN $__timeFrom() AND $__timeTo()
  AND model_version = 'v3.0.0'
