# syntax=docker/dockerfile:1.4
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .

COPY go.mod go.sum ./
RUN GO111MODULE=on go mod download && go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o /app/main .

WORKDIR /app

# -------------------------------
FROM gcr.io/distroless/static-debian12:debug-nonroot AS final

COPY --from=builder /app/main /app/main

RUN adduser -D -s /bin/sh migration

USER 65532:65532
ENTRYPOINT ["/app/main"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/main", "status"]
