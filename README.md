# sustainability-o11y

This repo houses everything related to sustainability observability at Grafana: data pipelines, dashboards, and documentation for collecting and visualizing cloud carbon emissions across the Cloud Service Providers AWS, GCP, and Azure.

## Data Pipelines

### AWS

Terraform-based pipeline that exports AWS Customer Carbon Footprint Tool (CCFT) data to S3, catalogs it with AWS Glue, and makes it queryable via Athena for Grafana to consume.

See [docs/aws-pipeline.md](docs/aws-pipeline.md).

### GCP

See [docs/gcp-pipeline.md](docs/gcp-pipeline.md).

### Azure

Exporter that fetches Azure carbon emissions data from the Carbon Optimization API and writes it to BigQuery for Grafana to consume.

See [docs/azure-pipeline.md](docs/azure-pipeline.md).

## Carbon Concepts

### Glossary

| Acronym | Term | Definition |
|---------|------|------------|
| CFE | Carbon-free energy | The percentage of renewable energy used as a proportion of total energy used. |
| CO2 | Carbon dioxide | One of the most common greenhouse gases. |
| CO2eq / CO2e | Carbon dioxide equivalent | Carbon is used as a common form of measurement for all greenhouse gases. This unit indicates the potential impact of non-CO2 gases on global warming in carbon terms. |
| gCO2eq/kWh | Grams of carbon per kilowatt hour | The standard unit of carbon intensity. |
| GHGs | Greenhouse gases | Gases that trap heat from solar radiation in the Earth's atmosphere, increasing the temperature on the surface of the Earth. |
| J | Joules | Energy is measured in joules (J). |
| kWh | Kilowatt hours | Energy consumption is measured in kilowatt hours (kWh). |
| PUE | Power usage effectiveness | The metric used to measure data center energy efficiency. |
| SCI | Software Carbon Intensity | A standard that gives an actionable approach to software designers, developers, and operations to measure the carbon impacts of their systems. |

Source: [Green Software Foundation Glossary](https://learn.greensoftware.foundation/glossary/)

### Greenhouse Gas Protocol

Carbon emissions are tracked according to the [Greenhouse Gas Protocol](https://ghgprotocol.org/corporate-standard), the most widely used greenhouse gas accounting standard. It categorizes emissions into three scopes:

- **Scope 1:** All **direct** emissions from owned or controlled sources (e.g., on-site fuel combustion, company-owned vehicles).
- **Scope 2:** **Indirect** emissions from purchased energy generation, calculated using either the Location-Based Method or Market-Based Method.
- **Scope 3:** All other **indirect** emissions in the value chain (upstream and downstream), including emissions from workloads running on CSPs and business travel.

> "Scope 3 can represent over 90% of a company's scope 1, 2 and 3 emissions."
>
> _Source: [GHG Protocol Scope 3 Detailed FAQ](https://ghgprotocol.org/sites/default/files/standards_supporting/Scope%203%20Detailed%20FAQ.pdf)_

### Location-Based Method (LBM) vs Market-Based Method (MBM)

The GHG Protocol defines two methods for calculating Scope 2 emissions from purchased electricity:

**Location-Based Method (LBM):**
* Uses average emission factors for the electricity grids where energy consumption occurs, based on actual regional grid data.
* Provides a baseline view of emissions from electricity consumption in a given location.

**Market-Based Method (MBM):**
* Uses emission factors based on contractual instruments (meaning, supplier-specific energy procurement choices), including:
  * Renewable Energy Certificates (RECs)
  * Power Purchase Agreements (PPAs)
  * Guarantees of Origin (GOs)
  * supplier-specific emission rates
* Reflects the impact of renewable energy purchasing decisions and the carbon savings this leads to.

The GHG Protocol's [Scope 2 Guidance](https://ghgprotocol.org/scope-2-guidance) requires reporting **both methods** when market instruments are used.

### Unit Conversions

Carbon data is surfaced in kilograms of CO₂e (kgCO2e) from some data sources. Dashboards display this as metric tons of CO₂e (MTCO2e), the industry-standard unit:

```
metric_tons_CO2eq = kgCO2e / 1000
```
