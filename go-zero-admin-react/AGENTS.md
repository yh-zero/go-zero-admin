# 前端开发规范

先读[根规范](../AGENTS.md)。本页补充 React 前端的目录、状态、对接与验证约定。

## 1. 技术栈与依赖

沿用现有 React、TypeScript 严格模式、Vite、antd、TanStack Query 和 zustand。Node、pnpm 与依赖版本见 [package.json](package.json) 和[锁文件](pnpm-lock.yaml)。新增依赖须事先说明理由并取得授权。

## 2. React 规范

使用函数组件和 Hooks。组件文件用 PascalCase，Hook 用 `useXxx`，工具函数用 camelCase。显式声明 props 类型，事件回调用 `onXxx`，列表 key 使用稳定标识。保持渲染纯净；订阅与资源释放成对处理，用户操作由事件或 Mutation 发起。组件或 Hook 按独立职责、复用或测试需要拆分。

## 3. 接口对接

- 常规业务请求经 [request.ts](src/utils/request.ts)，流式传输在 [src/api](src/api) 封装，复用认证、刷新、部门上下文和会话取消逻辑。组件通过 API 函数发起请求。
- 请求和响应类型从 [types.ts](src/api/types.ts) 导入，按[共享契约](../ai_document/api-contract.md)及后端 `.api` 手工核对。组件自行定义 Props 和局部 UI 类型。
- 请求层处理共用错误和会话失效。页面负责字段提示与可恢复的交互，不重复弹窗或将失败展示为成功。

## 4. 状态归属

服务端数据使用 Query 和 Mutation。写入成功后，仅在服务端结果完整且分页、排序和对象存续均确定时回填缓存，其他情况刷新受影响查询。乐观更新应验证失败回滚。表单值由 antd Form 管理，展开与选择由组件管理，会话、权限快照及主题由 zustand 管理。

尚未确认提交的流式增量或异步结果只作临时展示。断流、超时或取消后按契约保留原操作标识核查结果，重查不得重新提交。

## 5. 页面与权限登记

- 在 `src/pages/<模块>/` 实现页面，在 [registry.ts](src/router/registry.ts) 的 `pages` 注册。独立菜单页还要登记 `pageLinks`，核对后端菜单的 `name`、`path` 和 `component`。
- 沿用父菜单的详情、编辑等隐藏子页，在 [hiddenPages.ts](src/router/hiddenPages.ts) 登记 `hiddenPageLinks` 和 `menuPath`。含动态参数时核对 `hiddenPageForPath` 匹配实际 URL。
- 新增或调整表结构、独立菜单、按钮权限、受控接口、默认角色授权或必要种子时，将相应变更写入[完整初始化 SQL](../go-zero-admin/data/db/init/mysql-all.sql)，核对全新安装后的可见性与服务端授权。完全复用现有菜单、权限、接口和数据的展示子页只需核对。已有库按[数据初始化](../ai_document/dev-guide.md#数据初始化)补齐本次变更，保留既有数据与授权。

## 6. 安全

令牌经现有 zustand 会话存储和统一传输逻辑管理，退出时清理。权限码组件控制界面显示，服务端仍须鉴权。后端内容用文本转义渲染，禁止用 `dangerouslySetInnerHTML` 展示。客户端配置采用 `VITE_` 前缀，不存密钥。

## 7. 样式与易读

沿用 antd 主题和 `@ant-design/icons`，公共布局样式放入 `index.css` 或 CSS 模块。避免大块内联样式、用 emoji 充当图标；提交前移除调试输出。

## 8. 开发验证与提交前自检

开发阶段按[根规范的完成定义](../AGENTS.md#9-任务完成定义)验证直接受影响的查询、交互和失败状态。lint 优先限定改动文件；修改资源打包或构建配置时验证实际构建。

提交前端代码前，在 `go-zero-admin-react/` 执行以下三项。全部成功后再提交。纯文档修改按根规范核对内容、链接和差异。

```powershell
pnpm exec tsc -b
pnpm lint
pnpm exec vite build
```
