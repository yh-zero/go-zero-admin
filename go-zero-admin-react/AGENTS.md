# AGENTS.md — 前端开发规范（React 规范优先）

> 先读仓库根 `../AGENTS.md`（通用纪律），再读本文件。冲突时以本文件为准。

## 1. 技术栈（固定，禁止擅自新增依赖）

React 19 + TypeScript（严格）+ Vite 8 + antd 6 + @ant-design/pro-components + react-router-dom 7 + @tanstack/react-query + zustand + axios + dayjs。

- 新增依赖必须先经批准，并说明理由（同根规范 5）
- Node `^24`（`package.json` 的 `engines` 已锁定），包管理 `pnpm`（`packageManager` 字段已锁定）
- 具体版本以 `package.json`、锁文件和实际安装结果为准；当前 antd 声明为 `^6.6.8`，不得按陈旧文档擅自升降级。

## 2. React 规范

1. **函数组件 + Hooks**，禁止 class 组件
2. 组件文件 `PascalCase`（`UserList.tsx`）；hooks 文件 `useXxx.ts`；工具函数 `camelCase.ts`
3. props 必须显式类型；事件回调命名 `onXxx`；列表渲染 key 用稳定 id，**禁止 index 作 key**
4. 副作用进 `useEffect` 并保持纯净；自定义 hook 封装可复用逻辑
5. 组件行数超标（标准同根规范 §6）或嵌套超 3 层 → 拆分

## 3. 对接铁律（契约见 ../ai_document/api-contract.md）

1. **所有请求走 `src/utils/request.ts`**，禁止组件内裸 axios/fetch
2. **类型只从 `src/api/types.ts` 导入**，禁止自造与后端不符的类型；后端字段以 api-contract.md 为准
3. 错误处理：拦截器已统一弹错 + 401 跳登录；组件内 catch **只做降级**（如按钮恢复），不重复 `message.error`
4. 写操作成功后调用 `queryClient.invalidateQueries` 刷新相关列表，禁止手动 setState 同步列表
5. 分页/过滤参数与契约 §4 一致（`page`/`page_size` → `list`/`total`）

## 4. 状态纪律

| 状态 | 归属 |
|---|---|
| 服务端数据（接口返回） | **一律 TanStack Query**，`useQuery`/`useMutation`；禁止 useEffect 里裸请求 |
| 客户端状态（token/permCodes/主题） | zustand（`stores/`） |
| 表单状态 | antd Form 自管理，不用 useState 镜像表单值 |

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

## 8. 禁止事项速查

- ❌ 组件内裸 axios/fetch、useEffect 里裸请求、自造与契约不符的类型
- ❌ index 作 key、useState 镜像表单值、手动 setState 同步服务端列表
- ❌ `dangerouslySetInnerHTML` 渲染后端内容、emoji 当图标、提交残留 console.log

## 9. 提交前自检（必过）

工作目录为 `go-zero-admin-react/`：

```powershell
pnpm exec tsc -b      # 类型零错误
pnpm exec vite build  # 构建成功
```
