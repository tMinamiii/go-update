#!/bin/bash

set -Ceu

LATEST_GO_VERSION=$(curl -s 'https://go.dev/dl/?mode=json' | jq -r '[.[]][0].version')

echo "Latest go version is ---  ${LATEST_GO_VERSION}  ---"

GOGZ="${LATEST_GO_VERSION}.linux-amd64.tar.gz"

cd /tmp || return

trap 'sudo rm /tmp/${GOGZ} && cd - > /dev/null' EXIT

sudo curl -OL --progress-bar "https://go.dev/dl/${GOGZ}" && \
  sudo rm -rf /usr/local/go && \
  sudo tar -C /usr/local -xzf "/tmp/${GOGZ}"

echo "Install completed !"

