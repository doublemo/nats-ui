# NATS UI

[中文说明](./README.md)

A visual management tool for NATS Server built with `Golang + Gin + nats.go + Vue 3 + Vite + Element Plus + ECharts`.

The project now also includes an Electron desktop wrapper, so it can be packaged as a native app for Windows, macOS, and Linux. The desktop app automatically starts the embedded Go backend on launch and then loads the frontend UI.

## Project Structure

```text
.
├── cmd
│   └── server
│       └── main.go
├── desktop
│   ├── main.cjs
│   ├── preload.cjs
│   └── scripts
│       └── build-backend.mjs
├── internal
│   ├── config
│   │   └── config.go
│   ├── handlers
│   │   └── nats_handler.go
│   ├── models
│   │   └── types.go
│   └── service
│       └── nats_service.go
├── frontend
│   ├── package.json
│   ├── vite.config.js
│   └── src
│       ├── api
│       │   └── nats.js
│       ├── router
│       │   └── index.js
│       ├── views
│       │   ├── Dashboard.vue
│       │   ├── JetStreamView.vue
│       │   └── KVManager.vue
│       ├── App.vue
│       ├── main.js
│       └── styles.css
├── package.json
└── go.mod
```

## Start the Backend

```bash
set NATS_URL=nats://127.0.0.1:4222
set NATS_MONITOR_URLS=http://127.0.0.1:8222,http://127.0.0.1:8223
go run ./cmd/server
```

`NATS_MONITOR_URLS` is used to read `/varz` and `/connz` monitoring data, while `NATS_URL` is used for JetStream and KV management operations.

## Start the Frontend

```bash
cd frontend
npm install
npm run dev
```

## Electron Desktop App

Install Electron dependencies and frontend dependencies separately:

```bash
npm install
npm --prefix frontend install
```

Start the desktop app in development mode. It uses the built frontend assets together with the Go backend binary for the current platform:

```bash
npm run desktop:start
```

Common desktop build commands:

```bash
npm run assets:icons
npm run desktop:build:backend
npm run desktop:build:backend:all
npm run desktop:dist
npm run desktop:dist:win
npm run desktop:dist:mac
npm run desktop:dist:linux
```

Notes:

- `assets:icons` generates application icons, favicon assets, Windows installer sidebar images, and the DMG background image.
- `desktop:build:backend` builds the embedded Go backend for the current platform.
- `desktop:build:backend:all` additionally generates backend binaries for common Windows, macOS, and Linux architectures.
- `desktop:dist` packages the Electron app for the current operating system and outputs artifacts to `release/`.
- `desktop:dist:win`, `desktop:dist:mac`, and `desktop:dist:linux` generate installers for the corresponding platforms.
- The Electron desktop app starts its embedded backend on a local loopback address. Connection configs and secret files are written to the Electron app data directory under the current system user, so they do not overwrite the repository `data/` directory.

It is recommended to run the corresponding `desktop:dist:*` command directly on the target platform for the most reliable packaging result.

## Messaging and operations monitoring

- **Messages**: live Core NATS subscriptions with wildcards and optional queue groups, text/JSON publishing, custom headers, request/reply, and manual replies to message reply subjects. Subscriptions stop on navigation or connection changes. The browser keeps 200 messages with previews capped at 64 KiB each; binary payloads use Base64. Request timeout: 100–30000 ms.
- **JetStream**: durable pull consumer creation/deletion, pending/ack-pending/redelivery metrics, and read-only stored-message inspection by sequence. Stream details show the 10 most recent messages, with formatted JSON/text and hexadecimal previews for common binary formats. The viewer also offers String, JSON, Base64→String and Base64→JSON parsing. Inspection does not advance consumer progress. New consumers use explicit acknowledgments.
- **KV manager**: refresh the detail pane independently. Click an entry to inspect its full value, revision, and update time. Valid JSON is formatted automatically; other text is shown unchanged.
- **Monitoring**: inbound/outbound message and byte rates, 60-sample history, node CPU/memory, cumulative slow-consumer counts, connection backlog, JetStream account resources and API errors. Rates require two valid samples; topology changes and counter resets establish a new baseline.
- Node metrics require reachable NATS HTTP monitoring endpoints (e.g. port 8222). JetStream metrics require JetStream and account permissions. Connection monitoring samples up to 256 connections per node; the backlog table shows the top 50 from those samples, not an exhaustive total.

Validation: `npm --prefix frontend run build` and `go test ./...`. For integration tests, start a disposable JetStream server, set `NATS_TEST_URL`, and run `go test ./internal/service -run TestMessagingIntegration -v`. Tests create and clean up their own stream and consumer; use a test instance.
