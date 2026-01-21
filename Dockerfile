FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS builder

WORKDIR /build

ENV GOTOOLCHAIN=auto

ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
    -o netdebug \
    ./cmd/netdebug

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
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

# Grant CAP_NET_RAW to allow raw ICMP sockets when running as non-root
RUN setcap cap_net_raw=+ep /usr/local/bin/netdebug

RUN useradd -r -u 1000 -g root netdebug

USER netdebug

ENTRYPOINT ["/usr/local/bin/netdebug"]
CMD ["--help"]
