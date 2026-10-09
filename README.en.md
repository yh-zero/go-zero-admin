# go-zero-admin

An administration backend built with go-zero, GORM and Casbin, paired with the Vue Vben Admin web-antdv-next frontend.

The backend provides user/role/menu/API authorization, dictionaries, organization and file management, audit logs, device sessions and an optional AI Agent using external DeepSeek or Qwen models.

- HTTP API: port 7001; business RPC: 6001; independent AI RPC: 6002.
- Local development: Docker starts MySQL, Redis and etcd; the three Go services and the frontend run separately.
- AI settings are in `application/ai/rpc/etc/ai.yaml`; model API keys are supplied only to the AI process through its environment.
- Fresh installations import [data/db/gozero-admin.sql](data/db/gozero-admin.sql) into the configured empty database. It includes the current permission version/history tables, stable button keys, administrator editor access, essential seeds and migration records. It contains no audit, device session or AI history. Compose does not import SQL automatically; do not reapply the permission schema migration after this fresh import.
- Existing databases use `Status`, `Backup` and `Migrate` (`status`, `backup`, `migrate` in Shell). Never replay the full SQL over an existing database.

For local installation, start dependencies with `docker compose up -d` and wait until MySQL, Redis and etcd are healthy. Select the configured database in a database GUI and import the SQL file, or run these commands from the backend root in Windows PowerShell:

```powershell
docker cp .\data\db\gozero-admin.sql gozero-mysql:/tmp/gozero-admin.sql
docker exec gozero-mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root --database="$MYSQL_DATABASE" < /tmp/gozero-admin.sql'
```

The import uses the database and credentials already configured in the MySQL container. The SQL has no `CREATE DATABASE` or `USE` statement. After import, start business RPC, AI RPC, API and the frontend separately. For upgrades, stop all three backend services before backing up and applying new migrations.

Existing installations first apply [the permission schema migration](data/db/migrations/20261008_permission_revision_history.sql) with the normal migration tool. Review and run [the administrator recovery bootstrap](data/db/operator/20261008_permission_admin_recovery_bootstrap.sql) separately while services remain stopped. It only restores the built-in administrator's replacement editor and menu move preview when its legacy recovery grants exist. Verify the two grants before restarting; history, rollback and ordinary roles receive API permissions explicitly through API synchronization and role authorization. Upgrade the frontend and backend together; old clients without `expectedRevision` cannot save grants.

Permission edits use consistent snapshots and mandatory revisions; menu moves require a bound impact preview. Button keys remain stable. Rollback is conservative and rejects an event after any later configuration change. User deletion retains owned resources and audit identity; resource transfer is a separate action. Optional RPC `UserResources.AIDataSource` must describe the AI service's actual MySQL network/address/schema and match the applet database before AI transfer is available. Otherwise only file transfer is available, with an explicit AI reason.

Session maintenance is opt-in: `go run ./application/applet/rpc/cmd/sessioncleanup -f application/applet/rpc/etc/applet.yaml -retention-days 90 -batch-size 200` previews old terminal sessions. Add `-apply` explicitly after reviewing the result. Valid/recent sessions and audits remain; no retention schedule is created automatically.

See [README.md](README.md) for installation, startup, configuration, code generation, tests and deployment; [Docker deployment](docker/部署说明.md) for production service authentication and upgrades; [development plan](DEVELOPMENT_PLAN.md) for unfinished work and acceptance checks.

Licensed under [Apache License 2.0](LICENSE). The frontend preserves its upstream MIT license.
