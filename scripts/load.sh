#!/bin/bash

wget https://raw.githubusercontent.com/vulpemventures/taxi-protobuf/master/taxi.proto


./bin/taxi config init
./bin/taxi config set rpcserver localhost:7000

./bin/taxi add --asset 0f5243351c492dce4c7a8bc5cd01a69cd8f485f2c7686926a501692575af6835 --price 1.2

# ghz --insecure --proto taxi.proto --call Taxi.ListAssets localhost:8000 --total 800

ghz --insecure --proto taxi.proto --call Taxi.TopupWithAsset -d '{"asset_hash": "0f5243351c492dce4c7a8bc5cd01a69cd8f485f2c7686926a501692575af6835"}' localhost:8000 -t 0 --total 200

rm taxi.proto