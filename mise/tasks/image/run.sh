#!/usr/bin/env bash
#MISE description="Build the Docker image, then run it on :8080 as production does (read-only, no capabilities)"
#MISE depends=["image:build"]
set -euo pipefail

docker run --rm -p 8080:8080 --read-only --cap-drop=ALL ghcr.io/landazuripaul/unblogexutoire:latest
