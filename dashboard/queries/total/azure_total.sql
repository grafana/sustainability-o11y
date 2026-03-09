SELECT
    SUM(emissions_kg_co2e) / 1000 AS total
FROM `${bigquery_project}.${azure_dataset}.azure_carbon_data_prod`
WHERE $__timeFilter(TIMESTAMP(usage_month));
