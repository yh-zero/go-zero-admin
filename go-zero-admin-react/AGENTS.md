# AGENTS.md — 前端开发规范（React 实践 + 仓库约定）

> 先读仓库根 `../AGENTS.md`（通用纪律），再读本文件。冲突时以本文件为准。

## 1. 技术栈与依赖

沿用 React 函数组件、TypeScript 严格模式、Vite、antd、TanStack Query、zustand 与现有请求封装；版本和已安装依赖以 `package.json`、锁文件及实际安装为准，不在本文重复维护。

- 新增依赖必须先经批准，并说明理由（同根规范 6）
- Node 与 pnpm 分别按 `engines` / `packageManager`；依赖操作遵守根“安全基线”，不为迎合文档擅自升降级

## 2. React 规范

1. **函数组件 + Hooks**，禁止 class 组件
2. 组件文件 `PascalCase`（`UserList.tsx`）；hooks 文件 `useXxx.ts`；工具函数 `camelCase.ts`
3. props 必须显式类型；事件回调命名 `onXxx`；列表渲染 key 用稳定 id，**禁止 index 作 key**
4. render 保持纯净；外部订阅、资源生命周期用 `useEffect` 并清理，用户操作在事件/Mutation 中处理，不把所有副作用机械搬进 Effect
5. 抽组件/hook 以独立职责、复用或测试收益为准，不为行数、嵌套层数增加转发层；纯整理不改变交互和状态语义

## 3. 接口对接（契约见 ../ai_document/api-contract.md）

1. **业务 API 请求走 `src/utils/request.ts`**，API 函数放 `src/api/`，禁止组件内裸 axios/fetch 绕过认证、错误和会话处理
2. **接口请求/响应类型统一从 `src/api/types.ts` 导入**；当前按共享契约及核对后的后端类型手工维护，没有现成的 TypeScript 类型生成入口。禁止自造与契约不符的接口类型；组件 Props、UI 状态类型可在所属组件或模块定义。
3. 请求层统一处理错误及会话失效；组件可做字段提示、失败反馈和恢复交互，但不重复弹同一错误，也不吞掉失败伪装成功
4. 写成功后默认 `queryClient.invalidateQueries` 刷新受影响缓存；确需精确缓存更新/乐观更新时使用 Query 缓存 API 并验证回滚和一致性，不另用组件 state 长期维护服务器列表副本
5. 分页/过滤参数与契约 §4 一致（`page`/`page_size` → `list`/`total`）

## 4. 状态纪律

| 状态 | 归属 |
|---|---|
| 常规服务端查询与业务写操作 | 默认 `useQuery`/`useMutation`；不在 Effect 裸请求或复制缓存到 state |
| 会话、权限快照、主题等共享客户端状态 | 沿用 zustand 与既有会话流程，不强制把登录/刷新状态机改造成普通查询 |
| 表单与局部交互 | antd Form 管表单值，展开/选择/动画等用局部状态；避免与 Form 或 Query 双份维护 |

## 5. 目录规范

```
src/
├── api/         # 接口函数 + types.ts（与后端契约一一对应）
├── components/  # 通用组件（Auth 权限组件等）
├── layouts/     # 布局
├── pages/       # 业务页面（按模块分目录）
├── router/      # 路由表 + 页面注册表（新增页面必须登记）
├── stores/      # zustand
└── utils/       # request.ts 等
```

- 新增页面三步：`pages/<模块>/` 建文件 → 在 `router/registry.ts` 的 `pages` 与 `pageLinks` 登记（key 与后端菜单 `name` 一致）→ 后端菜单配置对应 `component`。`router/index.tsx` 负责组装路由，不维护页面注册表。

## 6. 安全

- token 只存 zustand（persist 到 localStorage），请求经拦截器注入；退出即清
- **禁止 `dangerouslySetInnerHTML` 渲染后端内容**（防 XSS）
- 环境变量只走 `VITE_` 前缀；任何密钥不进前端代码/仓库
- 权限判断以后端为准：前端 `<Auth code>` 只做渲染隐藏，**不做真正的访问控制**

## 7. 样式与易读

- 以 antd 组件 + 主题为准；禁止大规模内联 style 对象，公共布局样式进 `index.css` 或 CSS 模块
- 图标用 `@ant-design/icons`，禁止 emoji 当图标
- 注释解释"为什么"；提交前删掉调试代码/console.log

## 8. 代码提交前自检（必过）

工作目录为 `go-zero-admin-react/`：

```powershell
pnpm exec tsc -b      # 类型零错误
pnpm lint             # oxlint 零错误
pnpm exec vite build  # 构建成功
```

本地与 CI 应执行同一组门槛；当前 CI 尚缺 lint，见 `../ai_document/README.md` 待办，不因 CI 缺项省略本地检查。业务变更补可重复回归并验证受影响交互；认证/权限/状态竞争场景按根 §8。

纯文档或规则修改按根规范检查内容一致性、路径引用和差异，不要求运行上述构建。
