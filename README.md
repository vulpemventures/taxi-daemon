# liquid-taxi
![Go](https://github.com/vulpemventures/liquid-taxi/workflows/Go/badge.svg)

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
$ make KEY=<private> MONGO=<uri> STAGE=dev deploy
```

### Production

```sh
$ make KEY=<private> MONGO=<uri> STAGE=production deploy
```



