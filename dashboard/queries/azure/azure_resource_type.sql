SELECT
  SUM(emissions_kg_co2e) / 1000 AS total_carbon_footprint,
  resource_type
FROM `${azure_dataset}.azure_carbon_data_prod`
WHERE $__timeFilter(TIMESTAMP(usage_month))
  AND resource_type IS NOT NULL
GROUP BY resource_type
ORDER BY total_carbon_footprint DESC
LIMIT 10
