<!--
SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021

SPDX-License-Identifier: CC0-1.0
-->

# Go-SSB Room
[![REUSE status](https://api.reuse.software/badge/github.com/ssbc/go-ssb-room)](https://api.reuse.software/info/github.com/ssbc/go-ssb-room)

This repository implements the [Room (v1+v2) server spec](https://github.com/ssbc/rooms2), in Go.

It includes:
* secret-handshake+boxstream network transport, sometimes referred to as SHS, using [secretstream](https://github.com/ssbc/go-secretstream)
* muxrpc handlers for tunneling connections
* a fully embedded HTTP server & HTML frontend, for administering the room

![](./docs/images/screenshot.png)

See [this project](https://github.com/orgs/ssbc/projects/2) for current focus.

## :star: Features

* Rooms v1 (`tunnel.connect`, `tunnel.endpoints`, etc.)
* User management (allow- & denylisting + moderator & administrator roles), all administered via the web dashboard
* Multiple [privacy modes](https://ssbc.github.io/rooms2/#privacy-modes)
* [Sign-in with SSB](https://ssbc.github.io/ssb-http-auth-spec/)
* [HTTP Invites](https://github.com/ssbc/ssb-http-invite-spec)
* Alias management

For a comprehensive introduction to rooms 2.0, 🎥 [watch this video](https://www.youtube.com/watch?v=W5p0y_MWwDE).
For a description of MuxRPC APIs see https://github.com/ssbc/rooms2

## :rocket: Deployment

If you want to deploy a room server yourself, follow our [deployment.md](./docs/deployment.md) docs.
For packaging and running this server on [Cloudron](https://cloudron.io), see [docs/cloudron.md](./docs/cloudron.md).

### Quick start (docker-compose)
1. Copy `.env_example` to `.env` and set `HTTPS_DOMAIN`.
2. Build and start: `docker-compose build room && docker-compose up -d`.
3. Create the first admin inside the running container:
   ```bash
   docker-compose exec room sh
   /app/cmd/insert-user/insert-user -repo /ssb-go-room-secrets @your-ssb-pubkey
   exit
   ```
4. Dashboard is on port `3000`; SSB muxrpc on `8008`. Data/keys sync to `./ssb-go-room-secrets`.

### Cloudron install (example)
- Follow the Cloudron packaging tutorial: https://docs.cloudron.io/packaging/tutorial/
- Build locally (`cloudron build --set-version 2.0.7-1`) **or** use a published image (e.g. Docker Hub `pathab/go-ssb-room-cloudron:2.0.7-1`), then install:  
  `cloudron install --image <tag>`
- After install, create the first admin:  
  `cloudron exec -- /app/code/insert-user -repo /app/data/ssb-go-room @your-ssb-pubkey`
- Dashboard is proxied by Cloudron; data lives in `/app/data/ssb-go-room`. The SSB muxrpc port is configured via the `SSB_PORT` tcpPort from the manifest.

## :wrench: Development

For an in-depth codebase walkthrough, see the [development.md](./docs/development.md) file in the `docs` folder of this repository.

## :people_holding_hands: Authors

* [cryptix](https://github.com/cryptix) (`@p13zSAiOpguI9nsawkGijsnMfWmFd5rlUNpzekEE+vI=.ed25519`)
* [staltz](https://github.com/staltz)
* [cblgh](https://github.com/cblgh)

## License

MIT
