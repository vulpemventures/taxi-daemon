# first image used to build the sources
FROM golang:1.17-buster AS builder


ARG TARGETOS
ARG TARGETARCH


WORKDIR /taxi

COPY . .
RUN go mod download


RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/taxid cmd/taxid/*
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/taxi cmd/taxi/*
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o bin/fragmenter cmd/fragmenter/*


# running the executables
FROM debian:buster

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates



COPY --from=builder /taxi/bin/taxid /usr/local/bin/taxid
COPY --from=builder /taxi/bin/taxi /usr/local/bin/taxi
COPY --from=builder /taxi/bin/fragmenter /usr/local/bin/fragmenter

# Copy Ocean binary
COPY --from=ghcr.io/vulpemventures/oceand:v0.1.0-rc.9 /oceand-linux /usr/local/bin/oceand

COPY entrypoint.sh .
RUN chmod +x entrypoint.sh 

# Prevents volumes being created as owned by `root`
RUN useradd -ms /bin/bash user
USER user

RUN mkdir -p "$HOME/.taxid/"
RUN mkdir -p "$HOME/.oceand/"

ENV XDG_DATA_HOME="/.certmagic"
ENV TAXI_DATADIR="/.taxid"

ENTRYPOINT ["/entrypoint.sh"]
