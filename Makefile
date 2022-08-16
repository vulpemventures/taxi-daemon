.PHONY: build build-server build-cli build-fragmenter build-linux clean fmt proto-lint proto run alltest test servertest integrationtest create-cert infra-up infra-down infra-load

## build-server: builds taxid into ./bin
build-server:
	@echo "Building taxid..."
	@export GO111MODULE=on; \
	env go build -tags netgo -ldflags="-s -w" -o bin/taxid cmd/taxid/main.go

## build-cli: builds taxi CLI into ./bin
build-cli:
	@echo "Building CLI..."
	@export GO111MODULE=on; \
	env go build -tags netgo -ldflags="-s -w" -o bin/taxi cmd/taxi/*.go

## build-fragmenter: builds fragmenter into ./bin
build-fragmenter:
	@echo "Building fragmenter..."
	@export GO111MODULE=on; \
	env go build -tags netgo -ldflags="-s -w" -o bin/fragmenter cmd/fragmenter/*.go

## build: builds all binaries into ./bin
build: clean build-server build-cli build-fragmenter

## build-linux: uses a dockerized Linux environment to build binaries into ./bin
build-linux:
	@echo "Building taxid Linux..."
	@chmod u+x ./scripts/buildlinux
	@bash ./scripts/buildlinux

## clean: remove the compiled binaries
clean: 
	@echo "Cleaning..."
	@go clean
	@rm -rf ./bin

## gofmt: runs Go Format
fmt:
	@echo "Gofmt..."
	@if [ -n "$(gofmt -l .)" ]; then echo "Go code is not formatted"; exit 1; fi

## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## proto-lint: lints protos
proto-lint:
	@echo "Linting protos..."
	@buf lint

## proto: compiles proto stubs
proto: proto-lint
	@echo "Compiling stubs..."
	@buf generate buf.build/vulpemventures/taxi && buf generate

## run: runs the service (for development)
run: clean
	@echo "Running service..."
	@export TAXI_LOG_LEVEL=5; \
	export TAXI_OCEAN_ADDRESS=localhost:18000; \
	go run ./cmd/taxid

## test: runs short unit tests
test: fmt 
	@echo "Unit testing..."
	@go test -v -race ./...

## create-cert: creates localhost TLS certficate and key
create-cert:
	@echo "Generating TLS keypair..."
	@chmod u+x ./scripts/sslcert
	@bash ./scripts/sslcert

## infra-up: creates infra on Hetzner with Docker & Docker-Compose
infra-up:
	@echo "Building infrastructure (Terraform)..."
	@terraform -chdir=./resources/terraform  init
	@terraform -chdir=./resources/terraform  apply -auto-approve

## infra-down: tears down infra on Hetzner
infra-down:
	@echo "Tearing down infrastructure (Terraform)..."
	@terraform -chdir=./resources/terraform  destroy -auto-approve

## infra-load: loads taxid binary to remote host
infra-load:
	@echo "Loading binary to remote host..."
	@chmod u+x ./scripts/provision-docker
	@bash ./scripts/provision-docker
