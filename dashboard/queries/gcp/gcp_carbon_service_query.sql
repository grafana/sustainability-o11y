SELECT
  SUM(`carbon_footprint_total_kgCO2e`.`location_based`) / 1000 as total_carbon_footprint,
  --`location`.`location`,
  --`location`.`region`,
  --`project`.`id`,
  --`project`.`number`,
  `service`.`description`,
  --`service`.`id`
FROM
  `${bigquery_project}.${gcp_dataset}.carbon_footprint`
  WHERE $__timeFilter(TIMESTAMP(usage_month))
  GROUP BY
    -- usage_month,
    --location.region,
    service.description
  ORDER BY total_carbon_footprint DESC
  LIMIT 10;
