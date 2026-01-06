clean:
	rm -rf pours

gauge:
	go run ./cmd/foundryctl gauge -f ./tmp/casting.yaml

forge:
	go run ./cmd/foundryctl forge -f ./tmp/casting.yaml

docker:
	cd pours/docker && docker-compose up -d

docker-clean:
	docker stop `docker ps -aq` && docker rm `docker ps -aq`
