SELECT
  SUM(`carbon_footprint_total_kgCO2e`.`location_based`) / 1000 as total
FROM `${bigquery_project}.${gcp_dataset}.carbon_footprint`
WHERE $__timeFilter(TIMESTAMP(usage_month))

