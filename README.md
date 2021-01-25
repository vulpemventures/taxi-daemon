# taxi-daemon
![Go](https://github.com/vulpemventures/taxi-daemon/workflows/Go/badge.svg)

🚕 Liquid Taxi backend 

## Development

### Requirements

* Go
* Docker
* Docker Compose
* Nigiri


### Clone

```sh
$ git clone https://github.com/vulpemventures/taxi-daemon
```

## Run for development

```sh
# Run Nigiri first
$ make run
```

### Run test 

```sh
# Run Nigiri first
$ make test # Run short unit tests
$ make integrationtest # Run integration tests
```


## Docker

Build and use `taxid` with docker.

#### Build taxid docker image

_At the root of the repository_

```bash
docker build --pull --rm -f "Dockerfile" -t taxid:latest "."
```

#### Run the taxi server

```bash
docker run -d -it --name taxid -p 7000:7000 -p 8000:8000 -v `pwd`/taxid:/.taxid taxid:latest
```

#### Use the CLI

```bash
alias taxi="docker exec -it taxid taxi"
```

## Usage

### Set the `taxid` RPC admin server into `taxi` CLI

```sh
$ taxi config init
# Optional: default is localhost:7000
$ taxi config set rpcserver http://myremoteurl:7000
```

### Get funding addresses for LBTC

```sh
# This gets 10 addresses. It means we can serve 10 concurrent topups
$ taxi fund -n 10
```

### Deposit LBTC

Use the previous addresses to send an exact amount defined in `TAXI_FIXED_FEE_AMOUNT`

If running an Elements instance, you can use the `fragment.py` script to make a deposit to many addresses in a single transaction.

* Fragment script

`scripts/fragment.py` is a Python script used to create these fragments from an Elements/Liquid node.

NOTICE: the number of addresses specified as argument determines the number of fragments created.

```sh
$ git clone git@github.com:vulpemventures/taxi-daemon.git && cd taxi-daemon
$ python3 ./scripts/fragment.py --url http://127.0.0.1:7041 \
    --rpc_user admin1 \
    --rpc_pass 123 \
    --fragment_size 0.0001 \
    --addresses XQFscDmxh6N9L5SknW6XxDHvjie4xfH8Bu XNqdSvixmaVqgrgEEpSA9s1EEtGfPoAZoH 
```

tips: use `-h` to print the script's help guide.


### Add an asset and price

```sh
# Add Tether asset hash, make the USDt price per BTC of 34000 and use 10% as service fee for taxi
$ taxi add --asset ce091c998b83c78bb71a632313ba3760f1763d9cfcffae02258ffa9865a37bd2 --price 34000 --basis_point 1000
```
