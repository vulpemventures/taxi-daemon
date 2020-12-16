.PHONY: build clean deploy-dev deploy-prod run test

build: 
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/taxi cmd/server/main.go
	
clean: 
	@echo "Clean..."
	@rm -rf ./bin

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

run: clean build 
	export TAXI_FRAGMENT_AMOUNT=100000000; \
	export TAXI_EXPLORER_ENDPOINT=http://localhost:3001; \
	export TAXI_MNEMONIC="clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"; \
	./bin/taxi

test: fmt clean build 
	go test -v -short -race ./...