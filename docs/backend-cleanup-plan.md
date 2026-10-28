# 后端整理与规范治理方案

> 范围：`go-zero-admin` 后端（Go 595 个文件，go-zero v1.10.3 / goctl 1.10.2 / GORM / Casbin）
> 状态：方案待评审，**尚未执行任何代码修改**
> 日期：2026-10-28

## 一、现状摸底结论

### 1.1 架构现状（合理部分，保持不变）

| 层 | 位置 | 规模 | 说明 |
|---|---|---|---|
| HTTP 网关 | `application/applet/api` | 254 文件 | go-zero rest，14 个 desc/.api 分组，统一 `pkg/result` 响应包装 |
| 业务 RPC | `application/applet/rpc` | 255 文件 | 16 个 server 分组，GORM model（19 文件），Casbin 鉴权 |
| AI RPC | `application/ai/rpc` | 24 文件 | 独立服务，eino 框架 |
| 共享库 | `pkg/` | 18 个包 | xerr / audit / orm / middlecasbin / objectstorage 等 |
| 数据资产 | `data/` | migrations + seeds + goctl 模板 + 脚本 | gen.ps1 / verify.ps1 已有工程化雏形 |

架构骨架（api → rpc → model、pkg 复用、desc 单一事实源）符合 go-zero 推荐实践，**不做结构性推翻**。

### 1.2 已确认的不符合项清单

#### A. go-zero 规范偏离

| # | 问题 | 证据 | 影响 |
|---|---|---|---|
| A1 | ~~goctl 模板目录三层并存~~ **✅ 已收敛（2026-10-28）**：保留 `1.10.3/` 为唯一模板 home，删除 `1.6.0/` 旧快照与顶层 `api/` 重复层，gen.ps1 的 `-home` 与校验路径已指向 `1.10.3`，README 引用已更新。经 A/B 试生成验证：换 home 后 221 个生成文件唯一实质差异是 svc 中间件字段排序，无功能影响 |
| A2 | logic 内使用全局 `logx.Errorf` 而非内嵌的 `l.Logger`（go-zero 约定用 `logx.WithContext(ctx)` 才能携带 trace-id） | 48 处（含 pkg 与 logic） | 日志丢链路追踪上下文 |
| A3 | 中间件重复实现：`app_session_middleware.go`（手写增强版）与 `appsession_middleware.go`（goctl 脚手架残留，内部转调前者） | middleware 目录实测 | 职责不清，注册时易混用 |
| A4 | ServiceContext 字段顺序混乱、构造内 `panic`（rpc 端 5 处、api 端 2 处） | svc/service_context.go | 与 goctl 重新生成差异大；启动失败堆栈不友好 |
| A5 | goctl 命名残留文件名：`get_user_toke_logic.go`（toke→token）、`get_file_u_r_l_logic.go` | 文件清单 | 可读性差 |
| A6 | GORM 替代 go-zero sqlx model 层 | model 目录 | 属既定架构决策（README 已声明），但缺少 ADR 与 model 编写规约。经评估全量迁回 sqlx 需重写 82 个非测试文件（285 处链式查询 / 109 处写入 / 34 事务 / 67 clause / 43 软删除）+ casbin gorm-adapter + pkg/orm 基建 + 38 个测试文件，约 3~5 人周且回归风险集中在鉴权事务代码，**结论：不换，写 ADR 固化** |
| A7 | `gen.ps1 model` 子命令仍调用 `goctl model mysql datasource`（生成 sqlx 风格代码），与项目实际 GORM 模型不兼容，误跑会污染 `internal/model` | gen.ps1:56 | 陈旧工作流，需删除或改为生成 GORM 模型骨架 |

#### B. Go 通用开发规范问题

| # | 问题 | 证据 | 影响 |
|---|---|---|---|
| B1 | 错误码常量拼写错误 `REUQEST_PARAM_ERROR`（应为 REQUEST） | 定义于 `pkg/result/xerr/errCode.go`，**122 处引用**遍布 api/rpc/ai | 对外形象与检索成本 |
| B2 | 包名 `pkg/ctxJwt` 混合大小写 | Go 包名应为全小写 `ctxjwt` | 违反 Go Code Review Comments |
| B3 | import 分组混乱：标准库混在第三方之后（如 register_logic.go 中 `strconv`/`strings` 垫底） | 多文件实测 | 违反 goimports 分组约定 |
| B4 | 魔法错误码内联：`xerr.NewErrCodeMsg(300004, ...)` | register_logic.go:51 | 错误码不可集中管理 |
| B5 | `errors.New` 裸错误 97 处（部分在 pkg 属合理，logic 层应统一走 xerr） | 全仓扫描 | API 错误响应不一致风险 |
| B6 | 注释风格混用 `/** */` 块注释、中英混杂无规约 | xerr/errors.go 等 | 风格不统一 |
| B7 | `pkg` 内 `panic`：middlecasbin、orm | 库代码 panic 会炸掉调用方进程 | 应返回 error |

#### C. 工程化缺口

| # | 问题 | 说明 |
|---|---|---|
| C1 | 无 `.golangci.yml`、无 lint 门禁 | 上述 B 类问题无法防回归 |
| C2 | 无 `.github/workflows` CI | 测试/lint 全靠本地自觉 |
| C3 | 无 Makefile（仅有 ps1 脚本） | Windows 友好但跨平台贡献者不便（可低优先级） |
| C4 | `docs/` 目录不存在，无开发规范文档 | 规范只存在于口头 |
| C5 | 当前工作区有未提交改动（README 修改、sessioncleanup 删除） | 动手前需先收口基线 |

## 二、整理方案（四阶段，每阶段独立可交付、可回滚）

### 阶段 0：基线保护（前置，0.5 天）

1. 收口干工作区：提交或暂存 README 与 sessioncleanup 的未决改动。
2. 跑通 `data/scripts/verify.ps1` 与全部 Go 测试，记录通过基线（后续每阶段完成后重跑对比）。
3. 拉一个 `chore/backend-cleanup` 分支，所有修改进分支、分阶段独立提交。

**验收**：`go test ./...` 基线结果留档；分支就绪。

### 阶段 1：机械化低风险修复（不改行为，1 天）

> 原则：全部由工具或全局重命名完成，逐文件可审查。

| 项 | 动作 | 工具/方式 |
|---|---|---|
| B3 import 分组 | 全仓 gofumpt + goimports（local prefix = go-zero-admin） | 一条命令 |
| A5 文件名 | `get_user_toke_logic.go` → `get_user_token_logic.go`；评估 `get_file_u_r_l_logic.go` 改名（注意与 goctl 再生成的兼容） | git mv |
| B2 包名 | `ctxJwt` → `ctxjwt` 目录改名 + 全部 import 更新 | gopls rename / 全局替换 |
| B1 错误码拼写 | `REUQEST_PARAM_ERROR` → `REQUEST_PARAM_ERROR`，122 处一次性切换（仅常量名，不影响对外错误码数值，协议无破坏） | 全局重命名 |
| B4 魔法码 | `300004` 等内联码收敛为 xerr 具名常量 | 手工逐个 |
| B6 注释 | 统一 `//` 行注释；导出标识符补全 golint 风格注释 | 半自动 |

**验收**：`go build ./...` + `go test ./...` 全绿；`git diff --stat` 逐文件过审。

### 阶段 2：go-zero 规范对齐（2~3 天）

| 项 | 动作 |
|---|---|
| A2 日志 | logic 内 48 处全局 `logx.*` 改为内嵌 `l.Logger` 方法调用，保住 trace-id |
| A3 中间件 | 删除 goctl 残留的 `appsession_middleware.go` 包装层，svc 中直接引用函数式中间件；同类排查其余中间件 |
| A4 svc 构造 | NewServiceContext 返回 `(*ServiceContext, error)` 替代 panic，main 中处理；字段按字母序整理并加分组注释 |
| B7 pkg panic | middlecasbin/orm 的 panic 改 error 返回，调用点同步 |
| A1 模板收敛 | **✅ 已完成**：保留 `1.10.3/` 为唯一定制层并作为 gen.ps1 的 `-home`；已删除 `1.6.0/` 与顶层 `api/`；README 已注明 goctl 模板解析机制 |
| A7 生成脚本 | gen.ps1 的 `model` 子命令改为输出 GORM 模型骨架（或暂时移除并在帮助文本中说明原因），避免误生成 sqlx 代码 |
| A6 model 规约 | 沉淀《model 编写规约》：GORM 标签约定、软删除、事务边界（沿用 register_logic.go 的 `WithContext+Transaction` 范式） |
| B5 错误体系统一 | logic 层错误一律 xerr；pkg 层 error 在 api/rpc 边界转译；补 xerr 错误码段分配表 |

**验收**：全测试通过；抽查任意 logic 文件，日志/错误/事务三件套一致；swagger 重新生成无 diff（证明 desc 未被误伤）。

### 阶段 3：文档与规约沉淀（1 天）

1. 新建 `docs/`：
   - `docs/development-guide.md`：目录职责、新增一个接口的标准流程（desc → gen.ps1 → logic → 注册）、日志/错误/事务范式。
   - `docs/adr/0001-gorm-over-sqlx.md`：固化 GORM 选型理由与边界。
   - `docs/error-codes.md`：错误码段分配表。
2. README 补「开发规范」一节，链接上述文档。

**验收**：新人按文档能独立新增一个 API + RPC 接口。

### 阶段 4：工程化门禁（1 天）

1. `.golangci.yml`：启用 gofumpt、goimports、revive、errcheck、govet、staticcheck、misspell（可防 B1 类拼写回归）；先对存量问题设 baseline 或逐目录收敛。
2. `verify.ps1` 增加 lint 步骤；可选补 Makefile 薄封装。
3. GitHub Actions：build + test + lint 三连（test 需 MySQL/Redis service 容器，参考 docker-compose）。

**验收**：CI 绿灯；故意引入一处全局 logx 调用，lint 能拦截。

## 三、工作量与节奏建议

| 阶段 | 预计 | 风险 |
|---|---|---|
| 0 基线 | 0.5 天 | 低 |
| 1 机械修复 | 1 天 | 低（纯重命名/格式化） |
| 2 规范对齐 | 2~3 天 | 中（svc 签名变更波及所有 logic 构造点，编译器兜底） |
| 3 文档 | 1 天 | 低 |
| 4 门禁 | 1 天 | 低-中（CI 环境调试） |

合计约 5~6 个工作日。阶段 1、2 收益最大，建议必做；阶段 3、4 保障长期不腐化。

## 四、明确不做的事

- 不重写架构（不拆分单体 RPC、不引入 DDD 分层大改）。
- 不替换 GORM 回 sqlx。
- 不动前端仓库与 desc 定义的对外协议（错误码数值、字段名一律不变）。
- 不升级 go-zero / goctl 版本（保持 v1.10.3 / 1.10.2 钉死）。

---

*本方案基于 2026-10-28 对仓库的静态摸底（595 个 Go 文件、关键路径抽样与全仓反模式扫描）。执行前如对某阶段范围有异议，可先单独评审该阶段。*
