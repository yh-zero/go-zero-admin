# go-zero-admin 开发手册

> 本文回答“如何配置、启动、验证和排障”，按任务选读；不重复维护规范正文、接口契约或单次验收记录。
> 开发边界见[根规范](../AGENTS.md)、[后端规范](../go-zero-admin/AGENTS.md)、[前端规范](../go-zero-admin-react/AGENTS.md)；业务语义见[API 契约](api-contract.md)，已知缺口见[问题清单](README.md#后端剩余事项)。

## 1. 开发环境与配置

首次准备与工具版本见[快速开始](../README.md#快速开始)，以依赖声明、锁文件和实际工具版本为准；Docker 使用 Linux 容器模式，基础镜像须可拉取或已缓存。标准拓扑以 [Compose](../go-zero-admin/docker-compose.yml) 为准：

| 服务 | 宿主访问地址 | 容器网络 / 用途 |
|---|---|---|
| MySQL | `127.0.0.1:3307` | `mysql:3306`，业务库 `gozero_admin` |
| Redis | `127.0.0.1:16379` | `redis:6379`，缓存、会话与策略通知 |
| RustFS | `127.0.0.1:19000`，控制台 `:19001` | `rustfs:9000`，S3 兼容存储 |
| admin-api | `127.0.0.1:7001` | `platform-api:7001`，前端代理目标 |
| admin/app/file/ai/tenant RPC | 默认不映射宿主端口 | 各服务内部 `6001..6005`；tenant 仅平台栈 |
| Nginx | `:80` | 按域名路由；平台为 `localhost` / `platform.localhost` |
| 前端 dev | `http://127.0.0.1:5999` | 宿主运行，业务请求代理到 API |

### 配置双套制

| 运行方式 | 配置来源 | 地址与环境变量 |
|---|---|---|
| Compose | 后端 `docker/etc/*.yaml`，卷挂载到容器 | 使用容器服务名；AI/App 的 `env_file` 注入后端 `.env.local` |
| 宿主 `go run` | 后端 `application/*/etc/*.yaml` | 使用宿主映射端口；Go 不自动读取 `.env.local` |
| YAML 直填 | 所用配置字段的字面值 | 不因同名环境变量存在而自动覆盖 |
| YAML `${ENV_VAR}` | 进程环境变量 | 仅启用 `conf.UseEnv()` 的加载入口展开；字段留空仍为空 |

变更配置时核对宿主/容器两套及二次加载入口。当前仅 AI/App 主入口启用 `UseEnv()`；其他入口的统一覆盖见[待办](README.md#规范与自动化待对齐)，不能假定所有服务都支持占位符。

### AI 与邮件

在后端目录复制模板（不覆盖已有本地配置），再自行填写所需凭据；模板是变量说明，不是运行凭据文件：

```powershell
# 工作目录：go-zero-admin/
if (-not (Test-Path -LiteralPath .env.local)) { Copy-Item .env.local.example .env.local }
```

- 变量名见 [.env.local.example](../go-zero-admin/.env.local.example)，AI 启停按[契约](api-contract.md#ai-http-与执行约定)。默认占位符未取得密钥时模型降级；YAML 已直填有效密钥时，不要求存在 `.env.local`。公开主线禁止真实凭据。
- SMTP 须同时正确配置 Host/Port/ImplicitTLS，From 留空回退 Username。极简模板只有账号/授权码，不代表其他配置已正确；容器 Host 与宿主 465/TLS 的现有缺口见[问题清单](README.md#后端剩余事项)，不伪造发信成功。
- 宿主运行可直填 YAML，或在启动终端用 `$env:变量名=...` 设置进程环境；`setx` 不更新当前终端。env 变更须重新创建相关容器，单独 restart 不更新环境；租户不自动继承平台凭据。

### 数据初始化

**初始化 SQL 会 DROP 并重建业务表，不是已有库的升级脚本。** 唯一入口为 [mysql-all.sql](../go-zero-admin/data/db/init/mysql-all.sql)：
- 新建开发库：完整导入；Compose 的 MySQL 空数据卷首次启动自动导入，已有卷不自动覆盖或更新结构。
- 已有库改结构：先核对目标与数据处理方案；生成 model 时使用已导入完整 SQL 的新开发库，不为生成代码清空业务库。
- 重置已有库：须明确授权、确认目标并备份；不能把重跑初始化或删除数据卷当作通用排障手段。
- 演示账号与默认凭据见[快速开始](../README.md#快速开始)，仅限开发、部署前必须替换；后台图形验证码由 admin-rpc 的 `Captcha.Enabled` 控制。

## 2. 构建、启动与就绪检查

新建或重置数据库先看[数据初始化](#数据初始化)；常规代码部署不重复导入 SQL 或删除数据卷。

### Compose（日常默认）

[Dockerfile](../go-zero-admin/docker/Dockerfile)只复制宿主预编译的 Linux 二进制，`docker compose --build` **不会编译 Go 源码**。首次运行或后端代码变更后，先执行[构建入口](../go-zero-admin/data/scripts/build-images.ps1)；构建失败立即停止，不部署旧产物。

```powershell
# 终端 A：从仓库根目录执行
cd go-zero-admin
powershell -NoProfile -ExecutionPolicy Bypass -File ./data/scripts/build-images.ps1
if ($LASTEXITCODE -ne 0) { throw "后端编译失败，停止部署" }
docker compose up -d --build --wait
if ($LASTEXITCODE -ne 0) { throw "Compose 启动失败，先排查日志" }
docker compose ps
if ($LASTEXITCODE -ne 0) { throw "无法读取容器状态" }
```

前端另开终端；首次或依赖声明/锁文件变化后安装依赖：

```powershell
# 终端 B：从仓库根目录执行
cd go-zero-admin-react
pnpm install --frozen-lockfile
if ($LASTEXITCODE -ne 0) { throw "依赖安装失败，停止启动" }
pnpm dev
```

只改 YAML：重启读取它的服务；修改 `.env.local`：在后端目录 `docker compose up -d --force-recreate ai-rpc app-rpc`，检查退出码并复验对应功能。代码变更则先重新编译、构建镜像再创建容器，不用 restart 代替部署。

### 宿主单服务调试（手动重启，不是自动热更新）

先核对目标配置与 `svc` 依赖，不对所有服务套同一启动命令。例：从后端目录只启动依赖，再调试 admin-rpc：

```powershell
# 工作目录：go-zero-admin/；先确认不会干扰现有实例和数据
docker compose up -d mysql redis rustfs
if ($LASTEXITCODE -ne 0) { throw "基础依赖启动失败" }
go run ./application/admin/rpc -f application/admin/rpc/etc/admin.yaml
```

- 上例业务如需调用 FileRpc，还须保证对应端点可达；其他服务按实际依赖准备，不以进程能启动推定功能可用。
- 宿主 API 默认连 `127.0.0.1:6001/6003`，启动需访问 AdminRpc/FileRpc，并经 admin-rpc 加载 Casbin 策略；仅启动 MySQL/Redis/RustFS 不够。默认 Compose 的 RPC 不对宿主暴露，须运行所需宿主 RPC，或使用已明确授权的端口映射/混合环境。
- 端口冲突看宿主实际监听/映射：API 的 7001 会冲突，RPC 容器内部同号端口不必然冲突。不要顺手停掉现有服务，也不要把双实例当成同一运行版本；测试优先独立端口与隔离数据。

### 就绪不等于验收

确认启动命令、配置挂载及运行产物对应本轮改动；不确定时比较容器二进制与本地产物摘要。容器 Up / Compose wait 通过不等于全部业务可用：

```powershell
# 任意终端：仅探针示例，不代替业务验证
$ready = Invoke-RestMethod -Uri "http://127.0.0.1:7001/ready" -TimeoutSec 15 -ErrorAction Stop
if ($ready.code -ne 200 -or -not $ready.result.ready) { throw "API 尚未就绪" }
```

再检查 `/ping`、前端入口及受影响接口/页面；HTML 200 只说明页面可达，AI/邮件还需各自验证。持久失败查[排障表](#4-常见故障排查)，不靠不断重启掩盖错误。

## 3. 日常开发与验证流程

规范正文以各层 AGENTS 为准，这里只串执行顺序：

| 步骤 | 操作与完成边界 |
|---|---|
| 1. 确认范围 | 看现有差异、授权及相关规范；查实际字段/方法签名，不覆盖既有改动 |
| 2. 契约与来源 | 涉及字段/路由/权限先更新[契约](api-contract.md)；按[生成规则](../go-zero-admin/AGENTS.md#2-开发与生成规则)改 `.api` / `.proto` / SQL 并经 `gen.ps1` 生成，核对覆盖、注册及旧引用 |
| 3. 实现与回归 | 保持 handler/logic/model 职责；SQL、事务、缓存按[数据访问分层](../go-zero-admin/AGENTS.md#3-数据访问分层本仓库约定不表述为-go-zero-的强制限制)，每个受影响模块先做可重复回归再推进依赖模块 |
| 4. 静态门槛 | 后端运行[verify.ps1](../go-zero-admin/data/scripts/verify.ps1)；前端执行[类型/lint/构建](../go-zero-admin-react/AGENTS.md#8-代码提交前自检必过)。检查退出码；业务变更另跑受影响测试，无测试文件不代表已覆盖 |
| 5. 功能验收 | 用当前构建与配置做隔离联调/实际页面操作，按[根验收表](../AGENTS.md#8-任务完成定义)覆盖受影响正常、拒绝、状态竞争及失败路径；付费模型/真实邮件须在授权内 |
| 6. 交付 | 精确清理、保留回归测试，说明静态/回归/真实联调分别验证到哪里；未覆盖或失败列[待办](README.md#后端剩余事项)，不伪装全部通过 |

认证可复用[登录 HTTP 回归入口](../go-zero-admin/test/login_e2e.ps1)：只用专用测试账号，正常取得一次性图形码，不读数据库/缓存绕过验证，也不退出其他设备。纯文档只需检查内容、链接与 `git diff --check`，不强制构建或全量 E2E。CI、扫描及配置覆盖缺口见[待办](README.md#规范与自动化待对齐)，不能当成已修复。

## 4. 常见故障排查

把历史事故归纳成检查路径，不维护次数或重复禁令；查看日志时不回显凭据/完整请求，涉及写库或停服务先核对授权。下列 Compose 命令在后端目录执行：

| 症状 | 先检查与处理 | 如何确认 / 边界 |
|---|---|---|
| API 退出、反复重启 | `docker compose logs api admin-rpc --tail 50`；检查 admin-rpc 就绪、Casbin 策略与数据库/Redis | restart 策略只缓解启动时序；持续失败必须查因，再验证 `/ready` 与业务 |
| 7001 可达、Nginx 502 | API 重建后可能缓存旧上游地址，确认平台上游服务，再按下例校验并 reload | 用平台 Host 复验；未知 Host/IP 直访默认站点 404 是域名隔离设计 |
| 404 / 连接拒绝 / 请求到旧服务 | `netstat -ano` 查端口、PID 与容器映射，核对请求域名和代理目标 | 不停无关项目；选择独立测试端口，确认命中本轮实例 |
| `unknown service` / 新 RPC 不生效 | 核对调用方连接的模块/端口及 main 的服务注册，不能把 FileRpc 误连 admin-rpc；查 `gen.ps1 rpc` 告警 | 生成命令成功不保证已有 main 已补注册；调用新增方法复验 |
| pb 字段/路由签名不匹配 | 读取定义和生成类型，逐项核对，不用正则批改 proto 字段或手改生成物 | 按[生成规则](../go-zero-admin/AGENTS.md#2-开发与生成规则)重生成后检查编译与实际绑定 |
| 多 protobuf 包注册冲突 / etc 出现重复 YAML | 核对模块独立 proto package、`--name-from-filename` 和运行 `-f` 路径 | 不因重复就盲删配置；保留模块配置与手写内容，按规则消除冗余 |
| `not matching destination to scan` / 时间查询报数据库错误 | 自定义查询用带 `db` tag 的 row struct，不直扫 pb；时间转 int64 按规则用 `UNIX_TIMESTAMP()` 与显式别名 | 逐列检查查询/目标类型并真实读取，不以编译通过代替扫描验证 |
| 直接改库后仍读旧数据 | 查看实际生成 model 的 ID/旧新唯一索引缓存键，仅精确失效受影响记录；授权变更核对策略通知 | 重新读取确认；不以 FLUSH 或全库清空代替定位 |
| AI/SMTP 填了配置却不可用 | 核对[配置双套制](#配置双套制)、实际加载入口、Provider、SMTP 主机/端口/TLS 和容器是否重建 | 区分未配置降级与外部调用失败；不打印密钥或把占位符展开误认为读取 env 文件 |
| PowerShell 中文乱码、解析失败或 SQL 导入乱码 | 区分终端显示、文件编码与数据库存储；先 `Get-Content -Encoding UTF8`/原始字节核对，再考虑修改文件 | 兼容 Windows PowerShell 5.1 的含中文脚本保留 UTF-8 BOM；不要因显示乱码重写源文件 |
| 升级声明与运行结果不符 | 核对依赖声明、锁文件、实际 `go list -m` / `pnpm list` 及运行产物 | 未批准不升级；没有成功执行的命令不记作已生效 |

```powershell
# 后端目录；仅在本项目确需刷新上游时执行，校验失败不 reload
docker compose exec -T nginx nginx -t
if ($LASTEXITCODE -ne 0) { throw "Nginx 配置校验失败" }
docker compose exec -T nginx nginx -s reload
if ($LASTEXITCODE -ne 0) { throw "Nginx reload 失败" }
```

代理复验应带 `Host: localhost`（例如向 `http://127.0.0.1/ready` 发请求），不修改默认站点来绕过租户隔离。

SQL 传输优先复制 UTF-8 文件到容器后重定向导入，并显式 `--default-character-set=utf8mb4`，避免 PowerShell 管道转码；凭据走安全输入/进程环境而非命令明文。怀疑二次编码时先备份、核对实际字符与目标行列，再按授权精确修复，不套用无条件 UPDATE 模板。PowerShell 续行用反引号而非反斜杠，脚本执行策略仅对本次调用使用 `-ExecutionPolicy Bypass`，不修改机器全局策略。

## 5. 清理与恢复

按[根完成定义](../AGENTS.md#8-任务完成定义)处理，原则是只恢复本轮临时改动，而不是清空整个环境：
1. 记录并按准确 PID/任务 ID、容器名、路径与临时数据 ID 清理；保留用户要求运行的项目栈，不按进程名/通配符批量删除。
2. 恢复本轮临时配置与经授权调整的服务，确认测试端口释放；回归测试、用户数据及既有备份保留。
3. 用 `git status`/差异确认已有改动未被覆盖；可以保留本任务未提交内容，不为“工作区干净”执行 checkout/reset。测试 Git/hook 使用独立临时仓库，不在真实仓库造提交再撤销。

## 6. 专项维护入口

### 登录猫动画

维护细节与唯一生成命令见[素材说明](../go-zero-admin-react/public/mascots/README.md)、[生成器与测试](../go-zero-admin-react/scripts/build-login-cat.md)。来源图集为 [login-cat-atlas.png](../go-zero-admin-react/src/assets/login-cat-atlas.png)（3×3、1536×1536），生成 `.riv` 与静态备用图，不手改产物；依赖版本查前端声明/锁文件，历史授权不等于允许任意加依赖。

| 场景 | 保留的行为边界 |
|---|---|
| 闲置/指针 | 呼吸、眨眼、左右转头，进出阈值不同以防边界闪动 |
| 表单/密码聚焦 | 看向表单；密码框聚焦时捂眼、离开后放爪；动画只接收聚焦/结果状态，不读取密码内容 |
| 登录成功 | 成功反馈后约 1.1 秒保存会话并跳转原目标；资源未就绪/窄屏直接完成，不无限等待动画事件 |
| 登录失败 | 失败反馈不取代原错误提示、验证码清空与刷新流程 |
| 桌面/小屏 | 768px 及以上默认互动（当前包括减少动态效果设置，无手动开关）；以下隐藏装饰角色且不加载 Rive |
| 资源失败/卸载 | 素材和 WASM 本地托管；失败显示静态备用图，表单仍可用；清理事件、计时器和动画帧 |

这是图片姿态与 Rive 轨道组合，不是实时三维模型。涉及素材或交互时，按专项说明运行测试，再实际操作焦点、成功/失败、资源失败、快速切换与窄屏；Node 加载或模拟登录通过不等于真实后端登录验收。Rive API 兼容性按锁定版本核对，不为消除弃用提示擅自升级。效果参考：[登录页动图](screenshots/login-cat.gif)（不是当前版本的验收证据）。
