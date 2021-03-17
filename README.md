# taxi-daemon
![Go](https://github.com/vulpemventures/taxi-daemon/workflows/Go/badge.svg)

🚕 Liquid Taxi backend 

## Index 

1. [Deployment](#deployment)
2. [Usage](#usage)
3. [Development](#development)
4. [Docker](#docker)

## Deployment

Taxi Daemon is deployed onto a Hetzner Cloud VM in a box with 1vCPU, 2GB RAM, 20GB NVMe SSD, 20TB Traffic

### Prepare the infrastructure

In order to deploy you need to provide the following:

* Hetzner API Token `hcloud_token`
* Server name `server_name`
* SSH Public Key `server_sshkey`

```sh
$ make infra-up
```

This can be run against an already deployed infrastructure, it will re-create terraform state locally

### Build and upload

You need to have docker to perform this step. This will save and load the docker image up to the machine via ssh

```sh
$ make infra-load 
```

### Run

* Enter into the machine via SSH

```sh
$ ssh username@host
```

* start the taxi daemon 

```sh
$ cd compose
$ TAXI_MNEMONIC="<your words>" docker-compose up -d
```

* get logs 

```sh
docker logs taxid
```


### Tear down infrastructure

**WARNING** Backup taxid data directory and TLS certificates/keys before performing this action

```sh
make infra-down
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

Use the previous addresses to send an exact amount defined in `TAXI_FIXED_FEE_AMOUNT` (by default 0.00001 LBTC / 1000 staoshis)

If running an Elements instance, you can use the `fragment.py` script to make a deposit to many addresses in a single transaction.

* Fragment script

`resources/fragment/fragment.py` is a Python script used to create these fragments from an Elements/Liquid node.

NOTICE: the number of addresses specified as argument determines the number of fragments created.

```sh
$ git clone git@github.com:vulpemventures/taxi-daemon.git && cd taxi-daemon
$ python3 ./resources/fragment/fragment.py --url http://127.0.0.1:7041 \
    --rpc_user admin1 \
    --rpc_pass 123 \
    --fragment_size 0.00001 \
    --addresses XQFscDmxh6N9L5SknW6XxDHvjie4xfH8Bu XNqdSvixmaVqgrgEEpSA9s1EEtGfPoAZoH 
```

tips: use `-h` to print the script's help guide.


### Add an asset and price

```sh
# Add Tether asset hash, make the USDt price per BTC of 34000 and use 10% as service fee for taxi
$ taxi add --asset ce091c998b83c78bb71a632313ba3760f1763d9cfcffae02258ffa9865a37bd2 --price 34000 --basis_point 1000
```


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

### Build 

```sh
# Run Nigiri first
$ make build
```


### Run for development

```sh
# Run Nigiri first
$ make run
```

### Test 

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
$ docker build --pull --rm -f "Dockerfile" -t taxid:latest "."
```

#### Run the taxi server 

For in depth documentation of the environment variables look [here](https://github.com/vulpemventures/taxi-daemon/blob/master/config/config.go)

```bash
$ docker run -d -it --name taxid -p 7000:7000 -p 8000:8000 -v `pwd`/taxid:/.taxid -e TAXI_MNEMONIC="<seed words here>" taxid:latest
```

Run in regtest with Nigiri

```
$ docker run -d -it --name taxid --network host -v `pwd`/taxid:/.taxid -e TAXI_MNEMONIC="<seed words here>" -e TAXI_NETWORK=regtest -e TAXI_EXPLORER_ENDPOINT='http://127.0.0.1:3001' -e TAXI_CACHE_INTERVAL=5 -e TAXI_LOG_LEVEL=9 ghcr.io/vulpemventures/taxid:latest
```

#### Use the CLI

```bash
$ alias taxi="docker exec -it taxid taxi"
```

