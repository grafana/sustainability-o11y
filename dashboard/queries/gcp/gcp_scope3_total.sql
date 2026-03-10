SELECT
    'GCP' as source,
    SUM(`carbon_footprint_kgCO2e`.`scope3`) / 1000 as scope3
FROM `${gcp_dataset}.carbon_footprint`
WHERE $__timeFilter(TIMESTAMP(usage_month))
GROUP BY source;
