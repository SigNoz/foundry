#!/bin/bash
# install.sh - Downloads, verifies, and installs the foundryctl binary
# from GitHub releases.
#
# Usage:
#   curl -fsSL https://signoz.io/signoz.sh | bash
#   curl -fsSL https://signoz.io/signoz.sh | FOUNDRY_VERSION=v0.1.4 bash
#   bash install.sh -v v0.1.4
#   bash install.sh -d /usr/local/bin
#   bash install.sh -h
#
# OPTIONS:
#   -v <version>   Version to install (e.g. v0.1.4). Default: latest.
#   -d <path>      Install directory. Default: $XDG_BIN_HOME or ~/.local/bin.
#   -y             Auto-confirm upgrade prompt.
#   -h             Show help message.
#
# Environment:
#   FOUNDRY_VERSION       Equivalent to -v.
#   FOUNDRY_INSTALL_DIR   Equivalent to -d.
#   FOUNDRY_ASSUME_YES    Equivalent to -y. Set to "true" to enable.
#   NO_COLOR              When set, disables ANSI color output (https://no-color.org).

readonly NAME="install.sh"
readonly REPO="SigNoz/foundry"
readonly BINARY="foundryctl"

set -euo pipefail

# https://no-color.org honoured; auto-stripped when stderr is not a TTY.
if [[ -t 2 ]] && [[ -z "${NO_COLOR:-}" ]]; then
  readonly C_INFO=$'\033[32;1m'
  readonly C_WARN=$'\033[33;1m'
  readonly C_ERROR=$'\033[31;1m'
  readonly C_RESET=$'\033[0m'
else
  readonly C_INFO=""
  readonly C_WARN=""
  readonly C_ERROR=""
  readonly C_RESET=""
fi

info() {
  echo "${C_INFO}[INFO]${C_RESET} $*"
}

warn() {
  echo "${C_WARN}[WARN]${C_RESET} $*" >&2
}

err() {
  echo "${C_ERROR}[ERROR]${C_RESET} $*" >&2
}

help() {
  printf "NAME\n"
  printf "\t%s - Install %s, the SigNoz Foundry CLI\n\n" "${NAME}" "${BINARY}"
  printf "USAGE\n"
  printf "\t%s [-v version] [-d directory] [-y] [-h]\n\n" "${NAME}"
  printf "DESCRIPTION\n"
  printf "\tDownloads, verifies, and installs the %s binary from GitHub releases.\n\n" "${BINARY}"
  printf "OPTIONS\n"
  printf "\t-v <version>\tVersion to install (e.g. v0.1.4). [env: FOUNDRY_VERSION] [default: latest]\n"
  printf "\t-d <path>\tInstall directory. [env: FOUNDRY_INSTALL_DIR] [default: \$XDG_BIN_HOME or ~/.local/bin]\n"
  printf "\t-y\t\tAuto-confirm upgrade prompt. [env: FOUNDRY_ASSUME_YES]\n"
  printf "\t-h\t\tShow this help message.\n\n"
  printf "EXAMPLES\n"
  printf "\t%s -v v0.1.4\n" "${NAME}"
  printf "\t%s -d /usr/local/bin\n" "${NAME}"
  printf "\tcurl -fsSL https://signoz.io/signoz.sh | bash\n"
  printf "\tcurl -fsSL https://signoz.io/signoz.sh | FOUNDRY_VERSION=v0.1.4 bash\n"
}

init_arch() {
  local raw
  raw="$(uname -m)"
  case "${raw}" in
    x86_64 | amd64) ARCH="amd64" ;;
    aarch64 | arm64) ARCH="arm64" ;;
    *)
      err "Unsupported architecture: ${raw}"
      exit 1
      ;;
  esac
}

# Windows shells (Git Bash/MSYS2/Cygwin) map to OS=windows + .exe suffix; the
# windows tarball is then installed. Native PowerShell/cmd cannot run a .sh
# script at all and is not addressed here.
init_os() {
  local raw
  raw="$(uname | tr '[:upper:]' '[:lower:]')"
  BIN_SUFFIX=""
  case "${raw}" in
    darwin | linux) OS="${raw}" ;;
    mingw* | cygwin* | msys*)
      OS="windows"
      BIN_SUFFIX=".exe"
      ;;
    *)
      err "Unsupported OS: ${raw}"
      exit 1
      ;;
  esac
}

# Sets HAS_CURL/HAS_WGET and SHA256_CMD for downstream reuse.
verify_prereqs() {
  HAS_CURL="$(command -v curl >/dev/null 2>&1 && echo true || echo false)"
  HAS_WGET="$(command -v wget >/dev/null 2>&1 && echo true || echo false)"
  if [[ "${HAS_CURL}" != "true" ]] && [[ "${HAS_WGET}" != "true" ]]; then
    err "Either curl or wget is required."
    exit 1
  fi
  local cmd
  for cmd in tar mktemp install; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
      err "${cmd} is required."
      exit 1
    fi
  done
  if command -v sha256sum >/dev/null 2>&1; then
    SHA256_CMD="sha256sum"
  elif command -v shasum >/dev/null 2>&1; then
    SHA256_CMD="shasum -a 256"
  else
    err "Either sha256sum or shasum is required for checksum verification."
    exit 1
  fi
}

# Resolves latest by following the /releases/latest redirect (no GitHub API,
# no JSON, no rate limit).
resolve_version() {
  if [[ -n "${FOUNDRY_VERSION}" ]]; then
    case "${FOUNDRY_VERSION}" in
      v*) TAG="${FOUNDRY_VERSION}" ;;
      *) TAG="v${FOUNDRY_VERSION}" ;;
    esac
    return
  fi

  local latest_url="https://github.com/${REPO}/releases/latest"
  local resolved
  if [[ "${HAS_CURL}" == "true" ]]; then
    resolved="$(curl -sIL -o /dev/null -w '%{url_effective}' "${latest_url}")"
  else
    resolved="$(wget --max-redirect=5 --server-response --spider "${latest_url}" 2>&1 \
      | awk '/^  Location: /{u=$2} END{print u}')"
  fi

  TAG="${resolved##*/tag/}"
  if [[ -z "${TAG}" ]] || [[ "${TAG}" == "${resolved}" ]]; then
    err "Could not resolve the latest release tag from ${latest_url}"
    exit 1
  fi
}

resolve_install_dir() {
  if [[ -n "${FOUNDRY_INSTALL_DIR}" ]]; then
    INSTALL_DIR="${FOUNDRY_INSTALL_DIR}"
  elif [[ -n "${XDG_BIN_HOME:-}" ]]; then
    INSTALL_DIR="${XDG_BIN_HOME}"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
  mkdir -p "${INSTALL_DIR}"
  if [[ ! -w "${INSTALL_DIR}" ]]; then
    err "Install directory is not writable: ${INSTALL_DIR}"
    err "Set FOUNDRY_INSTALL_DIR to a writable path or run with appropriate permissions."
    exit 1
  fi
  DEST="${INSTALL_DIR}/${BINARY}${BIN_SUFFIX}"
}

# Same version: skip and exit 0. Different version: prompt if interactive,
# auto-proceed if piped (curl-pipe-bash, CI).
check_existing() {
  if [[ ! -f "${DEST}" ]]; then
    return
  fi
  local current
  current="$("${DEST}" version 2>/dev/null | awk '/^[[:space:]]*Version:/ {print $NF; exit}' || true)"
  if [[ -z "${current}" ]]; then
    return
  fi
  if [[ "${current}" == "${TAG}" ]]; then
    info "${BINARY} ${TAG} is already installed at ${DEST}"
    exit 0
  fi
  if [[ "${FOUNDRY_ASSUME_YES}" == "true" ]] || [[ ! -t 0 ]]; then
    info "Updating ${BINARY} ${current} -> ${TAG}"
    return
  fi
  local answer
  read -r -p "Update ${BINARY} ${current} to ${TAG}? [Y/n] " answer
  case "${answer}" in
    "" | y | Y | yes | YES) ;;
    *)
      err "Aborted by user."
      exit 1
      ;;
  esac
}

fetch() {
  local url="$1"
  local out="$2"
  if [[ "${HAS_CURL}" == "true" ]]; then
    curl -fsSL "${url}" -o "${out}"
  else
    wget -q -O "${out}" "${url}"
  fi
}

download_release() {
  TARBALL="foundry_${OS}_${ARCH}.tar.gz"
  CHECKSUMS="foundry_${TAG#v}_checksums.txt"
  local tarball_url="https://github.com/${REPO}/releases/download/${TAG}/${TARBALL}"
  local checksums_url="https://github.com/${REPO}/releases/download/${TAG}/${CHECKSUMS}"

  TMP_ROOT="$(mktemp -d -t foundry-installer-XXXXXX)"
  TARBALL_PATH="${TMP_ROOT}/${TARBALL}"
  CHECKSUMS_PATH="${TMP_ROOT}/${CHECKSUMS}"

  info "Downloading ${tarball_url}"
  fetch "${tarball_url}" "${TARBALL_PATH}"
  fetch "${checksums_url}" "${CHECKSUMS_PATH}"
}

# Checksums file lists "<sha256>  <filename>" or "<sha256> *<filename>".
verify_checksum() {
  local expected
  expected="$(awk -v f="${TARBALL}" '$2 == f || $2 == "*"f {print $1; exit}' "${CHECKSUMS_PATH}")"
  if [[ -z "${expected}" ]]; then
    err "Checksum for ${TARBALL} not found in ${CHECKSUMS}"
    exit 1
  fi

  local actual
  actual="$(${SHA256_CMD} "${TARBALL_PATH}" | awk '{print $1}')"

  if [[ "${expected}" != "${actual}" ]]; then
    err "Checksum mismatch for ${TARBALL}"
    err "  expected: ${expected}"
    err "  actual:   ${actual}"
    exit 1
  fi
}

install_binary() {
  local extract_dir="${TMP_ROOT}/extract"
  mkdir -p "${extract_dir}"
  tar -xzf "${TARBALL_PATH}" -C "${extract_dir}"

  local src="${extract_dir}/foundry_${OS}_${ARCH}/bin/${BINARY}${BIN_SUFFIX}"
  if [[ ! -f "${src}" ]]; then
    err "Expected binary not found in tarball: ${src#"${extract_dir}"/}"
    exit 1
  fi

  install -m 0755 "${src}" "${DEST}"
  info "Installed ${BINARY}${BIN_SUFFIX} ${TAG} to ${DEST}"
}

# Smoke test: catches arch/platform mismatches that slipped past checksum.
verify_install() {
  local output
  if ! output="$("${DEST}" version 2>&1)"; then
    err "Installed binary failed to run: ${DEST}"
    err "This may indicate a wrong-arch download or a permissions issue."
    err "${output}"
    exit 1
  fi
  echo
  echo "${output}"
}

print_path_hint() {
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) return ;;
    *) ;;
  esac

  local rc_file
  # shellcheck disable=SC2088
  case "${SHELL:-}" in
    */zsh) rc_file='~/.zshrc' ;;
    */bash) rc_file='~/.bashrc (Linux) or ~/.bash_profile (macOS)' ;;
    */fish) rc_file='~/.config/fish/config.fish' ;;
    *) rc_file='your shell config' ;;
  esac

  echo
  warn "${INSTALL_DIR} is not on your PATH."
  echo "To use ${BINARY} from any shell, add this to ${rc_file}:"
  echo
  echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
}

cleanup() {
  if [[ -n "${TMP_ROOT:-}" ]] && [[ -d "${TMP_ROOT}" ]]; then
    rm -rf "${TMP_ROOT}"
  fi
}

fail_trap() {
  local rc=$?
  if [[ ${rc} -ne 0 ]]; then
    err "Failed to install ${BINARY}."
    err "For support, see https://github.com/${REPO}/issues"
  fi
  cleanup
  exit "${rc}"
}

run() {
  init_arch
  init_os
  verify_prereqs
  resolve_version
  resolve_install_dir
  check_existing
  download_release
  verify_checksum
  install_binary
  verify_install
  print_path_hint
  cleanup
}

FOUNDRY_VERSION="${FOUNDRY_VERSION:-}"
FOUNDRY_INSTALL_DIR="${FOUNDRY_INSTALL_DIR:-}"
FOUNDRY_ASSUME_YES="${FOUNDRY_ASSUME_YES:-false}"

trap fail_trap EXIT

while getopts 'v:d:yh' opt; do
  case "${opt}" in
    v) FOUNDRY_VERSION="${OPTARG:-}" ;;
    d) FOUNDRY_INSTALL_DIR="${OPTARG:-}" ;;
    y) FOUNDRY_ASSUME_YES="true" ;;
    h)
      help
      trap - EXIT
      exit 0
      ;;
    ?)
      err "Invalid option: -${OPTARG:-}"
      exit 1
      ;;
    *)
      err "Unknown error while processing options"
      exit 1
      ;;
  esac
done

run
trap - EXIT
