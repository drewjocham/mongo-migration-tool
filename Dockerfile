# syntax=docker/dockerfile:1.4

# Builder Stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first for efficient caching
COPY go.mod go.sum ./

# Download modules
RUN go mod download && go mod tidy

# Copy the rest of the application source code
COPY . .

# --- DEBUG STEP: Verify migrations directory content ---
RUN echo "--- Contents of /app/migrations ---" && \
    ls -l /app/migrations && \
    echo "--- Content of /app/migrations/main.go ---" && \
    cat /app/migrations/main.go && \
    echo "--- Content of /app/migrations/20240101_001_add_user_indexes.go ---" && \
    cat /app/migrations/20240101_001_add_user_indexes.go && \
    echo "--- END DEBUG ---"

# Build production binary
# CGO_ENABLED=0 GOOS=linux are standard for static binaries in Alpine
# Explicitly build the ./cmd package
RUN CGO_ENABLED=0 GOOS=linux go build -v -ldflags="-w -s" -o /app/mongo-essential ./cmd

# Build profiling/debug binary (with gcflags for debugging info)
# Explicitly build the ./cmd package
RUN CGO_ENABLED=0 GOOS=linux go build -v -gcflags="all=-N -l" -o /app/mongo-essential-profile ./cmd

# -------------------------------
# Production Image
# ------------------------------
FROM gcr.io/distroless/static-debian12:debug-nonroot AS production

WORKDIR /app

# Copy production binary
COPY --from=builder /app/mongo-essential /app/mongo-essential

# Add non-root user (distroless images often run as non-root by default, but good to be explicit)
USER 65532:65532

ENTRYPOINT ["/app/mongo-essential"]

# Add healthcheck (assuming 'status' is a lightweight command)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/mongo-essential", "status"]

# --------------------------------
# Debug/Profiling Image
# --------------------------------
FROM gcr.io/distroless/static-debian12:debug-nonroot AS profiling

WORKDIR /app

# Copy profiling binary
COPY --from=builder /app/mongo-essential-profile /app/mongo-essential-profile

# Add non-root user
USER 65532:65532

ENTRYPOINT ["/app/mongo-essential-profile"]

# Add healthcheck (assuming 'status' is a lightweight command)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/mongo-essential-profile", "status"]
