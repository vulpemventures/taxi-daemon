# first image used to build the sources
FROM golang:1.18-buster AS builder

ENV GO111MODULE=on \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /app

COPY . .
RUN go mod download

RUN go build -ldflags="-s -w" -o bin/taxid cmd/taxid/main.go
RUN go build -ldflags="-s -w" -o bin/taxi cmd/taxi/*
RUN go build -ldflags="-s -w" -o bin/fragmenter cmd/fragmenter/*

# Second image, running the executable
FROM debian:buster-slim

# $USER name, and data $DIR to be used in the `final` image
ARG USER=taxi
ARG DIR=/home/taxi

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY --from=builder /app/bin/* /usr/local/bin/

# NOTE: Default GID == UID == 1000
RUN adduser --disabled-password \
            --home "$DIR/" \
            --gecos "" \
            "$USER"
USER $USER

# Prevents `VOLUME $DIR/.taxid/` being created as owned by `root`
RUN mkdir -p "$DIR/.taxid/"

# Expose volume containing all `taxid` data
VOLUME $DIR/.taxid/

# Expose ports of grpc server and profiler
EXPOSE 7000
EXPOSE 8000

ENV XDG_DATA_HOME="/.certmagic"

ENTRYPOINT [ "taxid" ]
