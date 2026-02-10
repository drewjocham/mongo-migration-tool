# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder

RUN apk --no-cache add git
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath \
      -ldflags="-s -w \
        -X github.com/drewjocham/mongo-migration-tool/internal/cli.appVersion=${VERSION} \
        -X github.com/drewjocham/mongo-migration-tool/internal/cli.commit=${COMMIT} \
        -X github.com/drewjocham/mongo-migration-tool/internal/cli.date=${DATE}" \
      -o /out/mongo-tool ./cmd

FROM alpine:3.23.0

RUN apk --no-cache add ca-certificates tzdata \
    && adduser -D -s /bin/sh migration

WORKDIR /app

COPY --from=builder /out/mongo-tool /usr/local/bin/mongo-tool

USER migration

ENTRYPOINT ["/usr/local/bin/mongo-tool"]
CMD ["--help"]
