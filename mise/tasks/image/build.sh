#!/usr/bin/env bash
#MISE description="Build the Docker image, tagged with the short commit hash (+ -dirty) and latest"
#
# Examples:
#   mise run image:build
#   mise run image:run    # build, then serve it on :8080
#   mise run image:push   # build, then push to ghcr.io (refuses -dirty tags)
set -euo pipefail

image="ghcr.io/landazuripaul/unblogexutoire"
root="${MISE_PROJECT_ROOT:-$(git rev-parse --show-toplevel)}"

tag="$(git -C "$root" rev-parse --short HEAD)"
if [[ -n "$(git -C "$root" status --porcelain)" ]]; then
  tag="${tag}-dirty"
fi

printf "> Building %s with tags:\n- %s\n- latest\n\n" "$image" "$tag"
DOCKER_BUILDKIT=1 docker build -t "$image:$tag" -t "$image:latest" "$root"
