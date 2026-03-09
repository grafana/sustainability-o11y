SELECT
  usage_period_start AS time,
  ROUND(SUM(total_lbm_emissions_value), 4) AS metric_tons_CO2eq
FROM "AwsDataCatalog"."${athena_database}"."${athena_table}"
WHERE usage_period_start BETWEEN $__timeFrom() AND $__timeTo()
  AND total_mbm_emissions_value > 0
GROUP BY usage_period_start
ORDER BY usage_period_start;

