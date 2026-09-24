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

## 消息工作台与专业监控

- **消息工作台**：Core NATS 实时订阅（支持 `*`、`>` 和 Queue Group）、文本/JSON 发布、自定义 Headers、Request/Reply 和对 Reply Subject 手动回复。切换连接或离开页面时自动停止订阅；最多保留 200 条，每条预览最多 64 KiB，非 UTF-8 内容以 Base64 展示。请求超时可设置为 100–30000 ms。
- **JetStream**：保留 Stream 管理，新增持久化 Pull Consumer 创建/删除、待投递/待确认/重投递指标，以及按消息序号只读查看持久化消息。Stream 详情还显示最近 10 条消息，自动识别并格式化 JSON、文本和常见二进制格式；也可手动按 String、JSON、Base64→String 或 Base64→JSON 解析。二进制内容默认显示十六进制摘要。查看消息不会推进 Consumer 的消费进度。创建的 Consumer 使用 Explicit ACK。
- **KV 管理**：详情区可单独刷新。点击条目可查看完整 Value、Revision 和更新时间；有效 JSON 会自动格式化，普通文本保持原样。
- **运行监控**：入站/出站消息和字节速率、最近 60 次采样趋势、节点 CPU/内存、慢消费者累计计数、连接积压、JetStream 账户资源及 API 错误计数。速率至少需要两次有效采样；节点变化和计数器重置会重建基线。
- 节点指标需要配置可访问的 NATS HTTP 监控端口（例如 `8222`）；JetStream 数据需要服务器启用 JetStream 且账户有相应权限。连接监控当前每节点最多取 256 条，积压表显示样本中最高的 50 条，并非全量统计。

验证命令：`npm --prefix frontend run build`、`go test ./...`。需要执行真实消息链路集成测试时，启动独立的临时 JetStream 服务器，将 `NATS_TEST_URL` 指向它，然后运行 `go test ./internal/service -run TestMessagingIntegration -v`。测试会创建并清理自己的 Stream 和 Consumer，请使用测试实例。
