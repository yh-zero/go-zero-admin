# 知识库小规模性能基线

本轮在已有串行质量评测之外增加独立的 `1 / 3 / 5` 并发观测，已完成本地桩回归及本地开发栈的真实小规模抽测。业务题集与回答复核另见[真实项目资料业务基线](business-baseline.md)，本文单独记录并发测量，不混用串行质量评测或本地桩耗时。

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
node go-zero-admin/data/scripts/bench-kb-quality.mjs --run --cases ai_document/ai-rag/business-cases.json --case-id business-dev-independent-embedding --auth-file tmp/<本轮目录>/auth-private.json --kb-map tmp/<本轮目录>/kb-map.json --context-file tmp/<本轮目录>/context.json --output tmp/<本轮目录>/performance.jsonl
```

本机 API 默认 `http://127.0.0.1:7001`；`--base` 只接受本机 HTTP 根地址。`--context-file` 与[质量评测](quality-cases.md)使用同一已消毒标签结构，必须包含真实构建的 64 位 `binary_sha`、`model`、`embedding`、`config`。配置里不能带 Key、令牌、密码或邮箱。工具不会自动登录、刷新、重试、取消任务或清理数据库。

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

当前维护方式和按影响范围选择的复验入口见[开发流程](../dev-guide.md#3-日常开发与验证流程)、[用量治理](../dev-guide.md#用量治理与持久结构维护)和[Qdrant恢复](../dev-guide.md#qdrant-启动恢复与孤儿索引对账)。

## 本地回归与边界

### 2026-11-08 管理页面只读 HTTP 小样本

在 Windows 本地主栈、正常管理员会话下，按表中顺序分别串行请求10次，无并发、重试或模型调用。采集时间为上海时间02:00前后，新用量账本当时为空；流程列表含本轮临时流程，知识库6仍为原项目资料。耗时包括本机HTTP、完整响应读取和JSON解码；响应字节为解压后的实际UTF-8 body，不包含响应头。

| GET接口 | 成功/请求 | 首次（ms） | 中位（ms） | P95/最大（ms） | 响应字节 |
|---|---:|---:|---:|---:|---:|
| workflows，page=1/page_size=10 | 10/10 | 243.983 | 24.473 | 243.983 | 863 |
| usage/summary，当天 | 10/10 | 86.923 | 27.667 | 86.923 | 529 |
| kbs/6/evaluations，page=1/page_size=10 | 10/10 | 33.871 | 21.921 | 33.871 | 1081 |

30次均为HTTP及业务信封200。P95采用nearest-rank，10个样本的P95等于最大值；首个请求明显更慢。原始观测保存于本地`tmp/ai-product-20261107/improvements-http-baseline.json`，没有前后对照，不据此宣称性能提升、生产吞吐或稳定尾延迟。

本轮避免无效外部调用的可重复证据另见`TestSearchDoesNotEmbedUntilVectorDependencyReady`等向量就绪用例：初始索引不可用时，同一请求的查询Embedding从1次降为0次；恢复后正常调用1次。摄取同样在就绪前保持队列状态，恢复后1次Embedding/1次Upsert。这是故障路径的调用数改善，不能换算成正常模型响应提速。

工具最终本地回归 **17/17** 通过，根开发流程联合业务题集、离线转换和性能工具回归 **34/34** 通过，均无跳过。两个新增脚本的 `node --check` 通过。默认 CLI 在禁用 fetch 的进程内完成 0 HTTP 计划；这项桩/离线结果与上面的真实测量分开记录。Windows 默认沙箱最初因 junction 夹具 EPERM 未通过该项，随后在允许建立独立临时链接的环境实际验证通过。

回归补充了“有效结果已观察，末尾语料 GET、版本复核或日志落盘/关闭失败”的场景：已知聊天调用与 tokens 继续保留，lane 的 `completed=false`、`evidence_status=invalid`，不进入成功耗时或质量统计。用量不完整单独标记，既不丢弃已知部分，也不按零费用解释。

可重复回归命令：

```powershell
node --test go-zero-admin/data/scripts/bench-kb-quality.test.mjs
```

回归覆盖 0 HTTP 计划、固定 9 次请求及唯一 UUID/任务 ID、桩并发峰值、各波次样本数、HTTP/元数据计数、未知提交不重放、失败终态、重复任务、跨波次语料变化、日志持久化失败、输出不覆盖及链接越界。Windows 沙箱可能限制 junction 夹具，需在允许创建独立临时链接的测试环境实际验证，不能把 EPERM 写成通过。
