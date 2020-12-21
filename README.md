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
* make integrationtest # Run integration tests
$ make stop # Stop and delete the containers
```

## Deployment 

### Requirements

* Go





