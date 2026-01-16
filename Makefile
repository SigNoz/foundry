clean:
	cd pours && docker compose down --remove-orphans --volumes
	cd ..
	rm -rf ./pours

gauge:
	go run ./cmd/foundryctl gauge --debug -f ./tmp/casting.yaml

forge:
	go run ./cmd/foundryctl forge --debug -f ./tmp/casting.yaml

cast:
	go run ./cmd/foundryctl cast --debug -f ./tmp/casting.yaml

docker:
	cd pours && docker-compose up -d

test:
	make forge
	make docker
