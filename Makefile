.PHONY: clean gauge forge cast mechanic test gen-examples gen-schemas gen-docs docs

FOUNDRYCTL   := go run ./cmd/foundryctl
GOTMPL       := go run go.opentelemetry.io/build-tools/gotmpl@latest
CASTINGS_JSON = $$(cat docs/examples/castings.json)
NO_LEDGER	:= "--no-ledger"
RESOURCE     ?= signoz/alert/019c8af3-416a-7562-838d-879aec44f566

clean:
	cd pours/deployment && docker compose down --remove-orphans --volumes
	cd ../..
	rm -rf ./pours

gauge:
	$(FOUNDRYCTL) gauge --debug $(NO_LEDGER) -f ./tmp/casting.yaml

forge:
	$(FOUNDRYCTL) forge --debug $(NO_LEDGER) -f ./tmp/casting.yaml

cast:
	$(FOUNDRYCTL) cast --debug $(NO_LEDGER) -f ./tmp/casting.yaml

# Override the resource path with RESOURCE, e.g.
#   make mechanic RESOURCE="signoz alert <id>"
#   make mechanic RESOURCE=telemetrystore/table/distributed_samples_v4
mechanic:
	$(FOUNDRYCTL) mechanic inspect --debug $(NO_LEDGER) -f ./tmp/casting.yaml $(RESOURCE)

test:
	make forge
	make docker

gen-examples:
	$(FOUNDRYCTL) gen --debug $(NO_LEDGER) examples

gen-schemas:
	$(FOUNDRYCTL) gen --debug $(NO_LEDGER) schemas

gen-docs:
	$(FOUNDRYCTL) catalog --debug $(NO_LEDGER) --format json --output "docs/examples/castings.json"
	$(GOTMPL) -b README.md.gotmpl -d "$(CASTINGS_JSON)" -o README.md
	$(GOTMPL) -b docs/examples/README.md.gotmpl -d "$(CASTINGS_JSON)" -o docs/examples/README.md

docs: gen-examples gen-schemas gen-docs
