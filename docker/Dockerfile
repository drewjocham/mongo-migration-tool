FROM alpine:3.23.0

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY mongo-migration /usr/local/bin/mongo-migration

RUN adduser -D -s /bin/sh migration
USER migration

ENTRYPOINT ["/usr/local/bin/mongo-migration"]
CMD ["--help"]
