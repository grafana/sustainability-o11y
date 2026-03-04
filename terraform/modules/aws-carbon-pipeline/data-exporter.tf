resource "aws_bcmdataexports_export" "carbon" {
  export {
    name = var.export_name

    data_query {
      query_statement = "SELECT last_refresh_timestamp, location, model_version, payer_account_id, product_code, region_code, total_lbm_emissions_unit, total_lbm_emissions_value, total_mbm_emissions_unit, total_mbm_emissions_value, total_scope_1_emissions_unit, total_scope_1_emissions_value, total_scope_2_lbm_emissions_unit, total_scope_2_lbm_emissions_value, total_scope_2_mbm_emissions_unit, total_scope_2_mbm_emissions_value, total_scope_3_lbm_emissions_unit, total_scope_3_lbm_emissions_value, total_scope_3_mbm_emissions_unit, total_scope_3_mbm_emissions_value, usage_account_id, usage_period_end, usage_period_start FROM CARBON_EMISSIONS"

      table_configurations = {
        "CARBON_EMISSIONS" = {}
      }
    }

    destination_configurations {
      s3_destination {
        s3_bucket = aws_s3_bucket.carbon.id
        s3_prefix = var.s3_prefix
        s3_region = var.region

        s3_output_configurations {
          overwrite   = "OVERWRITE_REPORT"
          format      = "PARQUET"
          compression = "PARQUET"
          output_type = "CUSTOM"
        }
      }
    }

    refresh_cadence {
      frequency = "SYNCHRONOUS"
    }
  }
}
