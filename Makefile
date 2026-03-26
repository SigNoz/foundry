.PHONY: clean gauge forge cast test gen-examples gen-schemas gen-docs docs

FOUNDRYCTL   := go run ./cmd/foundryctl
GOTMPL       := go run go.opentelemetry.io/build-tools/gotmpl@latest
CASTINGS_JSON = $$(cat docs/examples/castings.json)

clean:
	cd pours/deployment && docker compose -p dev down --remove-orphans --volumes
	cd ../..
	rm -rf ./pours

gauge:
	$(FOUNDRYCTL) gauge --debug -f ./tmp/casting.yaml

forge:
	$(FOUNDRYCTL) forge --debug -f ./tmp/casting.yaml

cast:
	$(FOUNDRYCTL) cast --debug -f ./tmp/casting.yaml

test:
	make forge
	make docker

gen-examples:
	$(FOUNDRYCTL) gen --debug examples

gen-schemas:
	$(FOUNDRYCTL) gen --debug schemas

gen-docs:
	$(GOTMPL) -b README.md.gotmpl -d "$(CASTINGS_JSON)" -o README.md
	$(GOTMPL) -b docs/examples/README.md.gotmpl -d "$(CASTINGS_JSON)" -o docs/examples/README.md

docs: gen-examples gen-schemas gen-docs
