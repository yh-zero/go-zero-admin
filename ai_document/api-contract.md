# API 契约（前后端共同遵守）

> 本文件是前后端唯一契约事实源。改契约 = 先改本文档 + 同步两端实现。

## 1. 响应格式

所有接口返回 HTTP 200，业务结果由 body 区分：

```json
{ "code": 200, "msg": "操作成功", "result": {} }
```

| 字段 | 说明 |
|---|---|
| code | 200 成功；其余为业务错误码（见 §5）；401 表示登录过期 |
| msg | 错误提示文案（成功时固定"操作成功"） |
| result | 成功数据；无数据时为 null 或省略 |

前端约定：axios 拦截器统一剥壳；`code !== 200` 由拦截器弹 `msg` 错误并 reject，**组件内不重复弹错**；code 401 时清 token 跳登录页。

## 2. 命名与类型

| 项 | 约定 |
|---|---|
| JSON 字段 | snake_case（如 `access_token`、`dept_id`） |
| 时间 | int64 Unix 秒（如 `created_at`） |
| ID | int64 |
| 枚举 | 字符串优先（如 `visibility: "public" | "private"`）；状态类用 int + 字典 |

## 3. 鉴权

- 请求头：`Authorization: Bearer <access_token>`
- 登录：`POST /v1/sys/auth/login`（需先 `GET /v1/sys/auth/captcha` 取验证码）
- 权限码格式：`sys:<模块>:<动作>`（如 `sys:user:add`），按钮级控制由后端下发，前端只做渲染判断
- 部门上下文：多部门用户可用请求头 `X-Dept-Id` 切换（可选）

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

### 无返回体的写操作

`result` 为 null，前端按成功处理，无需解析。

## 5. 错误码（与后端 pkg/xerr 一致）

| code | 含义 | 前端处理 |
|---|---|---|
| 200 | 成功 | 正常 |
| 400 | 参数错误 / 业务校验失败 | 弹 msg |
| 401 | 登录过期 / 未登录 | 清 token 跳登录 |
| 403 | 接口权限不足 | 弹 msg |
| 500 | 服务器内部错误 | 弹 msg |
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

1. 改本文档（注明变更日期）
2. 后端：改 desc/*.api 或 proto → `gen.ps1` 重新生成 → 实现 logic
3. 前端：改 `api/types.ts` + 对应 api 模块
4. 两端各自过自检（verify.ps1 / tsc+build）
