# go-zero-admin-react

React 管理后台使用 TypeScript、Vite、antd 和 TanStack Query，连接本项目的 admin-api。首次运行整套服务，请从[项目快速开始](../README.md#快速开始)进入。

## 本地开发

从仓库根目录打开终端。Node 和 pnpm 版本见 [package.json](package.json) 的 `engines` 和 `packageManager`。

```powershell
cd go-zero-admin-react
pnpm install --frozen-lockfile
pnpm dev
```

打开 [http://127.0.0.1:5999](http://127.0.0.1:5999)。端口已被占用时 Vite 会报错；如果已有本项目的开发服务器，可直接复用。

浏览器默认向 `/v1` 发送业务请求。开发服务器将 `/v1` 和 `/ping` 代理到 `http://127.0.0.1:7001`。要连接另一个后端，从前端目录将 [.env.example](.env.example) 复制为 `.env.local`，按需设置 `VITE_API_PROXY_TARGET`（代理目标）和 `VITE_API_BASE_URL`（浏览器请求前缀），重启 Vite。前端环境变量对浏览器可见。

## 参与开发

- **接入页面**。按[前端开发规范](AGENTS.md)登记页面、路由和权限。字段与业务行为查[共享 API 契约](../ai_document/api-contract.md)。
- **请求与会话**。常规请求使用 [request.ts](src/utils/request.ts)，AI 流式对话使用 [aiStream.ts](src/api/aiStream.ts) 的 fetch/SSE 传输，共用认证、部门上下文和取消语义。自动刷新依赖 Web Locks，请使用 localhost 或 HTTPS。
- **样式与素材**。全局主题在 [theme.ts](src/theme.ts)，布局在 [layouts](src/layouts)。替换登录页猫咪素材时按[生成器说明](scripts/build-login-cat.md)重新生成动画和备用图。
- **验证**。在前端目录按影响范围选择 [scripts](scripts) 中的回归，例如修改请求层后运行 `node --test scripts/request.test.mjs`。交互变化还需在对应页面验证；提交前的类型、lint 和构建命令见[前端规范](AGENTS.md#8-开发验证与提交前自检)。

新安装使用后端[完整初始化 SQL](../go-zero-admin/data/db/init/mysql-all.sql)的演示数据。需要维护旧联调样例时，先备份目标开发库，再使用[样例导入入口](scripts/seed-test-data.ps1)；固定 ID 和冲突处理见脚本引用的两份 SQL。
