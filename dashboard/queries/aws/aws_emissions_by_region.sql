SELECT
  DATE_TRUNC('month', usage_period_start) AS time,
  location AS metric,
  SUM(total_scope_1_emissions_value + total_scope_2_lbm_emissions_value + total_scope_3_lbm_emissions_value) AS value
FROM ${athena_database}.${athena_table}
WHERE model_version = 'v3.0.0'
  AND $__timeFilter(usage_period_start)
GROUP BY
  DATE_TRUNC('month', usage_period_start),
  location
ORDER BY time, location
