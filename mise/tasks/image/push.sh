#!/usr/bin/env bash
#MISE description="Build the Docker image, then push its tags to ghcr.io (needs `docker login ghcr.io`)"
#MISE depends=["image:build"]
set -euo pipefail

image="ghcr.io/landazuripaul/unblogexutoire"
root="${MISE_PROJECT_ROOT:-$(git rev-parse --show-toplevel)}"

if [[ -n "$(git -C "$root" status --porcelain)" ]]; then
  echo "Refusing to push: the working tree has uncommitted changes (the image would be tagged -dirty)." >&2
  exit 1
fi

tag="$(git -C "$root" rev-parse --short HEAD)"
printf "\n> Pushing %s:%s and %s:latest...\n" "$image" "$tag" "$image"
docker push "$image:$tag"
docker push "$image:latest"
