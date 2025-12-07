# syntax=docker/dockerfile:1.4

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod tidy

COPY . .

# Build 1: Production (Optimized, Static)
# Output name: /app/mongo-migration-prod
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o /app/mongo-migration-prod .

# Build 2: Profiling/Debug (Disable optimizations, Enable profiling)
# Output name: /app/mongo-migration-debug
RUN CGO_ENABLED=0 GOOS=linux go build -v -gcflags="all=-N -l" -o /app/mongo-migration-debug .

# -------------------------------
FROM alpine:3.19 AS production

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the specific production binary
COPY --from=builder /app/mongo-migration-prod /app/mongo-migration

RUN adduser -D -s /bin/sh migration
USER migration

ENTRYPOINT ["/app/mongo-migration"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/mongo-migration", "status"]

# --------------------------------
FROM alpine:3.19 AS profiling

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the specific debug binary
COPY --from=builder /app/mongo-migration-debug /app/mongo-migration-profile

RUN adduser -D -s /bin/sh migration
USER migration

ENTRYPOINT ["/app/mongo-migration-profile"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/mongo-migration-profile", "status"]
