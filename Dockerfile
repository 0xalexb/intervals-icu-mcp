# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG DI_VERSION=0.5.0
ARG COMPILED_AT

RUN apk add --no-cache ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY src/ ./src/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags "-s -w \
        -X github.com/0xalexb/hjarta-di.Version=${VERSION} \
        -X github.com/0xalexb/hjarta-di.DIVersion=${DI_VERSION} \
        -X github.com/0xalexb/hjarta-di.CompiledAt=${COMPILED_AT}" \
      -o /out/intervals-icu-mcp \
      ./src

FROM scratch

ARG VERSION=dev
LABEL org.opencontainers.image.title="intervals-icu-mcp"
LABEL org.opencontainers.image.description="MCP server for Intervals.icu with OAuth 2.1."
LABEL org.opencontainers.image.source="https://github.com/0xalexb/intervals-icu-mcp"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/intervals-icu-mcp /intervals-icu-mcp

EXPOSE 8080

ENTRYPOINT ["/intervals-icu-mcp"]
CMD ["--transport", "streamable", "--address", "0.0.0.0:8080"]
