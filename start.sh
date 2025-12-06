#!/bin/bash

# SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
#
# SPDX-License-Identifier: CC0-1.0

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ -f "${SCRIPT_DIR}/.env" ]; then
  # Allow local docker-compose/.env based development
  set -o allexport
  # shellcheck source=/dev/null
  . "${SCRIPT_DIR}/.env"
  set +o allexport
fi

HTTPS_DOMAIN="${HTTPS_DOMAIN:-${CLOUDRON_APP_DOMAIN:-}}"
LISTEN_ADDR_HTTP="${LISTEN_ADDR_HTTP:-:3000}"
LISTEN_ADDR_SHS_MUX="${LISTEN_ADDR_SHS_MUX:-:8008}"
ALIASES_AS_SUBDOMAINS="${ALIASES_AS_SUBDOMAINS:-false}"
REPO="${REPO:-/app/data/ssb-go-room}"

if [ -z "${HTTPS_DOMAIN}" ]; then
  echo "HTTPS_DOMAIN or CLOUDRON_APP_DOMAIN must be set" >&2
  exit 1
fi

umask 077
mkdir -p "${REPO}"

# Ensure data dir is writable by cloudron
chown -R cloudron:cloudron "${REPO}" 2>/dev/null || true
chown -R cloudron:cloudron /app/data 2>/dev/null || true

SERVER_BIN="${SERVER_BIN:-${SCRIPT_DIR}/server}"
ALT_SERVER_BIN="${SCRIPT_DIR}/cmd/server/server"

if [ ! -x "${SERVER_BIN}" ] && [ -x "${ALT_SERVER_BIN}" ]; then
  SERVER_BIN="${ALT_SERVER_BIN}"
fi

if [ ! -x "${SERVER_BIN}" ]; then
  echo "server binary not found (looked in ${SERVER_BIN})" >&2
  exit 1
fi

RUN_AS="env HOME=/app/data gosu cloudron"
if ! command -v gosu >/dev/null 2>&1; then
  RUN_AS=""
fi

exec ${RUN_AS} "${SERVER_BIN}" \
  -https-domain="${HTTPS_DOMAIN}" \
  -repo="${REPO}" \
  -aliases-as-subdomains="${ALIASES_AS_SUBDOMAINS}" \
  -lishttp="${LISTEN_ADDR_HTTP}" \
  -lismux="${LISTEN_ADDR_SHS_MUX}"
