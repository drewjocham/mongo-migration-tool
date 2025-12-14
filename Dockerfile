# syntax=docker/dockerfile:1.4

# Stage 1: Build the Go application
FROM golang:1.25-alpine AS builder

# Install git, which is needed by go mod download for some modules
RUN apk --no-cache add git

WORKDIR /app

# First, copy go.mod and go.sum to leverage Docker layer caching
COPY go.mod go.sum ./

# Download external dependencies
RUN go mod download

# Now, copy the entire source code, including local packages
COPY . .

# Build the application, targeting the current directory
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o /app/mongo-migration .

# Final Stage: Production image for the Go application
FROM alpine:3.19 AS production

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the Go binary from the builder stage
COPY --from=builder /app/mongo-migration /app/mongo-migration

RUN adduser -D -s /bin/sh migration
USER migration

ENTRYPOINT ["/app/mongo-migration"]
CMD ["--help"]
