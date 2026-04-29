.PHONY: clean gauge forge cast test gen-examples gen-schemas gen-docs docs

FOUNDRYCTL   := go run ./cmd/foundryctl
GOTMPL       := go run go.opentelemetry.io/build-tools/gotmpl@latest
CASTINGS_JSON = $$(cat docs/examples/castings.json)
NO_LEDGER	:= "--no-ledger"

clean:
	cd pours/deployment && docker compose -p dev down --remove-orphans --volumes
	cd ../..
	rm -rf ./pours

gauge:
	$(FOUNDRYCTL) gauge --debug $(NO_LEDGER) -f ./tmp/casting.yaml

forge:
	$(FOUNDRYCTL) forge --debug $(NO_LEDGER) -f ./tmp/casting.yaml

cast:
	$(FOUNDRYCTL) cast --debug $(NO_LEDGER) -f ./tmp/casting.yaml

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
