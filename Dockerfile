# first image used to build the sources
FROM golang:1.17-buster AS builder

ENV GO111MODULE=on \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /taxi

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o ./taxid cmd/taxid/main.go
RUN go build -ldflags="-s -w" -o ./taxi cmd/taxi/*
RUN go build -ldflags="-s -w" -o ./fragmenter cmd/fragmenter/*

# Second image, running the executable
FROM debian:buster

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

COPY --from=builder /taxi/taxid /
COPY --from=builder /taxi/taxi /
COPY --from=builder /taxi/fragmenter /

RUN install /taxi /bin
RUN install /fragmenter /bin

ENV XDG_DATA_HOME="/.certmagic"
ENV TAXI_DATADIR="/.taxid"

ENTRYPOINT ["/taxid"]
