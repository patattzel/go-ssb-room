<!--
SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021

SPDX-License-Identifier: CC0-1.0
-->

# Go-SSB Room (Cloudron packaging fork)
[![REUSE status](https://api.reuse.software/badge/github.com/ssbc/go-ssb-room)](https://api.reuse.software/info/github.com/ssbc/go-ssb-room)

This fork packages the upstream [go-ssb-room](https://github.com/ssbc/go-ssb-room) server as a Cloudron app and provides a ready-to-run Docker image. It still contains the full upstream codebase plus Cloudron-specific packaging files.

- Upstream project: https://github.com/ssbc/go-ssb-room
- Docker Hub image (example): `pathab/go-ssb-room-cloudron:2.0.7-1`
- Cloudron packaging tutorial: https://docs.cloudron.io/packaging/tutorial/

## What it is
- Implements the [Room (v1+v2) spec](https://github.com/ssbc/rooms2) in Go.
- Features: SHS+boxstream transport, muxrpc tunneling, web dashboard, invites, aliases, privacy modes, Sign-in with SSB.
- Default ports: HTTP dashboard `3000`, muxrpc/shs `8008`.

## Install on Cloudron
1. Use a published image (e.g. `pathab/go-ssb-room-cloudron:2.0.7-1`) **or** build locally:
   ```bash
   cloudron build --set-version 2.0.7-1
   cloudron install --image <tag>
   ```
2. Create the first admin:
   ```bash
   cloudron exec -- /app/code/insert-user -repo /app/data/ssb-go-room @your-ssb-pubkey
   ```
3. Dashboard is proxied by Cloudron; data lives in `/app/data/ssb-go-room`. The SSB muxrpc port is exposed via the manifest tcpPort `SSB_PORT` (defaults to 8008).

More details: [docs/cloudron.md](./docs/cloudron.md).

## Quick start with docker-compose (local)
1. Copy `.env_example` to `.env` and set `HTTPS_DOMAIN`.
2. Build and start:
   ```bash
   docker-compose build room
   docker-compose up -d
   ```
3. Inside the running container, create the first admin:
   ```bash
   docker-compose exec room sh
   /app/cmd/insert-user/insert-user -repo /ssb-go-room-secrets @your-ssb-pubkey
   exit
   ```
4. Dashboard: `http://localhost:3000`, muxrpc: `8008`, data: `./ssb-go-room-secrets`.

## Development
See [docs/development.md](./docs/development.md) for hacking on the codebase. Cloudron-specific files are `CloudronManifest.json`, `Dockerfile`, and `start.sh`.

## License
MIT
