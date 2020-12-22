.PHONY: build clean deploy-dev deploy-prod run test

build-server: 
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/taxid cmd/taxid/main.go
	
build-cli:
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/taxi cmd/taxi/*.go

build: clean build-server build-cli

clean: 
	@echo "Clean..."
	@rm -rf ./bin
	@echo "Clean test datadir..."
	@rm -rf ./taxi-server/test-taxi-datadir

cleantest:
	@echo "Clean test datadir..."
	@rm -rf ./taxi-server/test-taxi-datadir/

## gofmt: Go Format
fmt:
	@echo "Gofmt..."
	@if [ -n "$(gofmt -l .)" ]; then echo "Go code is not formatted"; exit 1; fi

deploy: clean build
	sls deploy --verbose --stage ${STAGE} --region eu-west-1

start: 
	nigiri start --liquid

stop: 
	nigiri stop --delete

run:
	export TAXI_FIXED_FEE_AMOUNT=100000000; \
	export TAXI_EXPLORER_ENDPOINT=http://localhost:3001; \
	export TAXI_MNEMONIC="clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"; \
	./bin/taxid

test: fmt 
	export TAXI_FIXED_FEE_AMOUNT=100000000; \
	export TAXI_EXPLORER_ENDPOINT=http://localhost:3001; \
	export TAXI_MNEMONIC="clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"; \
	go test -v -short -race ./...

servertest: fmt
	export TAXI_FIXED_FEE_AMOUNT=100000000; \
	export TAXI_EXPLORER_ENDPOINT=http://localhost:3001; \
	export TAXI_MNEMONIC="clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"; \
	go test -v -count=1 ./cmd/taxid;

integrationtest: servertest	cleantest
