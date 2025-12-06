FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy source code (invalidates cache on change)
COPY . .


RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mongo-essential .

# -------------------------------
# 2. PROFILING BUILD STAGE
# -------------------------------
FROM golang:1.25-alpine AS profiler_builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o mongo-essential-profile .

# -------------------------------
#  PRODUCTION IMAGE
# ------------------------------
FROM alpine:3.19 AS production

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/mongo-essential .

RUN mkdir -p migrations
COPY .env.example .
COPY examples/ examples/

RUN adduser -D -s /bin/sh migration
USER migration


ENTRYPOINT ["./mongo-essential"]

# --------------------------------
#  For debugging
# --------------------------------
FROM production AS profiling

COPY --from=profiler_builder /app/mongo-essential-profile .

# Entrypoint change to run the profiling binary
ENTRYPOINT ["./mongo-essential-profile"]
