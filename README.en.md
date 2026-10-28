# go-zero-admin

An administration backend built with go-zero, GORM and Casbin, paired with the Vue Vben Admin web-antdv-next frontend.

The backend provides user/role/menu/API authorization, dictionaries, organization and file management, audit logs, device sessions and an optional AI Agent using external DeepSeek or Qwen models.

Running the backend locally requires Docker and Docker Compose v2; a host Go installation is only needed for code generation and checks. Those tools require Go 1.27.0 or newer; the recommended toolchain and Docker builder use Go 1.27.1. The framework is pinned to go-zero v1.10.3, and code generation uses goctl v1.10.2. The framework and generator have independent release versions. Install the generator with `go install github.com/zeromicro/go-zero/tools/goctl@v1.10.2` and add `GOBIN` (or `GOPATH/bin` when unset) to PATH.

- HTTP API: `http://127.0.0.1:7001`; Swagger UI: `http://127.0.0.1:7000`. Business RPC port 6001 and AI RPC port 6002 are internal to the container network and are not published to the host.
- The root Compose starts seven services: MySQL, Redis, etcd, business RPC, AI RPC, API and Swagger UI. The frontend runs separately with Node.js `^22.18.0` or `^24.12.0` and pnpm `11.16.0`.
- Containers use one `docker/config.yaml.template` split into `# >>> api`, `# >>> rpc`, `# >>> ai-rpc` sections plus a shared `common` section; the entrypoint extracts the sections listed in `CONFIG_SECTIONS` and runs one `envsubst` pass to generate `/tmp/config.yaml`. They do not read host YAML files under `application/.../etc/`.
- On a fresh MySQL volume, the root Compose automatically imports the mounted [data/db/gozero-admin.sql](data/db/gozero-admin.sql). It includes the current permission version/history tables, stable button keys, administrator editor access, essential seeds and migration records. It contains no audit, device session or AI history. Existing volumes are neither reimported nor automatically migrated; do not reapply the permission schema migration after a fresh import.
- Maintain existing databases through a MySQL client using the upgrade steps below. Never replay the full initialization SQL over an existing database.

Run these commands from the backend root to build the backend and wait for healthy services:

```powershell
docker compose up -d --build --wait
docker compose ps
docker compose logs -f api rpc ai-rpc
```

In the sibling `go-zero-admin-vben` directory, run `pnpm install --frozen-lockfile`, then `pnpm dev`. Open `http://localhost:5999` and log in with `admin / 123456` and the image CAPTCHA. Stop the backend with `docker compose down`, which retains the database volume; start or update it with `docker compose up -d --build --wait`. Adding `-v` to `down` deletes the volume.

AI is disabled by default in the `ai-rpc` section of `docker/config.yaml.template`, and the root Compose explicitly supplies empty model keys. To enable it, set `AI.Enabled: true`, choose the provider and configure its model and base URL in that template. Only in `ai-rpc.environment`, replace the selected key's empty value with an environment reference, for example `API_KEY_DEEPSEEK: "${API_KEY_DEEPSEEK:-}"`, and store its value in the ignored root `.env` file. Keep the other key empty and do not pass model keys to API or business RPC. Apply the change with `docker compose up -d --build --wait --no-deps ai-rpc`. Editing host `ai.yaml`, setting a host environment variable alone, or restarting an existing container does not apply this configuration.

Maintain existing databases manually with a MySQL client; Docker does not upgrade them automatically:

1. Run `docker compose stop api ai-rpc rpc` and keep MySQL running. Connect to the actual application database, export a complete backup and verify its integrity and recoverability. Store SQL backups and their SHA256 checksum files in `data/db/backups/` and track them in Git; the directory remains excluded from the Docker build context. Manually retain only the newest historical backup in the repository by the snapshot timestamp in its filename, keeping the accompanying checksum file together. Initialization SQL and incremental migrations are excluded from backup rotation.
2. Compare `schema_migrations.filename` / `checksum` with `data/db/migrations/`. Apply only unapplied files in filename order. Stop on a checksum mismatch for an applied file; if historical records are missing, inspect the actual schema before rerunning anything.
3. Configure the client to stop immediately on SQL errors; do not run later statements or migrations, or record a failed file. Inspect any partial changes before proceeding. After each complete success, maintain its `filename`, `checksum` and `applied_at` record. Calculate SHA-256 over the file encoded as UTF-8 without a BOM, with CRLF normalized to LF, and store 64 lowercase hexadecimal characters.
4. Verify the schema, data and migration records, then run `docker compose up -d --build --wait` and check service health. Never replay the full initialization SQL over an existing database.

Migrations from 20261003 and earlier are merged into `data/db/baseline.sql` and their standalone files removed; their `schema_migrations` records remain as history and no checksum comparison applies. Installations deployed before 2026-10-07 are no longer provided upgrade files; recover them from Git history if truly needed.

Server deployment requires Docker and Docker Compose v2, plus the complete backend and sibling frontend sources. Run every deployment step through `docker/deploy.sh` (`init` → `check` → `up` → `import`); it pins `--env-file docker/.env.deploy -f docker/deploy-compose.yml` internally. This independent Compose builds Go and Node/pnpm inside Docker and includes the production `web` service. Its default frontend context is `../../go-zero-admin-vben`, relative to `docker/`; set `FRONTEND_CONTEXT` in the deployment environment file for a different layout. Web listens on `127.0.0.1:8080` and proxies `/api` to `api:7001`; API listens on `127.0.0.1:7001`, optional Swagger on `127.0.0.1:8081`, and database/RPC ports remain internal. A server HTTPS entry point forwards public traffic to local port `8080`; configure its domain and certificate for your deployment. Production volumes stay separate from local development. First start MySQL, Redis and etcd, explicitly import SQL into the empty database, then build and start the application and web services. Existing installations stop `api ai-rpc rpc`, back up and migrate before restarting. See [Docker deployment](docker/部署说明.md).

Older installations apply [the permission schema migration](data/db/migrations/20261008_permission_revision_history.sql) through the manual process above only if it is unapplied. Review and run [the administrator recovery bootstrap](data/db/operator/20261008_permission_admin_recovery_bootstrap.sql) separately while services remain stopped. It only restores the built-in administrator's replacement editor and menu move preview when its legacy recovery grants exist. Verify the two grants before restarting; history, rollback and ordinary roles receive API permissions explicitly through API synchronization and role authorization. Upgrade the frontend and backend together; old clients without `expectedRevision` cannot save grants.

Permission edits use consistent snapshots and mandatory revisions; menu moves require a bound impact preview. Button keys remain stable. Rollback is conservative and rejects an event after any later configuration change. User deletion retains owned resources and audit identity; resource transfer is a separate action. Optional RPC `UserResources.AIDataSource` must describe the AI service's actual MySQL network/address/schema and match the applet database before AI transfer is available. Otherwise only file transfer is available, with an explicit AI reason.

Device sessions are retained after expiry or revocation, and no automatic purge is scheduled. Terminal rows can be pruned manually when the table grows, for example `DELETE FROM sys_device_sessions WHERE created_at < NOW() - INTERVAL 90 DAY AND ((revoked_at IS NULL AND expires_at < NOW() - INTERVAL 90 DAY) OR (revoked_at IS NOT NULL AND revoked_at < NOW() - INTERVAL 90 DAY));`. Live and recently revoked sessions and audit records are never touched by that statement.

See [README.md](README.md) for installation, startup, configuration, code generation, tests and deployment; [Docker deployment](docker/部署说明.md) for production service authentication and upgrades.

Development tools in `data/scripts/` are `gen.ps1` for code generation, `build-fresh-sql.ps1` for regenerating `data/db/gozero-admin.sql` from `baseline.sql` and the tail migration/operator files (`-Check` verifies freshness without writing), and `verify.ps1` for Go formatting, static checks, tests and builds.

Licensed under [Apache License 2.0](LICENSE). The frontend preserves its upstream MIT license.
