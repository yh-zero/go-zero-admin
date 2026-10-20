# data/db 数据库资产

| 路径 | 角色 |
|------|------|
| `gozero-admin.sql` | 全新安装的出厂快照（**生成文件，勿手工编辑**）；新 MySQL 数据卷由 docker-entrypoint-initdb.d 自动导入一次，禁止导入已有库 |
| `baseline.sql` | 出厂快照的基线部分：完整表结构与种子数据 |
| `migrations/` | 已有库的增量迁移（仅 `20261008` 起；更早的已合并进 `baseline.sql` 并移除文件），按文件名排序手工执行，checksum 登记入 `schema_migrations` 表；升级流程见根 README「数据库升级」 |
| `operator/` | 迁移后的可选管理员授权脚本，需停 API/RPC 显式执行，非必经步骤 |
| `backups/` | 升级前的全量备份 + SHA-256 校验文件，只保留最新一份并纳入 Git |
| `*_test.go` | 守卫测试：逐字节校验出厂 SQL 与迁移文件一致、checksum 正确、结构可真实导入 |



## 开发库过期检查

开发库由 `docker-compose.yml` 挂载 `gozero-admin.sql` 到 `/docker-entrypoint-initdb.d/`，**只在新数据卷上执行一次**；
存量卷不会随SQL 更新而变化。更新 `gozero-admin.sql` 后对比两个数字（应相等，当前为 35）：

```sh
docker exec -e MYSQL_PWD=123456 gozero-mysql mysql -uroot -N -e \
  "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()"
grep -c 'CREATE TABLE' data/db/gozero-admin.sql
```

不一致就重置开发库（只影响 MySQL，其他服务与前端不受影响）：

```sh
docker compose down -v mysql && docker compose up -d mysql
```

## 新增迁移

1. 在 `migrations/` 新建 `YYYYMMDD_描述.sql`，幂等编写（`IF NOT EXISTS` / `NOT EXISTS` 守卫，可安全重试）；
2. 如需给角色 1 授权，另建 `operator/YYYYMMDD_描述.sql`；
3. 在 `data/scripts/build-fresh-sql.ps1` 的 `$TailEntries` 追加对应条目，运行该脚本重新生成 `gozero-admin.sql`（checksum 自动计算）；
4. 运行 `go test ./data/db/` 与 `data/scripts/verify.ps1` 验证。

已被任何环境应用的迁移文件不可再修改（checksum 已入 `schema_migrations`）。

2026-10-03 及更早的迁移已全部合并进 `baseline.sql`，独立文件已移除；`schema_migrations` 中的历史记录保留，但不再有对应文件可供校验。
