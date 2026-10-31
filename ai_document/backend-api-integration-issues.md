# 后端 API 待整改清单

更新：2026-10-31。后端已回退至 `1fad38b`（回退提交 `b534a82`），前端保留 `215dc46`。以下 **13 项未修复、3 项待确认**，记录不代表修复授权；接口契约见 [api-contract.md](api-contract.md)。

## 2026-10-31 修复验收（三批全部完成，真实调用通过）

环境：compose 全栈（gozero-platform），verify.ps1 六项全过。

| 编号 | 状态 | 验收证据 |
|---|---|---|
| R01 | ✅ | deny-by-default 白名单制（仅 captcha/login/ping）；无 token/坏 token → 401；demo_all(GET 通配)用户 POST user/create → 403；路由级 jwt 已移除（消除双重解析） |
| R02+R03 | ✅ | 弃用 CachedEnforcer，`pkg/casbinauth.Holder` 原子替换；中间件经 Holder 取值不再固化旧引用；Redis 订阅 + 60s 定时兜底 + 断线重订阅；热更新日志实测 "casbin policies refreshed" |
| R04 | ✅ | menu list/tree/my/perm-codes 由 505 变为正常返回；IN 切片改占位符展开；时间列 UNIX_TIMESTAMP+AS；查询下沉 model 扩展 |
| R05 | ✅ | SoftDeleteSubtree 按 parent_id 递归 + 事务 + 删 sys_role_menu + 清缓存；实测父删子随删、role_menu 零残留、其他角色授权不受影响；测试数据已物理清理 |
| R06 | ✅ | dict.api 补 `path:"code"`（/dict/code/demo_priority 实测返回完整数据）；file.api 双引号 typo 修复；均经 gen.ps1 重新生成 |
| R07 | ✅ | api/list、audit/list 由 505 变为正常返回；total 吞错已修；audit user_id/params COALESCE |
| R08 | ✅ | API 层补转发 Gender；实测 gender=1 保存后不再被文本更新清零（注意：proto3 无法区分"省略"与 0，前端表单需全量提交） |
| R09 | ✅ | gofmt 改临时副本 CRLF→LF 归一比较；go list 非零退出即失败；go.mod/go.sum 行尾已归一为 LF |
| R10 | ✅ | user/list 实测 dept_id=9001002（原静默归零），时间戳非零 |
| R11 | ✅ | update_role_menus 事务化 + 去重 + 错误检查；role/list 实测 super_admin menus=31 users=1 |
| R12 | ✅（别名部分） | dept/tree created_at 实测非零；移动部门未开放，防环不在本次范围 |
| R13 | ✅（别名部分） | file/list 实测 uploader_id=1；`expires_at` 与签名期限一致性（ttl 参数未生效）**遗留**，改动小待下轮 |

待确认 C01-C03 维持不变。遗留小项：R13 的 ttl 透传、verify.ps1 白名单与规范 §2.4 的措辞对齐（.agents/.claude 等工具目录）。

### 2026-10-31 复审追加修复（第二轮）

| 项 | 状态 | 验收证据 |
|---|---|---|
| 菜单 parent_id 成环死循环 | ✅ | SoftDeleteSubtree 收集加 visited 防环；update_menu 挂载前校验"不能挂到自身/后代"，实测两种成环场景均 400，正常移动与子树删除回归通过 |
| user_count 误计角色继承 | ✅ | CountUsersByRoleCodes 加 `v0 LIKE 'user:%'` 过滤 |
| 角色列表 N+1 | ✅ | 改批量 `ListMenuIdsByRoleIds`，实测 super_admin=31menus/1users 不变 |
| RPC in.Page 空指针 | ✅ | 全部 13 处改 `in.GetPage().GetPage()` nil 安全访问器（含既有未重写文件） |
| verify.ps1 robocopy 假通过 | ✅ | 拷贝退出码 ≥8 即失败 |

另发现既有拼写缺陷未动：`xerr.REUQEST_PARAM_ERROR`（REUQEST→REQUEST 拼错），改名波及面大，待专门处理。

前端字典路径参数、菜单子树删除和资料性别保存仍受旧后端影响，整体对接尚未通过。

## 待整改

| 编号 | 位置 | 待改内容 |
|---|---|---|
| R01 | [鉴权入口](../go-zero-admin/application/admin/api/admin.go) | 非公开路由统一鉴权，避免 token 格式差异跳过 Casbin；验证缺失、过期、无权限和合法请求。 |
| R02 | [Casbin 缓存](../go-zero-admin/pkg/casbinauth/enforcer.go) | 缓存过期曾触发并发崩溃，改用现有依赖中的并发安全实现，覆盖命中、未命中和过期。 |
| R03 | [策略刷新](../go-zero-admin/application/admin/api/internal/svc/service_context.go) | 修复中间件持有旧引用；完整构建后原子更新，补偿订阅窗口、断线及刷新失败，即使无后续通知也能恢复。 |
| R04 | [菜单查询](../go-zero-admin/application/admin/rpc/internal/logic/menuservice/) | 查询移入 model 扩展，修复时间/row 映射、直扫 pb 和 IN 参数；区分数据库错误与无角色/权限，核对非空数据及软删除过滤。 |
| R05 | [菜单写入](../go-zero-admin/application/admin/rpc/internal/logic/menuservice/) | 按 `parent_id` 事务删除子树及角色关联；编辑限定有效行，清理相关缓存。验证子树边界、删改并发、无变化更新与回滚。 |
| R06 | [字典](../go-zero-admin/application/admin/api/desc/sys/dict.api)、[文件](../go-zero-admin/application/admin/api/desc/sys/file.api) | 补字典 `path:"code"`、修复文件 URL 标签引号后重新生成；字典列表补 row tag、时间转换和别名，验证路径查询及增改删。 |
| R07 | [API 列表](../go-zero-admin/application/admin/rpc/internal/logic/apiservice/get_api_list_logic.go)、[审计列表](../go-zero-admin/application/admin/rpc/internal/logic/auditservice/get_audit_log_list_logic.go) | 查询移入 model 扩展，修复时间扫描及审计 `user_id/params` 的 NULL 处理；核对非空数据、筛选和分页。 |
| R08 | [个人资料](../go-zero-admin/application/admin/api/internal/logic/profile/update_profile_logic.go) | API 补转发 Gender，避免保存文本资料时将非零性别覆盖为零。 |
| R09 | [自检脚本](../go-zero-admin/data/scripts/verify.ps1) | 检查 gofmt/go list 退出码；CRLF 仅在只读比较或临时副本归一，不能掩盖格式、依赖和校验和差异。 |
| R10 | [用户列表](../go-zero-admin/application/admin/rpc/internal/logic/userservice/get_user_list_logic.go) | 补部门、头像文件 ID、登录及创建/更新时间的 SQL 别名，以非零数据库值核对响应。 |
| R11 | [角色管理](../go-zero-admin/application/admin/rpc/internal/logic/roleservice/) | 补时间别名、明确统计字段；菜单授权改为事务并处理错误、回滚和缓存，覆盖重复/空菜单。部分更新语义待确认。 |
| R12 | [部门管理](../go-zero-admin/application/admin/rpc/internal/logic/deptservice/) | 补时间别名与缓存失效；若开放移动，需事务维护后代路径并防环。当前前端禁移，扁平 tree 本身不是缺陷。 |
| R13 | [文件服务](../go-zero-admin/application/file/rpc/internal/logic/fileservice/) | 补上传者/时间别名，使 `expires_at` 与实际签名期限一致；验证非默认 TTL、公开/私有对象及引用删除限制。 |

判断边界：R01 不等于无 token 绕过 JWT；R06 缺 tag 不足以断言查询必败；旧菜单更新未写 `deleted_at`，不能认定会复活记录。已撤回的 `DeleteSubtree` 漏清角色菜单主键/唯一索引缓存，重做须覆盖，但尚未证明因此越权。

## 待确认

- **C01 响应规范**：成功信封、HTTP/业务错误码及迁移方式；前端目前兼容裸数据、空成功体和信封。
- **C02 资料更新**：头像用文件 ID 还是 URL，省略字段表示保留还是清空；明确文件授权后同步两端。头像编辑尚未开放，R08 是独立缺陷。
- **C03 缺失能力**：API 同步、文件上传、用户分配角色、App/AI/租户 HTTP 接口。`SyncApi` RPC 目前仅发通知并返回零计数，补路由不等于实现同步。

## 执行与历史

- 确认范围后逐模块修复并真实调用；契约先更新、生成物从来源生成、不擅自升级依赖。验收覆盖非空/NULL、权限、事务回滚和缓存；生成 CRUD 有限制时先确认方案。
- 新增后端测试放 `go-zero-admin/test/`，临时测试用完删除并保留命令/结果；原有测试、正式导入器和联调数据保留。
- 历史验证：原 `verify.ps1` 在 CRLF 工作区失败，隔离副本自检及前端构建曾通过；回退前联调未覆盖全部异常/权限，未运行 race detector。这些结果不能代替当前整改验收，本次仅简化文档。
- [完整源码证据、详细验收及历史记录（固定版本 b534a82）](https://github.com/yh-zero/go-zero-admin/blob/b534a820a85d39750dc4116d889599ff17178f39/ai_document/backend-api-integration-issues.md)：保留历史流程违规、当时仓库规则偏离及测试清理记录，不等同于 go-zero 官方限制。
