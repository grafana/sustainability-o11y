SELECT
    region_code AS region,
    SUM(total_lbm_emissions_value) AS total_carbon_footprint
FROM ${athena_database}.${athena_table}
WHERE usage_period_start >= CAST($__timeFrom() AS timestamp)
  AND usage_period_start <= CAST($__timeTo() AS timestamp)
  AND total_lbm_emissions_value > 0
GROUP BY region_code
