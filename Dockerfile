# syntax=docker/dockerfile:1

# Builds with the same pinned Go (mise/mise.lock) as local and CI builds.
FROM ghcr.io/jdx/mise:2026.10.13-debian AS builder

# Base mise config only (mise/config.toml: Go), not the dev tools (gopls)
ENV MISE_ENV=""
# Trust the project's mise config (no interactive prompt in a build)
ENV MISE_TRUSTED_CONFIG_PATHS=/src

WORKDIR /src

COPY mise/config.toml mise/mise.lock mise/
RUN mise install --locked

COPY go.mod go.sum ./
RUN mise exec -- go mod download && mise exec -- go mod verify

COPY . .
RUN CGO_ENABLED=0 mise exec -- go build -trimpath -ldflags="-s -w" -o /out/unblogexutoire .

FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.source=https://github.com/LandazuriPaul/unblogexutoire
LABEL org.opencontainers.image.description="Un blog exutoire: a single Go binary with the posts embedded"

COPY --from=builder /out/unblogexutoire /unblogexutoire

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/unblogexutoire"]
