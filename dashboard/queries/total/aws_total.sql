SELECT
  SUM(total_lbm_emissions_value) AS total
FROM "AwsDataCatalog"."${athena_database}"."${athena_table}"
WHERE usage_period_start BETWEEN $__timeFrom() AND $__timeTo()
  AND model_version = 'v3.0.0'

