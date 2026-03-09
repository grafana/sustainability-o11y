SELECT
  location AS region,
  SUM(total_scope_1_emissions_value) AS scope1,
  SUM(total_scope_2_lbm_emissions_value) AS scope2,
  SUM(total_scope_3_lbm_emissions_value) AS scope3
FROM ${athena_database}.${athena_table}
WHERE model_version = 'v3.0.0'
  AND $__timeFilter(usage_period_start)
GROUP BY location
ORDER BY (scope1 + scope2 + scope3) DESC

