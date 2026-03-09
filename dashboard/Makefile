.PHONY: generate check-drift fmt lint

generate: ## Generate carbon_emissions_report.json from dashboard code
	go run ./export

check-drift: generate ## Check that carbon_emissions_report.json is up to date
	@(git diff --exit-code 'carbon_emissions_report.json') && \
		( echo 'No drift detected'; exit 0 ) || \
		( echo 'ERROR: carbon_emissions_report.json is out of date, run "make generate" and commit the result' && exit 1 )

fmt: ## Run go formatter
	go fmt ./...

lint: ## Run linter
	golangci-lint run
