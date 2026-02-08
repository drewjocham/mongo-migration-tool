# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

RUN apk --no-cache add git
WORKDIR /src

RUN --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

ARG TARGETOS=TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown


RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath \
      -ldflags="-s -w \
        -X main.appVersion=${VERSION} \
        -X main.commit=${COMMIT} \
        -X main.date=${DATE}" \
      -o /mongo-migration ./cmd/

FROM alpine:3.21 AS final

LABEL org.opencontainers.image.source="https://github.com/drewjocham/mongo-migration-tool"

RUN apk --no-cache add ca-certificates tzdata \
    && adduser -D -u 10001 migration

COPY --from=builder /mongo-migration /usr/local/bin/mongo-migration

USER 10001

ENTRYPOINT ["/usr/local/bin/mongo-migration"]
CMD ["--help"]
