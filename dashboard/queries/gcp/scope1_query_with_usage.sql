SELECT SUM(`carbon_footprint_kgCO2e`.`scope1`) / 1000 as carbon_footprint,
    usage_month
    FROM  `${gcp_dataset}.carbon_footprint`
    WHERE $__timeFilter(TIMESTAMP(usage_month))
    GROUP BY    usage_month
    ORDER BY usage_month ASC;

