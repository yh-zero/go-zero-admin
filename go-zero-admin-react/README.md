# go-zero-admin-react

管理后台前端（React 19 + TypeScript + Vite + antd 5）。

> 骨架阶段：接口层保留类型定义与调用签名，尚未与后端联调。

## 技术栈

| 能力 | 选型 |
|---|---|
| 构建 | Vite 8 + React 19 + TypeScript |
| 组件 | antd 5（+ @ant-design/icons） |
| 效率组件 | @ant-design/pro-components（ProTable / ProForm） |
| 路由 | react-router-dom 7（静态路由 + 后端菜单动态注入） |
| 服务端状态 | @tanstack/react-query |
| 客户端状态 | zustand（persist 到 localStorage） |
| HTTP | axios |

## 目录

```
src/
├── api/          接口定义 + 类型（types.ts 与后端 REST 契约一致）
├── components/   通用组件（Auth.tsx 按钮级权限）
├── layouts/      BasicLayout（侧边菜单 + 顶栏）
├── pages/        业务页面（login / dashboard …）
├── router/       路由表 + 动态路由注入器
├── stores/       zustand（auth.ts）
├── utils/        request.ts（axios 封装）
├── App.tsx       Provider 组合
└── main.tsx      入口
```

## 约定

1. **接口契约**：REST 返回 `{ code, msg, result }`，`code === 200` 为成功；类型定义集中在 `api/types.ts`。
2. **权限**：
   - 菜单级 → `GET /sys/menu/my` 返回菜单树，`router.injectDynamicRoutes()` 注入路由。
   - 按钮级 → `GET /sys/menu/my/perm-codes` 返回权限码，存 zustand，用 `<Auth code="...">` 控制渲染。
3. **新增页面**：在 `pages/` 建文件 → 在 `router/index.tsx` 的 `pages` 注册表登记（key = 后端菜单的 `name`）。
4. **安全**：
   - token 只存 zustand + localStorage，请求经拦截器注入 `Authorization`。
   - 401 统一登出并跳登录页（拦截器内实现）。
   - 不在前端硬编码后端密钥或管理接口地址。
5. **样式**：以 antd 组件为准，`index.css` 只做必要 reset。

## 开发

```bash
pnpm install
pnpm dev        # http://localhost:5999
pnpm build      # 产物 dist/
npx tsc -b      # 类型检查
```

后端默认地址 `http://127.0.0.1:7001`，已配在 `vite.config.ts` 的 `/v1` 代理。
