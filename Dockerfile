# first image used to build the sources
FROM golang:1.15.5-buster AS builder

ENV GO111MODULE=on \
  GOOS=linux \
  CGO_ENABLED=1 \
  GOARCH=amd64

WORKDIR /taxi

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o taxid cmd/taxid/main.go
RUN go build -ldflags="-s -w" -o taxi cmd/taxi/*

WORKDIR /build

RUN cp /taxi/taxid .
RUN cp /taxi/taxi .

# Second image, running the executable
FROM debian:buster

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY --from=builder /build/taxid /taxid
COPY --from=builder /build/taxi /taxi


ENV XDG_DATA_HOME="/.certmagic"
ENV TAXI_DATADIR="/.taxid"

ENTRYPOINT ["/taxid"]
