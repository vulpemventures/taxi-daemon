# Instructions

Instructions to start a taxi daemon locally on regtest network.

## Requirements

- [Go](https://go.dev/dl/)
- [Nigiri](https://nigiri.vulpem.com/#install)

## Setup

Start Nigiri w/ Liquid services:

```bash
$ nigiri start --liquid
```

From the root folder of the project, run `./scripts/run-regtest` to start oceand and taxid containers.

## Run examples

```bash
# topup-with-asset example
$ go run examples/topup-with-asset/main.go

# topup-with-known-payout example
$ go run examples/topup-with-known-payout/main.go
```

## Stop services

```bash
# stop containers
$ docker-compose -f resources/compose-regtest/docker-compose.yml stop

# stop and remove containers
$ docker-compose -f resources/compose-regtest/docker-compose.yml down
```