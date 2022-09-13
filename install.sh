#!/bin/bash

set -Ceu

LATEST_GO_VERSION=$(curl -s 'https://go.dev/dl/?mode=json' | jq -r '[.[]][0].version')
OS="$(uname)"
ARCH="$(uname -m)"

echo "Latest go version is ---  ${LATEST_GO_VERSION}  ---"

echo "OS: ${OS} Arch: ${ARCH}"


if [ "${OS}" = "Darwin" ]; then
  GOPKG=""
  if [ "${ARCH}" = "x86_64" ]; then
    GOPKG="${LATEST_GO_VERSION}.darwin-amd64.pkg"
  elif echo "${ARCH}" | grep -sq "arm"; then
    GOPKG="${LATEST_GO_VERSION}.darwin-arm64.pkg"
  fi

  [ "$GOPKG" = "" ] && exit

  trap 'rm ${GOPKG}' EXIT
  sudo curl -OL --progress-bar "https://go.dev/dl/${GOGZ}"
  open "${GOPKG}"

elif [ "${OS}" = "Linux" ]; then
  GOGZ=""
  if [ "${ARCH}" = "x86_64" ]; then
    GOGZ="${LATEST_GO_VERSION}.linux-amd64.tar.gz"
  elif ${ARCH} | grep -sq "arm"; then
    GOGZ="${LATEST_GO_VERSION}.linux-arm64.tar.gz"
  fi

  [ "$GOGZ" = "" ] && exit

  trap 'sudo rm ${GOGZ}' EXIT

  sudo curl -OL --progress-bar "https://go.dev/dl/${GOGZ}" && \
    sudo rm -rf /usr/local/go && \
    sudo tar -C /usr/local -xzf "${GOGZ}"

  echo "Install completed !!"

  /usr/local/go/bin/go version
fi
