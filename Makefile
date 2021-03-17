.PHONY: build clean deploy-dev deploy-prod run test proto

## build the taxi server
build-server: 
	export GO111MODULE=on; \
	env go build -tags netgo -ldflags="-s -w" -o bin/taxid cmd/taxid/main.go

## build only the taxi CLI
build-cli:
	export GO111MODULE=on; \
	env go build -tags netgo -ldflags="-s -w" -o bin/taxi cmd/taxi/*.go

## build the project (CLI + server)
build: clean build-server build-cli

## buildlinux: use a docker container with ubuntu to build binaries and move into bin folder
buildlinux: 	
	chmod u+x ./scripts/buildlinux
	bash ./scripts/buildlinux

## remove the compiled binaries in ./bin
clean: 
	@echo "Clean..."
	@rm -rf ./bin

## remove the test data directory
cleantest:
	@echo "Clean test datadir..."
	@rm -rf ./cmd/taxid/test-taxi-datadir/

## gofmt: Go Format
fmt:
	@echo "Gofmt..."
	@if [ -n "$(gofmt -l .)" ]; then echo "Go code is not formatted"; exit 1; fi

## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## generate protobuf admin stubs
proto:
	chmod u+x ./scripts/compile-proto
	./scripts/compile-proto

## run the service (for tests)
run:
	export TAXI_CACHE_INTERVAL=5; \
	export TAXI_NETWORK=regtest; \
	export TAXI_EXPLORER_ENDPOINT=http://localhost:3001; \
	export TAXI_LOG_LEVEL=9; \
	export TAXI_MNEMONIC="clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"; \
	go run cmd/taxid/main.go

## run all the tests
alltest: fmt 
	export TAXI_NETWORK=regtest; \
	export TAXI_EXPLORER_ENDPOINT=http://localhost:3001; \
	export TAXI_MNEMONIC="clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"; \
	export TAXI_DATADIR="./test-taxi-datadir"; \
	go test -v -race ./...

## run the short tests
test: fmt 
	export TAXI_NETWORK=regtest; \
	export TAXI_EXPLORER_ENDPOINT=http://localhost:3001; \
	export TAXI_MNEMONIC="clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"; \
	export TAXI_DATADIR="./test-taxi-datadir"; \
	go test -v -short -race ./...

## integration test
servertest: fmt
	export TAXI_NETWORK=regtest; \
	export TAXI_EXPLORER_ENDPOINT=http://localhost:3001; \
	export TAXI_MNEMONIC="clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"; \
	export TAXI_DATADIR="./test-taxi-datadir"; \
	export TAXI_CACHE_INTERVAL=1; \
	go test -v -count=1 ./cmd/taxid;

## integration test + cleaning test datadir
integrationtest: servertest	cleantest

## create-cert: creates localhost ssl certficate and key
create-cert:
	chmod u+x ./scripts/sslcert
	bash ./scripts/sslcert

## infra-up: create infra on Hetzner with Docker & Docker-Compose
infra-up:
	terraform -chdir=./resources/terraform  init
	terraform -chdir=./resources/terraform  apply -auto-approve

## infra-down: tear down infra on Hetzner
infra-down:
	terraform -chdir=./resources/terraform  destroy -auto-approve

## infra-load: load remote infra
infra-load:
	chmod u+x ./scripts/provision-docker
	bash ./scripts/provision-docker
