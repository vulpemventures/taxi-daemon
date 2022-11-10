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

## vet: code analysis
vet:
	@echo "Vet..."
	@go vet ./...

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
	@buf generate buf.build/vulpemventures/taxi; \
	buf generate buf.build/vulpemventures/ocean; \
	buf generate

## run: runs the service (for development)
run: clean
	@echo "Running service..."
	@export TAXI_LOG_LEVEL=5; \
	export TAXI_OCEAN_ADDRESS=localhost:38000; \
	export TAXI_SWAGGER_PATH=api-spec/protobuf/gen/swagger/taxi/v1; \
	export TAXI_ADMIN_SWAGGER_PATH=api-spec/protobuf/gen/swagger/taxi-daemon/v1; \
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

#### Postgres database ####
## pg: starts postgres db inside docker container
pg:
	docker run --name taxid-pg -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres

## droppg: stop and remove postgres container
droppg:
	docker stop taxid-pg
	docker rm taxid-pg

## createdb: create db inside docker container
createdb:
	docker exec taxid-pg createdb --username=root --owner=root taxid-db

## dropdb: drops db inside docker container
dropdb:
	docker exec taxid-pg dropdb taxid-db

## createtestdb: create test db inside docker container
createtestdb:
	docker exec taxid-pg createdb --username=root --owner=root taxid-db-test

## droptestdb: drops test db inside docker container
droptestdb:
	docker exec taxid-pg dropdb taxid-db-test

## recreatedb: drop and create main and test db
recreatedb: dropdb createdb droptestdb createtestdb

## recreatetestdb: drop and create test db
recreatetestdb: droptestdb createtestdb

## psql: connects to postgres terminal running inside docker container
psql:
	docker exec -it taxid-pg psql -U root -d taxid-db

## migrate: creates pg migration file(eg. make FILE=init migrate)
migrate:
	migrate create -ext sql -dir ./internal/infrastructure/storage/postgres/migration/ $(FILE)

## sqlc: gen sql
sqlc:
	@echo "gen sql..."
	cd internal/infrastructure/storage/postgres; sqlc generate