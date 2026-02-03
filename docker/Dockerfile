# syntax=docker/dockerfile:1.7

FROM alpine:3.23.0

RUN apk --no-cache add ca-certificates tzdata \
    && adduser -D -s /bin/sh migration

WORKDIR /app

# The compiled binary is provided by GoReleaser in the build context.
COPY mongo-migration /usr/local/bin/mongo-migration

USER migration

ENTRYPOINT ["/usr/local/bin/mongo-migration"]
CMD ["--help"]
