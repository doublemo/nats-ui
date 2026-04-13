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
