# NATS UI

[English README](./README.en.md)

基于 `Golang + Gin + nats.go + Vue 3 + Vite + Element Plus + ECharts` 的 NATS Server 可视化管理工具基础架构。

现已增加 Electron 桌面端封装，可在 Windows、macOS、Linux 上打包为本地安装应用。桌面端会在应用启动时自动拉起内置 Go 后端，再加载前端页面。

## 目录结构

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

## 后端启动

```bash
set NATS_URL=nats://127.0.0.1:4222
set NATS_MONITOR_URLS=http://127.0.0.1:8222,http://127.0.0.1:8223
go run ./cmd/server
```

`NATS_MONITOR_URLS` 用于读取 `/varz` 和 `/connz` 监控数据，`NATS_URL` 用于 JetStream 和 KV 的管理操作。

## 前端启动

```bash
cd frontend
npm install
npm run dev
```

## Electron 桌面端

Electron 依赖和前端依赖分别安装：

```bash
npm install
npm --prefix frontend install
```

启动桌面端开发版（使用打包后的前端资源和当前平台 Go 后端二进制）：

```bash
npm run desktop:start
```

桌面端常用构建命令：

```bash
npm run assets:icons
npm run desktop:build:backend
npm run desktop:build:backend:all
npm run desktop:dist
npm run desktop:dist:win
npm run desktop:dist:mac
npm run desktop:dist:linux
```

说明：

- `assets:icons` 会生成应用图标、favicon、Windows 安装器侧边图和 DMG 背景图。
- `desktop:build:backend` 会为当前平台构建桌面端内置 Go 后端。
- `desktop:build:backend:all` 会额外生成 Windows、macOS、Linux 常见架构的后端二进制。
- `desktop:dist` 会为当前操作系统打包 Electron 应用并输出到 `release/`。
- `desktop:dist:win`、`desktop:dist:mac`、`desktop:dist:linux` 分别生成对应平台安装包。
- Electron 桌面端使用本机环回地址启动内置后端，连接配置与密钥文件会写入系统用户目录下的 Electron 应用数据目录，不会覆盖仓库里的 `data/`。

建议在目标平台本机执行对应的 `desktop:dist:*` 命令来生成安装包，这样最稳妥。
