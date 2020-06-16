.PHONY: protobuf 

protobuf:
	protoc --twirp_out=. --go_out=. ./rpc/taxi/service.proto