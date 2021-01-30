FROM ghcr.io/vulpemventures/golang:1.15.7-ubuntu20.04

ENV GO111MODULE=on \
  GOOS=linux \
  GOARCH=amd64 \
  CGO_ENABLED=1

WORKDIR /taxi-daemon

COPY go.mod .
RUN go mod download

COPY . .

RUN mkdir -p bin
RUN go build --ldflags '-linkmode external' -o ./bin/taxid cmd/taxid/main.go
RUN go build --ldflags '-linkmode external' -o ./bin/taxi cmd/taxi/*