.PHONY: build clean deploy-dev deploy-prod run test

build: 
	export GO111MODULE=on
	env GOOS=linux go build -ldflags="-s -w" -o bin/taxi cmd/server/main.go
	
clean: 
	rm -rf ./bin

deploy: clean build
	sls deploy --verbose --stage ${STAGE} --region eu-west-1


faucet:
	curl -X POST -d '{"address": "ert1qlg343tpldc4wvjxn3jdq2qs35r8j5yd5kjfrrt"}' http://localhost:3001/faucet

start: 
	nigiri start --liquid
	docker run -d --network host --name mongo-box mongo

stop: 
	nigiri stop --delete
	docker stop mongo-box 
	docker rm mongo-box

run: clean build faucet
	export TAXI_LAMBDA_CONTEXT=false; \
	export TAXI_MONGO_ENDPOINT=mongodb://127.0.0.1:27017/test; \
	export TAXI_EXPLORER_ENDPOINT=http://localhost:3001; \
	export TAXI_PRIVATE_KEY=1cc080a4cd371eafcad489a29664af6a7276b362fe783443ce036552482b971d; \
	./bin/taxi


test: clean build faucet
	export TAXI_LAMBDA_CONTEXT=false; \
	export TAXI_MONGO_ENDPOINT=mongodb://127.0.0.1:27017/test; \
	export TAXI_EXPLORER_ENDPOINT=http://localhost:3001; \
	export TAXI_PRIVATE_KEY=1cc080a4cd371eafcad489a29664af6a7276b362fe783443ce036552482b971d; \
	go test -v -race ./...