clean:
	cd pours/docker && docker compose down --remove-orphans --volumes && cd ../.. && rm -rf pours

gauge:
	go run ./cmd/foundryctl gauge -f ./tmp/casting.yaml

forge:
	go run ./cmd/foundryctl forge -f ./tmp/casting.yaml

docker:
	cd pours/docker && docker-compose up -d

test:
	make forge
	make docker
