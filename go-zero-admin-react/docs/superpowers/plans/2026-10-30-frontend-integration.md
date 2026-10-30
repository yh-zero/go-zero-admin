# 前端对接 Implementation Plan

> 历史计划（2026-10-30）：下列范围和验收数字记录当时的前端阶段。2026-10-31 用户已授权必要后端配合，后续修复、测试清理及当前状态以根目录 AGENTS.md 和 `ai_document/backend-api-integration-issues.md` 为准。

> **For agentic workers:** 按限定目录并行实施，父代理整合与验收。

**Goal:** 仅改前端完成现有 sys HTTP 模块对接。
**Architecture:** 统一请求与身份层，模块 API 解包，Query 管理服务端数据。
**Tech Stack:** 现有 React/TypeScript/Vite/antd/Query/zustand/axios。
**Spec:** ../specs/2026-10-30-frontend-integration-design.md

## Global Constraints
- 后端、契约、生成物、依赖版本及锁文件不改。
- REST类型集中，标准Bearer，不通过绕过鉴权掩盖后端问题。
- 测试数据仅写独立本机库，demo_与固定ID范围可识别。

## Review Focus
- HTTP200业务错误必须拒绝，401清身份。
- 切换账号不复用旧查询和权限。
- 菜单孤儿、循环和外部路径不会导致错误跳转或无限递归。
- 编辑先取详情，菜单读取失败不得清空角色授权。
- 正确处理验证码、分页、空列表和后端错误。

## Tasks
- [x] 请求响应、树与验证码先测试再实现。
- [x] 集中REST类型、共享查询/展示组件、API函数。
- [x] 并行完成user/dept/role与menu/dict/api，父代理完成file/audit/profile。
- [x] 真实登录、菜单、权限、路由和首页；后端失败显示错误与重试，不伪造成功。
- [x] 独立测试库关联数据，不改原管理员账号、密码和权限；精确修复默认种子中文误转码。
- [x] tsc严格模式、Vite、oxlint与12项测试通过；完成浏览器可运行路径及错误状态验收，后端阻塞如实记录，验证环境已清理，用户要求的测试数据保留。
