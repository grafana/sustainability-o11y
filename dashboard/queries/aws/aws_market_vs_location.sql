SELECT
  DATE_TRUNC('month', usage_period_start) AS time,
  SUM(total_mbm_emissions_value) AS "Market-Based",
  SUM(total_lbm_emissions_value) AS "Location-Based"
FROM ${athena_database}.${athena_table}
WHERE model_version = 'v3.0.0'
  AND $__timeFilter(usage_period_start)
GROUP BY DATE_TRUNC('month', usage_period_start)
ORDER BY time;
