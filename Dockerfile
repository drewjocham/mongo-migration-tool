
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
# GoReleaser places the platform-specific binary at the root of the Docker build
# context, so we just copy it into the image. For local builds run
#   GOOS=linux GOARCH=$(go env GOARCH) go build -o mongo-migration .
# first so the binary exists in the context.
COPY mongo-migration /usr/local/bin/mongo-migration

RUN adduser -D -s /bin/sh migration
USER migration

ENTRYPOINT ["/usr/local/bin/mongo-migration"]
CMD ["--help"]
