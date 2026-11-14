<!--
SPDX-FileCopyrightText: 2024 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021

SPDX-License-Identifier: CC0-1.0
-->

# Install and operate SSB Room on Cloudron

## Install from the dashboard

1. In the Cloudron 10+ dashboard, open **App Store → Add custom app → Community app**.
2. Paste the [SSB Room install URL](https://raw.githubusercontent.com/patattzel/go-ssb-room/master/CloudronVersions.json).
3. Choose a domain. Keep TCP 8008 enabled for TCP-only SSB clients, choose another free public TCP port, or disable the binding to use secure WebSockets on HTTPS port 443. Allow the chosen TCP port on any external firewall.
4. Open the deployed room from the Cloudron dashboard. Leave its URL publicly reachable for guests and their Manyverse clients.

For a CLI installation, use the current [Cloudron CLI](https://docs.cloudron.io/packaging/cli/):

```sh
cloudron login my.example.com
cloudron install --location room.example.com --versions-url https://raw.githubusercontent.com/patattzel/go-ssb-room/master/CloudronVersions.json
```

The CLI accepts `--port-bindings SSB_PORT=18008` to set a custom port and `--port-bindings SSB_PORT=x` to disable TCP and use WebSockets.

Cloudron terminates HTTPS and sends traffic to HTTP port 3000 in the container. It forwards the chosen TCP port to port 8008 inside the container. The room advertises the public endpoint in invitations, aliases and SSB sign-in links. Cloudron checks `/healthz` on the internal HTTP listener.

## Set up the first administrator

1. In Manyverse, copy your public SSB ID from your profile. It starts with `@` and ends with `.ed25519`.
2. In Cloudron, open the app's **Web Terminal** and run `/app/code/room-admin`. Paste your SSB ID and enter a new web password twice. The password prompt hides your input. From the Cloudron CLI, the equivalent is `cloudron exec -t --app room.example.com -- /app/code/room-admin`.
3. On the room site, choose **Sign in → Password sign-in**. Use the same public SSB ID and the web password you just set.
4. Use **Create invite** on the dashboard and share the single-use link. Guests can open it in Manyverse, paste the HTTPS link in Manyverse under **Connections**, or scan the QR code with another device. Once a peer connects, it appears in the room dashboard.

To recover a web password, run `/app/code/room-admin --reset-password` in the Web Terminal and enter the existing member's ID.

## Room settings

- **Restricted** limits connections to invited members and lets admins and moderators create invitations. **Community** lets visitors connect and gives members moderation options. **Open** lets anyone create an invitation. New rooms use the existing server default, **Community**. Choose the mode under **Settings** before sharing your first invite.
- Edit the description on the public homepage and manage the code of conduct and privacy notice under **Notices**.
- Cloudron uses path-based aliases on the app domain. Cloudron users and room members are managed separately through their respective dashboards.
- Keep the domain and connection mode stable after sharing the room. If either the domain or the public port changes, previously connected clients may need the updated address from **Connection details**.

## Backups and updates

Cloudron stores the room identity, SQLite database and web session keys under `/app/data/ssb-go-room`. The localstorage addon includes these in app backups and takes consistent SQLite backups. Restore through Cloudron to retain the room identity and existing membership. Treat these backups as sensitive because they contain the room's private key.

Cloudron receives new versions from the Community App catalog. Review and apply updates in the app's dashboard. For a rollback or restore, use the app's Cloudron backup controls.

For an existing source install with a custom `REPO`, stop the room and move its entire data directory to `/app/data/ssb-go-room` before updating. Remove the `REPO` override so the app and Cloudron's SQLite backup use the same path.

### Publish a release

Bump the package version and update its changelog in `CloudronManifest.json`. Push a matching `cloudron-v<version>` tag to build, smoke-test and publish the Linux AMD64 image to GitHub Container Registry. Once the image is available anonymously, run `cloudron versions add --image ghcr.io/patattzel/go-ssb-room@sha256:<digest>` and `cloudron versions verify`. Commit and push the updated catalog to `master`.

## Troubleshoot

- **The app shows as unhealthy.** Check `cloudron logs --app room.example.com` for startup errors. A healthy room answers HTTP 204 at `/healthz`.
- **The invitation is accepted, but Manyverse cannot connect.** Open **Connection details** in the room dashboard. A `net:` address must show the external TCP port configured in Cloudron. Check the external firewall and that the hostname resolves directly to Cloudron for TCP. A `wss:` address uses HTTPS on port 443 and requires WebSocket upgrades through any additional proxy.
- **The invitation did not open Manyverse.** Copy the HTTPS link and paste it under **Connections** in Manyverse, or download a supported client from the [official downloads page](https://www.manyver.se/download/). A consumed invitation requires a new link for another person.
- **The app cannot write to its database.** Check that its data directory is present and that the app starts through `/app/code/start.sh`, which restores ownership before running the server as the Cloudron user.

[Cloudron packaging tutorial](https://docs.cloudron.io/packaging/tutorial/) · [TCP port bindings](https://docs.cloudron.io/packaging/manifest/#tcpports) · [SQLite backups](https://docs.cloudron.io/packaging/addons/#sqlite)
