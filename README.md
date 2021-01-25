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



