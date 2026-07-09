# Cloudflare GitHub 登录与 Turso 代理

本方案用于“登录 GitHub 后使用线上 Turso 基准库”。本地应用只保存 Cloudflare Worker URL 和短期应用会话，不保存 Turso URL/token。

## 架构

```text
本地 sushiro Web UI
  -> GitHub 登录跳转
  -> Cloudflare Worker /auth/github/*
  -> GitHub OAuth App
  -> Worker 签发 sushiro cloud session
  -> 本地保存 session 到 ~/.sushiro/cloud_auth.json
  -> 本地后端带 Bearer session 请求 Worker 白名单 API
  -> Worker 用 secrets 里的 Turso token 查询固定 SELECT
```

Worker 只暴露这些数据接口：

- `GET /api/me`
- `GET /api/queue/baseline/export`
- `GET /api/queue/baseline/store?store_id=3006`

不暴露任意 SQL，也不下发 Turso token。

## 已验证的线上 schema

2026-06-08 只读查询确认当前 Turso 有这些表：

- `store_dimension`
- `store_latest`
- `store_bucket_rollups`
- `daily_store_bucket_rollups`
- `queue_snapshots`
- `collector_runs`
- `archive_runs`

当前 `store_latest` 没有 `display_called_no` / `group_queues_json`，`store_bucket_rollups` 没有 `called_sample_count` / `called_no_*`。本地和 Worker 都做了兼容：如果未来 schema 加上叫号列，会自动读取；现在只能提供等待分钟、等位桌数和排队压力基准。

用户本机采样会继续融合进本机主曲线。把用户采样上传到线上库需要先设计并迁移贡献表，本次不默认开启。

## 当前默认地址（workers.dev）

本仓库桌面端默认云端地址：

```text
https://sushiro-cloud.sushiro-ryujoxys.workers.dev
```

Turso / GitHub OAuth / SESSION 等密钥只配置在 Worker secrets，不写进源码、不下发到本机。

自定义域（如 `sushiro-cloud.ryujo.online`）可选；域名到期时改用 workers.dev 并更新 GitHub OAuth 回调即可。

## GitHub OAuth App

在 GitHub 创建 OAuth App：

- Homepage URL: Worker URL，例如 `https://sushiro-cloud.<account>.workers.dev`
- Authorization callback URL: `https://sushiro-cloud.<account>.workers.dev/auth/github/callback`
- Scope: Worker 只请求 `read:user`

创建后拿到：

- `GITHUB_CLIENT_ID`
- `GITHUB_CLIENT_SECRET`

## Cloudflare Worker

Worker 源码位于 `cloudflare/sushiro-cloud/`，无运行时 npm 依赖。

必需 secrets：

```text
GITHUB_CLIENT_ID
GITHUB_CLIENT_SECRET
SESSION_SECRET
TURSO_DATABASE_URL
TURSO_AUTH_TOKEN
```

可选变量：

```text
ALLOWED_GITHUB_LOGINS=
SESSION_TTL_SECONDS=2592000
```

`ALLOWED_GITHUB_LOGINS` 是 fail-closed：留空时 Worker 会拒绝所有 GitHub 登录（登录回调与每次 session 校验都会失败），任何账号都无法访问云端基准接口。要放行自己和朋友，用英文逗号分隔 GitHub login，并通过 secret 配置：

```bash
npx wrangler secret put ALLOWED_GITHUB_LOGINS
# 输入例如：alice,bob
```

## 部署命令

```bash
cd cloudflare/sushiro-cloud
npx wrangler login
npx wrangler secret put GITHUB_CLIENT_ID
npx wrangler secret put GITHUB_CLIENT_SECRET
npx wrangler secret put SESSION_SECRET
npx wrangler secret put TURSO_DATABASE_URL
npx wrangler secret put TURSO_AUTH_TOKEN
npx wrangler deploy
```

部署后把 Worker URL 填到本地 UI：

```text
设置 -> 云端数据 -> Cloudflare Worker URL -> 保存 URL -> 用 GitHub 登录
```

## 免费托管说明

- Worker 不使用 KV/D1/R2，不产生 Cloudflare 存储资源。
- Worker 每次 baseline 请求会向 Turso 发 3 个固定 SELECT；单店请求同样只查固定门店数据。
- 低频个人使用适合 Cloudflare Workers Free；如果开放给很多用户，需要再加缓存和限流。
