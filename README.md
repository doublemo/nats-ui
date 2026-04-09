# NATS UI

基于 `Golang + Gin + nats.go + Vue 3 + Vite + Element Plus + ECharts` 的 NATS Server 可视化管理工具基础架构。

## 目录结构

```text
.
├── cmd
│   └── server
│       └── main.go
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
