#!/bin/bash

while getopts a:t: flag
do
    case "${flag}" in
        a) asset=${OPTARG};;
        t) total=${OPTARG};;
    esac
done

echo "load testing script -- a program used to load test the TopupWithAsset request"

echo "asset = $asset"
echo "total = $total"

wget --quiet https://raw.githubusercontent.com/vulpemventures/taxi-protobuf/master/taxi.proto

# ghz --insecure --proto taxi.proto --call Taxi.ListAssets localhost:8000 --total 800
ghz --insecure --proto taxi.proto --call Taxi.TopupWithAsset -d '{"asset_hash": "'"$asset"'"}' localhost:8000 -t 0 --total $total

rm taxi.proto