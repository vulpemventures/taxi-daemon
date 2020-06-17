.PHONY: build clean protobuf

build: 
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/taxi cmd/server/main.go

clean: 
	rm -rf ./bin

protobuf:
	protoc --twirp_out=. --go_out=. ./rpc/taxi/service.proto


deploy-dev: clean build
	sls deploy --verbose --stage dev --region eu-west-1

deploy-prod: clean build
	sls deploy --verbose --stage production --region eu-west-1