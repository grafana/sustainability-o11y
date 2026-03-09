SELECT
  location AS region,
  SUM(CASE WHEN scope = 'Scope 1' THEN emissions_kg_co2e ELSE 0 END) / 1000 AS scope1,
  SUM(CASE WHEN scope = 'Scope 2' THEN emissions_kg_co2e ELSE 0 END) / 1000 AS scope2,
  SUM(CASE WHEN scope = 'Scope 3' THEN emissions_kg_co2e ELSE 0 END) / 1000 AS scope3
FROM `${azure_dataset}.azure_carbon_data_prod`
WHERE $__timeFilter(TIMESTAMP(usage_month))
  AND location IS NOT NULL
GROUP BY location
ORDER BY (scope1 + scope2 + scope3) DESC
