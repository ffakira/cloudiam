# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    go build \
    -trimpath \
    -buildvcs=false \
    -ldflags="-s -w -extldflags '-static'" \
    -o /out/access-server \
    ./cmd/access-server

FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="CloudIAM server" \
      org.opencontainers.image.description="CloudIAM MVP gRPC server" \
      org.opencontainers.image.source="github.com/ffakira/cloudiam"

COPY --from=build /out/access-server /usr/local/bin/access-server

WORKDIR /home/nonroot
ENV HOME=/home/nonroot

EXPOSE 50051 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/usr/local/bin/access-server", "healthcheck", "--url", "http://127.0.0.1:8080/healthz"]

ENTRYPOINT ["/usr/local/bin/access-server"]
CMD ["--addr", "0.0.0.0:50051", "--config", "/etc/cloudiam/cloudiam.yaml"]
