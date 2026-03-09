SELECT
    usage_month,
    SUM(emissions_kg_co2e) / 1000 AS metric_tons_CO2eq
FROM `${bigquery_project}.${azure_dataset}.azure_carbon_data_prod`
WHERE $__timeFilter(TIMESTAMP(usage_month))
GROUP BY usage_month
ORDER BY usage_month;
