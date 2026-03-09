SELECT
  SUM(total_scope_1_emissions_value) AS "Scope 1",
  SUM(total_scope_2_lbm_emissions_value) AS "Scope 2",
  SUM(total_scope_3_lbm_emissions_value) AS "Scope 3"
FROM ${athena_database}.${athena_table}
WHERE model_version = 'v3.0.0'
  AND $__timeFilter(usage_period_start);

