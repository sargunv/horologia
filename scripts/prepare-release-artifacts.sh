#!/bin/sh

set -eu

RELEASE_VERSION="${RELEASE_VERSION:-dev}"
GIT_COMMIT="${GIT_COMMIT:-$(git rev-parse HEAD)}"

export VITE_APP_VERSION="$RELEASE_VERSION"
export VITE_APP_COMMIT="$GIT_COMMIT"

mise run //api:generate
mise run //server:generate
mise run //web:build

rm -rf server/internal/webui/dist
mkdir -p server/internal/webui/dist
cp -R web/dist/. server/internal/webui/dist/
