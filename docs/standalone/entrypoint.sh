#!/bin/bash
# entrypoint.sh - Fetches the SigNoz binaries (signoz, otel-collector,
# foundryctl) at container start, then execs the container command (systemd,
# which runs foundry-setup.service -> foundryctl cast).
#
# Only ClickHouse is baked into the image; everything else is fetched here so
# versions can be chosen at `docker run`.
#
# Environment:
#   SIGNOZ_VERSION     SigNoz version to fetch (tag or "latest"). Default: latest.
#   INGESTER_VERSION   OTel Collector version to fetch (tag or "latest"). Default: latest.
#   FOUNDRY_VERSION    foundryctl version to fetch (tag or "latest"). Default: latest.

set -euo pipefail

# Constants.
readonly SIGNOZ_DIR="/opt/signoz"
readonly INGESTER_DIR="/opt/ingester"
readonly FOUNDRY_BIN="/usr/local/bin/foundryctl"
readonly FOUNDRY_INSTALL_URL="https://signoz.io/foundry.sh"

info() {
  echo "[INFO] $*"
}

err() {
  echo "[ERROR] $*" >&2
}

die() {
  err "$*"
  exit 1
}

# Sets PLATFORM_ARCH; the container is always linux, so only the arch varies.
init_platform() {
  local raw_arch
  raw_arch="$(uname -m)"
  case "${raw_arch}" in
    x86_64 | amd64) PLATFORM_ARCH="amd64" ;;
    aarch64 | arm64) PLATFORM_ARCH="arm64" ;;
    *) die "Unsupported architecture: ${raw_arch}" ;;
  esac
}

# install_release <repo> <version> <dest>: download the SigNoz GitHub release
# tarball for <repo> at <version> ("latest" or a tag) and extract it into <dest>.
install_release() {
  local repo="$1" version="$2" dest="$3"
  local asset="${repo}_linux_${PLATFORM_ARCH}.tar.gz" url

  if [[ "${version}" == "latest" ]]; then
    url="https://github.com/SigNoz/${repo}/releases/latest/download/${asset}"
  else
    url="https://github.com/SigNoz/${repo}/releases/download/${version}/${asset}"
  fi

  info "Fetching ${repo} ${version}"
  mkdir -p "${dest}"
  curl -fsSL "${url}" | tar -xz --strip-components=1 -C "${dest}"
}

# Installs foundryctl via the official installer into FOUNDRY_BIN's directory.
# The installer reads FOUNDRY_VERSION literally and rejects "latest", so for the
# default we unset it and let the installer resolve the newest release itself.
install_foundry() {
  local version="$1" dir
  dir="$(dirname "${FOUNDRY_BIN}")"
  info "Installing foundryctl ${version}"
  if [[ "${version}" == "latest" ]]; then
    curl -fsSL "${FOUNDRY_INSTALL_URL}" \
      | env -u FOUNDRY_VERSION FOUNDRY_INSTALL_DIR="${dir}" FOUNDRY_ASSUME_YES=true bash
  else
    curl -fsSL "${FOUNDRY_INSTALL_URL}" \
      | env FOUNDRY_INSTALL_DIR="${dir}" FOUNDRY_VERSION="${version}" FOUNDRY_ASSUME_YES=true bash
  fi
}

# Fetches any missing binaries, then hands their directories to the signoz user.
run() {
  init_platform

  if [[ ! -x "${SIGNOZ_DIR}/bin/signoz" ]]; then
    install_release signoz "${SIGNOZ_VERSION:-latest}" "${SIGNOZ_DIR}"
    chmod +x "${SIGNOZ_DIR}/bin/signoz"
  fi

  if [[ ! -x "${INGESTER_DIR}/bin/signoz-otel-collector" ]]; then
    install_release signoz-otel-collector "${INGESTER_VERSION:-latest}" "${INGESTER_DIR}"
    chmod +x "${INGESTER_DIR}/bin/signoz-otel-collector"
  fi

  if [[ ! -x "${FOUNDRY_BIN}" ]]; then
    install_foundry "${FOUNDRY_VERSION:-latest}"
  fi

  chown -R signoz:signoz "${SIGNOZ_DIR}" "${INGESTER_DIR}"
}

run
exec "$@"
