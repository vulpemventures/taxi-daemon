.PHONY: build clean deploy-dev deploy-prod protobuf run

build: 
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/taxi cmd/server/main.go
	
clean: 
	rm -rf ./bin

deploy-dev: clean build
	sls deploy --verbose --stage dev --region eu-west-1

deploy-prod: clean build
	sls deploy --verbose --stage production --region eu-west-1

run: clean build
	TAXI_LAMBDA_CONTEXT=false TAXI_EXPLORER_ENDPOINT=http://localhost:3001 TAXI_PRIVATE_KEY=1cc080a4cd371eafcad489a29664af6a7276b362fe783443ce036552482b971d ./bin/taxi