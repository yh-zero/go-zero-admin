# 前端验证证据

本页按功能保留 2026-10-30、2026-10-31 和 2026-11-11 的历史验收证据，供回归定位；2026-11-13 文档合并未重跑这些功能、构建或浏览器测试。当前开发约定见 [前端 README](../README.md)，接口字段和路由以 [API 契约](../../ai_document/api-contract.md) 为准。

原始记录没有为每次验收提供独立构建 SHA 或精确构建时间。11 月 11 日的原文预览、用量与 AI 模块调整可由提交 `f0f50b7`（提交时间 `2026-11-11T23:09:19+08:00`）定位；这不是对验收时运行构建 SHA 的补充证明。下文保留当时记录的源码加载方式、构建资源标识和覆盖边界，不将历史通过当作当前版本通过。

## 当前回归入口

下文历史命令保留当时的文件名和结果；涉及下表旧文件名时仅作证据引用，当前复验使用归并后的入口。测试数量变化不能解释为业务覆盖下降或性能提升。

| 历史文件 | 当前入口与必须保留的覆盖 |
|---|---|
| `aiChatExperience.test.mjs` | [聊天](../scripts/aiChat.test.mjs)保留知识库范围、原请求重查、完整导出失败及来源转义；[安全 Markdown](../scripts/safeMarkdown.test.mjs)保留外链协议、代码围栏转义和复制入口。 |
| `aiWorkspaceTransport.test.mjs` | [工作区](../scripts/aiWorkspace.test.mjs)保留已取消信号不进入请求适配器；普通请求预算仍由现有请求链路测试覆盖。 |
| `aiProductWorkspace.test.mjs` | [导入回执](../scripts/aiKnowledgeImportReceipts.test.mjs)、[资料](../scripts/aiKnowledgeDocuments.test.mjs)、[知识库](../scripts/aiKnowledge.test.mjs)、[运行参数](../scripts/aiWorkflowDataflow.test.mjs)、[草稿恢复](../scripts/aiDraftRunAttempts.test.mjs)及[任务](../scripts/aiTasksTools.test.mjs)分别承接未知/失败/未提交分区、批次标识及上限、模板与输入、身份隔离和结果优先。 |
| `aiWorkflowSimplification.test.mjs` | [流程](../scripts/aiWorkflows.test.mjs)、[任务](../scripts/aiTasksTools.test.mjs)和[编辑器](../scripts/aiWorkflowEditor.test.mjs)保留列表权限、状态、错误仅显示一次且转义、未知高级参数。浏览器交互不替代这些 CI 断言。 |
| `aiKnowledgePresentation.test.mjs` | [知识库](../scripts/aiKnowledge.test.mjs)和[质量验收](../scripts/aiKnowledgeQuality.test.mjs)保留默认入口、只读权限、未知请求恢复及首屏未执行详情模块。 |
| `aiUsage.browser.js` | [用量页面](../scripts/aiUsagePage.browser.mjs)保留 unknown/partial 实际展示、双击防重、409 后显式读新版本再保存及普通身份首次加载隔离。 |
| `aiWorkflowForm.browser.js` | [流程编辑与版本](../scripts/aiWorkflowRevisions.browser.js)以自包含响应桩覆盖实际 Form 提交的 enabled、工具缓存、未知结果刷新恢复和一次创建。 |

浏览器入口使用各自独立会话和响应桩前提。当前页面的分析区、新建评测、高级设置及“更多操作”需按真实交互展开；[导入回执浏览器回归](../scripts/aiKnowledgeImportReceipts.browser.js)的配额失败与部分确认场景分别验证，不能把未执行分支记为通过。

2026-11-13 测试归并的针对性验证：上表涉及的 12 个 Node 目标文件统一执行 `node --test --test-concurrency=2 --test-timeout=180000`，121 项通过、0 失败、0 跳过；7 个受影响浏览器脚本覆盖 8 个独立场景（导入回执 quota/partial 分开），均以响应桩验证，未写真实业务或调用模型。新增和修改脚本的语法、限定 lint 与差异检查通过。此轮只验证测试迁移和页面入口，不重跑其余前端回归、后端或真实供应商验收。

运行与失败复核证据位于本地忽略目录 `tmp/test-consolidation-20261113/`：`node-final.log`、`browser-targets.log`、`browser-usage-final.log`、`browser-dataset-comparison.log`、`browser-dataset-final.log` 和 `browser-import-receipts.log`。题集初次复验因高级设置再次收起而失败，最终通过记录单独保留；原始性能 JSON 及之前的质量/性能证据未改。归并后的测试文件为 32 个 Node 回归和 18 个浏览器脚本，共 50 个；文件数量减少不是覆盖率或性能提升的测量。

## AI 入口与操作减法（2026-11-13）

本轮基于 `e45fa76bcf977196641c0168ae09badde05fc50d` 后的工作区源码验证，仅修改前端 AI 页面和直接回归。常用流程列表移除重复能力目录链接，新建首屏移除 JSON 模板，保留编辑器能力选择、`json_formatter` 旧入口及仅工具查看权限用户的只读目录；界面统一“常用流程”“运行记录”。聊天自动知识库范围隐藏资料约束开关，指定知识库后显示，原发送值归一化与写入禁用保持。反馈采用“用此样本新建评测”，核对后通过“确认题集，前往新建评测”导入；两个下载入口分别收进“更多导出”，只有明确点击“开始评测”才执行。

针对性 Node 回归分组通过：流程相关 7 文件 75 项，编辑器/工具/草稿 3 文件 31 项，聊天 19 项，反馈及题集 63 项；术语收尾后反馈/质量/工具 3 文件 46 项复验通过。组间有重叠，不相加为独立用例数。新增入口、开关与评测主动作断言均保留实现前的预期失败和实现后的通过日志。`tsc -b`、改动文件限定 oxlint、浏览器脚本语法检查通过；历史面板验证需要生产资源，额外执行当前源码 `vite build` 成功，仅保留已有公共 chunk 体积提示。未运行全仓前后端回归。

浏览器执行以下 8 个入口，均取得实际完成结果：

| 入口 | 本次确认的行为 |
|---|---|
| [组合入口与聊天](../scripts/aiShellChatSubtraction.browser.mjs) | 隐藏重复入口、保留编辑器 JSON 能力与旧参数入口、只读目录兜底；新旧聊天指定库/自动范围开关切换；转义和 5 页 390px 布局。 |
| [流程列表与编辑](../scripts/aiWorkflowSubtraction.browser.mjs) | 常用/更多操作、删除取消、筛选与状态读取、进度/比较、参数引用、草稿和只读呈现。 |
| [流程编辑与版本](../scripts/aiWorkflowRevisions.browser.js) | 实际 Form 的 enabled/工具缓存、未知创建刷新不重提、冲突保留草稿、恢复需明确新操作、只读历史及 390px。 |
| [草稿运行权限](../scripts/aiDraftRunAccess.browser.js) | 最小权限、直接禁止路由、不读取无权运行记录、配额失败后 SPA 恢复及复用原请求。 |
| [已存流程运行](../scripts/aiWorkflowRun.browser.js) | 存储失败后关闭重开、双击核对仅追加一次请求、同一标识及输入、当前页读取并展示原运行结果。 |
| [反馈复核](../scripts/aiKnowledgeFeedback.browser.js) | 复核与事实校验、更多导出原样本、读取失败恢复和版本边界。 |
| [反馈题集](../scripts/aiFeedbackDataset.browser.js) | 资料 SHA 变化拒绝、导出与提交原字节一致、导入不自动运行、双击防重、未知请求原标识恢复、快照复用、负例须核对全部资料、身份隔离及 390px。 |
| [历史面板](../scripts/aiHistoryPanel.browser.mjs) | 生产资源加载失败提示、解除故障后重试成功、关闭返回原详情。 |

前 7 项加载原有 `5999` 开发服务当前源码；反馈、版本、草稿和运行提交均由响应桩接收，没有真实业务写入或模型调用。历史面板使用本轮构建的临时 `6019` 生产预览，资源含 `WorkflowHistory-DhM_3YKC.js`，通过本地演示账号正常登录，只读取已有流程及历史，未恢复版本。真实供应商、付费评测和后端逻辑不在本次验证范围。

原始日志、源码保护指纹和浏览器 JSON 保存在本地忽略目录 `tmp/ai-subtraction-20261113/`。Node 命令见 `workflow-commands.txt`、`chat-commands.log`、`feedback-green-node.log` 与 `final-followup-commands.log`；静态检查见 `static-*.log`、`final-followup-*.log`，构建见 `history-preview-build.log`。浏览器结果分别在 `live/`、`feedback-browser/`、`dataset-browser/`、`revisions-browser/`、`draft-browser/`、`run-browser/` 和 `history-preview/` 下。

两处初次验证结果保留：历史脚本依赖生产 `/assets/WorkflowHistory-*.js`，在开发服务无法注入该故障，改用生产预览后通过；运行脚本原先等待旧版跳转地址，已改为核对现有内联结果和实际读取的记录 ID，原标识、输入与防重断言保留。版本脚本遇到 CLI 的 beforeunload 提前返回 `{}`，该空值未计通过；通过本地包装器等待全部断言结束，`revisions-browser/completion.json` 明确记录 `ok:true`、零页面错误、零真实写入及完整结果。

1144 个范围外跟踪文件最终指纹一致，后端、model、生成物、API 契约、权限码和依赖未改。本轮真实演示登录均正常退出，独立浏览器全部关闭，临时 preview PID 36192 已结束且 6019 释放；原 5999 服务保留。本节功能验收结束时尚未提交，后续 Git 交付以提交记录为准。

提交前完整自检再次执行本地已安装工具入口：`node node_modules/typescript/bin/tsc -b`、`node node_modules/oxlint/bin/oxlint --format=json`、`node node_modules/vite/bin/vite.js build`，均退出 0。全量 lint 检查 248 个文件、零错误，保留两处既有 `react(refs)` 警告（`Conversation.tsx:71`、`NewConversationComposer.tsx:52`，两文件均与基线提交一致）；构建保留已有公共 chunk 大于 500 kB 提示。原始结果在同一本地任务目录的 `commit-tsc.log`、`commit-lint.json` 和 `commit-build.log`。源码与功能验收时指纹一致，本次交付未重复浏览器或真实模型调用。

## 基础管理模块与请求对接

2026-10-30 的初始对接记录覆盖 user/dept/role/menu/dict/api/file/audit/profile、登录、菜单、权限、路由和真实统计首页。记录称 TypeScript 严格检查、Vite、oxlint 与 12 项测试通过；Node 内置测试涉及响应解包、树和验证码，但没有保留完整命令、逐项报告或构建标识，不能据此复原每项断言。

当时浏览器验证了可运行路径和错误状态，后端错误按失败展示；独立本地库中的 `demo_` 与固定 ID 测试数据保留。该阶段 App/AI/租户尚缺 HTTP、文件缺上传入口，以及相关 SQL/配置问题只是当时边界，后续状态见 [后端事项](../../ai_document/README.md)。样例数据及幂等、冲突验证说明继续保留在 [测试数据](../README.md#测试数据历史导入记录)，不作为外部服务已联调的证明。初始临时测试已清理，不能将该次 12 项结果描述成当前可直接重跑的测试集。

## 公共主题与响应式布局

2026-10-31 的真实浏览器验证覆盖登录、工作台、个人中心和 8 个管理列表，共 11 页：

- 桌面 1280px 与手机 390px 均无页面横向溢出，宽表在容器内滚动。
- 用户名筛选 `admin` 返回 1 条，用户详情正常打开；手机导航可打开，切换页面后自动关闭。
- 新增用户空提交显示必填提示，未发起新增写入；手机表单可内部滚动，844px 高度下底部操作区约位于 802px，按钮可见，隐藏头像字段不可见。
- 桌面长弹窗上移；登录页在独立 localhost 标签检查，未再次填写验证码或提交登录。
- 独立静态审查未发现新增 P1/P2，冷刷新后浏览器无新增 error。

前端目录历史命令均退出 0：

```powershell
node node_modules/typescript/bin/tsc -b
node_modules/.bin/vite.cmd build
node node_modules/oxlint/bin/oxlint
```

构建仍提示已有公共 chunk 大于 500 kB，当时约 686 kB（gzip 226 kB）。本次历史验收没有覆盖真实写入和全部异常路径，未以临时 CSS/JSX 镜像测试代替浏览器验证。

## 密码规则、选项和只读页面回归

2026-10-31 的维护性整理包括移除无调用者的 `buildRoutes`、删除不可达的第二次 HTTP 401 判断、复用密码规则和状态/性别选项。原分页校正、权限加载失败保护、表单草稿、提交锁定和会话切换隔离经独立复核未发现行为变化。

前端目录执行下列命令，均退出 0：

```powershell
node node_modules/typescript/bin/tsc -b
node_modules/.bin/vite.cmd build
node node_modules/oxlint/bin/oxlint --format=json
git -c core.safecrlf=false diff --check
```

oxlint 检查 91 个文件、116 条规则，诊断为空；Vite 保留约 686 kB（gzip 226 kB）的公共 chunk 提示，未调高阈值。

- 历史临时命令 `node --test .tmp_password_rules.test.mjs` 使用 antd 实际依赖的验证器，3 组通过、36 次断言，覆盖 ASCII/中文/emoji 字符与 UTF-8 字节边界、空格、空值、确认密码匹配和源字段更新。该临时脚本已删除，命令当前不可直接重跑。
- `formatGender` 正常值与未知回退共 7 项通过；原记录未给出独立命令。
- 真实浏览器只读冒烟覆盖工作台、用户、角色、菜单、部门、字典、个人中心；停用用户筛选返回 4 条，新增用户空提交提示必填，性别选项和显示正常，浏览器无新增 error。

未填写新密码或提交写入；未重复全部写入、权限、并发和异常场景。密码边界由当时实际验证器测试覆盖，不能扩展为浏览器写入验收。

## AI 对话参考资料原文

2026-11-11 的浏览器加载当时 5999 开发服务最新源码，后端沿用已有镜像，没有重建后端。

```powershell
node --test --test-isolation=none scripts/aiChatExperience.test.mjs scripts/aiChat.test.mjs scripts/aiKnowledgeDocuments.test.mjs
node_modules/.bin/tsc.cmd -b
node_modules/.bin/oxlint.cmd src/pages/ai/MessageRecord.tsx src/pages/ai/ChatSourcePreview.tsx scripts/aiChatSourcePreview.browser.mjs
```

回归 21/21 通过；新增入口断言在实现前因缺少“查看原文”失败，实现后通过。类型与限定 lint 退出 0、无警告。pnpm 启动器首次受镜像源 DNS 限制；后续重试和直接使用本地工具完成检查，未改依赖或锁文件。

真实浏览器通过本地演示账号正常登录，读取已有“股票”对话及真实原文，未发送消息。[浏览器回归脚本](../scripts/aiChatSourcePreview.browser.mjs) 的隔离 GET 桩验证全部通过：

- 按需读取、全文末尾、换行与转义、历史摘录和引用编号保留。
- 双击只保留一个抽屉，关闭保留对话草稿及滚动位置。
- 403/404/503 读取失败与手动重试；关闭中止后切换资料，迟到响应不串文档，失败不展示旧全文为成功结果。
- 390px 窄屏无页面横向溢出。

历史验收发现并修复双击入口第二击落在遮罩上导致关闭，改用关闭按钮或 Esc 退出。自动化窄屏断言等待布局动画完成；切换视口前等待统一错误提示退场，避免鼠标悬停使提示续期并遮挡关闭按钮。请求桩在 finally 移除，未写业务数据。独立只读审查无必须修复项，`git diff --check` 通过；未运行全仓测试、生产验收或付费模型。可重复回归脚本保留，临时页面快照和控制台日志不属于现存原始证据。

## AI 用量与额度

2026-11-11 的 5999 开发服务加载当时最新源码，后端镜像和数据库未修改。

```powershell
node --test --test-isolation=none scripts/aiUsage.test.mjs scripts/aiUsageFilters.test.mjs scripts/aiUsagePresentation.test.mjs
node_modules/.bin/tsc.cmd -b
```

回归 15/15 通过；新增统计和日期断言在实现前因缺少辅助函数/额度组件、仍为四项统计而失败，实现后通过。改动 TypeScript 和浏览器回归文件的 `oxlint.cmd`、`git diff --check` 退出 0，原记录没有列出完整 lint 文件参数。

真实浏览器以本地演示账号读取本人用量与当日额度，检查桌面布局。[浏览器回归脚本](../scripts/aiUsagePage.browser.mjs) 的隔离请求桩全部通过：

- 上海日期、首尾包含及最多 31 日、折叠保留与重置；个人/全局条件隔离，全局用户筛选不进入个人查询。
- 期间 12 次与今日占用 2 次分开显示；请求结果未知和计量不完整分别告警；统计 503 时不显示旧值。
- 明细空状态与技术字段、只读/普通/无读取权限入口；390px 窄屏无页面横向溢出。
- 策略请求携带 `expected_version=4`，409 冲突保留草稿；PUT 被浏览器桩拦截，没有真实额度写入。

验收沿用 30 秒 Query 缓存；重置或切换命中缓存时，通过显式刷新核对实际请求，未为验收修改缓存策略。测试后恢复内存权限、卸载请求桩并重新打开真实页面。独立只读审查未发现必须修复项；未运行全仓测试、生产验证、真实策略写入或模型调用。

## AI 知识库、流程、运行记录与按需加载

2026-11-11 最终合并运行 22 个直接受影响回归脚本，166/166 通过、0 跳过；原记录未列出完整 22 个脚本及合并命令。`tsc -b`、限定改动文件的 oxlint、最终 Vite 生产构建均退出 0，未运行后端及前端其他模块测试。

隔离浏览器验收覆盖：

- 知识库/资料唯一入口、编辑和删除取消、两种添加方式、草稿折叠保持、未知原请求恢复、评测历史优先/选题保持、只读权限。
- 流程更多/复制/编辑/删除取消、3 标签及 5 状态的单请求筛选、结果/对比、参数来源摘要、试运行状态与只读权限。
- 工具入口归并、仅工具权限回退；聊天详情按需加载、安全转义及操作同一行；5 个主页面在 390px 下无页面横向溢出。

保留的浏览器入口：

- [知识库与反馈 Tab 按需](../scripts/aiKnowledgeSubtraction.browser.mjs)
- [流程与运行记录](../scripts/aiWorkflowSubtraction.browser.mjs)
- [导航及聊天](../scripts/aiShellChatSubtraction.browser.mjs)
- [异步运行面板](../scripts/aiAsyncPanel.browser.mjs)
- [版本历史面板](../scripts/aiHistoryPanel.browser.mjs)

独立审查发现 3 项 P2，历史记录均已修复并复核：

| 失败表现 | 修正与验证证据 |
|---|---|
| 动态模块加载失败会冒泡退出工作区 | `AsyncContent` 局部错误与重试；生产预览先复现失败，再验证恢复、加载中收起和草稿保持 |
| 导航归并遗漏隐藏/停用祖先 | 增加可见树判断，相关回归先失败后通过 |
| 版本历史异步失败提示不在当前详情中可见 | 提示进入当前详情抽屉；浏览器先复现不可见，再验证重试及关闭子抽屉返回 |

性能中间复测曾发现低频 UI 分割增加碎片请求，后续拆出 `WorkflowReadOnly`，列表详情、版本历史、对比与质量结果按需加载。最终只读复查未发现新的明确缺陷。

所有浏览器写请求均由桩拦截或提交前取消，没有真实业务写入、模型或 Embedding 调用；测试后恢复权限和精确存储键。任务/文档轮询仍按原条件暂停后台、错误停止并保留两分钟上限，由相关工作区/轮询回归覆盖，未执行付费模型的长期真实任务。

当时交付前重新执行本地工具的 `tsc -b`、完整 oxlint 与 `vite build`，均退出 0。完整 lint 保留 3 项既有警告，位于未改动的 `NewConversationComposer.tsx`、`Conversation.tsx` 和 `scripts/aiWorkflowForm.browser.js`；生产构建保留已有 antd 大块提示，没有通过改依赖或打包配置消除范围外警告。

## AI 首屏与交互性能

原始样本、对比和前后构建资源清单原样保留于 [ai-module-performance.json](superpowers/plans/2026-11-11-ai-module-performance.json)。复现入口为 [aiUiPerformance.browser.mjs](../scripts/aiUiPerformance.browser.mjs)，在已登录的独立生产预览运行：

```powershell
playwright-cli run-code --filename scripts/aiUiPerformance.browser.mjs --raw
```

历史测量使用同一浏览器、身份、数据及本机服务；基线 `http://127.0.0.1:6091`，调整后 `http://localhost:6092`，知识库 ID 为 7。每个生产构建 9 个页面、每页 3 次冷缓存，CDP 禁用缓存，各指标取中位数。首屏 API 包含 4 次公共初始化；脚本字节为浏览器 `decodedBodySize`，就绪时间取 API `responseEnd` 最大值，DOM 取主区域后代节点数；输入指标为 5 对填入/清空搜索草稿耗时的中位数。仅 GET 导航和搜索草稿编辑，不提交模型、Embedding 或业务写入。

以下为原始 `comparison` 中的历史汇总，质量页异常及解释限制见表后：

| 页面 | 首屏 API 前→后 | JS 请求 前→后 | JS 解码字节 前→后 | 主区域 DOM 前→后 | API 就绪 ms 前→后 | 输入 ms 前→后 |
|---|---|---|---|---|---|---|
| 对话 | 6→6 | 12→12 | 1,529,742→1,530,670 | 248→248 | 530→549 | 15→13 |
| 知识库列表 | 5→5 | 11→11 | 1,491,017→1,491,683 | 243→183 | 534→534 | 13→14 |
| 知识库工作区 | 6→6 | 21→18 | 1,632,588→1,538,386 | 236→185 | 606→583 | 未测 |
| 常用流程 | 5→5 | 19→16 | 1,573,936→1,517,905 | 266→236 | 539→556 | 26→27 |
| 新建流程 | 6→5 | 20→19 | 1,583,763→1,544,143 | 251→217 | 613→528 | 未测 |
| 运行记录 | 6→6 | 15→15 | 1,533,256→1,499,770 | 236→212 | 537→539 | 14→16 |
| 工具目录 | 5→5 | 13→13 | 1,496,567→1,497,049 | 155→155 | 526→574 | 未测 |
| 质量验收（含异常样本） | 7→7 | 16→16 | 1,582,221→1,552,160 | 130→96 | 814→536 | 未测 |
| 用量与额度 | 7→7 | 10→10 | 1,501,161→1,501,643 | 549→549 | 546→577 | 未测 |

质量页调整后 `sample=1`（原始 JSON 约第 689 行）仅有 4 次公共 API、6 个 JS 请求、1,479,197 解码字节、11 个主区域 DOM，`aiRequests=[]`，`readyMs=515.5`；与同页另两次 7 API/16 JS/96 DOM 的样本不一致。原始记录没有解释该异常，测量入口只等待 main 和 networkidle，也未断言质量页业务内容就绪。本次只保留并标注异常，未改原始值、未排除样本重算、未复测；不能把该汇总视为 3 个均有效质量页样本，或用它确认质量页性能与就绪时间提升。

其余历史汇总中首屏 API 和脚本请求数未增加，知识库工作区 JS 解码字节减少 94,202，流程列表减少 56,031。部分入口因公共导航变更增加不足 1 kB，不能描述为所有指标均下降。输入中位数约 13–27ms、变化约 1–2ms；API 就绪时间受本机服务波动影响，不据此声称普遍提速。DOM 减少不是网络收益的替代证据，未测输入不代表耗时为零。

原始 JSON 的 `beforeAssets` / `afterAssets` 包含构建资源文件名和字节数，可据其哈希化文件名定位当时产物，例如工作区 `KnowledgeWorkspacePage-6mU_CTFx.js` → `KnowledgeWorkspacePage-a9T6pB8F.js`，公共入口 `index-DVMs34yR.js` → `index-D27morXZ.js`；它们不包含独立构建时间戳或提交 SHA。布局截图的原始记录仅称保存于当次 Codex 可视化目录（知识库工作区、常用流程和运行记录各一张），未给出准确路径，不能将其当作仓库内现存附件引用。
