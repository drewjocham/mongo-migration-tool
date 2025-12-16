# Use a minimal base image
FROM alpine:3.19

# Install certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Set the working directory
WORKDIR /app

# Copy the pre-built binary from the GoReleaser build context
COPY mongo-migration /app/mongo-migration

# Create a non-root user for security
RUN adduser -D -s /bin/sh migration
USER migration

# Set the entrypoint and default command
ENTRYPOINT ["/app/mongo-migration"]
CMD ["--help"]
