<!--
SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021

SPDX-License-Identifier: CC0-1.0
-->

# SSB Room for Cloudron

Host a Scuttlebutt Rooms 2 server for Manyverse. Create invitations, manage members and choose who can connect from the room dashboard.

## Install

Use a Cloudron 10 or later server and the current [Cloudron CLI](https://docs.cloudron.io/packaging/cli/). Run these commands from your computer.

```sh
git clone https://github.com/patattzel/go-ssb-room.git
cd go-ssb-room
cloudron login my.example.com
cloudron install --location room.example.com
```

Cloudron builds the image from the checkout and serves the dashboard over HTTPS. The default public SSB port is TCP 8008. Set a different port at install time with `--port-bindings SSB_PORT=18008`, or use `--port-bindings SSB_PORT=x` to connect through secure WebSockets on port 443.

After installation, open the app's **Web Terminal** and run `/app/code/room-admin`. Enter your public Manyverse SSB ID and pick a web password. Open the room site, choose **Sign in → Password sign-in**, and create an invite from the dashboard.

See [Cloudron operations](docs/cloudron.md) for first setup, backups, updates and troubleshooting. See [development](docs/development.md) for working on the server.

## Run with Docker Compose

Copy `.env_example` to `.env` and set `HTTPS_DOMAIN` to your room hostname. Configure an HTTPS reverse proxy on that host to forward requests to `127.0.0.1:3000`, preserving the Host header and setting `X-Forwarded-Proto: https` and `X-Forwarded-For`. Then run:

```sh
docker compose up -d --build
docker compose exec room /app/code/room-admin
```

The Compose setup exposes the configured TCP SSB port and keeps data in `./ssb-go-room-secrets`. Its image targets Linux AMD64. Docker on ARM hosts needs AMD64 emulation.

The server is based on [go-ssb-room](https://github.com/ssbc/go-ssb-room) and implements the [Rooms 2 protocol](https://github.com/ssbc/rooms2). See [LICENSE](LICENSE).
