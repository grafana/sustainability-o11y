SELECT
    usage_month AS time,
    SUM(CASE WHEN scope = 'Scope 1' THEN emissions_kg_co2e ELSE 0 END) / 1000 AS `Scope 1`,
    SUM(CASE WHEN scope = 'Scope 2' THEN emissions_kg_co2e ELSE 0 END) / 1000 AS `Scope 2`,
    SUM(CASE WHEN scope = 'Scope 3' THEN emissions_kg_co2e ELSE 0 END) / 1000 AS `Scope 3`
FROM `${bigquery_project}.${azure_dataset}.azure_carbon_data_prod`
WHERE $__timeFilter(TIMESTAMP(usage_month))
GROUP BY usage_month
ORDER BY usage_month;

