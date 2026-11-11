# 对话参考资料原文预览实施计划

**目标与已批准设计：** 按用户确认的方案，在 AI 对话参考资料旁增加“查看原文”，使用右侧抽屉展示当前全文，保留回答时的引用片段。原文可能更新，明确区分当前全文和历史摘录。

**范围：** 仅前端，不改 API 契约、后端、依赖或锁文件，实施阶段未提交Git。任务开始时开发分支 `yh-dev` 工作区干净，在现有运行目录实施，便于直接验收。后续用户授权与用量页面、AI模块减法一起合为一个commit推送 `yh-dev`，交付检查见 `2026-11-11-ai-module-subtraction.md`。

**实施方式：** 使用 writing-plans / executing-plans 的步骤顺序，由主代理实现，独立代理只读检查查询接口及最终改动。用户已要求按方案实施，不重复请求批准。

**方案选择：** 已选对话内只读抽屉；跳转知识库会离开对话，复用可编辑资料弹窗会引入编辑与摄取职责，因此不采用。

**结构：** `MessageRecord.tsx` 维护当前预览引用；新增 `ChatSourcePreview.tsx` 按需挂载并复用 `knowledgeDocumentOptions`；`chat.module.css` 维护资料标题、按钮和全文换行。沿用 React / antd / TanStack Query。

## 检查重点

- 引用标题、历史摘录和当前全文均作为文本转义。
- 身份/部门与知识库读取权限沿用现有校验，关闭抽屉卸载查询。
- 当前读取失败时不把缓存旧全文显示为成功结果，允许手动重试。
- 快速点击、切换资料、迟到响应不能串文档，引用编号与历史摘录保持。
- 抽屉关闭保留对话、草稿与滚动位置，窄屏可阅读。

## 执行清单

- [x] 扩展现有 `scripts/aiChatExperience.test.mjs`：引用入口与历史片段保留、标题/片段转义；先运行确认新增断言因缺少“查看原文”失败，再实现并通过。
- [x] 实现资料入口、只读抽屉、加载/错误/重试状态和响应式样式。
- [x] 执行相关对话及文档回归、前端类型检查与改动文件 lint。
- [x] 浏览器验收正常全文、折叠摘录、关闭与双击、窄屏；用隔离请求桩验证读取失败和迟到响应，不修改业务数据、不调用模型。
- [x] 独立只读审查，检查 `git diff --check`，记录结果并清理本轮临时资源，保留项目服务。

## 结果

已完成，仅前端改动。浏览器加载现有 5999 开发服务的最新源码；后端沿用已有运行镜像，无需重建。

- `node --test --test-isolation=none scripts/aiChatExperience.test.mjs scripts/aiChat.test.mjs scripts/aiKnowledgeDocuments.test.mjs`：21/21 通过。
- `node_modules/.bin/tsc.cmd -b` 与 `node_modules/.bin/oxlint.cmd src/pages/ai/MessageRecord.tsx src/pages/ai/ChatSourcePreview.tsx scripts/aiChatSourcePreview.browser.mjs`：退出码 0，无警告。pnpm 启动器首次受镜像源 DNS 限制，沙箱外重试也成功；最终检查直接使用本地已安装工具，未更改依赖或锁文件。
- Playwright CLI 会话 `source-preview`：使用已授权本地演示管理员正常登录，读取真实“股票”对话及原文；未发送消息。
- `scripts/aiChatSourcePreview.browser.mjs`：本浏览器隔离 GET 桩验证按需读取、全文末尾/换行/转义、历史摘录保持、双击单抽屉、草稿及滚动位置、403/404/503 与手动重试、关闭中止后切换资料不串文档、390px 窄屏无横向溢出，全部通过。桩在 finally 移除，不写业务数据。
- 发现并修正双击入口时第二击触发遮罩关闭；抽屉使用关闭按钮或 Esc 退出。窄屏断言等待布局动画完成；统一错误提示退场后再切换视口，避免自动化鼠标悬停使提示续期遮挡关闭按钮。
- 独立只读审查无必须修复问题；`git diff --check` 通过。未运行全仓测试或生产验收，未调用付费模型。
- 本轮临时浏览器会话 `source-preview`（浏览器 PID 31240）关闭；清理本轮生成的三个页面快照和两个控制台日志，保留可重复浏览器回归脚本。前后端项目服务继续运行。
