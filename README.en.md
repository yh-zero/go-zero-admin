# go-zero-admin

An administration backend built with go-zero, GORM and Casbin, paired with the Vue Vben Admin web-antdv-next frontend.

The backend provides user/role/menu/API authorization, dictionaries, organization and file management, audit logs, device sessions and an optional AI Agent using external DeepSeek or Qwen models.

- HTTP API: port 7001; business RPC: 6001; independent AI RPC: 6002.
- Local development: Docker starts MySQL, Redis and etcd; the three Go services and the frontend run separately.
- AI settings are in `application/ai/rpc/etc/ai.yaml`; model API keys are supplied only to the AI process through its environment.
- Database upgrades use incremental migrations and backups. Do not replay the baseline SQL over an existing database.

See [README.md](README.md) for installation, startup, configuration, code generation, tests and deployment; [Docker deployment](docker/部署说明.md) for production service authentication and upgrades; [development plan](DEVELOPMENT_PLAN.md) for unfinished work and acceptance checks.

Licensed under [Apache License 2.0](LICENSE). The frontend preserves its upstream MIT license.
