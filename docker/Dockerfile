# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

RUN apk --no-cache add git
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-s -w \
        -X github.com/drewjocham/mongo-migration-tool/internal/cli.appVersion=${VERSION} \
        -X github.com/drewjocham/mongo-migration-tool/internal/cli.commit=${COMMIT} \
        -X github.com/drewjocham/mongo-migration-tool/internal/cli.date=${DATE}" \
    -o /tmp/mongo-migration ./cmd

FROM alpine:3.23.0

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /tmp/mongo-migration /usr/local/bin/mongo-migration

RUN adduser -D -s /bin/sh migration
USER migration

ENTRYPOINT ["/usr/local/bin/mongo-migration"]
CMD ["--help"]
