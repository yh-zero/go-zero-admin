# API 契约（前后端共同遵守）

本文规定后台与 App 的对外行为。改动先更新契约，再同步两端；🚧 表示已约定、实现或真实联调尚未完成，✅ 表示对应范围已有两端联调。桩环境覆盖和待验证的真实场景在相应条目注明。
完整 HTTP 签名见 [API 定义入口](../go-zero-admin/application/admin/api/desc/admin.api)。路由统一以 `/v1` 开头，模块内短路径继承相应前缀，`/ping`、`/ready` 除外。[根规范](../AGENTS.md)

## 1. 响应格式

业务成功与常规业务错误使用 HTTP 200 信封，不能把 HTTP 200 等同于业务成功；服务不可用还需处理非 2xx HTTP 响应，`/ready` 未就绪时返回 HTTP 503。

```json
{ "code": 200, "msg": "操作成功", "result": {} }
```

| 字段 | 说明 |
|---|---|
| code | 200 成功；其余为业务错误码（见 §5）；401 表示登录过期 |
| msg | 错误提示文案（成功时固定"操作成功"） |
| result | 成功数据；无数据写操作返回 `{}`；兼容旧接口的 null/省略，错误响应可不带此字段 |

前端由统一请求层剥壳并处理业务错误，组件不重复弹错；可刷新会话由请求层按既有流程尝试刷新，最终认证失败才清会话并跳登录，迟到响应不能恢复或清除另一有效会话。

成功由 `httpx.SetOkHandler` 统一包装，错误走现有 xerr 出口，不另造响应信封。

### 1.1 图形验证码

`GET /v1/sys/auth/captcha` 返回 `enabled/captcha_id/image_base64/expires_at`。

- 启用时，`captcha_id` 为 64 位 hex，`image_base64` 为 PNG data URI，`expires_at` 为 Unix 秒；服务端保存并原子消费答案。
- admin-rpc 的 `Captcha.Enabled` 默认 true。关闭时返回 `enabled:false`，验证码 ID、图片和到期时间分别为空字符串、空字符串、0；不生成验证码。登录字段 `captcha_id/captcha` 可省略，启用时仍强制校验，关闭时忽略；账号、密码、状态和限流始终生效。前端仅在明确收到 false 时隐藏验证码，配置请求失败时暂停登录。

### 1.2 登录响应（双令牌 + 服务端会话）

`POST /v1/sys/auth/login` 成功后返回 `access_token/expires_at/refresh_token/refresh_expires_at/session_id`。短期访问令牌携带会话标识；Redis 保存设备会话，访问及刷新时以服务端会话与用户状态为准。

- 刷新令牌通过 `POST /v1/sys/auth/refresh` 一次性轮换；并发使用同一令牌至多成功一次。

## 2. 命名与类型

| 项 | 约定 |
|---|---|
| JSON 字段 | snake_case（如 `access_token`、`dept_id`） |
| 时间 | int64 Unix 秒（如 `created_at`） |
| ID | 业务实体 ID 为 int64；会话等字符串标识按各接口定义 |
| 枚举 | 字符串优先，如 `visibility` 可为 `public` 或 `private`。数值状态配合字典展示。 |

## 3. 鉴权

- 请求头：`Authorization: Bearer <access_token>`
- 登录：`POST /v1/sys/auth/login`；先调用 `GET /v1/sys/auth/captcha` 获取验证码开关及启用时的图片。
- 权限码格式：`sys:<模块>:<动作>`（如 `sys:user:add`），按钮级控制由后端下发，前端只做渲染判断
- 部门上下文：多部门用户可用请求头 `X-Dept-Id` 切换（可选）

- 退出或撤销后旧访问令牌及刷新令牌均失效；存量无会话令牌需重新登录。
- 后台用户启停状态实际变化时先撤销该用户全部会话；重新启用不会恢复离线旧令牌，撤销失败则不提交状态变更。普通资料编辑和不改变状态的请求不撤销会话。刷新返回的 `expires_at` 不晚于会话绝对到期时间。
- `POST /v1/sys/auth/refresh`：请求 `refresh_token`，返回与登录相同的令牌字段；刷新令牌一次性轮换。
- `POST /v1/sys/auth/logout`：认证必需，请求 `all_devices`（默认 false）；默认撤销当前会话，true 撤销本人全部会话。
- `GET /v1/sys/auth/sessions`：返回 `list`，每项 `id/device_name/ip/user_agent/created_at/last_seen_at/expires_at/is_current`。`DELETE /v1/sys/auth/sessions/:id` 仅撤销本人会话。
- 登录签发与用户改密、启停使用同一用户行锁；签发前重验密码哈希和当前状态，改密或状态变更成功后，旧凭据产生的会话不得恢复。App 登录与审核状态变更遵守相同规则。
- 浏览器同源标签页共享会话更新和退出状态，刷新令牌轮换互斥；旧标签页不得清除另一标签页刚刷新的有效会话。 多窗口并发刷新与迟到响应的真实浏览器闭环仍待验证。
- 自动刷新使用浏览器原生 Web Locks，需要支持该能力的 HTTPS/本机地址；不支持时明确提示，不并发消耗共享刷新令牌。

## 4. 通用结构

### 分页

分页可来自 GET query 或 POST body。请求 `{ "page": 1, "page_size": 20 }`，成功信封 `result` 为 `{ "list": [], "total": 100 }`。过滤条件按各模块定义。

`page/page_size` 省略或 0 时使用服务端默认值，负数或 `page_size>100` 返回业务码 400。特殊游标分页见相应端点；部分 App 接口的默认值仍待真实 HTTP 闭环验证。

### 菜单与权限码

`GET /v1/sys/menu/my` → `result: { "list": [MenuInfo] }`，列表通过 `parent_id` 表达父子关系
`GET /v1/sys/menu/my/perm-codes` → `result: { "codes": ["sys:user:add", ...] }`

### 部分更新

- ✅ 角色、菜单、字典更新字段省略或 `null` 保持原值，显式空字符串/0 只清空允许清空的字段；角色编辑的 `menu_ids`、字典 `items` 省略或 null 保持，空数组清空；独立角色菜单授权接口仍执行完整替换。
- 用户 `role_ids` 省略或 null 保持，空数组清空。用户及个人资料使用 `avatar_file_id`，`avatar_url` 只读，不接受客户端提供可信操作者身份。
- 🚧 用户、个人资料和部门邮箱允许显式空字符串清空。
- 状态与解绑：用户/部门 `status` 仅为 1（启用）或 2（停用），0 不表示清空；`gender` 为 0/1/2。`dept_id:0`、`avatar_file_id:0` 表示解绑。

### 账号与密码策略（后台）

- 用户名：创建时 2–64 字符；登录/筛选不设最小长度（兼容历史账号），上限 64。
- 🚧 密码：设置类操作（创建用户、管理员重置、自助改密）至少 8 个字符，最多 72 字节（bcrypt 上限）；登录输入只校验字节上限。App 注册保留 8–64 字符，同时不得超过 72 字节。超过字节上限返回参数错误 400，不返回内部错误。 多字节边界仍待前后端真实 HTTP 验证。

## 5. 错误码（与后端 pkg/xerr 一致）

| code | 含义 | 前端处理 |
|---|---|---|
| 200 | 成功 | 正常 |
| 400 | 参数错误 / 业务校验失败 | 弹 msg |
| 401 | 登录过期 / 未登录 | 统一请求层尝试合法刷新，最终认证失败再清会话 |
| 402 | 令牌生成失败 | 弹 msg |
| 403 | 接口权限不足 | 弹 msg |
| 404 | 资源不存在 | 弹 msg |
| 409 | 版本、幂等内容或资源状态冲突，详见端点 | 保留草稿或请求标识，重新核对 |
| 429 | 用量准入或限流拒绝 | 提示限制，禁止自动重试付费调用 |
| 500 | 服务器内部错误 | 弹 msg |
| 503 | 服务不可用（含 AI/SMTP 未配置）；`/ready` 未就绪使用 HTTP 503 | 业务请求提示失败；探针判未就绪 |
| 505 | 数据库操作失败 | 弹 msg |
| 506 | 数据更新未生效 | 弹 msg |
| 10001-10007 | 用户/认证业务码 | 弹 msg |
| 20001-20007 | 角色/菜单/部门/字典/API 业务码 | 弹 msg |

通用及分模块常量见 [pkg/xerr](../go-zero-admin/pkg/xerr/errcode.go)。各端点还可返回表中的业务冲突和限流码。

## 6. 模块路由前缀

| 前缀 | 模块 | 说明 |
|---|---|---|
| /v1/sys/auth | 认证 | 验证码/登录公开；刷新校验刷新令牌，其他操作要求认证 |
| /v1/sys/user | 用户 | |
| /v1/sys/role | 角色 | |
| /v1/sys/menu | 菜单 | 含 /my、/my/perm-codes |
| /v1/sys/dept | 部门 | |
| /v1/sys/dict | 字典 | |
| /v1/sys/file | 文件 | |
| /v1/sys/audit | 审计 | |
| /v1/sys/profile | 个人中心 | |
| /v1/sys/api | API 资源 | |
| /v1/sys/tenant | 租户管理 | 仅平台栈；仅超管；创建/重置密码仅当次返回 |
| /v1/sys/ai | AI | 后台认证、接口权限及本人资源隔离 |
| /v1/sys/app | 后台 App 管理 | 后台认证域 |
| /v1/app | App 认证、资料、会员与反馈 | 独立 App 认证域 |

## 7. 模块行为约定

### 健康与审计

- `/ping` 返回存活状态；`/ready` 检查必要 RPC、数据库及 Redis，就绪返回 HTTP 200，未就绪 HTTP 503；不依赖可选 AI 外部模型。
- 审计通过 RPC 写现有审计表，记录操作者、路径、结果和耗时，不记录密码、令牌及完整请求体。
- 审计列表额外支持 username/method/path/ip、start_time/end_time 和 min_duration_ms/max_duration_ms 筛选；start/end 与耗时上下界必须合法。`POST /v1/sys/audit/export` 复用同一筛选并导出 UTF-8 CSV（含 BOM），单批最多 10000 行，超出时返回 400 要求缩小范围。导出字段做公式注入转义，不导出 params 或敏感请求体。admin-rpc 的 `Audit.RetentionDays` 默认 0 表示不自动删除；正值时按创建时间保留最近天数，`CleanupIntervalSeconds` 和 `CleanupBatchSize` 控制清理频率与单批上限。

### 用户、角色与数据范围

- ✅ 权限查看 `GET /v1/sys/user/:id/permissions` 仅全局超级管理员可调用。按目标用户可选择的部门返回角色及绑定来源、用户管理页面和操作授权、对应接口的 Casbin 判定，以及列表数据范围。列表接口允许时返回可见人数和按 ID 倒序排列的前 10 个账号示例。省略 `dept_id` 使用目标用户默认部门，显式选择超出其可用部门时返回 403。停用用户返回 `enabled=false` 和空权限。详情与修改类接口判定使用被查看的用户 ID 作为路径示例，具体记录仍使用业务数据范围。
- ✅ 后台用户导入 `POST /v1/sys/user/import` 要求导入和新增用户的接口权限。请求 `csv` 接受 UTF-8（可带 BOM），固定表头 `username,dept_id,real_name`，最多 64 KiB 和 100 条记录。省略 `confirm` 仅预检，返回物理行号及逐行结果。确认时提交相同 CSV、`confirm=true` 和选中的 `row_numbers`，服务端重新校验并处理选中行。部门必须有效且在操作者数据范围内，`dept_id=0` 代表未分配，仍受范围约束。重复账号不覆盖。新账号停用且没有可用密码，管理员先启用，再重置密码以开通账号。响应行状态为 `ready`、`invalid`、`created` 或 `failed`，并返回成功创建数量。
- ✅ 组织通讯录 `GET /v1/sys/dept/directory` 使用独立的菜单和接口授权。按可信当前部门分页查询启用且未删除的直属成员，支持姓名搜索和负责人标记。仅返回成员 ID、展示名、负责人标记及部门名。无有效当前部门时返回 403，切换部门沿用现有部门选择规则。

- 创建后台用户时 `avatar_file_id` 仅允许省略或 0，其他值返回 400；创建成功后通过用户编辑设置头像。已有头像按稳定用户 ID 对账。持续跨服务故障后的头像/文件引用恢复仍待验证，跨库修改无原子提交保证。
- `role_ids` 回读全部未删除角色 ID 的去重集合。显式列表中保留的角色维持原部门绑定，移除角色删除全部绑定，新普通角色绑定用户主部门（无部门为 `dept:0`）；更换主部门只迁移旧主部门绑定。分配角色限定超级管理员，禁止移除最后一个有效超级管理员；省略角色字段不影响授权。
- 用户主部门实际变更仅允许超级管理员，避免通过迁移部门绕过角色分配权限；传入与当前相同的 `dept_id` 不构成变更。角色 `code` 创建后不可修改，普通编辑需保留原值，否则返回 400，避免已有 Casbin 绑定失效。
- 重置密码和自助改密只更新密码列，目标必须未删除且启用；自助改密提交时再次校验旧密码哈希未变化，并发改密冲突返回旧密码错误，不覆盖头像、部门或状态。成功后撤销该用户全部会话。
- 用户管理的列表、详情、创建、编辑、删除和重置密码受数据范围约束：当前服务端验证部门内的有效角色（含全局绑定）取并集；`ALL` 为全部，`DEPT` 为当前部门，`DEPT_AND_CHILDREN` 含当前部门及有效下级，`SELF` 仅本人，全局超级管理员为全部。无有效范围拒绝访问；历史范围值去空白并忽略大小写，未知值不授予权限。`dept_id=0` 的部门范围仅代表未分配部门，不代表全部部门。客户端部门筛选只能进一步缩小列表。创建必须落在可管理部门，SELF 不允许创建他人；不存在目标保留用户不存在错误，越界目标返回 403。个人资料和认证状态读取固定本人，不以用户管理范围放行他人。
- 个人资料只读字段 `is_super_admin` 由服务端有效全局超级管理员绑定计算，供前端显示角色分配等入口，不根据用户名判断；后端仍独立执行权限校验。
- 删除角色与清理授权关系在同一事务内完成，并失效该角色 ID/编码缓存；角色编辑只更新可编辑列，不可恢复已删除角色。
- `POST /v1/sys/role/:id/menus` 替换授权时，同时传入读取角色时的 `expected_menu_ids` 和目标 `menu_ids`。两者都是正整数数组，`[]` 表示空集。`PUT /v1/sys/role/:id` 显式修改 `menu_ids` 时同样必须携带原集合。缺少或传 `null` 的原集合返回 400；当前有效授权与原集合不同返回业务码 409，整次保存回滚，客户端重新读取角色并核对后再编辑。集合比较忽略顺序和重复项。只更新角色资料和新建角色沿用现有调用方式。
- ✅ 角色、菜单、字典创建时省略 status 默认 1，显式状态只允许 1/2；角色省略 data_scope 默认 SELF。角色 code 编辑省略或 null 保持，显式不同于现值返回 400。普通角色资料编辑不自动提交旧菜单授权。

### 部门

- `GET /v1/sys/dept/my` 返回当前用户允许选择的部门 `list`。`X-Dept-Id` 必须属于服务端验证的部门身份，鉴权、context 和下游 Actor 使用同一个部门；部门移动事务维护后代路径并拒绝环。
- 部门负责人：沿用现有部门及用户接口，不新增 HTTP/RPC 方法。`DeptInfo` 包含 `leader_user_id`、`member_count`、`members_visible`。负责人 ID 为 0 表示未关联账号，`leader` 保留历史文本及只读显示名兼容；关联账号时显示其真实姓名、昵称或用户名，并标注已停用/已删除状态，不自动授予角色。部门创建/编辑使用 `leader_user_id`，省略或 null 保持编辑原值，编辑显式 0 清空关联和历史负责人文本，正值必须为当前操作者数据范围内的启用账号，且操作者具有用户列表读取权限。创建时正负责人 ID、编辑时任意显式负责人 ID，均不能与非空历史 `leader` 同时提交；创建时省略或 0 保留旧客户端文字录入兼容。历史文本仅在未关联账号时允许维护。
- 部门直属成员复用 `GET /v1/sys/user/list?dept_id=...`，保持接口读取权限及 Actor 数据范围。部门树的 `member_count` 仅统计当前身份可查询、未删除的直属用户（包含停用用户）；有用户列表读取权限且数据范围有效时 `members_visible=true`，否则 false、人数为 0，前端显示“—”，不将不可读取解释为部门无人。`/dept/my` 不附带成员统计。搜索及状态筛选在前端保留匹配节点的祖先，不改变组织层级。
- 部门停用与删除沿用现有更新/删除接口：存在未删除子部门、归属用户或部门域成员授权时拒绝，提示先处理关联关系，不自动级联停用或转移。校验与写入在同一事务中执行；负责人关联不是成员授权，不授予部门身份。当前仅展示直属成员，不实现兼职、专门调岗流程或独立影响预览。

### 菜单、字典与 API 资源

- `GET /v1/sys/dict/code/:code` 的 `code` 从路径绑定，不要求 GET JSON 请求体或额外 query。
- ✅ 菜单删除按 `parent_id` 确定当前节点及全部后代，在同一事务删除菜单授权关系并软删除菜单，提交后失效详情缓存；导航路径前缀相同的其他节点不受影响。创建、移动和子树删除共享数据库结构锁，校验和写入在同一事务内；普通资料编辑只锁目标行。已删除菜单不能被更新恢复，缺少结构锁记录时拒绝结构写入。
- `POST /v1/sys/api/sync` 使用实际注册路由同步 API 资源，返回 `added/updated/removed`。
- API 同步采用增量 upsert：保留已有人工标题和非本服务资源，`removed` 固定为 0；新增标题采用 `METHOD /path`。无来源标记时不自动删除其他资源。
- 菜单/按钮通过 `sys_permission(menu_id,api_id)` 显式关联 API。角色授权事务按映射重建该角色的 API 策略，菜单变更后重建所有角色；策略只使用 `sub/dom/obj/act` 字段，超级管理员 `/*` 通配规则和用户级直接授权不受影响。没有映射的菜单不会隐式获得 API 权限。`MenuInfo.api_ids` 用于维护映射。
- 非超级管理员不能修改内置超级管理员角色或自己已绑定的角色，也不能授予超出自身 API 权限和数据范围的操作。超级管理员不受此限制，但角色分配仍按原规则仅允许超级管理员执行。
- `POST /v1/sys/permission/check` 仅超级管理员可调用，接收可选 user_id/dept_id、method/path，按当前内存策略返回 allowed、命中 role/domain、data_scopes、policy_revision、generated_at 和 loaded_at。策略快照带递增 revision，发布失败保留旧快照并记录错误。

### 文件与引用

- 文件列表的 `url` 为可选字段，FileInfo 另含 `uploader_name/sha256`；历史文件的 sha256 为空。列表支持 name/mime/ext、大小与创建时间范围筛选，`cleanup_pending_only=true` 仅查看当前上传者的待清理记录。`GET /v1/sys/file/:id` 返回文件及引用明细，`GET /v1/sys/file/stats` 返回当前上传者有效文件和待清理记录的数量/字节统计；详情及引用仅上传者可读，统计按当前上传者汇总；列表沿用原有效文件数据范围。
- 删除先锁定文件并核对上传者及引用，软删除后即停止提供链接和新增引用。对象存储确认删除后记录清理完成。文件中心查询传 `include_cleanup_pending=true`，额外返回当前上传者尚未确认清理的软删除行，标记 `cleanup_pending: true`；其他用户不可见。默认列表及工作台仍只统计有效文件。刷新后可用原删除操作重试，重复完成幂等。历史软删除没有清理回执时同样允许重新确认。
- `POST /v1/sys/file/upload` 接收 `name/content_base64/scene/visibility`，原文件最大 10 MiB；服务端检查内容、类型及场景。引用增减仅由内部业务 RPC 调用，按业务类型及 ID 幂等维护。
- 上传响应为 `{file: FileInfo}`；`name/content_base64` 必填，`scene` 为 common/avatar/doc（默认 common），`visibility` 为 public/private（默认 private），原始字节 1..10 MiB。头像限可解码的 JPEG/PNG/GIF 并检查尺寸，不引入额外图像解码依赖；拒绝路径型文件名及 HTML/SVG。服务端推断 MIME，不接受客户端 uploader_id/Actor。
- `POST /v1/sys/file/upload-multipart` 接收单个 file 部件及 scene/visibility 表单字段，沿用 Base64 上传的权限、类型、10 MiB 上限和响应。API 经客户端流式 RPC 分块发送，file-rpc 写入有界临时文件并计算 SHA-256，再流式写入对象存储；结束、失败或取消均清理临时文件。新上传文件持久化 sha256，历史空值不回填。
- 私有文件链接、删除、引用新增验证上传者；公开文件链接允许已认证用户读取。头像引用类型为 admin_avatar/app_avatar，按 `(file_id,ref_type,ref_id)` 唯一幂等维护；删除引用不存在的明细不减少历史引用计数，删除文件同时检查计数和引用明细。
- 引用移除没有 HTTP 入口：仅由已完成业务对象授权的内部 RPC 编排，使用数据库读取的旧文件 ID 和精确业务类型/ID；不以历史上传者限制合法的后续管理员修改，也不信任客户端传入的旧文件 ID。
- 内部引用移除 RPC 的 `keep_file_id` 有字段存在性：省略时精确移除 `FileID`；传入时要求 `FileID=0`，对该业务类型/ID清理除保留文件外的全部引用，0表示全部释放。头像业务在用户行锁内新增引用，事务结束后重新锁行并按数据库当前头像对账；同一头像请求可重复对账，失败返回错误并保留引用明细用于重试。该参数不暴露给 HTTP 客户端。
- 内部文件链接 RPC 支持 `public_only`：App 自查头像固定使用该模式，只返回公开文件地址，私有文件拒绝，不将 App 用户 ID 冒充后台 Actor；后台调用保持真实 Actor 校验。该参数不暴露给 HTTP 客户端。
- 后台为 App 用户设置头像时只接受管理员本人上传的公开头像文件；新增引用验证真实后台操作者，App 自查只读取公开链接，解绑使用 `file_id:0`。

### App 认证与个人资料

- App 使用独立认证域：公开端点仅 register/login/options/email-code；refresh 校验刷新令牌；其余 App 请求校验有效 App 会话和用户当前状态。后台 App 管理、AI、租户接口仍受后台权限控制。
- App 认证：`POST /v1/app/auth/register` 接收 account/password/nickname/email_code；当前仅开放邮箱注册，默认要求一次性邮箱验证码，注册后为 pending，后台审核 active 后才能登录。`POST /auth/login` 接收 account/password；`POST /auth/refresh` 接收 refresh_token；二者返回 access_token/expires_at/refresh_token/refresh_expires_at/session_id/user，刷新原子轮换。`POST /auth/logout` 接收 all_devices（默认 false）。App 与后台使用不同 JWT 密钥、签发者及 Redis 会话命名空间，不能交叉访问。
- 邮箱账号规范化后最长 128 字节，与数据库列一致，超长参数返回 400。
- `GET /v1/app/auth/options` 返回 email_verification_required/email_code_enabled/phone_enabled；`POST /auth/email-code` 接收 account。验证码使用标准库 SMTP 发送，有效期由 YAML `SMTP.ValidSeconds` 配置（默认 600 秒，必须大于 0），到期自动失效、匹配成功后原子消费，输错不删除；邮箱/IP 限流保持。`SMTP.FromName` 设置收件人看到的发件人名称，未配置时仅显示发件邮箱。缺 SMTP 配置明确返回不可用，不返回验证码或伪造发送成功。
- `GET /v1/app/profile` 返回 user；`PUT /v1/app/profile` 接收可选 nickname，省略/null 保持、空字符串清空。user 字段为 id/account/identity_kind/nickname/avatar_file_id/avatar_url/status/review_reason/email_verified/phone_verified/last_login_at/created_at。`GET /v1/app/membership` 返回当前用户有效会员 list；不接受客户端 user_id 或 access_token 字段，访问令牌只从 Bearer 头读取。AppAccess 仅内部 RPC。

### 后台 App 管理、会员与反馈

- 后台 App 管理：`GET /v1/sys/app/users` 接收 page/page_size/status/account，公开头像返回 avatar_url；`POST /users/:id/review` 接收 status/reason（pending/active/disabled/rejected）；`PUT /users/:id/avatar` 接收 file_id（0 解绑）；`GET /users/:id/sessions` 返回 list，复用登录设备字段 id/device_name/ip/user_agent/created_at/last_seen_at/expires_at/is_current（管理其他 App 用户时固定 false），最多 20 条有效会话；`DELETE /users/:id/sessions/:session_id` 撤销该用户会话。所有请求使用服务端后台身份和接口权限；不能伪造 Actor，目标用户必须存在且未删除，停用/驳回撤销旧会话。仅展示安全设备字段，不返回令牌、摘要等内部会话字段。
- 后台会员：`GET /v1/sys/app/memberships` 接收 page/page_size/user_id/plan_code/active_only；`POST /memberships` 接收 user_id/plan_code/benefits/request_id/duration_days（1..3650），request_id 必填且幂等，重复键参数不一致拒绝；`DELETE /memberships/:id` 撤销。列表项为 id/user_id/plan_code/benefits/request_id/granted_by/granted_by_admin/starts_at/expires_at/revoked/revoked_at/created_at，有效期为 starts_at <= 当前时间 < expires_at 且未撤销。
- 后台用户详情与编辑：后台用户详情整合资料/会员/设备页签，审核与启停分入口，继续复用原审核及会话撤销规则。`GET /v1/sys/app/users/:id` 返回 `{user: AppAdminUserInfo}`；后台用户列表也改为 `AppAdminUserInfo`，在原用户字段上增加 `remark`。`PUT /users/:id` 仅允许可选 `nickname/remark`，省略/null 保持，空字符串清空，昵称最多 64 字符、备注最多 500 字符；账户、身份、审核状态由原专用接口维护。内部备注仅随后台用户响应返回，不出现在 App 注册、登录、刷新、个人资料及其他 App 侧响应中。
- 审核与启停提交到 `POST /v1/sys/app/users/:id/review`。请求须携带操作者打开表单时看到的 `expected_status` 及 `expected_reason`（原说明为空时传空字符串）。缺少快照返回业务码 400。在账号行锁内比较当前状态和原说明，任一不一致即返回业务码 409，保留已提交的决定和会话。发生冲突后，客户端重新读取该用户，管理员可核对最新决定后重新发起操作。
- `GET /v1/sys/app/membership-plans` 返回 `{list:[{label,value}]}`，仅从启用的 `app_membership_plan` 字典读取有效名称与编码，不开放任意字典读取，不新增套餐销售系统；开通字段和历史自由套餐编码兼容，后台新增开通使用下拉选择。此入口有独立后台接口授权，不要求开放全部字典权限。
- 反馈：App 登录用户 `POST /v1/app/feedback` 接收 `content`（非空纯文字，最多 2000 字符），服务端从有效 App 会话确定用户，初始 pending；`GET /v1/app/feedback` 分页读取本人反馈，不接受其他 user_id 授权。后台 `GET /v1/sys/app/feedback` 接收 `page/page_size/user_id/status`，`PUT /feedback/:id` 接收可选 `reply/status`；回复最多 2000 字符，空字符串清空，省略/null 保持；状态仅 pending/resolved，至少修改一项。列表返回 `list/total`，每项 `id/user_id/account/nickname/content/reply/status/created_at/updated_at`，App 侧仅含本人项。首版不实现附件、工单分配或实时客服。新分页默认 1/20、最大 100，非法 ID/状态/字段长度拒绝；后台使用可信 Actor 与 Casbin，App 使用独立令牌，不互相放行。
- 后台导航：顶层“APP管理”与“系统管理”同级，下设“APP用户”（/app/users）、“会员管理”（/app/memberships）和“意见反馈”（/app/feedback）。操作码为 sys:app:user:edit、sys:app:user:review、sys:app:user:avatar、sys:app:session:revoke、sys:app:membership:grant、sys:app:membership:revoke、sys:app:feedback:reply；读取登录设备使用 sys:app:session:list，头像上传复用文件管理的 sys:file:upload。新增菜单默认只关联超级管理员；菜单可见性与服务端 Casbin 接口授权独立，沿用既有普通角色读取策略。

### 租户

- 租户路由（后台超级管理员）：`POST /v1/sys/tenant/create` 接收 slug、可选 admin_username/admin_password；`GET /v1/sys/tenant/list` 接收 page/page_size/status；`GET /v1/sys/tenant/:id`；`POST /:id/start`、`POST /:id/stop`、`DELETE /:id`（请求体 confirm_slug 必须匹配）、`POST /:id/reset-admin`。创建返回 tenant/admin_password，重置返回 new_password，仅当次响应展示；列表/详情不返回密钥。创建异步进入 creating，初始化账号与桶就绪后可先标记 admin_account_ready=true；应用就绪并装载路由后才为 active；失败显示不含敏感信息的原因。
- `slug=platform` 为平台保留标识，创建返回 400；`localhost`、`platform.localhost` 保留给平台。平台默认网络专属别名 `platform-api` 与边缘网络上的租户别名 `<slug>-api` 分开寻址。
- 租户导航与 APP 管理同级；列表复用 `GET /v1/sys/tenant/list` 的状态筛选，详情复用 `GET /v1/sys/tenant/:id`，详情抽屉只读。公开初始化库提供完整菜单及创建/启停/重置/删除操作码，默认关联超级管理员，租户登记初始为空。

### AI HTTP 与执行约定

统一前缀 `/v1/sys/ai`，仅后台认证用户可访问；Actor 由服务端认证上下文生成。会话、工作流和任务仅其所属用户可读写，越权 ID 与不存在均返回 404。列表接收 `page/page_size`，默认 1/20，page_size 最大 100，响应 `list/total`。

- ✅ 一级“AI 功能”与租户管理同级，对话页 `/ai/chat`（`ai_chat`）显示本人会话、消息历史、发送状态、工具调用记录和已知 token 用量。
- ✅ 页面操作码：查看会话 `sys:ai:conversation:list`（含详情）、新建 `sys:ai:conversation:create`、删除 `sys:ai:conversation:delete`、发送 `sys:ai:message:send`。初始化仅关联超级管理员；菜单/按钮码不替代服务端 Casbin 路由权限，普通角色需要分别获得对应授权。
- ✅ 常用流程 `/ai/workflows`（`ai_workflow`）、运行记录 `/ai/tasks`（`ai_task`）、只读工具目录 `/ai/tools`（`ai_tool`）均在 AI 菜单下。操作码为 `sys:ai:workflow:list/create/edit/delete/run`、`sys:ai:task:list/approve/reject/cancel`、`sys:ai:tool:list`（末段各自独立）。初始化仅授予超管；路由权限和本人归属独立校验，已有库须通过菜单和角色授权入口补齐。
- ✅ 知识库详情 `/ai/knowledge/:id` 是隐藏子页，沿用 `ai_kb` 权限；资料管理、检索和就地分析共用该工作区。工作流和任务的用户可见名称为“常用流程”“运行记录”，菜单标识、URL 与操作码稳定。
- ✅ 工作流、任务、知识库和资料列表支持可选 `query`，去首尾空白、最多 128 字符，LIKE 通配符按文字匹配。查找先限本人有效资源，再筛选与分页；任务按标题、流程名快照或问题摘要匹配。文档另支持 `status=pending|ready|failed`。
- ✅ 能力状态：`GET /tools`、`GET /kbs`和`GET /kbs/:id`响应新增 `capabilities={retrieval_configured,analysis_configured,ingestion_mode}`，最后固定 `background`。布尔值表示服务已配置对应客户端，不等于供应商实时健康；不暴露密钥/地址。知识库Info和文档列表响应新增 `ready_count/pending_count/failed_count`，是该库未筛选有效文档计数，不能用文档总数代替可使用数。
- ✅ 发送中阻止重复提交，切换会话后的迟到响应不能覆盖当前会话。安全展示消息和工具参数，工具调用记录与已知 token 只反映已记录事实。

| 方法与路径 | 请求字段 | result 字段 |
|---|---|---|
| POST `/conversations` | title（可选，最多 128 字符） | conversation |
| GET `/conversations` | 分页、query（可选，最多128字符，标题或正文包含查询） | list、total，置顶优先 |
| PUT `/conversations/:id` | title/pinned（均可省略，至少一项，title非空≤128字符） | conversation |
| PUT `/conversations/:id/messages/:message_id/feedback` | feedback（helpful/irrelevant_source/missing_knowledge，空串清除）、note（≤1000 UTF-8字节） | `{}`，仅本人成功的助手消息 |
| GET `/conversations/:id` | id、before_id（0/省略取最新页）、page_size（0/省略默认20，最大20） | conversation、messages、has_more、next_before_id |
| DELETE `/conversations/:id` | id | `{}` |
| POST `/messages/stream` | request_id（规范 UUID，逻辑提交重试复用）、conversation_id（0 自动新建）、content（非空，最多 16000 字节）、kb_id（可选，0=不限定）、grounded（可选，true 要求 kb_id>0 且问题≤2000 UTF-8 字节）、doc_id（可选，正值要求 grounded=true） | SSE。done.result 含 conversation_id、conversation、user_message、ai_message。消息含实际来源、反馈及已知的请求范围。 |
| GET `/tools` | 无 | list：name、description、parameters_schema；capabilities |

### 会话、严格资料模式与流式协议

会话消息只读取本会话及本人数据。AI 按 YAML 中的 Provider 选择 `AI_<PROVIDER>_KEY`（如 `AI_DEEPSEEK_KEY` 对应 `AI.DeepSeekKey`），该 Key 去除首尾空白后非空即启用，未配置或为空白则关闭；YAML 的 Enabled 字段保留兼容，运行时启用状态以 Key 为准。模型、地址和超时继续使用 YAML。关闭时发送消息返回 503，不产生伪造回答；会话管理、工具列表和本地工作流仍可用。密钥不回显、不记录。真实模型长调用超时、供应商持续中断及付费调用中途重启仍待专项验证。

- ✅ 会话详情按消息 ID 游标分批读取，每页时间正序，响应同时受字节预算约束，低于默认 4 MiB RPC 接收上限。只读取本人未删除会话；单条历史异常数据不得导致游标无限循环。
- ✅ 消息增加 `status`（running/succeeded/failed）与安全 `error`；历史消息视为 succeeded。发送前持久化 running 消息对，逐轮保存已知累计 token，完成后原子更新两行终态；失败助手正文为空，HTTP 仍返回业务失败。模型历史只使用 succeeded 的 user/assistant 消息。
- ✅ `request_id` 在用户内幂等。原请求 ID 对应的内容、会话或范围不一致返回 400；成功重查返回原消息对，失败返回原安全错误，running 返回处理中。中断保留 incomplete；外部调用成功而未落库时可能用量未知，禁止自动重放模型或工具。用户主动再次调用使用新 ID。
- ✅ 会话整理及反馈：会话增加pinned，标题/正文查询仅本人数据；重命名/置顶复用新建会话操作权限，反馈复用发送权限且仅更新本人成功助手消息。反馈和说明原子保存，保留原调用元数据及用量；用于真实问题收集，尚不自动改检索参数。导出允许明确选择已加载或完整会话，完整导出逐页受控读取并支持取消，保持认证/部门隔离。
- ✅ 对话知识库：指定kb_id必须先校验本人归属；工具执行强制该范围，模型不能覆盖。grounded=true先检索，无片段直接说明资料不足且不调用聊天模型，有片段依据资料约束生成；检索故障不得降级成未标明的模型常识。消息sources为实际提供给模型的KbSearchHit列表，兼容旧调用元数据；来源不由模型伪造。kb_id/grounded纳入请求幂等指纹。来源标题与片段安全展示，不把相似度当正确率；有片段不保证含答案，生成质量仍需评测。
- ✅ 单篇追问通过发送接口传 `doc_id`，省略或 0 使用原范围；正值须配合 `kb_id>0` 与 `grounded=true`。服务端先校验本人、所属库及可检索状态，在候选截断前限定该文档，读正文时再次复核。指定资料未就绪返回 409；同篇追问只用该文档上下文；无命中明确依据不足，只有用户显式切换后才扩大至全库。重查保留原范围。
- ✅ 消息来源可含检索时的 `doc_version`。点击已保存回答中的有效 `[n]` 或来源入口时按当前权限读取原文，唯一定位摘录才滚动高亮；版本变化提示，重复匹配、摘录变化、无权或无法读取时保留历史摘录。旧消息可继续查看。
- ✅ SSE 终帧与会话历史保留已知的 `kb_id/grounded/doc_id`，零命中仍可恢复原范围；旧消息缺失的值保持未知。跨标签重开会话时，已发送的问题以最新服务端消息的范围为准，本标签为下一问显式选择的范围继续保留。旧版本地记录无法区分这两种情况时，若与服务端范围冲突，由用户选择后再发送。
- ✅ 知识库页允许显式查找原文或提交一次带引用分析；从库进入对话会预选该库。按登录会话、部门和对话保存选库及严格模式。权限撤销时暂停发送，只有显式切换才放宽范围；会话存储不可用时选择仅保留在当页。打开页面、保存流程、重查未知请求均不自动调用模型。
- ✅ 严格资料模式遇到明确的指代词时，可将同会话、同库且有来源的上一组成功问答附入本轮检索。追加内容最多含 320 字节旧问题和 640 字节旧回答，合计问题仍受 2,000 字节限制。仅将重新检索的片段用作依据。一次回答内跨轮检索的同一库与分块沿用首次返回的摘录和编号。工具结果的 `index` 对应消息 `sources` 的一基位置，历史重查恢复同一快照。无命中可返回资料不足，检索故障返回流式错误。集成覆盖使用隔离存储与本机模型桩。

✅ 发送与重查统一使用 `POST /v1/sys/ai/messages/stream`，无普通 `/messages` 或 `SendMessage` RPC 兼容别名。客户端必须携带 `Accept: text/event-stream`，服务端以 SSE 响应；重查提交原始请求 ID、会话、内容及范围。

| event | data 字段 | 说明 |
|---|---|---|
| `meta` | conversation_id、user_message_id、ai_message_id | 首帧，消息对预留成功后立即下发 |
| `delta` | content | 助手正文增量，仅最终回复轮流式；工具调用轮不下发参数片段 |
| `tool` | name、status（start/done） | 工具调用提示，不代表执行结果或费用 |
| `done` | tokens、usage_complete、result | ✅ 终帧；result 包含 conversation_id、conversation、user_message、ai_message，消息 status 均为 succeeded |
| `error` | code、msg | 业务失败终帧，语义同统一信封 |

- ✅ 失败语义不变：error 后消息对终态 failed、助手正文为空；流式已展示的部分内容**不落库、不构成最终回复**，客户端收到 error 后应以服务端状态为准重新拉取。
- ✅ 权威结果：done 只在消息对终态成功提交后下发；result 中消息正文、工具记录、时间和用量来自本次已保存结果。conversation 为当前会话摘要，执行期间被删除时为 null，客户端不得据此恢复已删除会话。成功重放返回相同消息对和当前有效会话摘要。前端据此精确回填 Query 缓存，保留历史消息和游标，正常成功路径不再重查全部历史页或立即刷新列表；断流、失败或无法确定缓存边界时保留读取兜底。跨标签新增消息超过最新页覆盖范围时重建最新页及服务端游标，历史可继续加载，不能伪装连续完整缓存。
- ✅ 流中断后不续传，按原始请求重查。服务端每 15 秒发送 `: ping` 注释心跳，成功时补发完整结果。
- ✅ SSE 框架接入：仅发送路由启用 go-zero `WithSSE()`，由框架设置 SSE 响应头并清除连接写入期限；普通 AI 接口继续返回 JSON。开流前的解析、校验及建立 RPC 流失败仍走统一 JSON 错误信封；开流后的写入或 Flush 失败立即结束连接，客户端按原请求重查。客户端继续携带 `Accept: text/event-stream`，事件字段、心跳与结果回填协议不变。
- ✅ 单一发送流程沿用权限与字节限制；AI.TimeoutSeconds（1..90 秒）作为历史读取、所有模型轮次及工具执行的整体业务截止时间，检查点与终态清理另有有界窗口。HTTP 层对 SSE 旁路超时，业务截止时间由 AI RPC 显式控制。

下列兼容行为沿用原有 🚧 状态，真实联调覆盖仍待逐项确认。
- 🚧 多轮历史按 `AI.MaxContextTokens`（默认 32000 tokens）裁剪。
- 🚧 `text_stats` 返回 `{"chars":N,"words":N,"lines":N}`，工具目录另有 `json_formatter`。
- 🚧 工作流进度：任务 `result` 每步前缀 `[当前步/总步数]` 标识执行进度。
- 🚧 对话导出：前端支持导出 Markdown / JSON 格式（浏览器下载）。
- 🚧 新建会话：标题自动取首条消息前 64 字符。

### 工作流执行定义与运行记录

`definition` 为 `{"steps":[{"name":"步骤名","tool":"text_stats","input":{"text":"示例"},"approval":true}]}`；1..32 步，工具仅使用服务端已有注册表，步骤 input 为对象。name 最多 128 字符，enabled 为 0/1；当前只提供手动执行，新增或显式更新 interval_seconds 非 0 返回 400，历史非 0 值保留但不调度。

| 方法与路径 | 请求字段 | result 字段 |
|---|---|---|
| POST `/workflows` | name、definition（JSON 字符串）、interval_seconds（默认 0） | workflow |
| GET `/workflows` | 分页、query（可选，最多128字符） | list、total；每项含step_count，definition为空串 |
| GET `/workflows/:id` | id | workflow |
| PUT `/workflows/:id` | id、expected_revision（必填≥1），name/definition/interval_seconds/enabled 均可省略；null 与省略不修改 | `{workflow}`；版本冲突409 |
| DELETE `/workflows/:id` | id | `{}` |
| POST `/workflows/:id/run` | id、request_id（必填UUID，同一逻辑运行复用）、input（可选JSON对象字符串，≤16000 UTF-8字节） | task_id、status，后台执行 |
| GET `/tasks` | 分页、status/workflow_id/query（可选，query最多128字符） | list、total；摘要投影，input/result/definition为空串、steps=[] |
| GET `/tasks/:id` | id、meta_only（可选，默认false） | task；meta_only=true时input/result/definition为空串、steps=[] |
| POST `/tasks/:id/approve` | id、expected_step（必填，0有效，来自详情 next_step） | `{}`，执行状态通过详情查询 |
| POST `/tasks/:id/reject` | id、expected_step（必填，0有效）、reason（非空，最多 512 字符） | `{}` |
| POST `/tasks/:id/cancel` | id | `{}`，本人非终态任务取消；操作码sys:ai:task:cancel |

- ✅ 工作流列表含 `step_count`、`definition:""`，编辑和运行前从详情读取定义。任务持久保存 `title/workflow_name/input_summary`；详情含本次 `definition` 执行快照，列表及元信息只读摘要。缺失的历史名称或定义使用历史标识。
- ✅ `POST /tasks/run`：接收 `request_id`（规范UUID）、`name`（1..128字符）、`definition`（有效顺序DSL，≤65536 UTF-8字节）、`input`（可选JSON对象字符串，≤16000 UTF-8字节）；用于一次性分析和未保存试运行，不创建工作流，返回 `task_id/status`。复用运行操作码 `sys:ai:workflow:run`，仍需服务端路由授权；任务归属、工具参数校验、快照、审批和取消与已保存流程相同。`workflow_id=0` 标识临时运行；同一用户request_id不得跨草稿/保存流程或不同规范化载荷复用，冲突400。重查已接受请求优先于并发容量检查，不重复模型调用。
- ✅ `GET /tasks/:id?meta_only=true` 返回状态、摘要和更新时间，`input/result/definition` 为空字符串，`steps=[]`。状态变化后读取完整结果；可见列表与详情协调刷新，失败可重连但不重提任务。
- ✅ 顺序编辑器新建默认启用，启停通过编辑保存。运行使用已保存的定义和本次输入；审批与拒绝使用详情的 `next_step`，提交期间阻止重复操作。创建结果未知先刷新核对，运行重查沿用原载荷和 ID。无任务查看权限时只展示返回的任务 ID、状态。工具目录提供用途、示例和编辑器入口，保持只读。
- ✅ 常用流程新建页 `/ai/workflows/new`、编辑页 `/ai/workflows/:id/edit` 继承父菜单并检查创建、编辑和工具查看权限。支持知识库分析、文本处理及空白模板，基础参数使用表单，复杂 Schema 可用高级 JSON，编辑时保留旧参数。
- ✅ 当前标签页草稿按会话 ID、所选部门和工作流 ID 隔离，刷新或权限重新加载时可恢复；不能以刷新会变化的版本号作为持久键。保存成功或明确丢弃后清除草稿，其他离开先确认；存储失败明确提示。请求发出前记录提交中，刷新或网络中断后视为结果未知，禁止自动重发创建或运行请求。版本前置条件见本节。
- ✅ 普通保存返回列表；有运行权限可选择“保存并准备运行”。确认页重读已启用的流程，显式开始才执行。运行记录 URL 保留 `task_id`，详情校验本人归属。此交互的浏览器验证使用本地接口桩。
- ✅ 默认知识库模板只有分析步骤，`kb_id/question` 可引用本次 input；JSON 模板用于格式整理。旧静态 DSL、多步骤流程仍可用。运行时核对本人知识库归属，总结和风险分析只覆盖检索到的片段。
- ✅ 工作流数据流：保留旧静态DSL；version:2中步骤带稳定id，参数字段可为单键对象{"$ref":"input.question"}、{"$ref":"input.text"}、{"$ref":"steps.<id>"}或其JSON字段路径。仅可引用前序已提交步骤，禁止表达式/脚本/原型属性；解析后按工具Schema校验类型、深度和大小，不能由前端直接提供已解析参数。审批展示本次任务快照和真实解析输入，工作流后续修改不能影响。
- ✅ 工作流版本与恢复：工作流信息新增正整数`revision`（初始1，单调递增）。PUT `/v1/sys/ai/workflows/:id` 必填`expected_revision>=1`，其他patch字段保持省略语义，成功返回`{workflow}`；持锁核对版本，过期返回409并保留用户草稿，同值/空patch核对后不增版本。变更在同一事务归档旧完整快照并递增版本，恢复亦递增，防止A→B→A的旧请求重放。
- ✅ 工作流历史：GET `/v1/sys/ai/workflows/:id/revisions` 以page/page_size分页返回`list/total`；摘要含`id,workflow_id,revision,name,definition:"",interval_seconds,enabled,step_count,created_at`（归档Unix秒）。GET `/v1/sys/ai/workflows/:id/revisions/:revision` 返回`{revision:完整历史项}`；POST 同路径加`/restore`，请求`{expected_revision}`，成功返回`{workflow}`。归属、当前工作流有效性及历史版本均由服务端核对，不存在/越权/删除均404；历史读取沿用`sys:ai:workflow:list`，恢复沿用`sys:ai:workflow:edit`且独立路由鉴权。恢复不执行任何任务、审批或工具。旧草稿缺基线版本时需先核对，不自动替用户提升版本；服务端历史不替代任务原快照。
- 任务保存 `created_by/workflow_snapshot/next_step`。审批在对应步骤执行前暂停（含最后一步）；原子领取后从该步继续，并发或重放不得重复执行。后续流程修改不影响快照，工具失败置 failed，缺快照的旧任务不能续跑，未知归属任务不对普通请求开放。进程中断不自动重放副作用步骤，当前无定时调度和跨人审批。
- ✅ 任务详情返回 `next_step`；批准、拒绝必须携带客户端读取的 `expected_step` 并原子比较，缺失返回 400。服务端不能以重新读取的新断点替代请求断点；迟到或重放请求不能改变下一审批点。
- ✅ 工作流后台任务：request_id在用户内唯一，同ID相同工作流/input返回原任务，不同载荷400；创建/批准返回后由后台执行，每次最多90秒、每实例8并发。任务状态增加cancelled，取消通过原子状态转换阻止后续步骤与迟到结果提交，不承诺撤销已发生的外部操作。进程中断或结果未知不自动重放工具；详情读取时超过100秒的running标记未知失败，不提供持久队列恢复、定时调度或未经保证的整任务安全重试。新增取消操作默认仅授予超级管理员，服务端独立Casbin及本人归属校验。
- ✅ 任务步骤记录：task新增steps/updated_at/request_id；steps包含index（0基）、id/name/tool、input/output（字符串）、status（pending_approval/running/succeeded/failed/cancelled）、error、started_at/completed_at（Unix毫秒）。步骤记录与断点/状态原子提交，累计记录≤128KiB，累计旧式result仍≤60000字节；任务时间字段仍Unix秒，列表和详情不以最新工作流推断历史步骤。
- ✅ 步骤调用用量：steps可选`usage`对象为`{scope:"chat",prompt_tokens,completion_tokens,total_tokens,complete}`，仅尝试模型调用后存在；缺字段或`null`表示未记录。失败、截断或安全校验拒绝答案时仍保留已知非负用量，未知部分以`complete=false`明确表示；零值不代表零费用。完整性只描述供应商计量，不代表任务成功或答案正确；供应商正常完整返回且分项/总量有效一致才可标完整，SDK拒绝/断流/取消未确认终点时标不完整。答案和output仍遵守原校验，不以计量元数据代替答案。流式多轮失败不能把前一轮已知用量标为全调用完整。步骤计量与失败状态原子保存；终态取消/失败后仅允许同一步的迟到计量受控补写，不改终态、正文或断点，详情读取保持本人隔离。
- ✅ 两次运行对照：在运行记录中选择两条本人终态任务，复用原列表及详情读取，比较实际输入、状态、步骤输入/输出、已知耗时及引用来源。只读比较不提交任务、不触发模型；同一流程按稳定步骤ID配对，旧记录或不同流程按执行顺序对照并说明。使用历史保存的定义/步骤/结果，不用最新流程补造；缺失计时/用量标为未记录。文字差异不等同于质量提升或性能因果，原质量批次同条件比较保持独立。
- ✅ 任务列表只读摘要，保留既有筛选、排序和归属；`input/result/definition` 为空字符串，`steps=[]`。完整输入、输出、步骤只由本人任务详情读取。

#### 分析工具与答案结构

- ✅ 工作流专用工具 `kb_analyze`：参数仅 `{kb_id: 正整数, query: 非空白字符串且≤2000 UTF-8字节}`，以任务创建者身份先核对知识库归属，再共用实际知识库检索器；无匹配时不调用聊天模型。正常路径一次检索、一次无工具的有界模型分析，依据实际返回资料回答，资料作为不可信文本处理；分析不能使用其他知识库或将模型常识冒充资料依据。仅在知识库检索和聊天模型均配置时注册，目录和工作流可用，模型自动工具列表及聊天工具执行均排除，避免嵌套调用漏计用量。
- ✅ `kb_analyze` 将单行 JSON 保存为任务及步骤结果。字段为 `version:1,type:"kb_analysis",status:"completed"|"no_matches"|"truncated",analysis,sources:[{index,chunk_id,kb_id,doc_id,doc_title,snippet,score}],usage:{scope:"chat",prompt_tokens,completion_tokens,total_tokens,complete}`。来源取实际送模片段，usage 只计已知聊天用量，缺失时 `complete=false`。输入按序列化字节预算预留回答空间；模型输出最多 2048 tokens，工具 JSON 最多 24000 UTF-8 字节，任务累计结果最多 60000 字节。达输出上限显式标截断，其他非法或过长输出安全失败。前端展示分析、来源、已知用量及原始记录，旧任务可读原文。运行/批准只等待提交（普通请求 30 秒），后续读任务详情，未知结果不自动重试。
- ✅ 回答组织约束：`kb_analyze` 的内部模型回复采用逐问项结构，服务端校验并渲染为上述原有 `analysis` 正文，外部字段及旧任务读取方式不变。每个问项对应原 query 的非空连续文字，状态为有据或不足；有据项保留转义后的问项标签及合法引用，资料缺失判断使用不足项，原文本身载明的缺失处理规则可逐字摘录并引用。不接受独立自由尾注、重复/互含目标、未知字段、重复 JSON 键或非法编号。不足项由服务端生成范围提示；全部不足时不复述 query 中的指令标记。模型输出不合法则安全失败，不清洗坏答案、不自动追加模型调用。模型截断时，完整且合法的结构可渲染并标记 truncated，语法不完整则只显示固定的截断提示并保留实际来源与已知用量，不展示破损模型正文；完整但违反约束的 JSON 仍失败。该约束只能校验结构及显式状态，不证明所有问项已覆盖或事实必然正确，跨问项语义矛盾仍需评测与原文复核。
- 🚧 问项完整性要求覆盖直接相关的对象/产物、形式、前置条件、适用范围、禁止项、原因和不可逆后果，综合实际候选并引用对应依据。不能省略决定性限制、凭常识添问或硬编码业务词。保持同一次模型调用、原预算和背景角色；效果仍须用原失败题及场景回归验证。
- ✅ 背景、示例和转述不自动成为待答目标。仅当前置文字为未分句的转述引导、无问句或请求标记，平衡引文结束后有独立明确请求且未要求回答该引文时，用 `background_quotes` 标明背景。此时内部 `query` 可仅含实际请求，`original_query` 保留原问，整份载荷计入预算，检索和任务记录仍用原问。仅位于背景引文中的 target，无论有据或不足均拒绝。引文本身是问题、显式要求回答、同一目标也在引文外、前部另有分句、只有引文、引号不平衡或归属不明时，保留原范围及其余结构校验。校验失败不删句或自动重试，日志只记静态原因。该有限句法守卫不保证语义完整。
- ✅ 分析问项及普通AI正文共用安全Markdown展示：后端问项中转义的标点按字面显示，不重新激活代码、强调、链接或图片语法；表格仅按未转义竖线分列。代码内容保留原始反斜杠，原有HTML/图片和非HTTP链接仍仅作文字处理。
- ✅ 对话和分析支持受限 Markdown、表格、代码块及复制，工具目录提供中文用途与示例，流程可复制整个定义或其中步骤。使用原有工具注册表与授权。

### 知识库与向量检索

| 方法与路径 | 请求字段 | result 字段 |
|---|---|---|
| POST `/kbs` | name（必填，≤128）、description（可选，≤512） | kb、capabilities |
| GET `/kbs` | 分页、query（可选，最多128字符） | list、total、capabilities |
| GET `/kbs/:id` | id | kb（含 doc/chunk 及ready/pending/failed统计）、capabilities |
| POST `/kbs/:id/search` | id、query（必填，非空白，≤2000 UTF-8 字节） | list（chunk_id/kb_id/doc_id/doc_title/snippet/score）、elapsed_ms |
| PUT `/kbs/:id` | name/description 均可省略；null 与省略不修改 | `{}` |
| DELETE `/kbs/:id` | id（级联软删文档与 chunk，并删向量） | `{}` |
| POST `/kbs/:id/docs` | title（必填，≤128）、content（必填，≤64 KiB）、request_id（必填规范UUID）、batch_id（可选规范UUID，默认request_id） | doc、job_id、batch_id；后台摄取，重复创建复用现有文档及其当前状态 |
| GET `/kbs/:id/docs` | 分页、query/status（可选，query最多128字符，status为pending/ready/failed） | list、total、ready_count、pending_count、failed_count（未筛选计数） |
| GET `/kbs/:id/docs/:doc_id` | id、doc_id、meta_only（可选，默认false） | doc、content；meta_only=true时content为空，不读取正文 |
| PUT `/kbs/:id/docs/:doc_id` | title/content（必填、完整替换，空白拒绝，≤128字符/65536 UTF-8字节）、expected_version（必填≥1）、request_id（必填规范UUID）、batch_id（可选规范UUID，默认request_id） | doc、job_id、batch_id；版本冲突400，原子更新正文/分块并提交后台摄取 |
| DELETE `/kbs/:id/docs/:doc_id` | id、doc_id | `{}` |
| POST `/kbs/:id/docs/:doc_id/retry` | id、doc_id、request_id（必填规范UUID）、batch_id（可选规范UUID，默认request_id）；仅在无queued/running作业，且文档为failed或超过120秒未更新的pending时可重试，状态冲突code 400；原请求重查先于状态检查 | doc、job_id、batch_id |

- 🚧 归属与权限：知识库仅创建者可读写，越权 ID 与不存在均返回 404；Qdrant 查询强制携带 owner/kb 过滤（引擎层）叠加 MySQL 复核（事实源层）。操作码：查看 `sys:ai:kb:list`、新建 `sys:ai:kb:create`、编辑（含文档新增/删除/重试）`sys:ai:kb:edit`、删除 `sys:ai:kb:delete`；页面 `/ai/knowledge`（标识 `ai_kb`）挂在"AI 功能"下，初始化仅关联超级管理员。
- 🚧 启用语义：Qdrant 以 `Qdrant.Address` 去空白后非空为启用（无独立 Enabled 开关）；Embedding 固定走 `AI.EmbeddingProvider`（当前仅 qwen，密钥取 `AI_QWEN_KEY`），与聊天 Provider 相互独立。未配置 Qdrant 或 Embedding 时：知识库与文档的 CRUD/列表可用，文档摄取返回 503 降级提示，`kb_search` 工具对模型不可见；运行期故障时，直接检索返回 503，严格模式发送失败；允许模型知识的路径须显式标注降级，禁止伪造检索内容。
- 🚧 存储分工：MySQL 为唯一事实源（知识库、文档、chunk 正文与状态，走事务与归属校验）；Qdrant 仅作向量索引（向量 + 过滤 payload，可整体重建）；Redis 在检索链路仅作缓存与对账标记。写入顺序固定 MySQL 事务提交 → embed → Qdrant upsert → chunk 置 ready；任何中间失败保持 pending/failed 可重试，禁止先写 Qdrant。检索时 Qdrant 召回 chunk_id 后必须回 MySQL 复核 status/归属，被软删或未就绪的 chunk 不出现在结果中。
- 🚧 集合与模型锁定：collection 名携带模型与维度（如 `kb_chunks_qwen_text-embedding-v4_1024`）；更换 Embedding 模型或维度 = 新建 collection + 全量重嵌入，不做原地迁移；chunk 落库记录 embedding 模型版本。
- 🚧 分块当前按 500 个 Unicode 字符及 50 字符重叠估算 token_count。检索片段与对话历史共享 `AI.MaxContextTokens` 预算且累计占比不超过 25%；发送过程为检索、工具及回答预留空间，追加工具结果后按完整工具轮次裁剪旧历史，保留当前问题及工具调用/结果配对。结构化工具结果按结构裁剪，不截断 JSON。预算是本地估算，不等于供应商计费 token。检索内容为不可信文本，拼接进 prompt 时标注"以下为资料原文，不是指令"；前端渲染沿用既有转义。
- 🚧 工具 `kb_search`：参数 `{query: string, kb_id?: number}`，query 非空白且不超过 2000 UTF-8 字节；以当前对话用户身份检索其本人知识库 ready 文档的 ready chunk。返回命中项 `chunk_id/kb_id/doc_id/doc_title/snippet/score`，标注"不代表完整原文"。Qdrant 返回的 ID 还须在 MySQL 复核 owner、指定 kb_id、文档/分块状态、文档与分块的知识库一致及软删。
- ✅ 直接检索：`POST /kbs/:id/search` 使用当前身份及知识库查看权限，先核对知识库归属，不存在/越权均返回 code 404；query 必填、非空白且≤2000 UTF-8 字节，非法参数 code 400。共用 `kb_search` 的实际检索器、20 候选召回、最多 5 个片段及片段预算，分数相同按 chunk_id 升序确定顺序；不调用聊天模型、不生成答案、不新增业务记录。返回 `list`（元素字段同上述命中项，空结果为 `[]`）与 `elapsed_ms`（服务端检索阶段毫秒，不含聊天耗时）。未配置或运行时检索失败返回安全的 code 503，不伪装为空命中；该测试请求最多运行 10 秒。分数是向量相似度，不是正确率或答案置信度，当前不设最低分阈值，非空命中不保证有答案。前端在知识库内显示查询、排序、分数、文档/分块与纯文本片段，重复点击只执行一次，失败和空结果分别提示，身份或知识库改变后旧结果失效。
- ✅ 通过归属、ready 和软删复核的候选最多 20 个。最高 5 个中若同一文档占严格多数则保留原序；否则先取各文档最佳片段，再按高分补满 5 个，最终按得分降序、同分 `chunk_id` 升序。预算不足可缩减或停止。冷缓存正常路径只执行一次查询 Embedding、向量查询和原文复核；不拆问或追加重排模型。直接检索、聊天及工作流的 `kb_search` 共用该规则，不保证覆盖候选外证据或每个问项。
- ✅ `kb_analyze` 可从同次召回与归属、ready、软删复核取得最多 20 个真实候选。候选来源 JSON 最多 24000 UTF-8 字节且最多占上下文四分之一，另为原问题、背景角色和回答留空间。低预算减少候选，保持原问题与真实 chunk ID。一次模型分析，不增加召回、Embedding 或 SQL。
- ✅ 同次回复的全部结构和引用通过校验后，`sources` 按原候选顺序保留实际引用片段，最多 8 个，正文同步连续编号。原文中的编号保持原样。唯一引用超过 8 个安全失败；送模前保证任意 8 个来源的 JSON 不超过 10000 字节。全部不足或语法截断的固定提示可附最多 8 个实际输入片段和已知用量，区别于未调用模型的 `no_matches`。旧 5 来源结果及历史快照可读。反馈仍最多 5 条摘录，`source_index` 可取 1..8；直接检索覆盖与分析引用分别评测。
- ✅ 成功、维度正确且数值有效的查询向量可缓存 5 分钟，拒绝 JSON null 与全零方向。键隔离本人、Embedding 端点/模型/维度及精确问题摘要；同实例并发可合并，等待者分别取消，已发外呼或已写缓存不会撤销。命中省去本次 Embedding 和相应供应商账目，每次仍核对依赖、重新向量召回和 MySQL 复核；正文和结果不缓存。预算内保留原片段，预算不足可从同一 chunk 按原问题选连续 UTF-8 原文，无匹配保留前缀。新摄取跳过纯重叠尾块，历史索引不自动重建。前端保留查询和筛选状态，可显式带入问题或就地预览，付费分析须点击开始。
- ✅ 向量依赖恢复与清理：已配置Qdrant启动暂不可达时保留客户端，通过有界退避重新探测集合；不把暂态故障变成永久未配置。检索Embedding与后台作业领取前核对恢复状态，未就绪时排队作业保持queued且不调用Embedding。运行期周期对账只扫描当前集合的真实ID，以MySQL未软删库/文档及pending或ready分块核对；保留正在摄取的ID，清除无效派生索引。全量扫描或SQL核对失败时不根据不完整集合删除，删除失败由后续周期继续核对，不自动重放Embedding/模型调用。当前集合必须由本平台MySQL独占，不用于跨独立库或租户栈共享；只清孤立索引，不补向量或自动重建。启动5秒后及每轮结束5分钟后运行，单轮120秒、最多1M numeric IDs、每批500；上界不代表容量承诺，超限明确记录人工处理。 真实持续故障、百万级上限和生产集合丢失恢复仍待验收。
- 🚧 支持内联文本和前端 TXT/Markdown 导入。文件模块接入、PDF/Office/OCR、逐篇全文与全库审阅、定时调度、按成员/部门共享、业务查询工具和独立租户栈的 Qdrant 接入均未提供。

#### 资料与后台摄取

- ✅ `GET /kbs?choices_only=true` 按本人、名称查询和分页返回 `choices:[{id,name}]/total`。每个选项只含整数 `id` 和字符串 `name`，无匹配项时 `choices` 为空数组。可叠加正数 `id` 精确回显；仍核对归属与 query，匹配总数为 0 或 1。省略 `choices_only` 或传 false 使用完整 `list/total/capabilities`；负 `id` 或在完整模式传非零 `id` 返回 400。
- ✅ `GET /kbs/:id/docs` 返回当前筛选及页码的 `page_signature`。同条件传 `meta_only=true` 返回空列表、同口径签名、筛选总数与全库三类状态计数；签名或计数变化才补读当前完整页。
- ✅ 创建、更新、重试须提供规范 UUID `request_id`，可选规范 UUID `batch_id`，省略时使用 `request_id`。一个事务持久化文档/分块代次、批次和作业，返回 `{doc,job_id,batch_id}`。`pending/stage=queued` 表示已接收。复用相同标题、正文、模型的文档时写 `reused` 作业，显示实际状态，不重复 Embedding。更新核对 `expected_version`；先按原请求重查，再检查版本或状态，同 ID 不同载荷返回 400。
- ✅ 对就绪资料仅改标题且正文和 Embedding 模型相同时，事务核对归属及 `expected_version`，只更新标题与版本。完全相同的提交保持版本。两者均返回 `reused` 作业，保留 ready、分块及向量；提交后缓存失效未确认返回 503，可按原 `request_id` 核对并恢复缓存后读取原作业。历史引用保留原标题与摘录。
- ✅ 批次路径：`GET /kbs/:id/imports`接收分页并返回 `list/total`；`GET /kbs/:id/imports/:batch_id`返回 `{batch,list}`；`POST`同路径`/stop`返回同一结构。batch_id为客户端提前生成的UUID，第一文档提交时原子创建，无单独创建请求；尚无任何已接受项的批次返回404。批次Info含 `id`（UUID）、`kb_id/stopped/total/queued/running/succeeded/failed/cancelled/reused/created_at/updated_at`。job含 `id/request_id/doc_id/operation/status/error/doc/created_at/updated_at`；job状态 `queued|running|succeeded|failed|cancelled|reused`，operation为create/update/retry，doc为当前元信息或null。只允许本人且归属此库；每批最多20项。停止只取消queued和后续新入队，running继续，已接受的旧请求仍可重查。
- ✅ MySQL 摄取队列有两个 worker，单次业务截止 90 秒、租约 120 秒。未领取作业可跨重启继续；运行租约过期标记失败，提示核对，禁止自动重放付费调用。领取、换版、删除及迟到完成比较代次与租约，仅按该代 chunk ID 补偿。关闭窗口后可返回批次，结果未知复用原 `request_id/batch_id`。
- 🚧 内联资料标题最多 128 字符，正文非空且最多 64 KiB。后台作业失败终态使用独立有界清理上下文。重试原子替换分块，终态核对本次分块集合及文档、知识库存续。Embedding 的 v3/v4 模型每批最多 10 条；旧分块及已删除资源均受代次保护。
- ✅ 文档维护：doc增加version（正文修订号，初始1；替换+1，重试不变）及stage（queued/embedding/indexing/ready/failed）。列表不读取正文；meta_only详情用于固定docID状态轮询，原文预览另读正文。更新以expected_version及摄取状态/本次分块代次原子比较，旧索引不得匹配新正文，迟到结果不能覆盖新版本或恢复删除。批量导入每批≤20个TXT/Markdown，UTF-8严格解码，每文件≤64KiB，最多2并发，逐项反馈成功/失败/结果未知，未知结果禁止自动新建重发。

### 知识库分析反馈与回归

✅ 复核针对本人知识库分析。已有聊天评价沿用原接口，历史聊天评价独立保存。当前题集经 AI 辅助编制，样本有限，尚无真人签署或生产效果结论。

| 方法与路径 | 请求字段 | result 字段 |
|---|---|---|
| GET `/tasks/:id/feedback` | 本人任务 id | list（最多32条，本任务已有评价；含快照） |
| PUT `/tasks/:id/feedback` | step_index（0起）、expected_revision（新建0）、feedback、note | feedback（已提交记录） |
| GET `/kbs/:id/feedback` | page/page_size、status（pending/accepted/dismissed，省略默认pending，all取全部） | list/total；列表不返回answer/snapshot/review大字段 |
| GET `/kbs/:id/feedback/:feedback_id` | 本人知识库及反馈 id | feedback（含快照与复核结论） |
| PUT `/kbs/:id/feedback/:feedback_id/review` | expected_revision、status（accepted/dismissed）、review | feedback（已提交复核结果） |

- 评价类型 `helpful/incorrect_answer/missing_knowledge/irrelevant_source`，note最多1000 UTF-8字节。对象固定为实际 `kb_analyze` 已成功步骤，后续步骤失败不影响该步评价；未执行/失败/非分析步骤拒绝。任务输入、答案及来源取服务端步骤记录，不信任客户端上传的原答案或来源。知识库仍需存在且属于本人，软删后读取、更新及导出均404。
- 新增操作码 `sys:ai:task:feedback`（评价）与 `sys:ai:kb:review`（复核），初始化仅授予超级管理员；读操作分别沿用任务/知识库查看权限，接口独立走Casbin与RPC归属校验，不新增跨用户审核授权。
- `AiKbFeedbackInfo`：id/kb_id/task_id/step_index/feedback/note/status/revision/question/answer/snapshot/review/created_at/updated_at/reviewed_at。snapshot是JSON字符串，含version=1、步骤实际input/output、实际sources、资料在反馈采集时可核对的版本/正文SHA256及状态；不是模型调用时全库版本证明。快照创建后保持，评价修改清空旧复核并回到pending，revision递增；完全相同载荷重查不重复改版本。旧revision更新不同内容返回409，不能覆盖新复核。
- review为 `{answerable,expected_answer,required_facts,evidence,reason}`：expected_answer≤4000 UTF-8字节，required_facts最多10项、每项≤200字节，evidence最多5项 `{source_index,quote}`、source_index为1..8且必须存在于该历史快照，quote≤500字节且必须逐字属于对应历史片段，reason≤1000字节。accepted须有非空预期答案和复核说明；有据题还须有事实检查项及原文摘录，资料不足题须无事实项和摘录。dismissed须有说明。复核在同一事务中按revision比较更新，冲突409，失败不改变记录。
- 已采纳记录可在页面下载 `{version:1,type:"kb_feedback_regression",feedback:AiKbFeedbackInfo}`。页面采纳前及离线转换均要求每项事实检查词在预期答案和至少一条依据摘录中同时出现（NFKC及空白归一，不将同义词自动视作相同），事实项及相同来源/原文摘录不可重复。离线转换工具将其并入指定质量题集：显式核对KB映射、素材标题/正文SHA、真实摘录及采集状态，生成新副本与来源指纹，不覆盖旧题、不自动修改提示词或触发模型。不匹配的历史资料必须先复核，不能静默换成当前资料。人工录入仅作为复核记录，词法事实检查不代表语义准确率。

### 知识库质量验收

✅ `/ai/knowledge/:id/quality` 从本人知识库进入，可读取内置项目文档题集。题目使用后即属已用数据；`checks` 为可选中文检查项（`label,kind,required_group`，索引对应 `answer.required_groups`），沿用原评分。黄金答案和检查项不得进入被测模型请求。

| 方法及路径 | 参数 | result |
|---|---|---|
| GET `/kbs/:id/evaluation-datasets` | 本人知识库id | list：`key/name/sha256/dataset`，dataset为冻结v1题集原始UTF-8 JSON字符串 |
| POST `/kbs/:id/evaluations` | request_id、dataset_key、dataset_sha256、group_key、case_ids | evaluation：批次详情，接受后后台串行执行 |
| GET `/kbs/:id/evaluations` | page/page_size、可选status/request_id | list/total，列表省略大JSON字段 |
| GET `/kbs/:id/evaluations/:evaluation_id` | 本人知识库及批次id | evaluation：批次详情 |
| POST `/kbs/:id/evaluations/:evaluation_id/stop` | 本人知识库及批次id | evaluation：停止请求的已提交状态 |

- ✅ 题集目录先用可选 `dataset_key` 选择范围，省略取全部内置题集，未知键返回 400。`meta_only=true` 时所选项返回 `key/name/sha256` 与空串 `dataset`；省略或 false 返回同一批项的完整冻结原文字节字符串。两个参数可合用以读取单项摘要；前端按键与 SHA 校验原文，自定义题集和历史冻结快照保持原校验。
- ✅ 评测详情默认返回完整冻结记录。`progress_only=true` 读取进度与 `observations_sha256`，`snapshot/bindings/context` 为空串。可带上次服务端返回的 `known_observations_sha256`；一致时 `observations` 为空串，否则返回最新观测。非空的已知 SHA 须为 64 位小写十六进制，仅能与 `progress_only=true` 合用，非法组合返回 400。两种模式均核对本人知识库、批次归属及租约。首次、终态、主动刷新或错误恢复可读取完整详情。

读取沿用 `sys:ai:kb:list`；创建/停止使用新增 `sys:ai:kb:evaluate`（仅授权内置知识库分析评测，初始化授予超级管理员）。所有接口保持后台认证、服务端Casbin及Actor归属检查，错知识库/跨用户/不存在均404。request_id为规范UUID；每批选择1..30个同group的有效题目，重复/未知ID或非法参数400，题集SHA变化、原UUID内容变化或语料不匹配409，配置缺失或执行槽占满503。同UUID同内容返回原批次，不新启动；已接受批次重查不依赖新模型调用。每实例最多2批并行、每批串行。

`evaluation` 字段：`id,kb_id,request_id,dataset_key,dataset_sha256,group_key,status,case_count,completed_count,failed_count,stop_requested,corpus_verified,error,snapshot,bindings,context,observations,created_at,updated_at,started_at,completed_at`。status为`pending/running/succeeded/failed/cancelled/unknown`，表示执行状态，不代表答案质量。completed_count仅计成功分析，failed_count计失败/未知；列表中`snapshot/bindings/context/observations`为空字符串。时间为Unix秒，耗时为毫秒。

`snapshot` 原样保留完整冻结题集字符串，SHA针对原字节；`context`沿用现有评测的脱敏模型/Embedding/预算/TopK结构，binary_sha来自实际运行二进制。`bindings`沿用CLI的`{documents,inventory}`（只当前group/知识库），包含文档真实ID、正文SHA、版本、ready及Embedding型号；不把全文塞入绑定。`observations`为按题集顺序的有界JSON数组，元素含`case_id,request_id,task_id,status(waiting/running/succeeded/failed/unknown),stage,query,list,search_client_ms,search_server_ms,analysis_elapsed_ms,analysis?,error`；没有有效分析时不填造usage或来源。每个成功项的analysis仍使用原kb_analysis结构，单次模型调用和原预算保持。

批次清单和逐题UUID先持久化，再领取题目执行；每题复用SearchKb与独立RunDraft任务，正常包含两次逻辑检索。停止只阻止未领取项，当前项结束后停止；不取消或重放已发生的模型调用。执行失败停止后续题；未知结果保留原UUID/task_id供核对，不自动换标识重试。领取/写进度受独占lease_token和120秒租约约束，10秒心跳；进程中断或租约过期标unknown，迟到结果不能恢复终态，不自动重领付费步骤。

- ✅ 采纳反馈与自定义冻结题集：质量验收开始请求可选`dataset_snapshot`（完整原字节JSON，最多256KiB），仅`dataset_key=custom-<SHA256前32位>`时使用，`dataset_sha256`须与原字节一致；内置目录及旧请求保持原语义。自定义v1题集保留原评分字段，最多100篇资料/100题、单批仍最多30题；严格拒绝重复字段、非约定小写字段名、过深结构、无效分组/证据/检查项。自定义资料必须携带正整数`source_doc_id/source_doc_version`、`source="inline:<key>"`、`content`及匹配的全文SHA；绑定时按本人当前库内精确ID及标题/ready/正文SHA/版本核对，不因自定义内容扩大读取范围。旧内置受限文件资料保持原语义。全文只存在题集快照，不复制进bindings，不将`expected_answer_example`或检查项送给模型。预览及溯源沿用可选`expected_answer_example`与`metadata.feedback_regressions`，不增加模型评分暗示。请求幂等指纹包含自定义原字节指纹，运行后仍冻结快照并复核语料。
自定义题集按相同严格格式校验：UTF-8文件不接受BOM或无效字节，不重新排版后计算SHA。根字段为`version,description,annotation,metadata,documents,cases`，`annotation/metadata`可选且须为对象；所有层字段名使用`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`，拒绝`constructor/prototype`、重复解码键及未知业务字段。`version`写作字面量`1`，文档ID/版本和检查项索引使用十进制整数字面量；文档ID唯一，key≤128、kb≤64、标题≤512、正文≤65536 UTF-8字节。题目id≤128、kb≤64、query≤2000字节且必须归属实际资料组；正例所需事实和证据每组至少一项经NFKC/空白归一后出现在对应冻结原文中。预期答案示例只受快照总量约束，仍须非空白；不因为示例长度另行拒绝合法服务端快照。内置旧文件题集保留原入口规则。

运行前后复核实际语料正文SHA、版本、ready、Embedding与库存；变化保留已有观测但corpus_verified=false，不允许比较提升。警报是字符串筛查且需人工复核。仅同题集SHA、group/题目集合、知识库/语料及模型预算的完整运行可比较，构建变化单列；未知/执行失败不计为空召回或错误答案。
🚧 现有片段可能遗漏同篇资料的关键内容；自由文本回答的引用编号与复杂指代仍需人工复核，不能用词法检查证明全部问项完整、引用正确或生产准确率。

### 用量与额度

页面`/ai/usage`展示新账本起算后的安全计量，不从历史消息/任务数补造供应商请求或Embedding用量。权限：`sys:ai:usage:read`读取本人；`sys:ai:usage:admin`读取全栈；`sys:ai:usage:manage`修改额度。初始仅超级管理员获得新增权限，普通用户的菜单或既有操作权限不自动扩大。 页面按业务筛选独立读取汇总，展开明细后再分页。

路由前缀`/v1/sys/ai/usage`：本人GET `/summary`、`/calls`、`/policy`；管理员GET `/admin/summary`、`/admin/calls`、`/admin/policies`；PUT `/admin/policies/:user_id`修改策略。本人接口归属只取会话身份、不提供user_id筛选；管理员可按user_id筛选，0表示全部/全局默认策略。日期from_day/to_day为Asia/Shanghai的YYYY-MM-DD，默认最近7日、最多31日；分页page/page_size默认1/20、最大100。明细可按source/operation/provider/model/status/usage_state筛选。汇总/策略分别返回`{summary}`/`{policy}`，列表统一`{list,total}`。

每次真实provider HTTP尝试独立attempt_id（Embedding按实际批次计），账本不存prompt、正文、Key或headers。来源source为chat/analysis/kb_search/kb_ingest/evaluation，operation为chat/embedding；保存原request_id、source_id、step_index、batch_index与provider/model。预留、已确认dispatch及不确定调用分开：status为reserved/running/succeeded/failed/cancelled/unknown/not_sent，usage_state为unknown/partial/complete/invalid；未知tokens字段省略，零不代表免费。已取得用量在SDK、向量或答案安全校验失败时仍保留；流断/取消/HTTP失败不推定免费，计量完整性须有供应商有效字段支持，Embedding不伪造输出token。

摘要含period、recording_started_at、confirmed_requests/reserved_requests/not_sent_requests/uncertain_requests、成功/失败/取消数量、完整/部分/未知/无效计量数量、known_prompt_tokens/known_completion_tokens/known_embedding_tokens/known_total_tokens，及source/operation/provider/model分组。明细只返回上述安全字段、时间、http_status、provider_request_id及安全error_code；不回显供应商原错误正文。原消息与步骤usage仅作原结果读取，不与账本重复相加。

策略PUT必填expected_version（0仅新建），含daily_request_limit（0无限）、daily_token_warning（0关闭）、inherit_default（用户恢复默认）、global_concurrent_limit（仅user_id=0且1..64）。初始每用户默认每日1000次、全栈并发8、token提醒关闭；额度值与继承状态明确展示。MySQL统一按全局策略→用户策略顺序锁行，同时核对当天已扣额度及全栈未过期的reserved/running调用再预留，Redis维护快速活跃租约；not_sent释放预留，崩溃unknown保守保留额度但不冒充已确认调用。Redis状态丢失不能绕过日额度或最终全局准入，活跃显示以可核对的MySQL状态/租约为准；并发仅覆盖本独立栈及共享同一MySQL/Redis的实例，不声称跨独立租户栈。

策略读返回quota_day、有效daily_request_limit、used_requests（包含预留/未知）、remaining_requests、unlimited/inherited、daily_token_warning、known_tokens、warning_reached、incomplete_requests、version；管理员另含global active_calls/limit/active_known。token仅按已知值提醒，不宣称金额或未知消耗硬封顶，不发邮件。准入拒绝返回业务code429（普通HTTP信封语义保持），账本/DB/Redis不可用返回503并不发起新provider请求；策略版本冲突409。SSE 开流后使用 `error` 事件的 `code/msg`，禁止自动重试被拒调用。

### 链路限制

- 上传链路 Nginx 接受最大 14 MiB JSON 请求体，原文件限制仍为 10 MiB。代理读取超时 125 秒；AI RPC 上限 120 秒，业务模型调用上限仍为 90 秒。App 邮件 HTTP/RPC 客户端/RPC 服务端上限分别为 20/15/12 秒，SMTP 内部上限 10 秒。 服务预算只描述配置，真实长调用超时及持续外部故障的恢复能力仍待验收。
- ✅ 上传时间预算：对象存储30秒、file-rpc服务端35秒、API的FileRpc客户端40秒、仅文件上传HTTP45秒、前端上传请求50秒；宿主/Compose及新建租户模板同步，已有租户环境快照需另行迁移。其他文件路由沿用原超时；取消或超时不保证存储操作回滚，仍由既有引用/补偿机制核对。
- ✅ 客户端地址与可信代理：API的`TrustedProxies`为显式IP/CIDR列表，默认空，不信任任意客户端传入的来源头。仅TCP对端命中时解析X-Forwarded-For，自右向左跳过可信代理，取首个不可信地址；无此头时可使用合法单IP的X-Real-IP，坏链回退TCP对端。Compose为Nginx配置明确地址与对应最小信任范围，宿主开发默认不信任代理；更换网络时需同步覆盖地址和信任配置。实际来源用于审计及IP配额，不能直接信任最左值。

维护入口见[根规范](../AGENTS.md)、[后端规范](../go-zero-admin/AGENTS.md)和[前端规范](../go-zero-admin-react/AGENTS.md)。
