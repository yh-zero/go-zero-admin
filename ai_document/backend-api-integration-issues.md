# 后端对接待办（2026-10-31）

仅记录**未关闭**事项。已修复历史见 git 提交（84245c4 / b22f359 / b258a2d）。前端联调详情见 [frontend-integration-plan.md](frontend-integration-plan.md)。

## 待修复

### 接口语义

| 编号 | 问题 | 处理建议 |
|---|---|---|
| B01 | `PUT /v1/sys/profile` 与 [UpdateUser](../go-zero-admin/application/admin/rpc/internal/logic/userservice/update_user_logic.go) 为全量覆盖语义：只提交昵称会清空姓名/邮箱、性别归零；部门/头像传 `0` 不能解绑 | 需 proto/desc 层 optional 改造支持部分更新；前端已全量提交暂无实际痛点，随契约版本一起做 |

### 静态确认的限制

| 范围 | 问题 | 边界 |
|---|---|---|
| 部门 | 移动部门不维护自身/后代 `path`（创建时维护）；[更新](../go-zero-admin/application/admin/rpc/internal/logic/deptservice/update_dept_logic.go)已补防环与缓存一致性，但开放"调整上级"必须连带做 path 重算 | 随部门移动功能一起 |
| 部门上下文 | [Authority](../go-zero-admin/application/admin/api/internal/middleware/authority.go) 的 `X-Dept-Id` 只改 Casbin domain，context 部门仍取 JWT；缺"当前用户部门清单"HTTP | 开放部门切换前必须补 |
| 响应与错误码 | 信封已统一，但[契约](api-contract.md)未列出实际使用的 402/404/505/506；文件模块 404 暂为无名常量 | 补契约错误码表时一并命名 |
| 租户表结构 | `gozero_platform.tenant_registry` 实测 13 列、无 `admin_account_ready`，proto/列表/详情四处读该列，调用即报错；[来源迁移](../go-zero-admin/data/db/migrations/platform/00001_tenant_registry.sql)同样缺列 | 随租户生命周期功能做列迁移（新增 goose migration） |
| 健康检查 | [`/ping`](../go-zero-admin/application/admin/api/internal/logic/ping_logic.go) 为 goctl 空桩，HTTP 200 仅证明 API 进程可达 | 上生产前实现 DB+Redis+RPC 探活 |

### 尚未接通的功能（有 RPC 无 HTTP / 能力缺失）

- App、AI、租户：有 RPC 无 HTTP 入口；租户 RPC 因缺列当前不可调用（见上）
- 文件上传、引用维护：仅 RPC；HTTP 只有列表、链接、删除
- 个人头像：资料 API 接受 `avatar_url` 却固定 `AvatarFileId: 0`；资料查询未解析头像 URL
- 用户分配角色：HTTP 新增/更新请求无 `role_ids`，RPC 创建/更新未处理绑定
- API 同步：无 HTTP；`SyncApi` 返回全零占位结果
- 认证刷新、服务端退出、设备会话：有 RPC 无 HTTP；登录不返回刷新令牌，前端退出仅清理本地会话
- 验证码为文本后门（`image_base64` 即答案，[get_captcha_logic.go](../go-zero-admin/application/admin/rpc/internal/logic/authservice/get_captcha_logic.go) 自述"简版"）；审计日志（`sys_audit_log`）只有读接口、无任何写入方
- 代码卫生：`xerr.REUQEST_PARAM_ERROR` 拼写（REUQEST→REQUEST，改名波及面大）、`xerr.Wrap` 死代码——合并为一个 `refactor:` 提交处理

## 验证边界

- 后端 `verify.ps1` 已在最近三轮修复提交前六项通过；App/AI/租户 RPC 调用与数据库/Redis 故障注入仍未执行，不计为验证通过。
