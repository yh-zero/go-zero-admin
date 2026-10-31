# 本轮后端对接问题（2026-10-31）

本文件按用户要求清空后重新记录，仅包含本轮发现。后端基线为 [`84245c4`](https://github.com/yh-zero/go-zero-admin/commit/84245c40bd1470ef0f662fdecb2d17a9e11998fb)；历史问题及修复见 [该提交中的原记录](https://github.com/yh-zero/go-zero-admin/blob/84245c40bd1470ef0f662fdecb2d17a9e11998fb/ai_document/backend-api-integration-issues.md)，不恢复历史问题正文。本轮没有修改后端源文件、配置、表结构、生成物或依赖。

## 1. 验证范围

当前注册 **41 个 HTTP 路由，全部至少一次正常路径真实调用通过**；本轮脚本验收批次共记录 **127 次 HTTP 请求、9 次预期拒绝**。权限使用真实 Bearer 和 Casbin，未替换权限响应。该结果不代表全部字段、异常、并发或权限组合已验收，也不代表下述问题已修复。

预期拒绝包括缺失/非法 token 401、受限账号访问用户列表 403、错误旧密码 10007、引用文件删除 400、已删除文件链接 404。临时账号首次自助改密因无角色返回 403 的原始失败仍保留；按授权补入该临时主体专属最小策略后，改密及资料链路通过。该测试授权不等于产品已提供用户角色分配功能。

## 2. 已真实复现

| 编号 | 路由与输入条件 | 实际结果与前端影响 | 代码依据及本轮处理 |
|---|---|---|---|
| B01 | `PUT /v1/sys/profile/`；临时账号原 `gender=2` 且姓名、邮箱非空，仅发送 `nickname` | 写操作 HTTP 200；随后读取 `gender=0`、姓名和邮箱为空。可选字段被按零值覆盖，部分保存会丢资料。 | [Profile RPC](../go-zero-admin/application/admin/rpc/internal/logic/profileservice/update_profile_logic.go) 直接覆盖文本与性别；API 已转发 Gender，旧“丢弃性别”问题没有恢复。前端完整提交全部可编辑字段；探测后已恢复临时账号完整资料。 |
| B02 | 同一临时角色，`GET /v1/sys/role/list` 与 `GET /v1/sys/role/9010002` | 均 HTTP 200；列表 `user_count=1`，详情 `user_count=0`。详情默认值不能表示真实人数。 | [详情逻辑](../go-zero-admin/application/admin/rpc/internal/logic/roleservice/get_role_logic.go) 没有填人数，[列表逻辑](../go-zero-admin/application/admin/rpc/internal/logic/roleservice/get_role_list_logic.go) 才统计；前端角色详情不展示该默认值。测试角色已清理。 |
| B03 | `GET /v1/sys/file/9001905/url`；专用公开对象 | HTTP 200，返回可下载的无签名公开 URL，但 `expires_at` 为未来时间。直接展示会把公开链接误报为临时链接。 | [GetFileURL](../go-zero-admin/application/file/rpc/internal/logic/fileservice/get_file_u_r_l_logic.go) 自行计算过期时间；[存储层](../go-zero-admin/pkg/objectstorage/storage.go) 公开 URL 无签名期限。前端按 `visibility` 展示公开链接长期有效。测试对象及元数据已清理。 |

本轮私有文件的签名期限与返回期限未发现不一致，不将其他配置下的可能偏差写成已复现。

运维实测：并发执行 `docker compose restart` 重启全部 11 个服务时，Nginx 因 `host not found in upstream "api" in /etc/nginx/nginx.conf:16` 退出 1；API/RPC 稳定后执行 `docker compose start nginx` 恢复，11 个容器均 running，`7001/ping`、`5999/`（前端根页面）、`80/ping` 均返回 200。属于重启依赖就绪时序问题，本轮未修改配置，不能泛化为服务无法启动。

## 3. 静态确认的残留限制

下表依据当前代码，不作为本轮已经触发的故障。普通增改删或正常路径通过不能代替这些限制的修复。

| 范围 | 代码确认与影响 | 前端边界 / 未覆盖项 |
|---|---|---|
| 部门移动与缓存 | [更新](../go-zero-admin/application/admin/rpc/internal/logic/deptservice/update_dept_logic.go) 缓存读取后裸 SQL 改 `parent_id`，未维护自身/后代 `path` 或校验循环；[删除](../go-zero-admin/application/admin/rpc/internal/logic/deptservice/delete_dept_logic.go) 同样未失效详情缓存。 | 本轮仅验证不移动的增改删；前端保持禁止调整上级部门。 |
| 普通角色编辑的授权写入 | [UpdateRole](../go-zero-admin/application/admin/rpc/internal/logic/roleservice/update_role_logic.go) 在改角色后删除并逐条重建授权，未包事务且忽略关联 SQL 错误；缺省 `menu_ids` 会清空授权。[CreateRole](../go-zero-admin/application/admin/rpc/internal/logic/roleservice/create_role_logic.go) 同样忽略授权插入错误。 | 编辑时保留完整授权 ID；本轮未注入数据库故障。独立 `POST /:id/menus` 已有事务和去重，并已实测通过，不能与此问题混同。 |
| 用户部分更新与清除关联 | [UpdateUser](../go-zero-admin/application/admin/rpc/internal/logic/userservice/update_user_logic.go) 全量覆盖文本、性别、状态；部门和头像仅在 ID 大于 0 时更新，传 0 不能清除既有绑定。 | 前端完整提交资料并保留关联；本轮未宣称“清空部门/头像”可用。 |
| 统一响应与错误码 | 成功 handler 直接 `OkJsonCtx` 或 `Ok`，尚未统一成功信封；[xerr.Handler](../go-zero-admin/pkg/xerr/handler.go) 的框架解析错误返回 HTTP 400，普通 gRPC 错误也可能落为 gRPC 数字码。[errcode.go](../go-zero-admin/pkg/xerr/errcode.go) 另有 402、505、506，文件逻辑使用 404，目标契约表尚未完整列出。 | 前端兼容裸成功、空响应、null 列表和错误信封；本轮未重定义目标响应或错误码。框架解析与普通 gRPC 退化分支未作为本轮故障触发。 |
| 部门上下文 | [Authority](../go-zero-admin/application/admin/api/internal/middleware/authority.go) 的 `X-Dept-Id` 只改变 Casbin domain，写入 context 的部门仍来自 JWT。 | 缺少当前用户部门清单 HTTP，前端不开放伪造的部门切换；跨部门一致性未实测。 |
| 租户表结构与查询 | 本轮只读查询 `information_schema.COLUMNS` 确认实际 `gozero_platform.tenant_registry` 缺少 `admin_account_ready`；[来源迁移](../go-zero-admin/data/db/migrations/platform/00001_tenant_registry.sql) 也未定义该列，但 [列表](../go-zero-admin/application/tenant/rpc/internal/logic/tenantservice/get_tenant_list_logic.go) 和 [详情](../go-zero-admin/application/tenant/rpc/internal/logic/tenantservice/get_tenant_logic.go) SQL 均读取 `COALESCE(admin_account_ready,0)`。 | 属于实际表结构与代码查询不一致；未调用租户 RPC，不记为 RPC 实测失败。未修改表结构或后端。 |
| `/ping` | [PingLogic](../go-zero-admin/application/admin/api/internal/logic/ping_logic.go) 仍为空实现。 | HTTP 200 只证明 API 可达，不证明数据库、Redis、RPC 或对象存储健康。 |

## 4. 无 HTTP 入口或尚未接通的能力

以 [已注册路由](../go-zero-admin/application/admin/api/internal/handler/routes.go) 和生成类型为准，不能根据 RPC 名称或前端菜单伪造接口。

| 能力 | 当前证据与阻塞 |
|---|---|
| App 用户、审核、会员、会话 | [app.proto](../go-zero-admin/application/app/rpc/desc/app.proto) 有 RPC，没有 HTTP；本轮仅保留演示数据，未验证 RPC。 |
| AI 对话、工具、工作流、任务审批 | [ai.proto](../go-zero-admin/application/ai/rpc/desc/ai.proto) 有 RPC，没有 HTTP；未执行外部模型或工作流。 |
| 租户创建、启停、删除、管理员重置 | [tenant.proto](../go-zero-admin/application/tenant/rpc/desc/tenant.proto) 有 RPC，没有 HTTP；未部署或操作租户栈。 |
| 文件上传与引用维护 | [file.proto](../go-zero-admin/application/file/rpc/desc/file.proto) 有上传/引用 RPC；HTTP 仅列表、链接、删除。测试对象使用独立 S3 夹具准备，不代表上传页面已接通。 |
| 个人头像 | [资料 API](../go-zero-admin/application/admin/api/internal/logic/profile/update_profile_logic.go) 接受 `avatar_url` 但固定传 `AvatarFileId: 0`；[GetProfile](../go-zero-admin/application/admin/rpc/internal/logic/profileservice/get_profile_logic.go) 没有解析头像 URL。现有字段不能完成头像保存。 |
| 用户分配角色 | [HTTP 生成类型](../go-zero-admin/application/admin/api/internal/types/types.go) 的新增/更新用户请求没有 `role_ids`；RPC 虽声明字段，当前创建/更新逻辑没有处理绑定。临时 SQL 授权已随验收清理。 |
| API 同步 | 未注册同步 HTTP；[SyncApi](../go-zero-admin/application/admin/rpc/internal/logic/apiservice/sync_api_logic.go) 仍返回全零的占位结果。 |
| 认证刷新、服务端退出、设备会话 | [admin.proto](../go-zero-admin/application/admin/rpc/desc/admin.proto) 声明 RPC，HTTP 未开放，HTTP 登录也不返回刷新令牌；本轮未验收这些 RPC。前端退出只清本地会话。 |

## 5. 证据与清理

本机证据目录：`%LOCALAPPDATA%/go-zero-admin/frontend-integration-20261031-152116/`。

- `http-results.json`：脱敏 HTTP 方法、路径、状态码、预期拒绝及实测偏差；保留初次 403。
- `http-coverage.json` / `http-coverage.md`：逐条 41 路由正向覆盖矩阵、保留数据计数及验证边界。
- `temporary-records.json`：本轮前缀 `fe_check_20261031_152116` 与登记 ID，最终 `cleaned=true`。
- 执行命令为 `node .tmp_integration-http.mjs <分组>`，按 `auth`、`user`、`user-resume`、`dept`、`role-menu`、`dict-api-audit`、`file`、`cleanup`、`summary` 分组推进；认证信息仅由环境变量提供，未写入结果。临时脚本和临时 SQL 已删除。

清理结果：登记的临时数据库行、专用 S3 对象和带本轮前缀的 Redis 键均为 **0**，没有全库清理或全量清缓存。原保留文件 `9001901–9001903` 共 **3** 条仍在；扩展种子按固定 ID 验证共 **45** 条：角色 1、字典 1、App 用户 8、会员 6、会话 4、消息 12、工作流 3、任务 7、租户 3，全部保留。

本记录限于上述 HTTP 与静态核对。浏览器、重启及前端自检结果见 [本轮计划](frontend-integration-plan.md)，未运行的后端自检或 RPC 验收不计为通过。
