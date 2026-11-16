#!/bin/bash

# SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
#
# SPDX-License-Identifier: CC0-1.0

set -euo pipefail

repo="${REPO:-/app/data/ssb-go-room}"
args=(-dbg= -nounixsock "-repo=${repo}")

if [[ -n "${CLOUDRON:-}" ]]; then
  : "${CLOUDRON_APP_DOMAIN:?Missing Cloudron app domain}"
  : "${CLOUDRON_PROXY_IP:?Missing Cloudron proxy IP}"
  if [[ "${repo}" != /app/data/ssb-go-room ]]; then
    echo 'Cloudron backs up SQLite at /app/data/ssb-go-room/roomdb. Move the existing room data there and remove the REPO override.' >&2
    exit 1
  fi
  args+=("-https-domain=${CLOUDRON_APP_DOMAIN}" "-proxy-ip=${CLOUDRON_PROXY_IP}"
         -aliases-as-subdomains=false -lishttp=:3000 -lismux=:8008)
  if [[ -n "${SSB_PORT:-}" ]]; then
    args+=("-public-mux-port=${SSB_PORT}")
  else
    args+=(-websocket-only)
  fi
else
  : "${HTTPS_DOMAIN:?Set HTTPS_DOMAIN to the hostname of your HTTPS reverse proxy}"
  args+=("-https-domain=${HTTPS_DOMAIN}" "-aliases-as-subdomains=${ALIASES_AS_SUBDOMAINS:-false}"
         "-lishttp=${LISTEN_ADDR_HTTP:-:3000}" "-lismux=${LISTEN_ADDR_SHS_MUX:-:8008}"
         "-public-mux-port=${SSB_PORT:-8008}")
  if [[ -n "${PROXY_IP:-}" ]]; then
    args+=("-proxy-ip=${PROXY_IP}")
  fi
fi

umask 077
mkdir -p -- "${repo}"
if [[ -n "${CLOUDRON:-}" && "$(realpath -- "${repo}")" != "${repo}" ]]; then
  echo 'The Cloudron room data directory must not be a symbolic link.' >&2
  exit 1
fi
chown -R -- cloudron:cloudron "${repo}"
chmod 0700 -- "${repo}"

exec gosu cloudron:cloudron /app/code/server "${args[@]}"
