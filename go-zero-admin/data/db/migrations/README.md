# 数据库迁移

001/002 已合并进 `data/db/init/mysql-all.sql`，全新初始化预置 `schema_migrations(version=2, dirty=false)`，历史变更 SQL 已清除。保留 `000002_baseline.down.sql` 空基线占位，因为 golang-migrate 在 Up 前需要确认当前版本仍存在于迁移源中。

后续新增迁移从 `000003_<name>.up.sql` / `000003_<name>.down.sql` 开始，使用 `pkg/cmd/migrate` 或 Docker `migrate` 服务执行。
