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
# The version comes from internal/version/VERSION embedded in the binary; no
# -X override, so a renamed or missing build argument cannot change it (ISSUE-111).
RUN go build -trimpath -ldflags "-s -w" -o /tunetrail ./cmd/api

FROM debian:bookworm-slim AS final

WORKDIR /app
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*
COPY --from=backend /tunetrail ./tunetrail
COPY assets/ ./assets/
COPY langs/ ./langs/
# Image label only; kept in sync with VERSION by scripts/version.py.
ARG TUNETRAIL_VERSION=0.9
LABEL org.opencontainers.image.title="TuneTrail" \
      org.opencontainers.image.description="ListenBrainz-compatible scrobbling with simple public listening profiles" \
      org.opencontainers.image.source="https://github.com/ebrgr/TuneTrail" \
      org.opencontainers.image.version=$TUNETRAIL_VERSION
# No configuration variables are fixed in the image (ISSUE-111): the defaults are
# /etc/koito, all interfaces and port 4110, and a fixed value would conflict
# with a user's own KOITO_* or TUNETRAIL_* setting.
ENV TUNETRAIL_LANGS_DIR=/app/langs
EXPOSE 4110
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD curl --fail --silent http://127.0.0.1:4110/apis/web/v1/health || exit 1
ENTRYPOINT ["./tunetrail"]
