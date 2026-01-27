clean:
	cd pours/deployment && docker compose down --remove-orphans --volumes
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

gen:
	go run ./cmd/foundryctl/*.go gen
