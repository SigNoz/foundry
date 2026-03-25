.PHONY: clean gauge forge cast test docs

clean:
	cd pours/deployment && docker compose -p dev down --remove-orphans --volumes
	cd ../..
	rm -rf ./pours

gauge:
	go run ./cmd/foundryctl gauge --debug -f ./tmp/casting.yaml

forge:
	go run ./cmd/foundryctl forge --debug -f ./tmp/casting.yaml

cast:
	go run ./cmd/foundryctl cast --debug -f ./tmp/casting.yaml

test:
	make forge
	make docker

docs:
	go run ./cmd/foundryctl gen --debug examples
	go run ./cmd/foundryctl gen --debug schemas
