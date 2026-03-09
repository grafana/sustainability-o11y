SELECT
    'GCP' as source,
    SUM(`carbon_footprint_kgCO2e`.`scope1`) / 1000 as scope1
FROM `${bigquery_project}.${gcp_dataset}.carbon_footprint`
WHERE $__timeFilter(TIMESTAMP(usage_month))
GROUP BY source;
