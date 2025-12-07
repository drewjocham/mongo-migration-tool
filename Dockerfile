FROM golang:1.25-alpine AS builder

WORKDIR /app

# Dependency Caching
COPY go.mod go.sum ./

RUN go mod download

COPY . .

# 4. Final Build
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o mongo-migration .

# PROFILING BUILD STAGE
FROM golang:1.25-alpine AS profiler_builder

WORKDIR /app

# 1. Dependency Caching
COPY go.mod go.sum ./

COPY . .

# Download/Tidy and Build
RUN go mod download && \
    go mod tidy && \
    go build -v -o mongo-migration .

#  PRODUCTION IMAGE
FROM alpine:3.19 AS production

# Setup environment
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy final binary from the builder stage
COPY --from=builder /app/mongo-migration .

RUN mkdir -p migrations
COPY .env.example .

COPY examples/ examples/ 

RUN adduser -D -s /bin/sh migration
USER migration

ENTRYPOINT ["./mongo-migration"]

#  For debugging
FROM production AS profiling

# Copy the profiling-enabled binary
COPY --from=profiler_builder /app/mongo-migration .

ENTRYPOINT ["./mongo-migration"]
