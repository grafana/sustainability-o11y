SELECT
  usage_month,
  SUM(`carbon_footprint_kgCO2e`.`scope1`) / 1000 AS `Scope 1`,
  SUM(`carbon_footprint_kgCO2e`.`scope2`.`location_based`) / 1000 AS `Scope 2`,
  SUM(`carbon_footprint_kgCO2e`.`scope3`) / 1000 AS `Scope 3`
FROM `${gcp_dataset}.carbon_footprint`
WHERE $__timeFilter(TIMESTAMP(usage_month))
GROUP BY usage_month
ORDER BY usage_month ASC;
