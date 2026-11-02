# API 契约（前后端共同遵守）

> 本文件是前后端唯一契约事实源。改契约 = 先改本文档 + 同步两端实现。
> 状态标记：✅ 两端已实现并联调通过；🚧 契约已定、实现未完成。新增/变更条目默认 🚧，两端同步完成后改 ✅。

> **2026-10-31 当前基线：** 后端为 [`84245c4`](https://github.com/yh-zero/go-zero-admin/commit/84245c40bd1470ef0f662fdecb2d17a9e11998fb)，本轮只调整前端并进行真实 HTTP 验收，未修改后端代码。下述目标契约语义不变，当前实现及验证边界见 §8；历史回退说明不再代表当前代码状态。

## 1. 响应格式

业务接口返回 HTTP 200，业务结果由 body 区分；健康检查 `/ready` 未就绪时返回 HTTP 503：

```json
{ "code": 200, "msg": "操作成功", "result": {} }
```

| 字段 | 说明 |
|---|---|
| code | 200 成功；其余为业务错误码（见 §5）；401 表示登录过期 |
| msg | 错误提示文案（成功时固定"操作成功"） |
| result | 成功数据；无数据时为 null 或省略 |

前端约定：axios 拦截器统一剥壳；`code !== 200` 由拦截器弹 `msg` 错误并 reject，**组件内不重复弹错**；code 401 时清 token 跳登录页。

### 1.1 验证码（已升级为图形码）

`GET /sys/auth/captcha` 返回：

```json
{ "code": 200, "result": {
    "enabled": true,
    "captcha_id": "<64位hex>",
    "image_base64": "data:image/png;base64,<PNG图形>",
    "expires_at": 1793586328
} }
```

- `image_base64` 是 **data URI 格式的 PNG 图形码**（非明文文本），前端直接置 img src
- e2e 取答案（联调用）：`docker exec gozero-redis8 redis-cli GET captcha:<captcha_id>`

### 1.2 登录响应（双令牌 + 服务端会话）

`POST /sys/auth/login` 的 `result`：

```json
{ "access_token": "...", "expires_at": 1793586328,
  "refresh_token": "...", "refresh_expires_at": 1793586408,
  "session_id": "..." }
`

- `access_token` 短期；`refresh_token` 走 `POST /sys/auth/refresh`（服务端会话校验，旧令牌撤销）
- 会话管理见 `pkg/jwtx/session.go`（Redis 服务端会话，支持设备列表/踢出）

## 2. 命名与类型

| 项 | 约定 |
|---|---|
| JSON 字段 | snake_case（如 `access_token`、`dept_id`） |
| 时间 | int64 Unix 秒（如 `created_at`） |
| ID | int64 |
| 枚举 | 字符串优先（如 `visibility: "public" | "private"`）；状态类用 int + 字典 |

## 3. 鉴权

- 请求头：`Authorization: Bearer <access_token>`
- 登录：`POST /v1/sys/auth/login`；先调用 `GET /v1/sys/auth/captcha` 获取验证码开关及启用时的图片。
- 权限码格式：`sys:<模块>:<动作>`（如 `sys:user:add`），按钮级控制由后端下发，前端只做渲染判断
- 部门上下文：多部门用户可用请求头 `X-Dept-Id` 切换（可选）

### 2026-10-31 历史基线绑定说明

以下绑定在当前基线已恢复实现。路径编码详情和菜单子树删除已在本轮真实调用；事务、缓存失效及软删除保护同时有代码依据，不据此推定全部异常或并发场景已通过。

- `GET /v1/sys/dict/code/:code` 的 `code` 从路径绑定，不要求 GET JSON 请求体或额外 query。
- 文件列表的可选 `url` 字段修正描述文件标签拼写，字段名和类型不变。
- 菜单删除按 `parent_id` 确定当前节点及全部后代，在同一事务删除菜单授权关系并软删除菜单，提交后失效详情缓存；导航路径前缀相同的其他节点不受影响。已删除菜单不能被更新恢复。
- 列表与详情中的时间仍为 Unix 秒；菜单、权限、API资源、审计查询修复不改变已有字段与分页格式。
- 历史基线成功出口存在裸对象和空响应；本轮已按 §7 统一成功信封。旧行为保留为问题来源，不作为当前实现结论。

## 4. 通用结构

### 分页

请求（GET query 或 POST body）：

```json
{ "page": 1, "page_size": 20, /* 过滤字段 */ }
```

响应：

```json
{ "code": 200, "result": { "list": [], "total": 100 } }
```

### 菜单与权限码

`GET /sys/menu/my` → `result: { "list": [MenuInfo] }`（树形）
`GET /sys/menu/my/perm-codes` → `result: { "codes": ["sys:user:add", ...] }`

### 无返回体的写操作

`result` 为 `{}`，前端按成功处理，无需解析；兼容历史接口的 null 响应。

## 5. 错误码（与后端 pkg/xerr 一致）

| code | 含义 | 前端处理 |
|---|---|---|
| 200 | 成功 | 正常 |
| 400 | 参数错误 / 业务校验失败 | 弹 msg |
| 401 | 登录过期 / 未登录 | 清 token 跳登录 |
| 402 | 令牌生成失败 | 弹 msg |
| 403 | 接口权限不足 | 弹 msg |
| 404 | 资源不存在 | 弹 msg |
| 500 | 服务器内部错误 | 弹 msg |
| 503 | 就绪探针依赖不可用（HTTP 503） | 运维探针判为未就绪 |
| 505 | 数据库操作失败 | 弹 msg |
| 506 | 数据更新未生效 | 弹 msg |
| 10001-10007 | 用户/认证业务码 | 弹 msg |
| 20001-20007 | 角色/菜单/部门/字典/API 业务码 | 弹 msg |

新增错误码须同步：`go-zero-admin/pkg/xerr/errcode.go` + 本表。

## 6. 模块路由前缀

| 前缀 | 模块 | 说明 |
|---|---|---|
| /v1/sys/auth | 认证 | 验证码/登录（公开） |
| /v1/sys/user | 用户 | |
| /v1/sys/role | 角色 | |
| /v1/sys/menu | 菜单 | 含 /my、/my/perm-codes |
| /v1/sys/dept | 部门 | |
| /v1/sys/dict | 字典 | |
| /v1/sys/file | 文件 | |
| /v1/sys/audit | 审计 | |
| /v1/sys/profile | 个人中心 | |
| /v1/sys/api | API 资源 | |

## 7. 变更流程

### 2026-10-31 全清单实施约定

本节是已确认的契约，具体实现与验证状态见本节末表；尚未联调的条目不得标记为完成。历史基线及失败记录保留在 §8，不能作为本轮完成证据。

- 成功统一经 go-zero `httpx.SetOkHandler` 输出 §1 信封；无数据写操作返回 `EmptyResp`，`result` 为 `{}`。错误仍走现有 xerr 出口。
- 后台登录新增 `refresh_token`、`refresh_expires_at`、`session_id`；访问令牌包含会话标识。服务端每次校验会话及用户状态，退出、撤销后旧访问与刷新令牌失效。存量无会话令牌需要重新登录。
- 后台用户启停状态实际变化时先撤销该用户全部会话；重新启用不会恢复离线旧令牌，撤销失败则不提交状态变更。普通资料编辑和不改变状态的请求不撤销会话。刷新返回的 `expires_at` 不晚于会话绝对到期时间。
- `POST /v1/sys/auth/refresh`：请求 `refresh_token`，返回与登录相同的令牌字段；刷新令牌一次性轮换。
- `POST /v1/sys/auth/logout`：认证必需，请求 `all_devices`（默认 false）；默认撤销当前会话，true 撤销本人全部会话。
- `GET /v1/sys/auth/sessions`：返回 `list`，每项 `id/device_name/ip/user_agent/created_at/last_seen_at/expires_at/is_current`。`DELETE /v1/sys/auth/sessions/:id` 仅撤销本人会话。
- 登录验证码开关（✅ 两端已实现并联调通过）：admin-rpc 的 `Captcha.Enabled` 默认 true。`GET /v1/sys/auth/captcha` 新增 `enabled` 布尔值；启用时返回原有 `captcha_id/image_base64/expires_at`，关闭时这三项分别为空字符串、空字符串、0，不生成或保存验证码。登录请求的 `captcha_id/captcha` 可省略，但启用时服务端仍强制验证；关闭时忽略这两项，用户名、密码、用户状态和登录限流照常校验。前端仅在明确收到 `enabled:false` 时隐藏验证码，配置请求失败时不放行登录。已验证显式开关、遗漏默认值、限流及新 API/旧 RPC 兼容；浏览器完成关闭模式登录成功/失败、退出、恢复验证码与配置失败重试。
- 验证码 `image_base64` 为 PNG data URL，答案只保存在服务端，校验时原子消费；不再返回可直接解出答案的文本。
- `/ping` 返回存活状态；`/ready` 检查必要 RPC、数据库及 Redis，就绪返回 HTTP 200，未就绪 HTTP 503；不依赖可选 AI 外部模型。
- 健康检查代码使用 `health` 分组：由 `desc/admin.api` 的 `@server(group: health)` 经 `gen.ps1 api` 生成 `handler/health`、`logic/health`，业务实现补入 logic；HTTP 路径、响应字段和公开访问规则保持不变。
- 用户及个人资料更新采用部分更新：省略字段保持原值，显式空字符串/0 清空对应允许清空的字段；`role_ids` 省略保持授权，空数组清空。头像使用 `avatar_file_id`，`avatar_url` 为只读查询结果。禁止客户端传入可信操作者身份。
- 创建后台用户时 `avatar_file_id` 仅允许省略或 0，正值/负值返回 400，先创建成功再通过用户编辑设置头像。已有用户头像按稳定用户 ID 对账；本轮未验证进程崩溃和跨服务连续故障的全部恢复路径，不声称跨库原子提交。
- `GET /v1/sys/dept/my` 返回当前用户允许选择的部门 `list`。`X-Dept-Id` 必须属于服务端验证的部门身份，鉴权、context 和下游 Actor 使用同一个部门；部门移动事务维护后代路径并拒绝环。
- 极简部门管理（✅ 两端已实现并联调通过）：沿用现有部门及用户接口，不新增 HTTP/RPC 方法。`DeptInfo` 增加 `leader_user_id`、`member_count`、`members_visible`。负责人 ID 为 0 表示未关联账号，`leader` 保留历史文本及只读显示名兼容；关联账号时显示其真实姓名、昵称或用户名，并标注已停用/已删除状态，不自动授予角色。部门创建/编辑增加 `leader_user_id`，省略或 null 保持编辑原值，编辑显式 0 清空关联和历史负责人文本，正值必须为当前操作者数据范围内的启用账号，且操作者具有用户列表读取权限。创建时正负责人 ID、编辑时任意显式负责人 ID，均不能与非空历史 `leader` 同时提交；创建时省略或 0 保留旧客户端文字录入兼容。历史文本仅在未关联账号时允许维护。
- 部门直属成员复用 `GET /v1/sys/user/list?dept_id=...`，保持接口读取权限及 Actor 数据范围。部门树的 `member_count` 仅统计当前身份可查询、未删除的直属用户（包含停用用户）；有用户列表读取权限且数据范围有效时 `members_visible=true`，否则 false、人数为 0，前端显示“—”，不将不可读取解释为部门无人。`/dept/my` 不附带成员统计。搜索及状态筛选在前端保留匹配节点的祖先，不改变组织层级。
- 部门停用与删除沿用现有更新/删除接口：存在未删除子部门、归属用户或部门域成员授权时拒绝，提示先处理关联关系，不自动级联停用或转移。校验与写入在同一事务中执行；负责人关联不是成员授权，不授予部门身份。第一期仅展示直属成员，不实现兼职、专门调岗流程或独立影响预览。
- `POST /v1/sys/api/sync` 使用实际注册路由同步 API 资源，返回 `added/updated/removed`。
- API 同步采用增量 upsert：保留已有人工标题和非本服务资源，本轮 `removed` 固定为 0；新增标题采用 `METHOD /path`。无来源标记时不自动删除其他资源。
- `POST /v1/sys/file/upload` 接收 `name/content_base64/scene/visibility`，原文件最大 10 MiB；服务端检查内容、类型及场景。引用增减仅由内部业务 RPC 调用，按业务类型及 ID 幂等维护。
- 上传响应为 `{file: FileInfo}`；`name/content_base64` 必填，`scene` 为 common/avatar/doc（默认 common），`visibility` 为 public/private（默认 private），原始字节 1..10 MiB。头像限可解码的 JPEG/PNG/GIF 并检查尺寸，不引入额外图像解码依赖；拒绝路径型文件名及 HTML/SVG。服务端推断 MIME，不接受客户端 uploader_id/Actor。
- 私有文件链接、删除、引用新增验证上传者；公开文件链接允许已认证用户读取。头像引用类型为 admin_avatar/app_avatar，按 `(file_id,ref_type,ref_id)` 唯一幂等维护；删除引用不存在的明细不减少历史引用计数，删除文件同时检查计数和引用明细。
- 引用移除没有 HTTP 入口：仅由已完成业务对象授权的内部 RPC 编排，使用数据库读取的旧文件 ID 和精确业务类型/ID；不以历史上传者限制合法的后续管理员修改，也不信任客户端传入的旧文件 ID。
- 内部引用移除 RPC 的 `keep_file_id` 有字段存在性：省略时精确移除 `FileID`；传入时要求 `FileID=0`，对该业务类型/ID清理除保留文件外的全部引用，0表示全部释放。头像业务在用户行锁内新增引用，事务结束后重新锁行并按数据库当前头像对账；同一头像请求可重复对账，失败返回错误并保留引用明细用于重试。该参数不暴露给 HTTP 客户端。
- 内部文件链接 RPC 支持 `public_only`：App 自查头像固定使用该模式，只返回公开文件地址，私有文件拒绝，不将 App 用户 ID 冒充后台 Actor；后台调用保持真实 Actor 校验。该参数不暴露给 HTTP 客户端。
- 后台为 App 用户设置头像时只接受管理员本人上传的公开头像文件；新增引用验证真实后台操作者，App 自查只读取公开链接，解绑使用 `file_id:0`。
- 审计通过 RPC 写现有审计表，记录操作者、路径、结果和耗时，不记录密码、令牌及完整请求体。
- App 使用独立认证域：公开端点仅 register/login/options/email-code；refresh 校验刷新令牌；其余 App 请求校验有效 App 会话和用户当前状态。后台 App 管理、AI、租户接口仍受后台权限控制。
- App 认证（🚧）：`POST /v1/app/auth/register` 接收 account/password/nickname/email_code；当前仅开放邮箱注册，默认要求一次性邮箱验证码，注册后为 pending，后台审核 active 后才能登录。`POST /auth/login` 接收 account/password；`POST /auth/refresh` 接收 refresh_token；二者返回 access_token/expires_at/refresh_token/refresh_expires_at/session_id/user，刷新原子轮换。`POST /auth/logout` 接收 all_devices（默认 false）。App 与后台使用不同 JWT 密钥、签发者及 Redis 会话命名空间，不能交叉访问。
- `GET /v1/app/auth/options` 返回 email_verification_required/email_code_enabled/phone_enabled；`POST /auth/email-code` 接收 account。验证码使用标准库 SMTP 发送，有效期由 YAML `SMTP.ValidSeconds` 配置（默认 600 秒，必须大于 0），到期自动失效、匹配成功后原子消费，输错不删除；邮箱/IP 限流保持。`SMTP.FromName` 设置收件人看到的发件人名称，未配置时仅显示发件邮箱。缺 SMTP 配置明确返回不可用，不返回验证码或伪造发送成功。已通过本地 SMTP 桩与隔离 RPC 验证；2026-11-01 临时使用 465 + 隐式 TLS 的真实 SMTP 发信通过，TTL 符合 600 秒配置，指定收件人确认收到且显示正常。真实邮箱验证码注册仍未覆盖，正式配置同步见剩余事项。
- `GET /v1/app/profile` 返回 user；`PUT /v1/app/profile` 接收可选 nickname，省略/null 保持、空字符串清空。user 字段为 id/account/identity_kind/nickname/avatar_file_id/avatar_url/status/review_reason/email_verified/phone_verified/last_login_at/created_at。`GET /v1/app/membership` 返回当前用户有效会员 list；不接受客户端 user_id 或 access_token 字段，访问令牌只从 Bearer 头读取。AppAccess 仅内部 RPC。
- 后台 App 管理（✅ 两端已实现并联调通过）：`GET /v1/sys/app/users` 接收 page/page_size/status/account，公开头像返回 avatar_url；`POST /users/:id/review` 接收 status/reason（pending/active/disabled/rejected）；`PUT /users/:id/avatar` 接收 file_id（0 解绑）；`GET /users/:id/sessions` 返回 list，复用登录设备字段 id/device_name/ip/user_agent/created_at/last_seen_at/expires_at/is_current（管理其他 App 用户时固定 false），最多 20 条有效会话；`DELETE /users/:id/sessions/:session_id` 撤销该用户会话。所有请求使用服务端后台身份和接口权限；不能伪造 Actor，目标用户必须存在且未删除，停用/驳回撤销旧会话。仅展示安全设备字段，不返回令牌、摘要等内部会话字段。
- 后台会员（✅ 两端已实现并联调通过）：`GET /v1/sys/app/memberships` 接收 page/page_size/user_id/plan_code/active_only；`POST /memberships` 接收 user_id/plan_code/benefits/request_id/duration_days（1..3650），request_id 必填且幂等，重复键参数不一致拒绝；`DELETE /memberships/:id` 撤销。列表项为 id/user_id/plan_code/benefits/request_id/granted_by/granted_by_admin/starts_at/expires_at/revoked/revoked_at/created_at，有效期为 starts_at <= 当前时间 < expires_at 且未撤销。
- APP 通用管理五项（✅ 两端已实现并联调通过）：后台用户详情整合资料/会员/设备页签，审核与启停分入口，继续复用原审核及会话撤销规则。`GET /v1/sys/app/users/:id` 返回 `{user: AppAdminUserInfo}`；后台用户列表也改为 `AppAdminUserInfo`，在原用户字段上增加 `remark`。`PUT /users/:id` 仅允许可选 `nickname/remark`，省略/null 保持，空字符串清空，昵称最多 64 字符、备注最多 500 字符；账户、身份、审核状态由原专用接口维护。内部备注仅随后台用户响应返回，不出现在 App 注册、登录、刷新、个人资料及其他 App 侧响应中。
- `GET /v1/sys/app/membership-plans` 返回 `{list:[{label,value}]}`，仅从启用的 `app_membership_plan` 字典读取有效名称与编码，不开放任意字典读取，不新增套餐销售系统；开通字段和历史自由套餐编码兼容，后台新增开通使用下拉选择。此入口有独立后台接口授权，不要求开放全部字典权限。
- 反馈（✅ 两端已实现并联调通过）：App 登录用户 `POST /v1/app/feedback` 接收 `content`（非空纯文字，最多 2000 字符），服务端从有效 App 会话确定用户，初始 pending；`GET /v1/app/feedback` 分页读取本人反馈，不接受其他 user_id 授权。后台 `GET /v1/sys/app/feedback` 接收 `page/page_size/user_id/status`，`PUT /feedback/:id` 接收可选 `reply/status`；回复最多 2000 字符，空字符串清空，省略/null 保持；状态仅 pending/resolved，至少修改一项。列表返回 `list/total`，每项 `id/user_id/account/nickname/content/reply/status/created_at/updated_at`，App 侧仅含本人项。首版不实现附件、工单分配或实时客服。新分页默认 1/20、最大 100，非法 ID/状态/字段长度拒绝；后台使用可信 Actor 与 Casbin，App 使用独立令牌，不互相放行。
- APP 五项验收（2026-11-01）：124 项真实 HTTP 断言通过；后台浏览器验证详情、会员开通、设备读取、资料保存与清空、启停、反馈回复与状态、空结果、切换反馈草稿、窄屏及只读按钮。
- 后台导航（✅ 已实现并经浏览器验证）：顶层“APP管理”与“系统管理”同级，下设“APP用户”（/app/users）、“会员管理”（/app/memberships）和“意见反馈”（/app/feedback）。操作码为 sys:app:user:edit、sys:app:user:review、sys:app:user:avatar、sys:app:session:revoke、sys:app:membership:grant、sys:app:membership:revoke、sys:app:feedback:reply；读取登录设备使用 sys:app:session:list，头像上传复用文件管理的 sys:file:upload。新增菜单默认只关联超级管理员；菜单可见性与服务端 Casbin 接口授权独立，沿用既有普通角色读取策略。
- 租户新增 HTTP 对接现有生命周期 RPC，保留现有数据，补齐缺列及 NULL/别名扫描。AI 审批在步骤执行前暂停，批准后续跑，失败不得标成功；会话/工作流/任务均检查归属。
- 租户路由（后台超级管理员）：`POST /v1/sys/tenant/create` 接收 slug、可选 admin_username/admin_password；`GET /v1/sys/tenant/list` 接收 page/page_size/status；`GET /v1/sys/tenant/:id`；`POST /:id/start`、`POST /:id/stop`、`DELETE /:id`（请求体 confirm_slug 必须匹配）、`POST /:id/reset-admin`。创建返回 tenant/admin_password，重置返回 new_password，仅当次响应展示；列表/详情不返回密钥。创建异步进入 creating，初始账号与探针确认成功后才为 active/admin_account_ready=true；失败显示不含敏感信息的原因。
- APP 后台管理页面已补齐用户、头像、登录设备与会员管理；AI、租户仍提供后端与 HTTP 接口。当前后台保留角色分配、上传/头像、刷新/退出/设备会话、部门切换及 API 同步入口。
- 更新请求中 `null` 与省略都保持原值；用户/部门 `status` 仅为 1（启用）或 2（停用），0 不表示清空；`gender` 为 0/1/2。`dept_id:0`、`avatar_file_id:0` 表示解绑。
- `role_ids` 回读全部未删除角色 ID 的去重集合。显式列表中保留的角色维持原部门绑定，移除角色删除全部绑定，新普通角色绑定用户主部门（无部门为 `dept:0`）；更换主部门只迁移旧主部门绑定。分配角色限定超级管理员，禁止移除最后一个有效超级管理员；省略角色字段不影响授权。
- 用户主部门实际变更仅允许超级管理员，避免通过迁移部门绕过角色分配权限；传入与当前相同的 `dept_id` 不构成变更。角色 `code` 创建后不可修改，普通编辑需保留原值，否则返回 400，避免已有 Casbin 绑定失效。
- 重置密码和自助改密只更新密码列，目标必须未删除且启用；自助改密提交时再次校验旧密码哈希未变化，并发改密冲突返回旧密码错误，不覆盖头像、部门或状态。成功后撤销该用户全部会话。
- 用户管理的列表、详情、创建、编辑、删除和重置密码受数据范围约束：当前服务端验证部门内的有效角色（含全局绑定）取并集；`ALL` 为全部，`DEPT` 为当前部门，`DEPT_AND_CHILDREN` 含当前部门及有效下级，`SELF` 仅本人，全局超级管理员为全部。无有效范围拒绝访问；历史范围值去空白并忽略大小写，未知值不授予权限。`dept_id=0` 的部门范围仅代表未分配部门，不代表全部部门。客户端部门筛选只能进一步缩小列表。创建必须落在可管理部门，SELF 不允许创建他人；不存在目标保留用户不存在错误，越界目标返回 403。个人资料和认证状态读取固定本人，不以用户管理范围放行他人。
- 个人资料只读字段 `is_super_admin` 由服务端有效全局超级管理员绑定计算，供前端显示角色分配等入口，不根据用户名判断；后端仍独立执行权限校验。

#### AI HTTP 与执行约定（本地链路已验证，模型调用待配置）

统一前缀 `/v1/sys/ai`，仅后台认证用户可访问；Actor 由服务端认证上下文生成。会话、工作流和任务仅其所属用户可读写，越权 ID 与不存在均返回 404。列表接收 `page/page_size`，默认 1/20，page_size 最大 100，响应 `list/total`。

| 方法与路径 | 请求字段 | result 字段 |
|---|---|---|
| POST `/conversations` | title（可选，最多 128 字符） | conversation |
| GET `/conversations` | 分页 | list、total |
| GET `/conversations/:id` | id | conversation、messages |
| DELETE `/conversations/:id` | id | `{}` |
| POST `/messages` | conversation_id（0 自动新建）、content（非空，最多 16000 字节） | conversation_id、user_message、ai_message |
| GET `/tools` | 无 | list：name、description、parameters_schema |
| POST `/workflows` | name、definition（JSON 字符串）、interval_seconds（默认 0） | workflow |
| GET `/workflows` | 分页 | list、total |
| GET `/workflows/:id` | id | workflow |
| PUT `/workflows/:id` | id，name/definition/interval_seconds/enabled 均可省略；null 与省略不修改 | `{}` |
| DELETE `/workflows/:id` | id | `{}` |
| POST `/workflows/:id/run` | id、input（可选 JSON 字符串，保存为本次输入记录） | task_id、status |
| GET `/tasks` | 分页、status/workflow_id（可选） | list、total |
| GET `/tasks/:id` | id | task |
| POST `/tasks/:id/approve` | id | `{}`，执行状态通过详情查询 |
| POST `/tasks/:id/reject` | id、reason（非空，最多 512 字符） | `{}` |

`definition` 为 `{"steps":[{"name":"步骤名","tool":"text_stats","input":{"text":"示例"},"approval":true}]}`；1..32 步，工具仅使用服务端已有注册表，步骤 input 为对象。name 最多 128 字符，enabled 为 0/1；本轮只实现手动执行，新增或显式更新 interval_seconds 非 0 返回 400，历史非 0 值保留但不调度。

任务保存 created_by、workflow_snapshot 和 next_step。审批在对应步骤执行前暂停（包括最后一步），批准用原子状态变更领取执行权后从该步骤继续；重复或并发审批不能重复执行。工作流修改不改变已创建任务快照；工具失败标 failed，不伪装成功。旧任务缺少快照不能续跑；未知归属的历史任务不对普通请求开放。进程中断后 running 任务不自动重放有副作用步骤，本轮不提供自动恢复或定时调度。

会话消息只读取本会话及本人数据。AI 按 YAML 中的 Provider 选择 `AI_<PROVIDER>_KEY`（如 `AI_DEEPSEEK_KEY` 对应 `AI.DeepSeekKey`），该 Key 去除首尾空白后非空即启用，未配置或为空白则关闭；YAML 的 Enabled 字段保留兼容，运行时启用状态以 Key 为准。模型、地址和超时继续使用 YAML。关闭时发送消息返回 503，不产生伪造回答；会话管理、工具列表和本地工作流仍可用。密钥不回显、不记录；名称与启用规则已通过本地模型桩验证，2026-11-01 最新代码隔离 RPC 的真实 DeepSeek 单轮调用通过，未调用工具；长调用超时仍待验收。

#### 本轮实现状态（2026-10-31）

| 范围 | 状态与证据 |
|---|---|
| 后台认证、响应、审计、就绪 | ✅ 前后端已实现，认证真实回归 18 项通过；故障注入未覆盖 |
| 用户角色、部分更新、部门与数据范围 | ✅ 前后端已实现，用户/部门 17 项、数据范围与会话 35 项通过；浏览器部门切换通过 |
| 文件上传、引用与头像 | ✅ HTTP/RPC 19+16 项通过，前端入口及构建完成；🚧 浏览器实际上传受扩展文件权限限制 |
| API 同步 | ✅ HTTP 5 项通过，前端入口已接，保留真实新增 48 条路由 |
| App | ✅ 按约定交付后端 16 个 HTTP 接口，本地 31 项通过；🚧 SMTP 与邮箱注册成功链路待配置 |
| AI | ✅ 按约定交付后端 16 个 HTTP 接口，本地 33 项通过；🚧 外部模型调用待配置 |
| 租户 | ✅ 按约定交付后端 7 个 HTTP 接口，完整生命周期 24 项通过；未验证所有中断/大数据场景 |

当前生成路由共 96 个。后端完整 `verify.ps1`、前端 `tsc -b`/`vite build`/lint 均退出 0；APP管理页面已补齐，AI/租户页面未在本轮新增。剩余待办与验收边界见 [后端待办](backend-api-integration-issues.md)。

#### 复查补充约束（2026-11-01，✅ 本地定向回归完成）

- 登录签发与用户改密、启停使用同一用户行锁；签发前重验密码哈希和当前状态，改密或状态变更成功后，旧凭据产生的会话不得恢复。App 登录与审核状态变更遵守相同规则。
- 浏览器同源标签页共享会话更新和退出状态，刷新令牌轮换互斥；旧标签页不得清除另一标签页刚刷新的有效会话。
- 跨标签页轮换使用浏览器原生 Web Locks，自动刷新要求支持该能力的 HTTPS/本机地址；不支持时明确提示，不并发消耗共享刷新令牌。App 邮箱账号规范化后最长 128 字节，与现有数据库列一致，超长参数返回 400。
- AI 批准、拒绝同时绑定本次读取的 `next_step`，迟到请求不能重复执行旧步骤或改变下一审批点。
- 上传链路 Nginx 接受最大 14 MiB JSON 请求体，原文件限制仍为 10 MiB。代理读取超时 125 秒；AI RPC 上限 120 秒，业务模型调用上限仍为 90 秒。App 邮件 HTTP/RPC 客户端/RPC 服务端上限分别为 20/15/12 秒，SMTP 内部上限 10 秒。
- 租户 `slug=platform` 为平台保留标识，创建返回 400；平台 `localhost`、`platform.localhost` 域名不能作为租户路由。平台上游使用独有网络别名，避免共享网络内同名 `api` 解析到租户服务。
- 删除角色与清理授权关系在同一事务内完成，并失效该角色 ID/编码缓存；角色编辑只更新可编辑列，不可恢复已删除角色。

#### 后续契约变更流程

1. 改本文档（注明变更日期）
2. 后端：改 desc/*.api 或 proto → `gen.ps1` 重新生成 → 实现 logic
3. 前端：改 `api/types.ts` + 对应 api 模块
4. 两端各自过自检（verify.ps1 / tsc+build）

## 8. 接口实现状态矩阵

### 历史基线真实 HTTP 验证（本轮修复前）

修复前基线注册 **41 个 HTTP 路由**，前端均有对应 API 函数。历史脚本验收批次记录 **127 次 HTTP 请求，41/41 路由至少一次正常路径调用通过，9 次预期拒绝通过**。以下表格保留当时结果；本轮新增接口与修复状态见 §7，剩余事项见 [后端待办](backend-api-integration-issues.md)，不能把旧路由数量和旧问题当作当前状态。

| 范围 | 路由数 | 本轮真实调用通过的场景 | 未完成或未覆盖边界 |
|---|---:|---|---|
| 认证 `/v1/sys/auth` | 2 | 验证码、登录；缺失/非法 Bearer 返回 401 | 刷新、服务端退出和设备会话没有 HTTP 入口 |
| 用户 `/v1/sys/user` | 7 | 非空分页、空筛选、详情、增改删、重置密码、自助改密、新密码登录 | 用户角色分配未接通；无角色临时用户初次改密返回 403，补入专属测试策略后通过 |
| 部门 `/v1/sys/dept` | 4 | 非空树及临时部门增改删 | 未验证移动；现有路径维护和详情缓存问题为静态确认，前端继续禁止移动 |
| 角色 `/v1/sys/role` | 6 | 列表、详情、增改删、菜单授权及重复 ID 去重 | 详情 `user_count` 与列表不一致；普通编辑的授权事务问题仍为静态确认 |
| 菜单 `/v1/sys/menu` | 8 | 列表、详情、增改删、树的 `type/status` 筛选、当前用户菜单/权限码；父节点删除后子树及授权关系移除 | 未穷尽异常父节点、并发和缓存故障场景 |
| 字典 `/v1/sys/dict` | 5 | 列表、新增、仅路径 `code` 详情、完整编辑、删除 | 未穷尽编码冲突等异常 |
| 文件 `/v1/sys/file` | 3 | 非空列表；专用公开/私有对象取链接并下载；有引用删除拒绝；无引用删除和删除后 404 | 公开链接错误报告过期时间；上传和引用维护没有 HTTP 入口 |
| API 资源 `/v1/sys/api` | 2 | 非空筛选、删除本轮专用资源 | 无创建/同步 HTTP；删除夹具通过授权 SQL 插入 |
| 审计 `/v1/sys/audit` | 1 | 非空分页、时间字段、`result=2` 筛选 | 未验证所有筛选组合及审计写入链路 |
| 个人资料 `/v1/sys/profile/` | 2 | 临时账号读取、完整保存并保留非零性别 | 仅提交昵称会清空其他字段；头像未接通 |
| `/ping` | 1 | HTTP 可达 | logic 仍为空实现，不能代表数据库、RPC 或存储健康 |
| **合计** | **41** | **全部有正常路径真实调用** | **残留偏差不因路由通过而视为修复** |

9 次预期拒绝覆盖缺失/非法 token 401、受限账号访问用户列表 403、错误旧密码 10007、引用文件删除 400、已删除文件链接 404。初次无角色账号改密的非预期 403 保留在原始结果，未删除失败记录或改写为通过。

### 历史基线的实现能力缺口（修复状态见 §7）

| 项目 | 当前事实与前端边界 |
|---|---|
| 统一响应与错误出口 | 成功裸响应/空响应仍存在；框架参数解析返回 HTTP 400 的分支为静态确认。代码还存在 §5 未列的 402、404、505、506，不能宣称该表已完整反映运行行为；本轮未新增或重定义错误码。 |
| 部门上下文 | `X-Dept-Id` 改变 Casbin domain，但注入 context 的部门仍取 JWT；缺少当前用户部门清单 HTTP。前端不据此伪造部门切换能力。 |
| App / AI / 租户 | 只有 RPC 声明和实现，没有注册 HTTP 路由；本轮未复验 RPC 或创建可操作的 REST 页面。 |
| 文件上传、引用、头像 | 上传/引用维护仅 RPC；资料 HTTP 接受的 `avatar_url` 未转成文件 ID。前端不模拟上传或头像保存成功。 |
| API 同步、用户角色分配、认证扩展 | API 同步仅 RPC 且为占位实现；用户角色字段未进入 HTTP 且 RPC 写入未处理；认证刷新/退出/设备会话没有 HTTP。 |

剩余事项见 [后端待办](backend-api-integration-issues.md)，历史问题与修复证据见该文件中的 Git 记录链接。本机脱敏证据位于 `%LOCALAPPDATA%/go-zero-admin/frontend-integration-20261031-152116/`：`http-results.json`、`http-coverage.json`、`http-coverage.md`、`temporary-records.json`。浏览器与重启验收单独记录在 [本轮计划](frontend-integration-plan.md)，不从 HTTP 结果推定页面全部通过。

较早的历史基线见 [84245c4 基线记录](https://github.com/yh-zero/go-zero-admin/blob/84245c40bd1470ef0f662fdecb2d17a9e11998fb/ai_document/backend-api-integration-issues.md)，后续修复记录见后端待办中的 Git 链接；不把已修复的旧问题重新列为本轮阻塞。

**最近更新：2026-11-01**
