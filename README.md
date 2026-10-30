# Sushiro Overdose 公开排队数据

寿司郎门店公开排队快照的半小时代表值与统计，由 `.github/workflows/public-data.yml` 定时从维护者数据库只读导出。
不含任何账号、手机号、票号或凭证。格式说明见主分支 `docs/public-data.md`。

- `meta.json`：版本、来源、时间范围、覆盖统计、采集器状态、节假日表和各文件 SHA-256
- `stores.json`：门店与覆盖天数
- `latest.json`：每店最近一次公开状态（不是实时）
- `patterns/<门店ID>.json`：按日期类型的半小时规律（等待 P50/P80、叫号 P20/P50/P80）
- `daily/<门店ID>.json`：每天每半小时的代表值
