#!/usr/bin/env bash
set -euo pipefail

# ------------------------------------------------------------
# Multi-arch Docker release script for modbus-exporter
# Builds and pushes linux/amd64 + linux/arm64 images.
#
# Usage:
#   ./release-docker.sh v0.1.1
#
# Optional:
#   ./release-docker.sh v0.1.1 latest
# ------------------------------------------------------------

IMAGE="atrabilis/modbus-exporter"

if [ $# -lt 1 ]; then
  echo "Usage: $0 <version-tag> [also-tag-latest]"
  exit 1
fi

VERSION="$1"
TAG_LATEST="${2:-}"

PLATFORMS="linux/amd64,linux/arm64"

echo "==> Releasing ${IMAGE}:${VERSION}"
echo "Platforms: ${PLATFORMS}"

# ------------------------------------------------------------
# Ensure buildx is available
# ------------------------------------------------------------
if ! docker buildx inspect multiarch-builder >/dev/null 2>&1; then
  echo "==> Creating buildx builder..."
  docker buildx create multiarch-builder
  docker buildx use multiarch-builder

else
  docker buildx use multiarch-builder
fi

docker buildx inspect --bootstrap >/dev/null

# ------------------------------------------------------------
# Build & push
# ------------------------------------------------------------
CMD=(
  docker buildx build
  --platform "${PLATFORMS}"
  -t "${IMAGE}:${VERSION}"
  --push
)

if [ "${TAG_LATEST}" = "latest" ]; then
  CMD+=(-t "${IMAGE}:latest")
fi

CMD+=(.)

echo "==> Running:"
printf ' %q' "${CMD[@]}"
echo

"${CMD[@]}"

echo "==> Done."
echo "Published:"
echo "  - ${IMAGE}:${VERSION}"
if [ "${TAG_LATEST}" = "latest" ]; then
  echo "  - ${IMAGE}:latest"
fi
