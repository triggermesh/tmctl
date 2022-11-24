#!/bin/sh
set -e

PKG_NAME=tmctl
PKG_VERSION=
GITHUB_URL=https://github.com/triggermesh/tmctl

PLATFORM=
EXT=
ARCH=

SUDO=sudo
CURL=curl
BIN_DIR=

info() {
  echo '[INFO] ' "$@"
}

warn() {
  echo '[WARN] ' "$@" >&2
}

fatal() {
  echo '[ERROR] ' "$@" >&2
  exit 1
}

if [ $(id -u) -eq 0 ]; then
  SUDO=
fi

# if BIN_DIR is not specified, install to "/usr/local/bin"
if [ -z "${BIN_DIR}" ]; then
  BIN_DIR="/usr/local/bin"
fi

# os platform
if [ -z "${PLATFORM}" ]; then
  case $(uname) in
    Linux)
      PLATFORM="linux"
      ;;
    Darwin)
      PLATFORM="macOS"
      ;;
    Windows)
      PLATFORM="windows"
      EXT=".exe"
      ;;
    *)
      fatal "Unsupported platform ${PLATFORM}"
      ;;
  esac
fi

# system architecture
if [ -z "${ARCH}" ]; then
  case $(uname -m) in
    x86_64|amd64)
      ARCH="amd64"
      ;;
    arm64|aarch64)
      ARCH="arm64"
      ;;
    *)
      fatal "Unsupported architecture ${ARCH}"
      ;;
  esac
fi

# check if curl command is available
if ! [ -x "$(command -v ${CURL})" ]; then
  fatal "Command not found: ${CURL}"
  return 1
fi

# generate download URL
if [ -n "${PKG_VERSION}" ]; then
  DOWNLOAD_URL="${GITHUB_URL}/releases/download/${PKG_VERSION}/${PKG_NAME}_${PLATFORM}_${ARCH}${EXT}"
else
  DOWNLOAD_URL="${GITHUB_URL}/releases/latest/download/${PKG_NAME}_${PLATFORM}_${ARCH}${EXT}"
fi

info "Downloading ${DOWNLOAD_URL}..."
TMP_BIN=$(mktemp -u /tmp/${PKG_NAME}.XXXXXX)
${CURL} -sfL ${DOWNLOAD_URL} -o ${TMP_BIN}

info "Installing to ${BIN_DIR}/${PKG_NAME}"
chmod 755 ${TMP_BIN}
${SUDO} chown root ${TMP_BIN}
${SUDO} mv -f ${TMP_BIN} ${BIN_DIR}/${PKG_NAME}${EXT}
