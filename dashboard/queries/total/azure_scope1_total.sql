SELECT
    'Azure' AS source,
    SUM(CASE WHEN scope = 'Scope 1' THEN emissions_kg_co2e ELSE 0 END) / 1000 AS scope1
FROM `${bigquery_project}.${azure_dataset}.azure_carbon_data_prod`
WHERE $__timeFilter(TIMESTAMP(usage_month));
