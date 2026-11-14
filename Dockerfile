FROM golang:1.25-bookworm AS backend

ENV CGO_ENABLED=0
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY engine/ ./engine/
COPY imagecache/ ./imagecache/
COPY internal/ ./internal/
COPY queue/ ./queue/
COPY db/ ./db/
ARG KOITO_VERSION=0.9
RUN go build -trimpath -ldflags "-s -w -X main.Version=$KOITO_VERSION" -o /tunetrail ./cmd/api

FROM debian:bookworm-slim AS final

WORKDIR /app
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*
COPY --from=backend /tunetrail ./tunetrail
COPY assets/ ./assets/
COPY langs/ ./langs/
ARG KOITO_VERSION=0.9
LABEL org.opencontainers.image.title="TuneTrail" \
      org.opencontainers.image.description="ListenBrainz-compatible scrobbling with simple public listening profiles" \
      org.opencontainers.image.source="https://github.com/ebrgr/TuneTrail" \
      org.opencontainers.image.version=$KOITO_VERSION
ENV TUNETRAIL_LANGS_DIR=/app/langs \
    KOITO_CONFIG_DIR=/etc/koito \
    KOITO_BIND_ADDR=0.0.0.0 \
    KOITO_LISTEN_PORT=4110
EXPOSE 4110
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD curl --fail --silent http://127.0.0.1:4110/apis/web/v1/health || exit 1
ENTRYPOINT ["./tunetrail"]
