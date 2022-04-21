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

* Initialize and unlock the ocean wallet
```sh
$ cd compose
# Start the ocean wallet with a list of node peers to connect to.
$ NODE_PEERS=<peers> NETWORK=<network> docker-compose up -d oceand
# Configure the CLI and create a new wallet
$ ocean config init --no-tls
$ ocean wallet create --password <password>
# Make sure the backup the mnemonic of the wallet in response.
$ ocean wallet unlock --password <password>
```

* Run taxi daemon:
```sh
# Start taxi
$ EMAIL=info@example.com DOMAIN=taxi.example.com EXTERNAL_URL=taxi.example.com ADMIN_EXTERAL_URL="taxi.example.com:7000" docker-compose up -d taxid

# If everything's ok, 2 new wallet's accounts 'lbtc" and 'payout' should have been created.
$ ocean wallet info
# {
#   <...>
#   "accounts": [s
#     {
#       "name": "lbtc",
#       "index": 0,
#       "derivationPath": "m/84'/0'/0'",
#       "xpub": "<...>"
#     },
#     {
#       "name": "payout",
#       "index": 1,
#       "derivationPath": "m/84'/0'/1'",
#       "xpub": "<...>"
#     }
#   ]
# }
```


### Tear down infrastructure

**WARNING** Backup taxid data directory and TLS certificates/keys before performing this action

```sh
make infra-down
```


## Usage

### Deposit LBTC

* Generate some receiving addresses from your Ocean wallet:
```sh
# in this example we derive 10 new addresses for the account named "lbtc".
# This is the name of the wallet account taxi uses to pay for network fees,
# and you should take care of keeping it funded with LBTC fragments intended
# to be used for covering fee amounts without any change in return.
$ ocean account derive --account-name lbtc --num-addresses 10
```

* Fragmenter

The fragmenter is a utility service that makes use of an Elements node to send some funds to a list of receiving addresses with a single transaction. It requires an Elements node with a wallet funded with LBTCs.

Compile the binary with (you can skip this step if you previously used `make build`):

```sh
$ make build-fragmenter
```

Use the fragmenter by providing a list of outputs as a list of comma separated pairs of `<addr>:<amount_in_sats>`:

```sh
$ ./bin/fragmenter --rpcserver http://admin1:123@localhost:18884 --outputs "el1qqtqts296m20vhptp2tens8nh40nn62h7hg04cpml3zntwy75qnlwetzywyyes0yayhzhr8un43vj4rxd28aeurcnkfwp07pc3=500","el1qq2rgldhypkjqkehnkypyhr0vhjsq6780sgp9kmne5ykwz3w8ur3d340puy2fx5ch8xkxssf4cpwz98a6ry4m7kqmwk7c92w6z=500,el1qqtcrkd0che6ed97muhy62w0wmv7lnjl8wj0e9swn2jf36y7mftymtkxkwhfmcjeud68sd35p3a4y9cnwfe4m6mp3ashr8tp3d=500,el1qqfjzrkawqz4usfgepc25wstleczknlpjst8gzu4tel0rwdnvjyctnzf9d385nzadvft2mh58s0k7ndg6y8d6shed30pvzdhl6=500,el1qqgqndw32cs6zyr6qhfhd8ppuzx3aku02lesuqg332ngppquakl5cxtsd0m45h08davmk5u604l8cuyv59c4pu6qagmvx77ajt=700,el1qqvxvjt5sscyy5pkdgz26mrx797sz7acks8rtrhvnx2vm9tsvx0a07raj0t583xv597tkew4sxqwsswhx8nz6692jz8htuywnp=700,el1qqfqvmj5yvlehk2mv4a23edqk5am2u95awqcp8ke3x5fkv0pxhxxd23nk862x0xljj5m8q0lyuc3rqa9fvy2tclqstdkwkngty=700,el1qq0et3gx2uzj2jwdp6y48yq0qvhnejxcj3w2clft372ke7zuaj67nudt7jl65r855e7mf8hmzcdfaherxyantqu8duc3umntwt=1000,el1qqgur9ttlcsrdq7kl8m3xqr9jyds3kjfw7zk2dg9qh5q64yrwsvf42yqf2fcw9ewr5hwu7gek9kt6hkhwgtmk5xzzh8g0r2ycd=1000,el1qq2usnr5penldzjtje8rvdk34afp5rlx6p94l5t46nugjkh2630e4ypzvgt55n6t4lf7msazutxrfkrm9vver5h32ve0y8hwku=1500"
```

### Configure Taxi CLI

```sh
$ taxi config init
# You can connect to an insecure taxi service (only for dev purposes)
$ taxi config init --insecure
# You can connect to a remote taxi service with the CLI installed on your local machine
$ taxi config set rpcserver myremoteurl:7000
```

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
$ make test # Run tests
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
$ docker run -d -it --name taxid -p 7000:7000 -p 8000:8000 -v `pwd`/taxid:/.taxid -e TAXI_OCEAN_ADDRESS="<ocean wallet address>" taxid:latest
```

#### Use the CLI

```bash
$ alias taxi="docker exec -it taxid taxi"
```

