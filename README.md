<!--
SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021

SPDX-License-Identifier: CC0-1.0
-->

# SSB Room for Cloudron

Host a Scuttlebutt Rooms 2 server for Manyverse. Create invitations, manage members and choose who can connect from the room dashboard.

## Install on Cloudron

In the Cloudron 10+ dashboard, open **App Store → Add custom app → Community app** and paste the [SSB Room install URL](https://raw.githubusercontent.com/patattzel/go-ssb-room/master/CloudronVersions.json).

```text
https://raw.githubusercontent.com/patattzel/go-ssb-room/master/CloudronVersions.json
```

Choose the room domain and an SSB TCP port. The default is 8008. Disable the TCP port to use secure WebSockets on HTTPS port 443.

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
