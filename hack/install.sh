#!/bin/sh
set -e

PKG_NAME=tmctl
GITHUB_URL=https://github.com/triggermesh/tmctl

PLATFORM=
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
EXT=".tar.gz"
BIN_NAME="tmctl"
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
      EXT=".zip"
      BIN_NAME="tmctl.exe"
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

TMP_DIR=$(mktemp -d -t tmctl-install.XXXXXX)
TMP_ARCHIVE=${TMP_DIR}/${PKG_NAME}_${PLATFORM}_${ARCH}${EXT}
TMP_BIN=${TMP_DIR}/${BIN_NAME}
cleanup() {
  code=$?
  set +e
  trap - EXIT
  rm -rf ${TMP_DIR}
  exit $code
}
trap cleanup INT EXIT

info "Downloading ${DOWNLOAD_URL}... to ${TMP_ARCHIVE}"
${CURL} -sfL ${DOWNLOAD_URL} -o ${TMP_ARCHIVE}

if [ "${EXT}" = ".zip" ]; then
  unzip ${TMP_ARCHIVE} -d ${TMP_DIR}
else
  tar xzf ${TMP_ARCHIVE} -C ${TMP_DIR}
fi

info "Installing to ${BIN_DIR}/${PKG_NAME}"
chmod 755 ${TMP_BIN}
${SUDO} chown root ${TMP_BIN}
${SUDO} mv -f ${TMP_BIN} ${BIN_DIR}/${BIN_NAME}
