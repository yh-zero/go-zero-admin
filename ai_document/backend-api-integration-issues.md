# 后端 API 对接偏差与待确认事项

首次记录：2026-10-30；后端回退状态更新：2026-10-31。

## 当前状态：撤回本轮全部后端修改

2026-10-31 用户要求“撤回全部后端的修改，前端不变”。回退范围为提交 `215dc46` 中 `go-zero-admin/` 下的 24 个文件，目标版本为父提交 `1fad38b`，包含后端 `AGENTS.md`、自检脚本、描述来源和生成类型；本次新增的 `policy_holder.go` 与 `verify-helpers.ps1` 随回退移除。前端目录、根目录规则、正式种子导入脚本和已交付测试数据保留。

下表中的后端修复及其联调结果均属于回退前历史，后端回退后不再生效。原编号 2、3、7 路径绑定、8 查询、9、10 性别、11、14 及鉴权刷新、文件描述标签与自检脚本问题恢复；其他未实施问题继续保留。前端路径参数请求、菜单删除提示和资料保存仍依赖被撤回的后端行为，因此不能宣称当前组合已完整对接。详见 `api-contract.md` §8 的当前状态。

本轮只恢复仓库文件并验证版本范围，不修改数据库、对象存储或运行中的服务；不把历史测试通过视为回退后通过。

回退验证：`git diff --exit-code 1fad38b -- go-zero-admin` 与 `git diff --exit-code 215dc46 -- go-zero-admin-react` 均退出 0，分别确认后端与基线一致、前端无改动。原版 `verify.ps1` 在当前 CRLF 工作区的 gofmt 阶段退出 1，后续步骤未执行，不记录为全部通过；独立执行 `go list -mod=readonly ./...` 成功识别 104 个包，`go vet -mod=readonly ./...` 和 `go build -mod=readonly ./...` 均退出 0。本轮未启动服务或复测接口，未生成新的测试代码。

2026-10-31 提交前复核：用户要求将回退及待办文档提交到主分支。后端从同一 Git 基线导出到隔离副本，仅在副本内对 282 个 Go 文件及 `go.mod/go.sum` 归一 CRLF 为 LF，未修改仓库源码、锁文件或自检脚本；副本 gofmt 无差异、包枚举成功且为 104 个，原版 `verify.ps1` 六项全部执行并退出 0。自检前后副本文件哈希一致，临时目录已清理；前端 `tsc -b` 与 `vite build` 均退出 0，保留既有共享包体积提示。该验证不代表 Windows 工作区自检兼容问题或接口缺陷已经修复。本次提交范围为 24 个后端回退文件及 2 份共享文档，前端目录和根规则保留。

## 回退后待修改清单（当前，均未实施）

本节以恢复后的 `1fad38b` 为代码基线，汇总原问题及后续审查发现；下面的修改方案用于后续确认和实施，本轮只回退代码、更新文档。路径相对后端 `go-zero-admin/`，原编号对应下方保留的证据章节。历史实测与本轮静态核对分开记录，未在回退后重新复现所有故障。

### 优先修复：鉴权、数据写入与现有接口阻塞

| 编号 | 需要修改的位置与问题 | 建议修改内容 | 验收要求 |
|---|---|---|---|
| R01 | `application/admin/api/admin.go`：仅精确以 `Bearer ` 开头才调用 Authority，其他格式进入后续处理。 | 所有非公开路由统一进入 Authority，保留 go-zero JWT 校验；不能让框架可接受但自定义前缀判断不识别的 token 格式跳过 Casbin。 | 有效 JWT 但无接口权限必须拒绝；覆盖标准和非标准请求头格式、缺失/过期 token，并确认公开登录和验证码可用。不能把该缺陷描述成“无 token 必然绕过框架 JWT”。 |
| R02 | `pkg/casbinauth/enforcer.go`：默认缓存过期删除 map 的并发问题恢复，原编号 14 曾实测 API 退出。 | 在现有依赖版本内使用并发安全的缓存实现，核对真实缓存读写和过期路径；无需为此擅自升级 Casbin。 | 并发请求覆盖缓存未命中、命中和过期，API 不退出；条件允许时运行 race detector，未运行须记录。 |
| R03 | `application/admin/api/internal/svc/service_context.go`、`application/admin/api/internal/middleware/authority.go`：刷新重新赋值不能更新中间件持有的旧实例；首次快照/订阅间有窗口，刷新失败与重连无补偿。 | 中间件持有稳定容器，完整构建后原子发布；设计订阅就绪后的补拉、失败重试或周期对账，并管理后台任务生命周期。只恢复历史 `PolicyHolder` 仍不足以解决漏通知。 | 同一已登录会话撤权后应被拒绝；模拟订阅窗口、断线重连、RPC/构建失败、没有后续消息的场景，验证恢复后在规定时间内收敛。 |
| R04 | `application/admin/rpc/internal/logic/menuservice/` 的列表、树、详情、用户菜单与权限查询：时间扫描、直扫 pb、切片绑定单个 IN 参数等问题恢复（原 2、3）。 | 查询封装到 model 扩展，使用明确 `db` tag 和 Unix 秒别名；权限关联用参数化 JOIN 或正确展开占位符；详情读取有效记录，区分数据库错误与确实无权限。 | 用非空菜单、多角色用户和无角色用户验证列表/树/详情/权限码；检查时间值、软删除过滤以及前端真实权限按钮。 |
| R05 | `application/admin/rpc/internal/logic/menuservice/delete_menu_logic.go`、`application/admin/rpc/internal/logic/menuservice/update_menu_logic.go` 及对应 model：删除仍按导航 path 前缀，编辑缺少有效行限定、并发及缓存保护（原 11）。 | 按 `parent_id` 得到真实子树，在事务内处理菜单及角色关联；编辑只允许有效记录并保护删除字段，检查写入错误；提交后维护菜单及受影响关联的所有缓存键。先核对生成 CRUD 的限制，必要的受保护写入方案需明确说明并确认，不能直接照搬旧扩展。 | 空路径目录、按钮后代、同前缀非后代、含 LIKE 通配字符的路径、删除与编辑并发、重复/无变化更新、事务回滚及缓存读取均需覆盖。当前前端已允许目录/按钮删除并提示“删除子树”，与旧后端不匹配。 |
| R06 | `application/admin/api/desc/sys/dict.api`、`application/admin/api/desc/sys/file.api`，以及 `application/admin/rpc/internal/logic/dictservice/get_dict_list_logic.go`：字典 path 绑定和文件 URL 标签错误恢复（原 7及补充项）。 | 在来源中声明 `path:"code"`，修正 URL 标签多余引号后通过 `data/scripts/gen.ps1 api` 生成；字典列表补明确 row tag、SQL 时间转换及别名。 | 保持现有前端只传路径即可读取字典详情；完整增改删、字典项保存及 API validate/vet 通过。缺少 tag 本身只属规范风险，不能直接认定列表必然失败。 |
| R07 | `application/admin/rpc/internal/logic/apiservice/get_api_list_logic.go`、`application/admin/rpc/internal/logic/auditservice/get_audit_log_list_logic.go`：非空时间扫描及审计 NULL 处理问题恢复（原 8、9）。 | 移入对应 model 扩展，使用明确 row；时间表达式加 Unix 秒转换和别名，可空 `user_id`、`params` 明确处理。 | 非空 API/审计记录正常返回，覆盖 NULL 和非 NULL、时间值、筛选与分页，不能只验证空表。 |
| R08 | `application/admin/api/internal/logic/profile/update_profile_logic.go` 与 `application/admin/rpc/internal/logic/profileservice/update_profile_logic.go`：API 未转发 Gender，而 RPC 无条件写回（原 10）。 | 正确传递已有性别字段，避免保存姓名/邮箱等文本时丢失性别；字段省略语义另按下方 C02 确认。 | 原性别为非零值时，使用保留的前端修改文本，HTTP 返回后重新读取数据库和资料接口确认性别不变。当前前端虽提交原性别，仍不能抵消旧 API 丢弃字段的问题。 |
| R09 | `data/scripts/verify.ps1`：旧脚本未检查 gofmt/go list 退出码，且 Windows 行尾兼容修复撤回。 | 原生命令失败立即终止并保留诊断；只有成功且确实无包才能记录跳过。CRLF 仅在只读比较或临时副本中归一，不重写源码、依赖或锁文件掩盖差异。 | 分别验证工具失败、包加载失败、CRLF、真实格式/依赖/校验和差异，保证失败退出非零；最后实际执行全部六项自检。当前 CRLF 工作区在 gofmt 阶段失败，隔离 LF 副本六项通过，不代表此脚本缺口已修复。 |

R05 的关联缓存要求来自对回退前 `DeleteSubtree` 的审查：当时删除 `sys_role_menu` 后只清菜单缓存，未维护关联主键及唯一索引缓存。该扩展现已撤回，不能继续把它列为当前已有函数的缺陷；它是后续重做关联删除时的验收要求。当时也未发现业务调用关联缓存读取方法，未证明现有请求因此越权。

### 继续保留的既有问题

| 编号 | 需要修改的位置与问题 | 建议修改内容及验收要求 |
|---|---|---|
| R10 | `application/admin/rpc/internal/logic/userservice/get_user_list_logic.go`（原 4）。 | 为部门、头像文件 ID、登录时间和创建/更新时间表达式添加与 row tag 一致的列别名；用非零数据库值逐字段对照响应，不能以列表返回 200 作为字段正确证据。 |
| R11 | `application/admin/rpc/internal/logic/roleservice/` 的列表、详情、角色编辑和菜单授权（原 5）。 | 修复时间别名并明确统计字段是否填充；角色菜单关系删除/重建置于同一事务、检查每次写入并维护缓存。保持现有完整提交可用；验证错误回滚、重复/空菜单集合及已有授权保留。变更部分更新语义或统计字段承诺前先确认契约。 |
| R12 | `application/admin/rpc/internal/logic/deptservice/` 的树查询、更新和删除（原 6）。 | 补时间别名；修复直接写库后的 model 缓存失效。若提供父级移动，必须防环并事务更新节点及后代物化路径，覆盖缓存读取与回滚；当前前端禁止移动的限制保留。扁平 tree 响应本身不是缺陷，不应擅自改成嵌套结构。 |
| R13 | `application/file/rpc/internal/logic/fileservice/get_file_list_logic.go`、`application/file/rpc/internal/logic/fileservice/get_file_u_r_l_logic.go`（原 12）。 | 补上传者与时间表达式别名；返回的 `expires_at` 与实际签名有效期一致。验证非零上传者、非默认存储 TTL、公开/私有对象及引用删除限制，不能仅依据默认配置正常就判定全部语义正确。 |

### 需要另行确认的契约或能力范围

| 编号 | 待确认内容 | 确认后才能实施的内容 |
|---|---|---|
| C01（原 1） | 是否统一成功响应信封及 HTTP/业务错误码语义。 | 当前前端兼容裸成功数据、空成功体和信封。确认迁移方式后先更新 `api-contract.md`，再调整统一出口并验证前端；不能手改生成 handler 或用 HTTP 200 代替业务成功判断。 |
| C02（原 5、10） | 头像采用文件 ID 还是 URL；更新请求省略字段代表保留原值还是清空。 | 明确头像读取、授权、写入及字段存在性语义后同步契约与两端。当前头像编辑仍未开放；性别转发的现有缺陷见 R08，不应与新增头像能力混为一项。 |
| C03（原 8、13） | 是否增加 API 资源同步、文件上传、用户分配角色及 App/AI/租户 HTTP 接口。 | 先确认具体能力、授权和字段，再改 `.api`/proto 来源并生成实现。`SyncApi` 当前 RPC 逻辑仅发刷新通知并返回零计数，不能只补 HTTP 路由就宣称同步完成；没有 HTTP 的能力也不能按前端已对接验收。 |

### 后续实施约束

- 本清单不代表已获修复授权；当前要求是保留前端、撤回后端并记录待办。涉及前端或契约的后续修改需按用户确认的范围执行。
- 按模块逐项修复并立即真实调用，记录允许/拒绝、非空/NULL、事务回滚和缓存结果；不能重用回退前“41 路由通过”的结论。
- 后端后续新增测试仍按用户要求放在 `go-zero-admin/test/`，临时测试用完删除，保留命令、结果、未覆盖项；不删除原有测试、正式导入器或保留的联调数据。
- 不手改生成物、不自主升级依赖；生成物先读实际签名。若生成 CRUD 无法满足有效行、事务或并发约束，先说明具体限制与方案，不为套用生成方法破坏数据保护。

## 2026-10-31 回退前修复状态（历史）

初次记录时用户要求先做前端、后端问题只记录，随后授权“要配合可以改后端”。以下记录当时的修复和验证结果，保留用于追溯；未增加或升级依赖、未修改锁文件。

| 原编号 | 回退前状态 | 后端配合与前端结果 |
|---|---|---|
| 2、3 | 已修复并真实验证 | 菜单列表/树用 model 扩展、带 `db` tag 的 row 和 Unix 时间别名；详情用生成 model 读取并转换时间；用户菜单和权限用参数化 JOIN，非空数据及真实权限按钮正常。 |
| 7 路径绑定 | 已修复并真实验证 | 契约先更新，再将 `.api` 改为 `path:"code"`，经 `gen.ps1 api` 重生成；前端只传路径。字典新增、按编码详情、完整编辑、删除通过。列表 row 的规范整理仍待处理。 |
| 8 查询、9 | 已修复并真实验证 | API/审计列表移到 model 扩展，时间转 Unix 秒并加明确别名；审计兼容 `user_id`、`params` 为 NULL。已有 41 条 API、12 条审计及临时可空记录正常返回。API 同步入口未新增。 |
| 10 性别 | 已修复并真实验证 | API 正确转发 `Gender`；真实浏览器保存文本资料后，临时账号原有性别 2 保持不变。头像 URL/文件 ID 语义仍待处理，前端不开放头像上传。 |
| 11 | 已修复并真实验证 | 按真实父子关系事务删除菜单子树及角色菜单关系，提交后清详情缓存；同路径前缀的无关节点保留。编辑只锁定并更新有效记录的可编辑字段，不能复活软删除记录；未变化值仍可成功保存。前端删除提示同步为子树语义。 |
| 14 及鉴权刷新 | 已修复并真实验证 | 使用现有 Casbin 依赖的同步缓存；中间件持有稳定策略容器，刷新构建完成后原子替换。80 次并发请求成功；Redis 通知撤销角色后已有会话返回 403，恢复后正常放行；非标准请求头不能绕过 Authority。 |
| 文件描述标签、自检脚本 | 已修复并验证 | 修正 `.api` 可选 `url` 标签多余引号后重新生成；Windows 自检只归一 CRLF，不掩盖实际格式/依赖差异，不改写源文件或锁文件。完整 `verify.ps1` 六项通过。 |
| 1、4、5、6、12、13及上列保留项 | 未实施 | 成功信封、其他列表字段别名、角色授权事务与统计、部门移动和缓存、头像/文件有效期、缺少的 HTTP 能力仍按原记录处理，不宣称整个后端已规范化或完成安全整改。 |

原四项审查问题同时修复：种子按固定 ID 与自然身份双向核对，冲突不追加权限且写入事务回滚；后台刷新保留表单草稿并更新未编辑字段；删除末页唯一记录后实际重新请求有效页；文件元数据不再触发无关表的整段保留 ID 冲突。14 项真实 MySQL 场景、22 项前端回归测试通过。真实 Chrome 验证 Profile/User/Role/Dict 草稿、保存后新基线及分页回退；另用真实后端权限完成登录、新增和末页删除、资料保存与所有数据页面加载。

后端定向测试覆盖路径绑定、资料 RPC 转发、权限并发/刷新、中间件入口、数据库扫描、菜单子树与编辑事务。当前 `CGO_ENABLED=0`，未执行 Go race detector；普通并发回归和真实并发请求已通过。构建保留既有 Vite 共享包体积提示。

后续补验收使用本轮独有临时对象，完成用户详情/编辑/重置密码/个人改密（新密码与恢复密码均真实登录）、角色 CRUD 与菜单授权、部门 CRUD（不移动）、临时 API 资源删除。浏览器另验证空路径目录及按钮后代按层级删除、独立新建 S3 对象与元数据实际删除，并确认三份交付文件仍保留。现有 41 个 HTTP 路由正常调用路径均已覆盖；完整页面表单组合与全部失败场景未穷尽。

最终清理已核对：本次启动的 API、两个 RPC、Vite 及专用 MySQL/Redis/RustFS 均停止，容器退出码均为 0；5999/6001/6003/17001 监听释放，仓库外临时配置、凭据、二进制、脚本和调试日志已删除。只按本轮唯一前缀与明确缓存键清理，没有清空 Redis；其他项目容器保持运行。持久数据恢复并核对为 29 用户、6 部门、7 角色、31 菜单、41 API、4 字典、3 文件、12 审计、37 Casbin 规则、75 角色菜单关系；数据库和对象存储卷保留。

## 历史证据与验收记录的口径

以下保留首次问题发现、回退前修复及验收记录。其中“本轮”“当前”“前端处理”以记录当时为准；回退后的实际状态与待修改内容以上方清单为准，不能沿用旧的前端临时兼容措施或删除限制。

- 源码路径均相对本仓库根目录，行号以本次只读核对时的文件为准。
- “静态确认”表示已核对 `.api`、生成类型、handler、API/RPC logic；不等同于真实请求通过。
- “此前实测”引用本会话前一轮隔离库结果；“本轮实测”是随后前端联调的真实请求，另行标明，不把静态结论冒充运行结果。
- 本轮前端统一使用标准 `Authorization: Bearer <token>`，不会改变请求头格式绕过权限；请求失败显示真实错误及重试入口，不以伪造列表、权限码或成功结果遮盖后端阻塞。
- 数据库扫描规则另核对了本机实际 `go-zero v1.10.3` 的 `core/stores/sqlx/orm.go:89`：存在 `db` tag 时按列名匹配；完全没有 tag 时按字段顺序匹配。不能把“缺少 tag”直接等同于已证明运行失败。

## 2026-10-31 后端规范复核

审查范围为本次后端工作树差异、新增测试、自检和生成来源，结合实际已安装 go-zero v1.10.3 源码；未将仓库强制规则等同于框架官方限制。本轮未修改业务实现。

| 分类 | 位置与证据 | 结论 |
|---|---|---|
| P2，仓库规则偏离 | `application/admin/rpc/internal/model/sys_menu_model.go:60` 的 `UpdateActive` 在锁定有效行后仍手写单表 UPDATE，由 `logic/menuservice/update_menu_logic.go:34` 调用 | 与后端 AGENTS §2.7“CRUD 用生成 model”不一致。旧实现已存在手写更新，本次迁入扩展后仍保留同类偏离，不是新增运行时回归。修正时必须保留行锁及防软删除复活保护，不能直接退回无保护的生成 Update。跨表 `DeleteSubtree` 有真实事务需要，不按此项认定违规。 |
| 操作流程偏离 | 后端 AGENTS §2.11 要求每模块完成立即真实调用 | 上轮集中完成菜单、API、审计、字典、资料等修复后才统一联调，未逐模块立即 e2e。后续补充真实调用通过不改变这项流程偏离。 |
| P2，既有可靠性风险 | `application/admin/api/internal/svc/service_context.go:60` 先取策略快照，`:91` 后启动订阅；`:104` 刷新 RPC 失败直接 continue | 订阅建立前的通知或刷新失败可能使撤权长时间不生效；当前没有重试/定期对账。快照、通知窗口和失败处理在 HEAD 已存在，并非本次新引入，普通热更新验证未覆盖这些失败场景。 |
| P2，既有自检缺口 | `data/scripts/verify.ps1:16` 未检查 go list 退出码，`:18` 只按 stdout 包数决定跳过 | 模块加载/工具链错误导致无输出时，可能误判骨架并跳过 vet/build/tidy。HEAD 中已存在该逻辑，本次 CRLF 修复未引入也未修复它。 |

其余已核对事项：查询扩展位置、本地 db tag row、Unix 时间别名、xerr 错误出口、API 通过 RPC、独立 file-rpc 连接、公开路径之外统一鉴权、同步 Casbin 缓存、原子发布策略及 pkg 导入边界未发现本次新的高置信问题。生成文件与描述来源一致；未更改 go.mod/go.sum、生成模板或目录白名单。CRLF 检查仅归一行尾，真实格式、依赖和锁文件顺序差异仍会失败。

临时测试清理前重新运行 `go test -mod=readonly ./...` 和 6 项自检脚本回归，均通过。按用户要求，删除本次新增的 12 个后端 Go 测试、1 个后端测试脚本、7 个前端单元测试和 2 个种子回归脚本，共 22 个文件；原有 `test/login_e2e.ps1`、业务代码、种子导入脚本、测试数据和验证结果保留。后续后端新增测试统一在 `test/`，临时测试用完删除。

## 1. 成功响应与统一信封契约不一致

**路由：** 所有现有 REST 模块；代表接口为 `GET /v1/sys/menu/list`、`POST /v1/sys/menu/create`，其他列表和写接口采用同类 handler。

**证据：**

- `ai_document/api-contract.md:5` 约定成功响应为 `{code:200,msg,result}`。
- `go-zero-admin/application/admin/api/internal/handler/menu/get_menu_list_handler.go:28` 直接 `httpx.OkJsonCtx(..., resp)`，返回 `{list,total}`，没有成功信封。
- `go-zero-admin/application/admin/api/internal/handler/menu/create_menu_handler.go:28` 使用 `httpx.Ok(w)`，写成功为 HTTP 200 空体。
- `go-zero-admin/pkg/xerr/handler.go:15`、`:20` 将业务/RPC 错误作为 HTTP 200 的 `{code,msg}` 返回；`:25` 的框架参数错误另用 HTTP 400。

**确认方式：** 静态确认；此前真实登录、用户读取/删除链路也观察到裸成功数据和错误信封。不能仅检查 HTTP 200 判断成功。

**前端处理：** `go-zero-admin-react/src/utils/response.ts` 统一兼容裸成功数据、空写响应及约定信封；数值业务码非 200 时仍拒绝请求。详情请求分别解 `{menu}`、`{dict}`、`{role}`、`{user}`，不在组件中假设 `result` 总存在。

**后端建议（待确认）：** 统一成功出口与契约。确定迁移方式后，再同步契约、生成来源和两端验证；本轮不自行改变既有响应格式。

## 2. 菜单列表、树和详情的数据扫描阻塞

**路由：** `GET /v1/sys/menu/list`、`GET /v1/sys/menu/tree`、`GET /v1/sys/menu/:id`。

**证据：**

- `go-zero-admin/application/admin/rpc/internal/logic/menuservice/get_menu_list_logic.go:58` 将 `created_at` 声明为 int64，`:62` 却直接查询 DATETIME 原列。
- `go-zero-admin/application/admin/rpc/internal/logic/menuservice/get_menu_tree_logic.go:47`、`:51` 存在相同问题。
- `go-zero-admin/application/admin/rpc/internal/logic/menuservice/get_menu_logic.go:24` 直接把查询结果扫描到生成的 `adminpb.MenuInfo`，`:26` 仍查询 DATETIME 原列。生成结构实际见 `go-zero-admin/application/admin/rpc/adminpb/admin.pb.go:2682`，字段为 int64 时间且没有数据库扫描 tag。
- `go-zero-admin/data/db/migrations/business/00001_init.sql:91` 明确 `sys_menu.created_at` 为 DATETIME。

**确认方式：** 列表和树已在前一轮有真实菜单行的隔离库请求中返回业务码 505；详情的直扫 pb、字段类型问题为静态确认，没有在本文编写阶段新增详情请求结果。

**前端处理/仍阻塞：** 已实现分页、类型/父菜单筛选、树视图、详情和重试入口。`/tree` 成功体是扁平 `{list}`，前端按 `parent_id` 构树。编辑先调用详情，失败时禁止保存，避免用列表缺失字段覆盖已有菜单。后端返回 505 时，真实菜单数据查看及编辑仍阻塞，前端无法恢复丢失的数据。

**后端建议（待确认）：** 使用带明确 `db` tag 的本地 row；时间采用 `UNIX_TIMESTAMP(created_at) AS created_at`；row 转 pb，禁止直接扫描生成消息。

## 3. 当前用户菜单与权限码查询未展开 IN 参数

**路由：** `GET /v1/sys/menu/my`、`GET /v1/sys/menu/my/perm-codes`。

**证据：**

- `go-zero-admin/application/admin/rpc/internal/logic/menuservice/get_my_menus_logic.go:49` 至 `:53` 使用 `r.code IN (?)`，将 `[]string roleCodes` 作为一个 SQL 参数。
- `go-zero-admin/application/admin/rpc/internal/logic/menuservice/get_my_perm_codes_logic.go:32` 至 `:35` 同样绑定整个 slice，没有逐项生成占位符。
- `get_my_menus_logic.go:46` 的 int64 时间与 `:50` 的 DATETIME 查询同时保留菜单时间扫描问题。
- 两个方法在角色查询失败或没有角色时直接返回空列表：分别见 `get_my_menus_logic.go:29`、`get_my_perm_codes_logic.go:28`。因此空数组不能单独证明后续关联查询正常。

**确认方式：** 静态确认。本文不把“没有角色时返回空数组”算作菜单/权限关联链路已经实测通过。

**前端处理/仍阻塞：** 按实际 `{list}`、`{codes}` 解析，并严格按服务器返回的权限码显示按钮。菜单/权限查询错误应有显式状态与重试；不能靠硬编码全部权限或特殊 token 格式完成验收。

**后端建议（待确认）：** 按角色数量展开占位符和参数，或改成参数化关联查询；修正时间转换并区分数据库错误与无角色结果。前端页面注册名称使用 `menu`、`dict`、`api`，后端 `name` 应与注册项一致；`component` 仍应保留真实配置。

## 4. 用户列表表达式缺少别名，部分字段静默归零

**路由：** `GET /v1/sys/user/list`。

**证据：**

- `go-zero-admin/application/admin/rpc/internal/logic/userservice/get_user_list_logic.go:23` 至 `:38` 的 row 使用 `db:"dept_id"`、`db:"avatar_file_id"`、`db:"last_login_at"`、`db:"created_at"`、`db:"updated_at"`。
- 同文件 `:77` 查询 `COALESCE(dept_id,0)`、`COALESCE(avatar_file_id,0)`、`COALESCE(UNIX_TIMESTAMP(last_login_at),0)`、`UNIX_TIMESTAMP(created_at)` 等表达式，却没有为表达式提供与 tag 一致的 `AS` 别名。

**确认方式：** 静态确认；前一轮真实请求已观察到列表 `dept_id`、`created_at` 为 0，而数据库该用户的部门为 1、创建时间有有效值。其余同类表达式的归零风险由相同映射规则确定，未逐字段新增请求。

**前端处理/仍阻塞：** 时间 0 显示缺失占位，不伪造时间；编辑应先读取 `GET /v1/sys/user/:id` 的完整详情，不能使用列表零值覆盖有效部门、头像等。列表无法准确表达这些字段的根因仍在后端。

**后端建议（待确认）：** 为每个表达式逐一添加准确列别名，保持 row tag、SQL 列和 REST 字段一致；用数据库中非零字段验证返回值。

## 5. 角色列表未填充统计与菜单，更新会整体重建菜单关联

**路由：** `GET /v1/sys/role/list`、`GET /v1/sys/role/:id`、`PUT /v1/sys/role/:id`、`POST /v1/sys/role/:id/menus`。

**证据：**

- `go-zero-admin/application/admin/rpc/internal/logic/roleservice/get_role_list_logic.go:62` 的时间表达式没有 `AS created_at/updated_at`；`:69` 没有填充 `UserCount`、`MenuIds`。这些字段确实存在于生成消息，见 `go-zero-admin/application/admin/rpc/adminpb/admin.pb.go:2022`、`:2023`。
- `go-zero-admin/application/admin/rpc/internal/logic/roleservice/get_role_logic.go:35` 至 `:37` 在详情额外读取 `menu_ids`，仍未统计 `user_count`。
- `go-zero-admin/application/admin/rpc/internal/logic/roleservice/update_role_logic.go:29` 至 `:34` 全量覆盖角色属性；`:41` 至 `:43` 先删除再重建所有 `sys_role_menu`，省略 `menu_ids` 就会按空集合处理。
- `go-zero-admin/application/admin/rpc/internal/logic/roleservice/update_role_menus_logic.go:28` 至 `:30` 同样先删后插，插入错误被忽略，没有事务保护。

**确认方式：** 静态确认。列表 `user_count=0` 不代表真实统计结果为零；本文不新增“角色授权更新已经真实通过”的结论。

**前端处理/仍阻塞：** 编辑先读详情，提交完整属性并携带详情中的现有 `menu_ids`。授权窗口单独读取详情与菜单树；菜单树错误时禁止提交。不能用角色列表中的空菜单集合直接保存。时间和用户数量仍不能由前端补算。

**后端建议（待确认）：** 明确列表/详情统计字段是否保证填充，修复表达式别名；定义全量更新语义或使用具有字段存在性的部分更新；菜单关系更新使用事务并检查每次写入结果。

## 6. 部门 tree 为扁平列表，父级更新未维护物化路径与缓存

**路由：** `GET /v1/sys/dept/tree`、`PUT /v1/sys/dept/:id`；创建与删除分别为 `POST /v1/sys/dept/create`、`DELETE /v1/sys/dept/:id`。

**证据：**

- `go-zero-admin/application/admin/rpc/internal/logic/deptservice/get_dept_tree_logic.go:48` 至 `:52` 直接返回扁平 `List`；`:44` 的时间表达式也缺少 `AS created_at`。
- `go-zero-admin/application/admin/rpc/internal/logic/deptservice/create_dept_logic.go:33` 至 `:42` 创建时计算物化 `path`。
- `go-zero-admin/application/admin/rpc/internal/logic/deptservice/update_dept_logic.go:29` 更新 `parent_id`，但没有同步节点及后代 `path`，也没有拒绝环形父级。
- 同方法 `:24` 先经缓存 model 读取，再在 `:28` 直接 `DB.ExecCtx` 更新，未失效 model 缓存。生成缓存读取见 `go-zero-admin/application/admin/rpc/internal/model/sys_dept_model_gen.go:75` 至 `:80`；生成的标准 Update 包装见同文件 `:101` 至 `:106`。
- 删除同样直接写库：`go-zero-admin/application/admin/rpc/internal/logic/deptservice/delete_dept_logic.go:36`。

**确认方式：** 静态确认；不把已有空/非空 tree 查询成功推断为父级移动或缓存更新正确。

**前端处理/仍阻塞：** 前端按 `parent_id` 构树；当前编辑保持原父级并禁止调整上级，避免触发未实现的子树移动。刷新列表不等于清理后端 model 缓存；依赖 model 的其他接口仍可能读到旧部门信息。

**后端建议（待确认）：** 明确 tree 是扁平还是嵌套结构；若支持移动则校验环、事务更新整棵子树路径；所有改写遵循 model 缓存失效规则，补齐时间别名。

## 7. 字典详情路径参数未绑定；列表缺少 db tag 属规范偏差

**路由：** `GET /v1/sys/dict/code/:code`、`GET /v1/sys/dict/list`、`POST /v1/sys/dict/create`、`PUT /v1/sys/dict/:id`、`DELETE /v1/sys/dict/:id`。

**证据：**

- `go-zero-admin/application/admin/api/desc/sys/dict.api:38` 的 `GetDictByCodeReq` 只有 `json/form` 标签，路由在 `:71` 却使用 `:code`。生成类型原样保留此结构，见 `go-zero-admin/application/admin/api/internal/types/types.go:208`、`:209`。
- `go-zero-admin/application/admin/rpc/internal/logic/dictservice/get_dict_list_logic.go:54` 至 `:63` 使用无 `db` tag 的本地 row，`:66` 按相同字段顺序 SELECT，`:79` 将 `time.Time` 转 Unix 秒。
- 此处**并不是直接扫描 pb**，也不是 int64 时间直扫 DATETIME。实际 sqlx 在完全无 tag 时按位置扫描，所以不能仅凭 tag 缺失认定本接口已经字段错位或失败。
- `go-zero-admin/application/admin/rpc/internal/logic/dictservice/update_dict_logic.go:31` 至 `:36` 全量替换名称、编码、字典项、描述和状态。

**确认方式：** 静态确认；未在本轮新增真实字典请求，列表能否在当前服务配置下运行应由联调单独记录。

**首次联调时的前端处理（历史）：** 详情曾使用 `/code/<编码>` 并额外附带 query `code`；编辑先读详情，再完整提交字典及 `items`。字典项由 `Form.List` 管理。列表无 tag 记作规范及维护风险，不伪造“必然 505”的状态。回退后保留的前端已移除额外 query，只传路径，当前需要修复后端绑定，见 R06。

**本轮实测补充：** 即使路径和 query 同时传入编码，详情仍返回 HTTP400，业务码400，msg 为 `field "code" is not set`。新增和列表可以成功，编辑所需详情仍被绑定问题阻塞。前端显示详情错误并禁用保存，没有用列表数据覆盖详情；补 query 不能被标为已解决。

**后端建议（待确认）：** 在 `.api` 来源正确声明 path 参数后重新生成；为 row 补齐明确 tag，并按后端规范在 SQL 中转换时间且加别名，避免以后字段调序引入问题。

## 8. API 资源非空列表时间扫描阻塞，HTTP 没有同步入口

**路由：** `GET /v1/sys/api/list`、`DELETE /v1/sys/api/:id`。

**证据：** `go-zero-admin/application/admin/rpc/internal/logic/apiservice/get_api_list_logic.go:55` 把 `CreatedAt` 定义为 int64，`:59` 直接查 DATETIME `created_at`。`go-zero-admin/application/admin/api/internal/handler/routes.go:34` 至 `:49` 仅注册查询和删除；`go-zero-admin/application/admin/rpc/desc/admin.proto:584` 虽定义 `SyncApi`，实现 `go-zero-admin/application/admin/rpc/internal/logic/apiservice/sync_api_logic.go:23` 仅发布刷新通知并返回零计数。

**确认方式：** 静态确认。空表返回列表不能证明存在 DATETIME 数据行时的扫描通过；没有新增真实同步调用。

**前端处理/仍阻塞：** 只实现分组、方法、路径筛选与删除登记，不提供不存在的新增、编辑、同步按钮。非空列表扫描问题仍由后端解决，错误通过统一状态显示。

**后端建议（待确认）：** 修正时间转换及别名；只有用户确认需要同步功能后，才补 HTTP 契约与真实同步实现。

## 9. 审计列表时间未转换，当前没有独立别名丢失证据

**路由：** `GET /v1/sys/audit/list`。

**证据：** `go-zero-admin/application/admin/rpc/internal/logic/auditservice/get_audit_log_list_logic.go:67` 为 int64 `created_at`，`:71` 查询 DATETIME 原列；表结构见 `go-zero-admin/data/db/migrations/business/00001_init.sql:203`。当前 SELECT 使用原列名，和现有 `db` tag 一致，不能把用户列表中的表达式别名问题直接套用于审计。`params` 在表结构中可为 NULL，row 则为 string，亦需在非空数据验证时覆盖。

**确认方式：** 时间与可空字段类型为静态确认；未新增真实审计记录请求，不宣称审计非空链路通过。

**前端处理/仍阻塞：** 已有 REST 能力仅查询；时间缺失和请求错误应如实展示。不能用前端当前时间补齐审计时间，也不能将空表结果当作完整验证。

**后端建议（待确认）：** 使用显式 row 和空值处理，时间转换写成 `UNIX_TIMESTAMP(created_at) AS created_at`，同时保证别名匹配；对允许 NULL 的字段采用 `COALESCE` 或相应可空类型。

## 10. 个人中心性别未转发，头像 URL 未消费

**路由：** `GET /v1/sys/profile`、`PUT /v1/sys/profile`。生成路由的子路径是 `/`，go-zero `WithPrefix` 使用 `path.Join` 合成为上述路径。

**证据：**

- `go-zero-admin/application/admin/api/desc/sys/profile.api:24` 至 `:30` 声明更新可接收 `avatar_url`、`gender`。
- `go-zero-admin/application/admin/api/internal/logic/profile/update_profile_logic.go:27` 至 `:30` 没有转发 `Gender`；`Gender: req.Gender` 位于注释中。`AvatarFileId` 固定为 0，`req.AvatarUrl` 没有消费。
- `go-zero-admin/application/admin/rpc/internal/logic/profileservice/update_profile_logic.go:31` 无条件把 RPC `Gender` 写回用户，因此通过当前 HTTP 更新资料可能把原性别覆盖为 0；`:36` 仅在文件 ID 大于 0 时更新头像。
- `go-zero-admin/application/admin/rpc/internal/logic/profileservice/get_profile_logic.go:32` 至 `:38` 只填 `AvatarFileId`，没有解析或填 `AvatarUrl`，API 映射却会读取它：`go-zero-admin/application/admin/api/internal/logic/profile/get_profile_logic.go:34`。
- 已核对生成 RPC 请求确有 `Gender`、`AvatarFileId` 字段：`go-zero-admin/application/admin/rpc/adminpb/admin.pb.go:1923`、`:1924`。

**确认方式：** 静态确认，不宣称资料 HTTP 更新已经完整保存所有声明字段。

**前端处理/仍阻塞：** 不提供会误导为已持久化的头像编辑功能；性别字段应明确只读或未支持。即使前端不提交性别，当前 API 仍会将零值传到 RPC，因此姓名/邮箱等基本资料更新也不能保证保留原性别，需要主代理在交付时明确风险和限制。

**后端建议（待确认）：** 明确头像使用文件 ID 还是 URL，再同步契约；真实转发性别，并定义字段省略时保留原值的更新语义。补齐头像读取与文件授权检查。

## 11. 菜单删除按路径前缀扩散，不等同于层级删除

**路由：** `DELETE /v1/sys/menu/:id`。

**证据：** `go-zero-admin/application/admin/rpc/internal/logic/menuservice/delete_menu_logic.go:25` 读取导航 `path`，`:27` 执行 `WHERE path LIKE path+"%"`；空 path 则只删当前 ID。该 `path` 是导航路径，不是由 `parent_id` 生成的物化层级路径。前缀可匹配其他并非后代的菜单，`%`、`_` 还会作为 LIKE 通配符解释。

**确认方式：** 静态确认，未新增破坏性删除测试。

**首次联调时的前端处理（历史）：** 当时禁止目录、按钮、空/根/异常路径及含上述通配字符路径的删除；其他菜单删除前提示路径前缀影响。回退后保留的前端已允许目录/按钮删除并提示真实子树与授权关系清理，旧限制不再存在，当前必须按 R05 处理前后端删除语义偏差。

**后端建议（待确认）：** 按真实父子关系确定删除集合，校验引用并使用事务；不要以未转义的导航 URL 前缀替代层级关系。

## 12. 文件列表字段与有效期语义存在静态偏差

**路由：** `GET /v1/sys/file/list`、`GET /v1/sys/file/:id/url`。

**证据：**

- `go-zero-admin/application/file/rpc/internal/logic/fileservice/get_file_list_logic.go:62` 的 `COALESCE(uploader_id,0)`、`UNIX_TIMESTAMP(created_at)` 没有匹配 row tag 的列别名，属于用户列表同类字段归零问题。
- `go-zero-admin/application/file/rpc/internal/logic/fileservice/get_file_u_r_l_logic.go:38` 至 `:47` 计算请求 TTL，然后调用按存储配置签名的 `Store.URL`，忽略返回的有效期，最终返回自行计算的 `ExpiresAt`。HTTP 层当前不暴露自选 TTL，默认配置恰好同值时可能不显现偏差。

**确认方式：** 静态确认；不以文件列表此前返回成功推断全部非零字段和不同存储 TTL 都正确。

**前端处理/仍阻塞：** 仍只展示服务器返回的元数据和获取链接结果，不凭空补上传者、创建时间或延长签名链接有效期。

**后端建议（待确认）：** 添加表达式别名；统一实际签名有效期与响应 `expires_at` 的语义，覆盖存储配置非默认值的验证。

## 13. HTTP 能力边界：缺少的接口不得由前端假装完成

| 能力 | 当前准确入口与证据 | 确认方式 | 本轮前端边界 |
|---|---|---|---|
| 文件上传 | REST 只有 `GET /v1/sys/file/list`、`GET /v1/sys/file/:id/url`、`DELETE /v1/sys/file/:id`，见 `go-zero-admin/application/admin/api/desc/sys/file.api:55`、`:58`、`:61`；上传仅定义于 `go-zero-admin/application/file/rpc/desc/file.proto:53` 的 `UploadFile` | 静态核对 desc 与生成 routes | 不添加请求不存在 HTTP 地址的上传按钮；文件三项现有能力继续对接 |
| 用户分配角色 | `go-zero-admin/application/admin/api/desc/sys/user.api:97` 至 `:115` 只有用户查询、增改删、重置/修改密码；创建/更新字段没有 `role_ids`。`go-zero-admin/application/admin/rpc/desc/admin.proto:531` 的 UserService 同样没有角色分配方法 | 静态核对 REST/RPC 描述与实际注册路由 | 不把用户编辑中的多选框或本地状态当成真实角色分配 |
| App 用户与会员 | `go-zero-admin/application/app/rpc/desc/app.proto:96`、`:109` 的 AppAuthService/AppAdminService | 静态核对 RPC 与统一 API imports | 当前没有可对接的 REST 路由，不构造 `/v1/app/*` 等不存在入口 |
| AI 会话、工作流、任务 | `go-zero-admin/application/ai/rpc/desc/ai.proto:73` 的 AiService | 静态核对 RPC 与统一 API imports | 不将 RPC 就绪标记成 AI 网页功能已联调 |
| 租户管理 | `go-zero-admin/application/tenant/rpc/desc/tenant.proto:47` 的 TenantService | 静态核对 RPC 与统一 API imports | 不将租户创建/启停/重置伪装成 REST 已完成 |

统一 API 的真实 imports 见 `go-zero-admin/application/admin/api/desc/admin.api:9` 至 `:18`，生成 HTTP 注册清单见 `go-zero-admin/application/admin/api/internal/handler/routes.go:27`。App/AI/tenant 没有出现在这份 REST 注册清单中。

**后端建议（待确认）：** 只有用户确认增加具体 HTTP 能力后，才先更新共享契约、修改 `.api`/proto 来源、重新生成并实现服务及授权，再联调相应前端。此次不增加新接口，也不更新原有状态矩阵。

## 14. 正常并发鉴权触发 API 进程退出

**影响：** 共享 Authority 的受保护 HTTP 接口；API 退出后所有页面请求会表现为代理 502/连接失败。

**本轮实测：** 2026-10-30 浏览器正常访问、筛选、查看详情和刷新页面时，API 在 23:22 退出，stderr 包含以下最小调用链：

```text
fatal error: concurrent map writes
github.com/casbin/casbin/v2/persist/cache.(*DefaultCache).Get
  casbin/v2@v2.77.1/persist/cache/default-cache.go:45
github.com/casbin/casbin/v2.(*CachedEnforcer).Enforce
go-zero-admin/application/admin/api/internal/middleware.(*Authority).Handle
```

**证据与判断：** 当前项目使用共享 CachedEnforcer；依赖默认缓存的 Get 在过期分支删除 map 项，需要核验其并发锁策略。已直接观察到进程退出，不把它归为前端空列表或单个接口失败。原始鉴权格式、后端源码和依赖均未修改；后续只重启同一 API 继续短时验收。

**前端处理：** 网络失败统一提示并提供重试，不为规避后端竞态串行化正常业务请求，也不伪造成功。

**后端建议（待确认）：** 核实缓存过期路径并发安全及可用的框架内实现；任何替换、依赖升级或鉴权修改先取得用户确认。

## 2026-10-30 前端验收记录（历史）

现有生成路由共41项（40项业务接口及1项健康检查），前端API函数、类型及对应页面均已接入。未改变任何后端源码、契约、生成物或依赖文件。

| 验证项 | 结果与边界 |
|---|---|
| 前端自检 | TypeScript严格模式、Vite生产构建、oxlint通过；12项Node内置测试通过。Vite保留共享包体积提示。 |
| 真实浏览器 | 16项检查完成：未登录跳转、验证码登录、9个页面、用户筛选与详情、刷新恢复、权限失败隐藏写操作、文件链接与实际对象、401清会话。页面无运行时异常；受阻页面的错误状态也计入检查，不计为后端成功。 |
| 真实用户写入 | 新增、详情、完整编辑、删除和列表刷新通过；只操作本轮临时账号。 |
| 真实字典写入 | 新增、删除和列表刷新通过；详情400时确认编辑保存禁用，未把字典编辑标为真实通过。 |
| 写入测试的权限来源 | 由于真实权限接口505，仅在独立测试浏览器中用与测试菜单一致的权限夹具替代该查询；上述业务接口仍全部请求真实后端、使用标准Bearer，没有改变产品代码或后端授权。正常运行时仍安全隐藏未取得权限的写操作。 |
| 其他写操作 | 前端已实现且按现有契约核对；未宣称所有角色、菜单、部门、文件删除及个人改密均已全量真实验收。 |

用户要求的测试数据保留：28个demo用户、5个demo部门、4个demo角色、31个菜单、41项API资源、4个字典、12条明确标记模拟审计，以及3份真实文件及元数据。原有管理员账号、密码、权限和状态未变；仅将8处已精确证明是UTF8被latin1误解码的默认种子文本修回原SQL中文，保留更新时间。临时验收账号与字典另行清理，不作为交付数据。

**清理核对：** 活跃数据最终为29用户（含原管理员）、6部门、7角色、31菜单、41API、4字典、3文件、12审计。临时账号和字典均为0条。本轮API/RPC/Vite已停止、仓库外临时配置/密钥/日志已删除；专用MySQL、Redis、RustFS均正常退出，数据库与对象存储持久卷保留。其他项目容器未改动。

## 回退前交付结论（历史）

回退前，前端现有接口接入、四项审查修复和当时必要后端配合已完成验证。保留偏差继续明确列出，不把正常路径通过等同于整个后端符合全部规范或已经完成安全审计整改。本段不适用于后端恢复到 `1fad38b` 后的组合。

当时授权范围内的配合修复、契约来源修改与重新生成已经完成；新增 HTTP 能力、依赖升级及其余架构整改不包含在当时结果中。后续工作以本文顶部“回退后待修改清单”为准。
