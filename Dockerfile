ARG GO_VER=1.25
ARG OS_VER=3.22

FROM dhi.io/golang:${GO_VER}-alpine${OS_VER}-dev AS builder

WORKDIR /build
COPY go.mod ./
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -o docker-socket-proxy ./cmd/proxy

FROM scratch AS runner

COPY --from=builder /build/docker-socket-proxy /docker-socket-proxy
COPY policies /policies

USER 65532

EXPOSE 2375
ENTRYPOINT ["/docker-socket-proxy"]

ENV POLICY="/policies/traefik.json"
ENV LISTEN_ADDR=":2375"
ENV DOCKER_SOCKET_PATH="/var/run/docker.sock"

LABEL org.opencontainers.image.description="Docker Socket Proxy"
LABEL org.opencontainers.image.url="https://github.com/andrmr/docker-socket-proxy"
LABEL org.opencontainers.image.source="https://github.com/andrmr/docker-socket-proxy.git"
LABEL org.opencontainers.image.license="MIT"
