# AGENTS.md — 采集器部署指南（给 LLM / 自动化）

> 目标：**零猜测地把 sushiro 采集器部署到一台服务器并跑起来**。按顺序执行，每步有验证检查点。人类看 README.md。

## 这是什么

`collector/` 是 Python 采集器（3.9+，纯标准库）：每 15 分钟调寿司郎公开接口，把全国 130+ 家店的排队压力和叫号写进 Turso 的 `queue_records`（每店每半小时一条）。Turso 走 HTTP，无需 SDK。库结构与读法见 [查询指南](../docs/turso-readonly-query-guide.md)；公开副本见 [public-data.md](../docs/public-data.md)。

## 前置条件

1. `python3 --version` ≥ 3.9。
2. Turso 库 `libsql://su-shiro-ryujoxys.aws-us-west-2.turso.io` 的**读写** token（只读 token 写入会失败）。
3. 出站 HTTPS 能访问 Turso 和寿司郎接口。

> **当前状态（2026-10-20）：采集方式待适配。** 寿司郎换了接口地址和认证方式，本采集器用的固定公共 token 已不被接受（每轮 HTTP 403）。新 token 绑定会员账号、一小时过期，脱离微信续期尚未验证。适配前部署也采不到数据，见 [线上现状](../docs/cloudflare-github-turso.md#数据库与采集2026-10-20)。

## 部署步骤

### 1. 代码与 venv

```bash
git clone https://github.com/Ryujoxys/sushiro-overdose.git
cd sushiro-overdose/collector
python3 -m venv venv
```

### 2. 配置（不进 git）

```bash
cp config.example.json config.json   # 或 cp .env.example .env 用环境变量
```

填 `turso.url`、`turso.auth_token`（读写）、`sushiro.token`。**验证**：
`venv/bin/python -c "from collector.config import load_config; print(bool(load_config()['turso']['auth_token']))"` → `True`。

### 3. 建表、日历、门店

```bash
venv/bin/python -m collector.main init-schema     # ✅ 现有表含 queue_records / store_latest / store_dimension / holiday_calendar / collector_runs
venv/bin/python -m collector.main seed-holidays    # ✅ 写入 N 条节假日
venv/bin/python -m collector.main bootstrap        # ✅ store_dimension 现有 1xx 行
```

库若还是 v1（有 `queue_snapshots`），先按 README「从 v1 升级」迁移，不要直接启动新代码：新代码写 v2 的 `store_latest` 列。

### 4. 跑一轮

```bash
venv/bin/python -m collector.main run-once
```

**验证**：`{'stores_seen': 1xx, 'snapshots_written': 1xx, 'detail_ok': 1xx, ...}`；营业时段 `detail_ok` 接近门店数。

### 5. systemd 常驻

改 `collector.service` 的 `WorkingDirectory` 和 `ExecStart` 路径，然后：

```bash
sudo cp collector.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now collector
```

**验证**：`systemctl is-active collector` → `active`；`journalctl -u collector -f` 每 15 分钟一轮。

### 6. 确认数据在涨

```bash
venv/bin/python -c "
from collector.config import load_config
from collector.turso import TursoClient
c=load_config(); t=TursoClient(c['turso']['url'], c['turso']['auth_token'])
print(t.execute('SELECT MAX(slot) AS s FROM queue_records')[0]['s'])
print(t.execute(\"SELECT started_at, ok, substr(error_message,1,80) AS e FROM collector_runs ORDER BY started_at DESC LIMIT 3\"))
"
```

`slot` 应是当前半小时；最近几轮 `ok=1`。

## 命令一览

| 命令 | 作用 | 频率 |
|------|------|------|
| `init-schema` | 建表（幂等） | 首次 |
| `seed-holidays [--year N]` | 节假日日历 | 首次 / 每年 |
| `bootstrap` | 门店维度 | 首次 |
| `run-once [--stores id...] [--no-detail]` | 一轮采集 | 验证 |
| `migrate-v2 [--drop]` | v1 → v2；`--drop` 不可逆 | 升级一次 |
| `run` | 每 15min 采集 + 每天 03:00 清理 | systemd |

## 配置项

```json
{
  "sushiro": { "base_url", "token", "referer", "user_agent" },
  "turso": { "url", "auth_token" },
  "collect": { "interval_seconds": 900, "store_ids": [], "active_hours": [10, 22], "concurrency": 8 },
  "retention": { "records_days": 0, "runs_days": 30 }
}
```

`records_days=0` 永久保留半小时记录（约每年 60MB）；v1 的 `archive.retention_days` 已不生效。

## 缩小数据库文件

Turso HTTP 接口不允许 `VACUUM`。删表后空闲页会被之后的写入复用，文件本身不会立刻变小。要立刻缩小：

1. 停采集器；用只读查询把 5 张表复制进一个新的本地 SQLite 文件（`journal_mode=WAL`）。
2. 用 Turso CLI 从该文件建新库（如 `turso db create su-shiro-v2 --from-file <file>.db`，以 `turso db create --help` 为准），为新库生成读写和只读 token。
3. 核对新库 `queue_records` 行数和 `MAX(slot)`，更新采集器配置和仓库 Actions secrets，再启动采集器。确认无误后删除旧库。

## 故障排查

| 现象 | 原因 / 处理 |
|------|------|
| 401 Unauthorized | Turso token 错或过期；控制台重生成**读写** token |
| `HTTP 403 stores: <html>…403 Forbidden` | 寿司郎已换接口地址和认证方式，固定公共 token 失效；要改成新接口 + 会员授权 token，见上方「当前状态」 |
| `stores_seen: 0` / `detail_ok=0` | 接口不通，同上 |
| 营业时段 `display_called_no` 全是 NULL | 门店都关了，或 `getStoreById` 结构变了，查日志 |
| `数据库还是 v1 结构` | 库里还有旧表，先按 README「从 v1 升级」执行 `migrate-v2` |
| systemd 秒退 | 路径错；`journalctl -u collector` |
| `缺少配置` | config.json 仍是 `PUT_` 占位 |

## 安全

- `config.json` / `.env` / `venv/` 都在 `.gitignore`；读写 token 只放 EnvironmentFile 或 config.json，权限 0600。
- 日志里 token 已 redact；错误信息只存前 300 字符。
- 公开导出只用**只读** token（仓库 Actions secrets），不复用采集器的读写 token。

## 关键设计

- **一张事实表**：每店每半小时一条，`(slot, store_id)` 主键、`WITHOUT ROWID`，按时间追加；同一半小时后一帧覆盖前一帧，乱序旧帧不覆盖。
- **两个接口**：`stores?`（全国压力，无叫号）+ `getStoreById?`（叫号 `groupQueues`）；详情失败用列表帧兜底，叫号记 NULL。
- **未知不当 0**：等待、桌数、叫号缺失都是 NULL；等待超过 180 分钟按原值存、读时当未知。
- **不存聚合**：规律由读取方算（`scripts/export-public-data.mjs` 是参考实现），库里没有需要夜间重算的表。
