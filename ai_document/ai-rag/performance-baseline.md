# 知识库与前端性能基线

各节使用独立的构建、语料和测量口径。复验时保留原始记录，另写新文件。

## 固定测量范围

- 必须显式选择一条 `development` 正例。同一问题依次运行 1、3、5 个客户端工作，共 9 次重复，不消耗新的 `validation` 保留集。
- 每次工作执行一次正式资料检索，再提交一次单步骤 `kb_analyze` 草稿任务。正常路径共 9 次 search POST、9 次分析 POST，预计 18 次逻辑查询 Embedding；供应商的实际 HTTP 尝试、重试及费用不能由此直接推算。
- 每个波次等全部工作完成语料预检后一起放行。每个 lane 仍单独保留原版本的严格串行质量日志，不修改原评测脚本或将交错事件塞入 v1 日志。
- 每个 lane 都核对语料前后哈希、版本、ready 状态及库存；所有 lane 与首个已核对语料指纹保持一致。额外元数据 HTTP 次数单列，不能隐藏在“只有 18 次请求”的描述中。
- 模型、Embedding、预算、TopK 和构建摘要由操作者核对实际加载配置后提供。认证文件只用于正常登录，不进入报告。

## 执行方式

从仓库根目录执行，先创建被 Git 忽略的输出父目录。`--plan` 为默认行为，不发 HTTP，也不创建任务：

```powershell
node go-zero-admin/data/scripts/bench-kb-quality.mjs --plan --cases ai_document/ai-rag/business-cases.json
```

确认选择一条开发正例后显式执行。以下选择资料存储和独立 Embedding 配置题；认证、映射、上下文和输出需替换为本轮实际路径：

```powershell
node go-zero-admin/data/scripts/bench-kb-quality.mjs --run --cases ai_document/ai-rag/business-cases.json --case-id business-dev-independent-embedding --auth-file tmp/perf-run/auth-private.json --kb-map tmp/perf-run/kb-map.json --context-file tmp/perf-run/context.json --output tmp/perf-run/performance.jsonl
```

本机 API 默认 `http://127.0.0.1:7001`；`--base` 只接受本机 HTTP 根地址。`--context-file` 与[质量评测](README.md#检索与回答质量评测)使用同一已消毒标签结构，必须包含真实构建的 64 位 `binary_sha`、`model`、`embedding`、`config`。配置里不能带 Key、令牌、密码或邮箱。工具不会自动登录、刷新、重试、取消任务或清理数据库。

工具独占创建 11 个新文件，全部名称先检查，不覆盖已有记录：

- `performance.jsonl`：独立性能 manifest，包含固定题目/题集 SHA、构建配置、工具摘要、各波次及 lane 的结果、原 UUID、任务 ID、HTTP 计数、已知聊天用量和日志摘要。
- `performance.md`：同次观测生成的简要表格，分开列各波次成功数与样本数。
- `performance.c1.l1.jsonl`、`performance.c3.l1.jsonl` 至 `performance.c3.l3.jsonl`、`performance.c5.l1.jsonl` 至 `performance.c5.l5.jsonl`：9 份原 v1 串行质量日志，保留回答、实际来源、原载荷与前后语料核对记录。后续波次未运行时预留文件为空，不能当完成日志。

遇到提交超时等未知结果时保留原 UUID 和载荷；有接受响应时保留任务 ID。等待同波次已开始的工作收束后停止下一波，退出非零。日志写失败时，manifest 标记日志不可校验、摘要为空，并保留已观察到的 ID；不把不完整记录报成有效样本。运行中断后按精确 ID/UUID/owner 核对服务端结果，不换 UUID 自动重放。

## 指标口径

| 指标 | 含义与边界 |
|---|---|
| `search_client_ms` | 正式 search POST 的客户端请求往返耗时 |
| `search_server_ms` | search 响应中的现有服务端 `elapsed_ms`，不是模型分阶段耗时 |
| `analysis_elapsed_ms` | 分析 POST 发起至 GET 观察到终态，含提交、服务端排队、模型和轮询误差 |
| `observed_active_lanes_peak` | 放行后客户端工作未返回结果/失败的最大数量，不等于服务端实际模型并发 |
| `observed_accepted_waiters_peak` | 已观察到接受响应、尚未观察到终态的客户端计数峰值 |
| `workload_wall_ms` | 全部 lane 语料预检完成后放行至最后结果/失败，含正常执行期间日志等客户端开销 |
| `full_wave_wall_ms` | 整个波次的观测耗时，另含预检、复核及报告落盘 |
| `metadata_http_requests` | 文档分页、正文等语料核对请求，与检索/提交/轮询次数分开 |
| `accepted_task_responses / unique_task_ids` | 接受响应数量与去重后的已知任务数量；收到重复任务 ID 会判定本轮失败 |
| `known_chat_total_tokens` | 观察到有效 result 时即保留供应商已返回的聊天 token；后续语料或日志复核失败也不丢弃，不含 Embedding 费用 |
| `analysis_observed / evidence_status` | 有效分析结果是否已观察到，与本 lane 的证据是否核验通过分别记录；证据无效的 lane 不进入成功耗时或质量统计 |
| `unknown_usage_lanes` | 已提交分析但尚无有效结果，或已知聊天 usage 不完整的 lane 数；预检未提交、明确 no_matches 均不算未知聊天用量，不能由证据无效推断用量未知 |

P50/P95 用 nearest-rank，分别注明成功样本 `N=1/3/5`；没有成功样本时为 `null`，不能写零耗时。失败工作不混入成功分位数，但保留失败数量与原记录。这个样本量不代表生产容量、稳定尾延迟、服务最大并发或完整语义质量；固定波次顺序还可能受到缓存、供应商预热和网络波动影响。没有高精度内部埋点，不拆出精确排队或纯模型耗时。

## 本轮真实观测（2026-11-07）

固定题目为 `business-dev-independent-embedding`：询问聊天使用 DeepSeek 时的 Embedding 模型及 MySQL/Qdrant 数据分工，属于 `development` 正例。题集 SHA256 为 `f3a4016eabc10a974e3acffc6d7347477976d00923c196c608de55a6db99538b`。三个波次合计 **9/9** 完成，分别观察到客户端活动 lane 峰值 1、3、5；这仍然只有 **1 个独立问题**，不是新增 9 道质量验收题。

实际聊天模型为 `deepseek / deepseek-chat`，查询 Embedding 为 `qwen / text-embedding-v4`、1024 维；上下文预算 32000、回答上限 2048 tokens、召回 Top20 后返回 Top5，分析 temperature=0。运行 AI RPC 二进制 SHA256 为 `86698e4d8a9d8d0cb93cf706fed7317e7c136fa8e6673f3042d75d54056551c2`，宿主与容器摘要一致。

| 目标并发 | 成功/尝试 | 检索客户端 P50/P95（ms） | 分析提交到观察终态 P50/P95（ms） | 客户端活动峰值 | 元数据 HTTP |
|---:|---:|---:|---:|---:|---:|
| 1 | 1/1 | 531.89 / 531.89（N=1） | 1674.33 / 1674.33（N=1） | 1 | 10 |
| 3 | 3/3 | 231.13 / 485.58（N=3） | 2401.96 / 2444.60（N=3） | 3 | 30 |
| 5 | 5/5 | 287.82 / 315.45（N=5） | 2043.70 / 2087.78（N=5） | 5 | 50 |

本次实际 HTTP 为 **136 次**：9 次 search POST、9 次分析 POST、28 次任务 GET、90 次语料元数据/正文核对 GET，HTTP 失败为 0；9 个原 UUID 和任务 ID 均唯一。已知聊天调用 9 次，已知聊天总用量 **14768 tokens**，9 份 usage 均完整。正常路径的逻辑查询 Embedding 上限为 18 次，**不含资料摄取的 Embedding**；这里不把逻辑次数换算为供应商实际 HTTP 次数、重试次数或费用。

每份 lane 日志均保留原始回答、实际来源及前后语料核对。9 次观察均未触发词法规则标记、截断或 no_matches，这不等于语义准确率 100%。同题重复与固定 1→3→5 波次顺序可能受到缓存、供应商预热、网络波动和轮询误差影响，不能据此认定 5 并发更快、获得稳定尾延迟或已经测出生产容量；也不能把分析耗时拆成纯模型耗时或精确排队时间。

原始记录保留在本地被忽略的目录，不写入认证信息：

- `tmp/ai-product-20261107/feedback-performance.jsonl`：独立性能 manifest，保留波次、原 ID、完整精度耗时、配置、各 lane 日志 SHA256 和调用计数。
- `tmp/ai-product-20261107/feedback-performance.md`：本次观测生成的简要报告。
- 同目录下 `feedback-performance.c1.l1.jsonl`、`feedback-performance.c3.l1.jsonl` 至 `feedback-performance.c3.l3.jsonl`、`feedback-performance.c5.l1.jsonl` 至 `feedback-performance.c5.l5.jsonl`：9 份原 v1 串行质量日志。

## 历史工作区投影与请求量

以下观测从开发手册归档，适用各自注明的历史版本与样本，属于不同批次，不能合并为同一轮测量或汇总成性能提升。

- **此前同步摄取版本**：相同 owner/KB/软删过滤下，11篇演示资料的列表 CLI UTF-8 序列化结果从13095降到914字节（剔除12181字节正文），COUNT+分页仍为2次SQL；单文档状态只查1次元信息。任务摘要固定夹具从197068降到80字节，仍为2次SQL；后台取消观察每秒只读status。阶段可见性当时增加一次摄取CAS事务。这些历史数据不作为后续后台批次、搜索字段或轮询协调实现的性能验收，也不证明模型时延或并发吞吐提升；这些行为需独立数据集与隔离负载重新测量。
- **2026-11-07工作区版本**：两篇合成文本的后台接收回执分别约471/573毫秒，接收耗时不包含Embedding完成；真实浏览器就绪原文和终态任务各为一次完整详情GET、零额外meta GET。普通任务详情由原先两次SELECT改为一次，超时状态恢复分支例外；流程列表不返回定义，同批三条流程详情合计1166字节定义从列表省去。KB计数会增加统计查询工作，检索的Embedding/向量/原文调用链没有减少，不能将这些局部证据解释为模型提速或生产吞吐结论。
- **用量治理保留样本**：一题公开项目反馈记录3次实际供应商请求（1聊天/2Embedding）、已知合计4596 tokens；这是本地主栈保留样本，不是供应商账单或生产容量验收。
- **孤儿索引对账小集合**：真实验证自动删除1个专用孤立点，原40个点及payload保持。恢复、代次竞争及失败另有本地桩/隔离回归，持续真实故障、大集合和生产恢复未由该小样本覆盖；对账不补缺失向量、不重放Embedding，也不自动清空集合。

## 2026-11-08 管理页面只读 HTTP 小样本

在 Windows 本地主栈、正常管理员会话下，按表中顺序分别串行请求10次，无并发、重试或模型调用。采集时间为上海时间02:00前后，新用量账本当时为空；流程列表含本轮临时流程，知识库6仍为原项目资料。耗时包括本机HTTP、完整响应读取和JSON解码；响应字节为解压后的实际UTF-8 body，不包含响应头。

| GET接口 | 成功/请求 | 首次（ms） | 中位（ms） | P95/最大（ms） | 响应字节 |
|---|---:|---:|---:|---:|---:|
| workflows，page=1/page_size=10 | 10/10 | 243.983 | 24.473 | 243.983 | 863 |
| usage/summary，当天 | 10/10 | 86.923 | 27.667 | 86.923 | 529 |
| kbs/6/evaluations，page=1/page_size=10 | 10/10 | 33.871 | 21.921 | 33.871 | 1081 |

30次均为HTTP及业务信封200。P95采用nearest-rank，10个样本的P95等于最大值；首个请求明显更慢。原始观测保存于本地`tmp/ai-product-20261107/improvements-http-baseline.json`，没有前后对照，不据此宣称性能提升、生产吞吐或稳定尾延迟。

本轮避免无效外部调用的可重复证据另见`TestSearchDoesNotEmbedUntilVectorDependencyReady`等向量就绪用例：初始索引不可用时，同一请求的查询Embedding从1次降为0次；恢复后正常调用1次。摄取同样在就绪前保持队列状态，恢复后1次Embedding/1次Upsert。这是故障路径的调用数改善，不能换算成正常模型响应提速。

评测器自身的本地桩回归可从根目录运行 `node --test go-zero-admin/data/scripts/bench-kb-quality.test.mjs`。

## 前端首屏与交互

2026-11-11 采用同一浏览器、账号、数据与本机服务，前后生产预览各测 9 页，每页 3 次冷缓存取中位数。CDP 禁用缓存，首屏 API 含 4 次公共初始化。JS 字节使用浏览器资源计时中的 `decodedBodySize`，API 就绪取最大 `responseEnd`，DOM 取主区域后代节点数，输入测量 5 次填入和清空草稿。原始 `beforeAssets`、`afterAssets` 记录资源名与字节数。源码提交 `f0f50b7` 可定位当日改动，独立受测构建 SHA 和构建时间未记录。[原始样本、汇总与资源清单](../../go-zero-admin-react/docs/superpowers/plans/2026-11-11-ai-module-performance.json)

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

质量页调整后的第 1 个样本只有 4 次公共 API、6 个 JS、1,479,197 解码字节、11 个 DOM，`aiRequests=[]`、`readyMs=515.5`。另外两次各有 7 API、16 JS 和 96 个 DOM。脚本只等待主区与网络空闲，未断言质量页业务内容就绪，异常原因未知。表中保留原汇总，不能用此差异确认质量页提速。其他页面中，工作区与流程列表 JS 分别减少 94,202 和 56,031 字节。输入耗时中位数为 13–27 ms，部分页面 API 就绪时间增加。

在前端目录对已登录的独立生产预览运行 `playwright-cli run-code --filename scripts/aiUiPerformance.browser.mjs --raw`。历史基线使用 `127.0.0.1:6091`，候选使用 `localhost:6092`，知识库 ID 为 7。脚本执行页面导航和搜索草稿输入。[浏览器测量脚本](../../go-zero-admin-react/scripts/aiUiPerformance.browser.mjs)

## 查询缓存与队列

2026-11-17 对同一库的两篇第 1 版就绪资料，用同一问题分别串行查询 3 次。两次测量的正文、分块、排序、分数与返回片段指纹一致。旧、新 AI RPC 产物 SHA256 分别为 `f46b3353ba8792eccea33be59f792d6fa681b489e4d29b6497969ec99f798b63`、`96b5294a73e292c47918d0be3a02754d5db02c4c535cccf4a35c725ef388a7bc`。模型与额度为 DeepSeek `deepseek-chat`、Qwen `text-embedding-v4` 1024 维、32,000 上下文及 2,048 输出上限。

| 指标 | 原版本 | 新版本 |
|---|---:|---:|
| 客户端第 1 / 2 / 3 次耗时（ms） | 1082.772 / 536.713 / 482.718 | 464.785 / 46.785 / 44.206 |
| 三次中位耗时（ms） | 536.713 | 46.785 |
| 真实查询 Embedding 账目 | 3 次，45 已知 tokens | 1 次，15 已知 tokens |
| 直接检索片段 | 同 2 块、同分数与正文 | 同 2 块、同分数与正文 |
| 空闲队列采样 | 10.546735 秒，84 Expired + 84 Claim | 10.290196 秒，2 Expired + 10 Claim |
| 对应空闲 SQL 速率 | 15.929 次/秒 | 1.166 次/秒 |

空闲 SQL 时段取自计数器同一次 MySQL 查询。预热、网络和只测一个问题限制了外推。原始对比在本机 `tmp/kb-opt-20261117/comparison.json` 及其引用的 4 份 JSON。单次实际分析任务 470 引用了两个可逐字定位的片段，聊天 1,063 个已知 token、查询 Embedding 15 个。对应截图在同目录 `analysis-preview.jpg` 与 `analysis-result.jpg`。

前端桩观察到快速输入 3 次发 1 次搜索、重复解码同一题集计算 1 次 SHA、同一库选择器两处挂载共用 1 次请求。任务详情隐藏 51.408 秒时计数保持 10，展开后续至 23。本机保存 `tmp/kb-review-20261117/browser-{active-start,hidden-observed,lifecycle-final}.json`。隔离 MySQL、Redis、Qdrant 下的缓存回归记录 300 秒 TTL、8 次检索触发 1 次本地 Embedding，过期资料仍由 MySQL 过滤。跨进程恢复仅观测到 queued 作业恢复，running 中断由租约用例覆盖。

## 资料改名与分块

2026-11-18 的单分块资料改名试验，对照 `5557c49` 的旧更新路径和当时工作树，使用相同的隔离 MySQL/Redis 及 Embedding/Qdrant 计数桩。改名新增的 Embedding 调用由 1 次变为 0 次，向量写入由 1 次变为 0 次，原分块 ID 保留数由 0/1 变为 1/1。正文与模型均相同、文档已就绪是复用条件。

分块对照使用[8 篇冻结原文](business-corpus-provenance.json)和运行前拟定的 9 道本地题。现行 500 字符、重叠 50 的硬切与同上限、同重叠的优先换行策略使用相同的字符二元组余弦桩。

| 指标 | 当前策略 | 优先换行 |
| --- | ---: | ---: |
| 分块数 | 24 | 27 |
| 至少在一块内完整保留的非空原文行 | 123/132 | 129/132 |
| 含重叠的实际输入字符数 | 10,062 | 10,196 |
| 按每篇每批最多 10 块估算的 Embedding 批数 | 8 | 8 |
| 文档 Recall@5 | 9/9 | 9/9 |
| 目标文档的完整证据行进入前五 | 7/9 | 6/9 |
| 本机无缓存检索批均耗时的 P95 | 70.892 µs | 96.190 µs |

计时为 9 题各 4 批、每批 64 次，P95 取 36 个批平均值。优先换行多保留 6 行，但前五名中完整证据少 1 题，候选仅保留在实验里。此实验未调用模型，对供应商 token、网络延迟或生产流量没有测量。后端目录运行以下两项，第二项需要显式配置隔离 MySQL 和 Redis 及摄取表。

```sh
go test ./application/ai/rpc/internal/logic/aiservice -run '^TestKbChunkStructureComparison$' -count=1 -v
go test ./application/ai/rpc/internal/logic/aiservice -run '^(TestKb(Documents|Background).*|TestKbChunkStructureComparison|TestChunkText.*)$' -count=1 -v
```

## 工作区轻量读取

2026-11-18，本机已登录会话逐项读取同一资源的完整与轻量投影。下表单位为 HTTP JSON 正文字节。

| 实际读取 | 完整 | 轻量 |
|---|---:|---:|
| 5 个知识库的管理列表 / 选库目录 | 1,641 | 289 |
| 6 条资料的当前页 / 同页变更探测 | 1,446 | 199 |
| 4 份内置题集的完整目录 / 元信息 | 257,335 | 674 |
| 已完成评测的完整详情 / 已知观测指纹的进度 | 90,297 | 655 |

按需加载一份题集为 105,622 字节，题集 SHA 与完整目录一致。资料列表计数和行读取由 3 条 SQL 合为一条快照查询。一次真实导航中详情和资料页各请求 1 次，两者启动相差 0.4 ms、重叠 22.1 ms。用量首屏读取一次摘要和一次额度策略，展开明细再发一次请求。仅测本机给定库与页码。该轮 AI RPC SHA 为 `bf3263e96907394f15c2e922688e12d38228f8dde8ef76b7b81c206c8f130416`。

## 单篇问答与原文定位

2026-11-18，隔离服务使用同库两份分别规定 300 元和 800 元的资料。800 元资料有 22 个更高分候选，其中一条向量的文档标识与原块不一致。限定 300 元资料仍命中原条款、原分块 ID 及第 7 版；限定 800 元仅返回对应第 2 版。独立 Chrome 桩以 400/900 元资料验证首问、追问、原请求重查与引用定位。开发栈以只读方式打开既有资料和历史引用。当时 API SHA 为 `65317b52d6c1e16a2da8bfe3355f626ec175d4229f34b8cc616d33d67ddd9f5e`，AI RPC 为 `e4c10b0bca2fbc4096d613960e3399ca7874d2145e84a5918c118778fa4ef6ee`。[单篇问答浏览器回归](../../go-zero-admin-react/scripts/aiChatDocumentScope.browser.js)

从后端目录复测。前两项涉及集成时，将 `GOZERO_AI_TEST_DSN`、`GOZERO_AI_TEST_REDIS` 指向已初始化的隔离实例并核对跳过项。最后一项还需要 `GOZERO_QDRANT_TEST_ADDR`。

```powershell
go test ./pkg/agenttools ./application/ai/rpc/internal/logic/aiservice -run 'TestKbCitations|TestKbSearch|TestContextBudget|TestAIStream(NumberedKbToolRounds|Budget)|TestGroundedReference' -count=1
go test ./application/ai/rpc/internal/logic/aiservice -run 'TestChatDocumentScope|TestChatExperience|TestAIStreamIntegration' -count=1 -v
go test ./application/ai/rpc/internal/svc -run TestKbQueryVectorCacheRealDependenciesRecheckFacts -count=1 -v
```
