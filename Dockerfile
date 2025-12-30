FROM golang:1.23-bookworm AS builder

WORKDIR /build

# Set GOTOOLCHAIN to allow auto-download of newer Go versions
ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
    -o netdebug \
    ./cmd/netdebug

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    iputils-ping \
    dnsutils \
    curl \
    iperf3 \
    iproute2 \
    netcat-openbsd \
    tcpdump \
    procps \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/netdebug /usr/local/bin/netdebug

RUN useradd -r -u 1000 -g root netdebug

USER netdebug

ENTRYPOINT ["/usr/local/bin/netdebug"]
CMD ["--help"]
