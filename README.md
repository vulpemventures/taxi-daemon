# liquid-taxi
![Go](https://github.com/vulpemventures/liquid-taxi/workflows/Go/badge.svg)

🚕 Liquid Taxi backend 

## Development

### Requirements

* Go
* Docker
* Docker Compose
* Nigiri


### Clone

```sh
$ git clone https://github.com/vulpemventures/liquid-taxi
```

## Run

```sh
$ make build # build binary server + admin cli
$ make start # Run Nigiri
$ make run
$ make stop # Stop and delete the containers
```

### Run test 

```sh
$ make start  # Run Nigiri
$ make test # Run short unit tests
$ make integrationtest # Run integration tests
$ make stop # Stop and delete the containers
```

## Deployment 

### Fragment script

Taxi needs utxos with a fixed amount (i.e fragment size). `scripts/fragment.py` is a Python script used to create these fragments from an Elements/Liquid node.

 NOTICE: the number of addresses specified as argument determines the number of fragments created.
```sh
python3 scripts/fragment.py --url http://127.0.0.1:7041 \
    --rpc_user admin1 \
    --rpc_pass 123 \
    --fragment_size 0.0001 \
    --addresses XQFscDmxh6N9L5SknW6XxDHvjie4xfH8Bu XNqdSvixmaVqgrgEEpSA9s1EEtGfPoAZoH 
```

tips: use `-h` to print the script's help guide.

## Requirements

* Go
* Python 3 (fragment script)




