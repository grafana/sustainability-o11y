SELECT
  SUM(`carbon_footprint_total_kgCO2e`.`location_based`) / 1000 as metric_tons_CO2eq,
  usage_month,
FROM
  `${bigquery_project}.${gcp_dataset}.carbon_footprint`
  WHERE $__timeFilter(TIMESTAMP(usage_month))
  GROUP BY
    usage_month
  ORDER BY usage_month ASC
  LIMIT 1000;
