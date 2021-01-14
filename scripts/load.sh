#!/bin/bash

wget https://raw.githubusercontent.com/vulpemventures/taxi-protobuf/master/taxi.proto


./bin/taxi config init
./bin/taxi config set rpcserver localhost:7000

asset=$(nigiri rpc --liquid issueasset 1 1 | jq -r '.asset')
./bin/taxi add --asset $asset --price 1.2

ghz --insecure --proto taxi.proto --call Taxi.ListAssets localhost:8000 --total 800

ghz --insecure --proto taxi.proto --call Taxi.TopupWithAsset -d '{"asset_hash": "'"$asset"'"}' localhost:8000 -t 0 --total 2

rm taxi.proto