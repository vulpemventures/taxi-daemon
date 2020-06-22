# liquid-taxi
🚕 Liquid Taxi backend 

## Development

### Requirements

* Go
* Docker
* Docker Compose


### Clone

```sh
$ git clone https://github.com/vulpemventures/liquid-taxi
```

## Run

```sh
$ make start # Run Nigiri and Mongo
$ make run
$ make stop # Stop and delete the containers
```

### Run test 

```sh
$ make start  # Run Nigiri and Mongo
$ make test
$ make stop # Stop and delete the containers
```

## Deployment 

### Requirements

* Go
* Serverless CLI
* AWS CLI

### Staging

```sh
$ make deploy-dev
```

### Production

```sh
$ make deploy-prod
```



