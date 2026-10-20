# go-zero-admin

基于 **go-zero + GORM + Casbin** 的管理后台，配套前端为 **Vue Vben Admin**。包含用户、角色、菜单、API 权限、字典、组织、文件、审计、AI 助手，以及独立的 App 用户与会员模块。

- [后端仓库](https://github.com/yh-zero/go-zero-admin) : https://github.com/yh-zero/go-zero-admin
- [前端仓库](https://github.com/yh-zero/go-zero-admin-vben) : https://github.com/yh-zero/go-zero-admin-vben

## 接口文档

- [管理后台 Swagger](http://localhost:7000) · [管理接口 JSON](data/api/generated/go-zero-admin.swagger.json)
- [App Swagger](http://localhost:7000/?audience=app) · [App 接口 JSON](data/api/generated/go-zero-app.swagger.json)

两份文档由同一生成命令产出。管理文档包含 92 个 HTTP 接口（本模块新增 8 个），App 文档包含 12 个；App 接口不会进入后台 API 授权资源同步。

## 环境要求

- 后端日常运行只需 Docker 与 Docker Compose v2，无需在本机安装 Go。
- 本机代码生成与检查需要 Go ≥ 1.27.0，项目推荐工具链固定为 `go1.27.1`，与 Docker 构建环境一致。
- 后端框架固定为 `go-zero v1.10.3`，代码生成工具固定为 `goctl v1.10.2`。两者独立发布，不要求版本号相同。
- 前端：Node.js `^22.18.0` 或 `^24.12.0`，pnpm `11.16.0`。
- 下列本机脚本使用 Windows PowerShell 5.1；命令均在对应仓库根目录执行。

## 快速开始

```powershell
# 在后端根目录构建并启动完整后端，等待服务健康
docker compose up -d --build --wait
docker compose ps

# 查看应用日志（Ctrl+C 退出日志查看，服务继续运行）
docker compose logs -f api rpc ai-rpc
```

根目录 Compose 启动 MySQL、Redis、etcd、业务 RPC、AI RPC、API、Swagger UI 和 RustFS 八个常驻服务，以及执行完成后退出的 `storage-init`。API 为 `http://127.0.0.1:7001`，Swagger 为 `http://127.0.0.1:7000`；两个 RPC 的 `6001` / `6002` 仅容器内部可达，不映射到宿主机。默认配置供本机开发使用。

新 MySQL 数据卷会自动执行挂载的 `data/db/gozero-admin.sql`，无需手动导库。该生成文件在历史基线、迁移和管理员初始化脚本后依次追加 `data/db/seeds/demo.sql`、`data/db/seeds/activity.sql`，提供完整可体验的演示数据，只供空库初始化；已有卷不会重新导入，也不会自动应用新增迁移，升级按下文操作。

前端在同级 `go-zero-admin-vben` 目录单独运行：

```powershell
pnpm install --frozen-lockfile
pnpm dev
```

打开 `http://localhost:5999`，用 `admin / 123456` 和图片验证码登录。全新安装提供四个后台账号：`admin`、`operator`（运营人员）、`viewer`（只读观察员）、`disabled`（停用示例）。后三者密码均为 `Demo@123456`，`disabled` 的 `enable=2`，不能登录；三个后台角色 ID 分别为 1、2、3，停用账号绑定只读角色 3。运营可读取 App 用户与会员列表，审核和维护用户状态、头像，并上传和读取本人文件，无设备撤销、会员开通或撤销、系统配置权限；只读只开放首页及 App 用户、会员列表读取，无操作按钮。

App 提供六个示例邮箱，密码均为 `Demo@123456`：`member@example.test`（普通）、`vip@example.test`（导入 UTC 时间起 30 天 VIP）、`pending@example.test`（待审核）、`disabled@example.test`（禁用）、`rejected@example.test`（拒绝）、`expired@example.test`（VIP 已过期的普通账号）。状态分布为 3 个 active、1 个 pending、1 个 disabled、1 个 rejected，仅 active 账号可登录；三条 VIP 授予分别展示有效、过期和撤销，普通账号无有效 VIP。示例邮箱的 `verified_at` 均为空，不代表真实邮箱验证；App 账号不能登录管理后台。

演示数据还包含 3 个部门、2 个岗位、3 张实际 RustFS PNG（2 公开、1 私有）、1 条手工引用用于体验删除保护、3 条明确标记的内置审计，以及 1 条 AI 对话、2 条消息和 1 条已完成任务。图片素材随 `storage-init` 镜像提供，仅补齐缺失对象、不覆盖已有同键对象；不绑定用户头像或固定外部地址。AI 历史为预置内容，不调用模型、不产生费用；有效设备会话由真实登录生成。对外提供服务前应修改密码并处理不需要的示例账号。完整数据与权限说明见 [数据库文档](data/db/README.md)。

仓库随附的 `data/db/backups/` 只保留最新可公开演示库快照 `gozero-admin-demo-latest.sql` 及 SHA-256 校验文件。部署脚本后续写入此目录的私有备份继续由 Git 忽略，不能提交；真实业务备份应按部署方策略存放在受控位置，不能混入公开示例快照。本机历史私有备份归档在 Git 忽略的 `.cache/db-backup-archive/`，不作为生产备份方案。

停止并保留数据：`docker compose down`；再次启动或更新：`docker compose up -d --build --wait`。重置开发库前须确认目标并备份；删除持久卷属于有损操作，不用于常规更新。

## 配置

文件资源使用 RustFS 1.0.3 的 S3 接口，已移除原云存储 SDK。根 Compose 会自动创建 `gozero-public` / `gozero-private` 两个 Bucket、公开读取策略、下载跨域配置和独立应用身份，不会清空已有对象。存储数据保存在 `rustfs_data` 卷，日志在 `rustfs_logs` 卷，正常停止时保留持久卷。

- S3 地址：`http://127.0.0.1:9000`；控制台：`http://127.0.0.1:9001`。
- 本地控制台默认身份 `GOZEROLOCALROOT`，密钥由根 Compose 的 `RUSTFS_SECRET_KEY` 开发默认值提供，可在忽略的 `.env` 中覆盖；正式部署必须使用独立随机值。
- RPC 使用受限的 `STORAGE_ACCESS_KEY_ID` / `STORAGE_SECRET_ACCESS_KEY`，不接收 RustFS 管理凭据。API 和 AI RPC 不接收存储凭据。
- `STORAGE_EXTERNAL_ENDPOINT` 必须能从浏览器访问，本地默认 `http://127.0.0.1:9000`；正式部署使用反向代理的 HTTPS 文件域名，保留 Host、路径和查询参数，不把容器服务名返回给浏览器。
- `STORAGE_CORS_ORIGINS` 是允许下载的前端完整来源，多个用逗号分隔，本地默认 `http://localhost:5999,http://127.0.0.1:5999`。

“文件资源”支持 PNG/JPEG/GIF/WebP、最大 10MB，上传后自动登记资源并刷新列表。公开文件返回稳定地址；私有文件通过数据范围校验后生成 300 秒签名链接，到期可重新获取。下载前重新申请地址，删除受业务引用保护，存储删除失败可重试。用户头像使用公开图片，默认头像为前端自带的 `/default-avatar.svg`。

后台用户和 App 用户头像在保存事务中自动维护文件引用，替换或清空解除旧引用；共享图片只有所有使用者解除后才可删除。系统头像引用不能手工增删。App 用户管理新增“设置头像”（`app-members-users:avatar`）；App 接口读取 `avatarUrl`，自助资料仍只更新昵称。

已有库按数据库升级流程应用 `20261020_rustfs_avatar.sql`，再应用 `20261020_user_avatar_references.sql` 新增 App 头像列、回填公开头像引用并注册权限。需给内置管理员开通新功能时执行 `data/db/operator/20261020_avatar_admin_bootstrap.sql`；其他角色按需授权。新库初始化 SQL 已包含迁移和管理员 bootstrap，历史迁移保持不变，fresh-only seed 只整理全新安装数据。

自定义公开 Bucket 时，在同一迁移连接设置 `@avatar_public_bucket`；`docker/deploy.sh upgrade` 自动读取 `STORAGE_PUBLIC_BUCKET`（默认 `gozero-public`）。历史手工百分号转义地址应先规范化并重新保存头像。

- [配置模板](docker/config.yaml.template)：按 `# >>> api`、`# >>> rpc`、`# >>> ai-rpc` 分段，`common` 段供两个 RPC 复用
- [部署脚本](docker/deploy.sh)：`init` / `check` / `up` / `import` / `migrate-check` / `upgrade` / `rollback`
- [Swagger 反代](docker/swagger.conf.template) 与 [UI 初始化](docker/swagger-initializer.js)

`docker/entrypoint.sh` 按 `CONFIG_SECTIONS` 取出对应段落，通过一次 `envsubst` 生成 `/tmp/config.yaml`，不读取 `application/.../etc/` 下的本机 YAML。修改模板后需要重新构建相应镜像。API 与业务 RPC 的 `JwtAuth.AccessSecret` 必须一致；独立的 `AppJwtAuth.AccessSecret` 也必须在两者间一致，并使用不同于后台的 `APP_JWT_ACCESS_SECRET`（至少 32 字节）。业务 RPC 与 AI RPC 使用同一业务库。

## App 用户与会员

App 使用邮箱或手机号加密码公开注册，注册后为待审核状态；管理员在“App 用户”页面审核通过后才能登录。账号审核与 VIP 开通分别操作，审核通过只得到普通 App 账号。邮箱统一小写；手机号支持国际 E.164 或中国大陆 11 位号码。每个账号首版绑定一个登录身份。

后台管理员仍保存在 `sys_users`。App 新增 `app_users`、`app_user_identities`、`app_sessions`、`app_membership_plans`、`app_membership_grants` 五张表，使用独立令牌、会话和业务授权；App 用户及 VIP 均不获得后台访问权。访问令牌默认 15 分钟，刷新令牌默认 30 天，刷新时轮换并检测旧令牌复用；退出、撤销设备、禁用账号使相应会话失效，恢复账号需重新登录。

“App 会员”页面支持人工开通、续期和撤销。首版提供 `vip` 方案及 `profile.export` 权益，用于导出本人资料的完整接入示例；本人资料接口无需 VIP，会员也不能读取别人的资料。授予记录保留方案/权益快照，续期重试使用同一 `requestId`，撤销只影响指定记录。有效期实时判断，账号禁用期间会员仍按自然时间计时。本模块尚未接入支付、短信、基金业务或移动端客户端。

邮箱注册复用已有 Mail RPC 的 SMTP 发送、5 分钟验证码、限流和一次性校验，使用独立的 `app_register` 场景。同一客户端 IP、同一邮箱每天各最多请求 5 次验证码，按 `Asia/Shanghai` 自然日计数。默认 `APP_EMAIL_VERIFICATION_REQUIRED=false`，允许不提交验证码后人工审核；设为 `true` 后，邮箱注册必须提交 `emailCode`，仍需管理员审核。手机号首版由管理员人工核实。审核页面会区分邮箱已验证、邮箱未验证及手机号待人工核实。

SMTP 仅注入业务 `rpc`：在本地根 `.env` 或生产 `docker/.env.deploy` 配置 `SMTP_HOST`、`SMTP_PORT`（默认 465，隐式 TLS）、`SMTP_USER`、`SMTP_PASSWORD`、`SMTP_FROM`，再重建 RPC。App 客户端通过 `GET /v1/app/auth/options` 查询配置，用 `POST /v1/app/auth/email-code` 请求验证码。未配置 SMTP 不影响默认注册及基础后台启动；真实邮件投递仍需使用部署方的 SMTP 凭据验收。

注册时将解析出的客户端 IP 保存为 `app_users.registration_ip`，归属地保存为 `registration_region`，在 App 用户管理中展示。归属地使用 ip2region 离线 IPv4/IPv6 数据库，文件位于 `data/ipregion/`，不向外部接口发送用户 IP。容器中的实际查询已验证，升级前的旧账号保留空值，不推测历史注册位置。

注册、登录和验证码请求按同一来源的客户端 IP 限流。通过 Nginx 等代理访问时，`APP_TRUSTED_PROXY_CIDRS` 优先填写实际反向代理容器的单个 IP：IPv4 加 `/32`，IPv6 加 `/128`，多个用逗号分隔。空值忽略转发头，同一代理后的请求会共享代理 IP 的限额。本地 Swagger 启动后，可分两步获取其实际 IP，PowerShell 与 sh 均适用：

```sh
docker compose ps -q swagger-ui
# 将上一步输出的容器 ID 代入，选择该容器与 API 通信的网络 IP
docker inspect <容器ID> --format '{{range .NetworkSettings.Networks}}{{println .IPAddress}}{{end}}'
```

将确认后的 IPv4 地址加 `/32` 写入根 `.env` 的 `APP_TRUSTED_PROXY_CIDRS`，重新创建 API 容器。不要把整个 Docker 子网、宿主/NAT 网关或客户端地址范围列为可信代理；容器重建后 IP 可能变化，须重新检查并更新配置。生产 `web` 和其他实际受控反向代理同理，步骤见部署说明。

## AI 助手

默认关闭：`docker/config.yaml.template` 的 `ai-rpc` 段中 `AI.Enabled: false`，根 Compose 的两个模型 Key 显式为空，无 Key 也能启动 AI RPC。启用时在该模板设 `AI.Enabled: true`，选择 `Provider`（deepseek 或 qwen），填写对应 `Model` / `BaseURL`。

仅为 `ai-rpc` 配置所选提供商的环境变量引用，例如将其 Compose `environment` 中的 DeepSeek 项改为 `API_KEY_DEEPSEEK: "${API_KEY_DEEPSEEK:-}"`，实际 Key 写入已被 Git 和 Docker 构建上下文忽略的根目录 `.env`；另一项保持为空。业务 RPC 和 API 不接收模型 Key。随后执行 `docker compose up -d --build --wait --no-deps ai-rpc`，使模板和容器环境生效。容器不使用本机 `ai.yaml` 中的 `APIKey` 兜底；仅修改本机 YAML、设置宿主机环境变量或执行 `restart` 都不会替代上述配置与重建。

## 数据库升级

已有库通过 MySQL 客户端手工维护，Docker 不会自动升级现有库，也不要重放全量初始化 SQL：

1. 执行 `docker compose stop api ai-rpc rpc`，保留 MySQL 运行。连接实际业务库，导出完整备份并核验完整性及可恢复性；SQL 备份及配套 SHA256 校验文件保存到 `data/db/backups/`。本机备份不属于 fresh 初始化资产，包含本地数据时不应公开；新备份默认被 Git 忽略，Docker 构建上下文也排除该目录。
2. 核对 `schema_migrations` 的 `filename` / `checksum` 与 `data/db/migrations/`，按文件名顺序只执行未应用迁移。已应用文件的校验和不符时停止；历史记录缺失时先核实实际结构，不直接重跑。
3. 配置客户端在 SQL 失败时立即停止，不继续后续语句或迁移，也不登记失败项；核实部分执行状态后再处理。每个文件完整成功后维护对应的 `filename`、`checksum` 和 `applied_at` 记录；校验和为文件按 UTF-8 去 BOM、将 CRLF 转为 LF 后计算的 SHA-256，保存为 64 位小写十六进制。
4. 核验结构、数据和迁移记录后，执行 `docker compose up -d --build --wait` 并检查服务健康。

2026-10-03 及更早的迁移已合并进 `data/db/baseline.sql`，独立文件随之移除；其 `schema_migrations` 记录保留为历史事实，无需再核对校验和。2026-10-07 之前部署的库不再提供升级文件，确需时从 Git 历史取回。

备份及配套校验文件按部署方的保留策略存放到受控位置。数据库升级状态由迁移记录、校验和与实际结构确认，不能通过表数量判断。`seeds/demo.sql`、`seeds/activity.sql` 只用于全新安装，已有库升级不会执行它们，也不会重置已有账号或权限。

App 模块依次使用 [20261019_app_members.sql](data/db/migrations/20261019_app_members.sql) 和 [20261019_app_registration_ip.sql](data/db/migrations/20261019_app_registration_ip.sql)。已有库先完成此前未应用迁移，再按序应用这两份文件并分别登记校验和；已应用的文件和校验和保持不变，新增 IP/归属地字段通过后者升级，旧账号数据保留。两份迁移都不自动授权旧角色。需要给角色 1 开通该模块时，在维护窗口明确执行 [可选管理员授权](data/db/operator/20261019_app_member_admin_bootstrap.sql)，其他角色通过现有权限编辑器分别配置菜单、按钮和 API。全新安装 SQL 已包含两份迁移及角色 1 初始化。

## 代码生成

需要 goctl v1.10.2（RPC 另需 protobuf）：

```powershell
go version
go install github.com/zeromicro/go-zero/tools/goctl@v1.10.2

powershell -NoProfile -ExecutionPolicy Bypass -File data/scripts/gen.ps1 api applet      # API
powershell -NoProfile -ExecutionPolicy Bypass -File data/scripts/gen.ps1 rpc applet applet  # RPC
powershell -NoProfile -ExecutionPolicy Bypass -File data/scripts/gen.ps1 swagger          # Swagger 文档
```

API 生成使用项目自定义模板 [data/goctl/api/](data/goctl/api/)（统一响应格式与 JWT 失效处理）。

## 检查与测试

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File data/scripts/verify.ps1
```

检查包含 Go 格式、静态检查、测试与构建，统一执行各源码包中的 `*_test.go` 测试。

2026-10-20 已通过完整后端检查、真实 MySQL/Redis 测试及 Docker 重建，七个本地服务健康；App 注册、审核、登录、会员和会话的 31 项 HTTP 流程及 3 项注册 IP/归属地检查通过。管理页面的浏览器审核/VIP 按钮闭环尚未完成；未配置真实 SMTP，未验证实际邮件投递。

## 开发脚本

| 脚本 | 用途 |
|------|------|
| `data/scripts/gen.ps1` | 代码生成：`api` / `rpc` / `model` / `swagger` |
| `data/scripts/build-fresh-sql.ps1` | 由历史基线、迁移、管理员初始化脚本及 fresh-only `seeds/demo.sql`、`seeds/activity.sql` 重新生成空库初始化 SQL；`-Check` 仅校验不写入 |
| `data/scripts/verify.ps1` | 一键检查与测试 |

## 服务器部署

服务器只需 Docker 与 Docker Compose v2，同时保留后端及同级前端完整源码。独立的 `docker/deploy-compose.yml` 会在容器内构建 Go 后端和 Node/pnpm 前端，`web` 默认通过 `127.0.0.1:8080` 提供页面，并把 `/api` 转发到 `api:7001`；API 默认仅绑定 `127.0.0.1:7001`，Swagger 为 `127.0.0.1:8081`，数据库与 RPC 不发布端口。对外 HTTPS 由服务器入口转发到本机 `8080`，域名和证书按实际入口配置。

服务器上用 `docker/deploy.sh` 完成全部操作（`init` → `check` → `up` → `import`）；脚本内部固定使用 `--env-file docker/.env.deploy -f docker/deploy-compose.yml`。数据卷与本机根 Compose 分开；首次先启动依赖，向空库手动导入 SQL，再构建启动全部服务。已有库先停止 `api ai-rpc rpc`、备份与迁移。完整步骤见 [部署说明](docker/部署说明.md)。

## 相关文档

- [配套前端](https://github.com/yh-zero/go-zero-admin-vben) · [Apache 2.0 许可证](LICENSE)
