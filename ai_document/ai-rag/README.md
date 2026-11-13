# ai-rag — 知识库联调素材（演示合成数据）

## 内容

| 目录 | 知识库 | 文件类型覆盖 |
|---|---|---|
| kb1-a股投资知识/ | A 股投资知识（交易规则、板块、两融、分红、指标） | md ×2、txt ×2、csv ×1、json ×1 |
| kb2-运维发布SOP/ | 运维与发布 SOP（发布、值班、变更窗口、告警、DB 变更） | md ×2、txt ×1、csv ×1、json ×1 |
| test-questions.md | 检索测试问题集（每库命中题 + 跨库隔离/注入/降级边界题） | — |

## 使用方式（文本批次导入）

共 11 篇素材（投资知识 6 篇、运维 SOP 5 篇），通过知识库工作区提交后台文本批次，每篇正文最多 64 KiB。批次回执、停止与未知结果核对见[当前操作说明](../../README.md#从资料到有来源的回答)。

1. 登录后台（admin）→ AI 功能 → 知识库，新建两个知识库：“A股投资知识”“运维发布SOP”。
2. 进入各知识库工作区，用“粘贴资料”把每个文件的正文**全文复制**为一篇文档，包括 CSV/JSON 的完整文本；TXT/Markdown 也可从文件导入。标题均用文件名去掉序号与扩展名，保持题集的标题绑定。
3. 从“查看导入批次”核对处理结果，资料变为“可使用”后，按 [test-questions.md](test-questions.md) 在 AI 对话中逐题验证。

## 直接召回测试

在知识库列表点击“召回测试”，输入查询即可查看实际命中的文档、分块、原文片段、相似度和检索耗时。它复用对话使用的检索器，不调用聊天模型；查询最多 2000 UTF-8 字节。当前没有最低相似度阈值，有返回片段不代表资料有答案，分数也不等于答案正确率。

固定题集 [recall-cases.json](recall-cases.json) 含 12 个命中题和 2 个指定库无答案题，覆盖上述 11 篇资料。以下命令从仓库根目录执行：

```powershell
# 默认只校验素材与展示计划，不请求服务
node go-zero-admin/data/scripts/eval-kb-recall.mjs --plan

# 使用正常登录的 result 保存为忽略文件 tmp/auth-private.json；不得提交会话令牌
# tmp/kb-map.json 填实际知识库 ID，例如 {"stocks":1,"ops":2}
node go-zero-admin/data/scripts/eval-kb-recall.mjs --run --auth-file tmp/auth-private.json --kb-map tmp/kb-map.json --output tmp/recall-20261107.jsonl

# 评测工具回归使用内存 HTTP 桩，不调用真实模型
node --test go-zero-admin/data/scripts/eval-kb-recall.test.mjs
```

也可用 `--manifest <导入记录.json>` 替代 `--kb-map`，按来源、哈希和文档 ID 绑定；两种方式都会用正式分页列表确认文档就绪，HTTP 不提供正文哈希，不能把导入记录当成本轮重读运行库正文。每次结果文件必须是 `tmp/` 下新文件；逐题追加 JSONL，失败保留已完成结果并停止，不自动重试。继续前核对旧结果，用 `--case-ids <尚未执行的题目ID,另一个ID>` 显式选择题目及新输出文件。一次完整评测最多 14 次查询 Embedding 调用，无聊天调用。

指标中的 K 是实际返回的前 K 个片段；本题集每个正例仅一个相关文档，Hit@K 与 Recall@K 同值，另报 MRR@5 和正确文档内的证据字符串覆盖。无答案题单独记录返回数量、最高分和知识库归属，不纳入正例分母，也不将其当作生成答案/拒答评测。小库命中结果不能代表生产准确率或并发性能；相似度阈值需扩大正负例及独立验证集后再确定。

## 检索与回答质量评测

[quality-cases.json](quality-cases.json) 在同一批11篇素材上固定40题：development24题、validation16题；题目分类、标注口径见 [quality-cases.md](quality-cases.md)。分组保留首轮设计；当前这些题已用于复核和失败修复，应作为历史回归，后续独立验证须另留新的业务样本。

本地首轮测量、失败样本及仍存问题见 [质量与耗时基线](quality-baseline.md)。这批验证题已用于回退核对，后续调优需另留新验证样本。

语料按原始字节校验SHA256：[.gitattributes](../../.gitattributes)固定两个演示素材目录以CRLF检出、根层题集JSON以LF检出，避免不同平台的自动换行改变冻结指纹。已有工作树若仍校验失败，先核对本地差异并与新的隔离检出比对，不自动覆盖本地资料；该规则不改变运行库正文或历史报告。

```powershell
# 默认只校验题集与素材，不请求服务；也可选择 --split development 或 --case-ids <ID,ID>
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan

# auth仍使用正常登录result；kb-map填本人知识库ID，context只放核对后的构建/模型/Embedding/预算标签
node go-zero-admin/data/scripts/eval-kb-quality.mjs --run --auth-file tmp/auth-private.json --kb-map tmp/kb-map.json --context-file tmp/quality-baseline-context.json --phase baseline --output tmp/quality-baseline.jsonl
node go-zero-admin/data/scripts/eval-kb-quality.mjs --run --auth-file tmp/auth-private.json --kb-map tmp/kb-map.json --context-file tmp/quality-candidate-context.json --phase candidate --output tmp/quality-candidate.jsonl

# 离线导出及比较，不产生模型调用；使用自定义题集时三种模式都传同一个 --cases <路径>
node go-zero-admin/data/scripts/eval-kb-quality.mjs --report tmp/quality-baseline.jsonl --output tmp/quality-baseline.md
node go-zero-admin/data/scripts/eval-kb-quality.mjs --compare tmp/quality-baseline.jsonl tmp/quality-candidate.jsonl --output tmp/quality-compare.md
node --test go-zero-admin/data/scripts/eval-kb-quality.test.mjs go-zero-admin/data/scripts/eval-kb-recall.test.mjs
```

context JSON记录 `binary_sha`、`source_revision`、`model:{provider,name}`、`embedding:{provider,model,dimension}`、`config:{max_context_tokens,analysis_answer_token_cap,recall_top_k,return_top_k}` 和可选 `experiment:{label,change}`，不得放密钥、密码、令牌或邮箱。工具不读取环境密钥；由执行者先核对实际运行配置及构建，标签不替代运行环境核验。

`binary_sha`须为64位十六进制摘要；模型/Embedding字段须为非空对象和名称，维度、预算及TopK须为正安全整数，回答预算和返回TopK不超过对应上限，不能用空对象或字符串代替完整标签。温度标签允许非负有限数或`provider_default`，旧日志可保留省略值，但省略与显式默认不自动视为相同配置。

每题串行请求一次正式search和一次draft任务的kb_analyze，正常路径最多两次查询Embedding、一次有界聊天；完整40题最多40次分析、80次查询Embedding，不追加LLM裁判。调用前读取正式分页列表及全文，核对唯一标题、原文SHA256、版本、Embedding模型和ready；结束再次核对全文与库存，变化则本轮不可比较。JSONL在POST前持久化原UUID/载荷，后续仅GET轮询原任务，最多100秒。失败或未知结果停止并保留已有记录，不自动重试；新输出必须位于忽略的 `tmp/`，拒绝覆盖历史文件。

Hit/Recall/MRR@1/3/5按原始前K片段排名计算，文档Recall去重并使用全部预期文档作分母；证据组只在对应文档匹配。无答案题单列。回答筛查必需事实、引用编号、实际来源归属、已引用片段证据覆盖、资料不足措辞及禁答短语，保存实际答案、来源和待复核原因；规则未标记也需人工复核，不能称为语义准确率或人审通过率。

报告按split/category列出成功题指标；失败不算空召回或错误回答。耗时区分search client/server与分析提交到终态，P50/P95注明串行经验分位数和N，不代表生产并发容量。chat tokens仅为供应商已知聊天用量，不含Embedding费用；正常调用上限和成功分析推定次数不等于供应商实测计费次数。比较前用同SHA256冻结题集及原始记录复算，要求同题、同语料版本和同模型/预算/TopK；仅构建、源码版本及实验说明可变化，失败、不完整或记录不一致时拒绝计算提升。JSONL不带数字签名，一致性核对不能证明日志来源未被修改。

## 知识库质量验收页与24题项目场景

从 **AI 功能 → 知识库 → 本人知识库工作区 → 质量验收** 进入 `/ai/knowledge/:id/quality`。页面提供[24题原场景](business-acceptance-cases.md)、[24题证据覆盖验收校准版](business-evidence-cases.md)、[32题历史业务题](business-cases.md)和[14题失败回归](business-failure-regression-cases.md)，适合在同一批资料上核对回答、实际引用、关键条件与风险，以及检索和分析耗时。题目基于仓库公开项目文档，由AI辅助编制，不是真实用户历史，也没有真人签署。

24题复用 `business-docs/` 中8篇冻结资料：16题开发、8题初始预留，运维与AI题组各12题。`project_ops`、`project_ai`分别对应正文及SHA一致的本人知识库；已有相同资料可直接复用。题集原始SHA为 `176e20b4563d8c892b375fe45af89a4a27b78ee1d62c9686e9b49b3956696760`。这8道初始预留题已在本轮实际执行并用于复核，现为已使用回归数据；页面保留的“初始预留题”标签不代表未见过的独立验证集。实际观察与剩余问题见[业务基线续记](business-baseline.md#质量验收页与24题场景的中间观察)。

1. 选择题集、与当前资料对应的题组及1..30道题。读取使用 `sys:ai:kb:list`，开始和停止评测另需 `sys:ai:kb:evaluate`；页面入口可见不代替服务端权限及知识库归属检查。模型与检索配置齐备、资料正文/版本/ready/库存核对一致后才能运行。
2. 点击“开始评测”创建持久化批次，每批串行执行。正常每题包含一次直接检索和一次分析内部检索，最多两次逻辑查询Embedding及一次有界聊天；真实运行按已配置供应商计费。执行失败会停止余题；“停止后续题目”仅阻止未开始项，已发生的调用不会撤回。
3. 刷新或返回页面后从“历史批次”查看进度。提交待确认时先“核对原请求”，保留原UUID、题目与载荷；需要继续原提交时使用原标识，不自动换UUID重跑。逐题结果未知时先按请求标识核对原运行记录，不能直接单题复测；明确进行单题复测会新建独立批次并产生新的调用。
4. 展开题目，对照实际答案、实际来源和中文检查项。执行成功只代表任务完成；规则是字符串筛查，0警报也不等于事实完整、语义正确或人审通过。执行失败/未知单列，不算空召回或错误答案，未取得的用量不能按零解释。成功分析可使用原有评价入口加入反馈复核；缺少来源时不能补造引用。
5. 在完整成功、语料前后复核通过的历史批次中“设为对照”。只有题集SHA、题目集合、题组、知识库、语料版本/正文指纹/库存及模型、Embedding、预算、TopK等context一致时才比较；构建变化单列。不完整、失败、未知或配置不同的批次不能计算提升，也不能把不同批次的成功片段拼成一轮完整结果。

CLI仍可使用同一份题集，离线计划不会请求服务：

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/business-acceptance-cases.json --split development
```

页面批次保存内部调用的观测，CLI保存客户端HTTP事件，两种记录不能冒充相同的HTTP日志。真实模型效果、费用与小样本耗时必须以对应构建的实际结果为准；本轮中间结果不代表后续构建已通过验收。

## 关键证据覆盖与完整题集基线

直接召回测试与普通 `kb_search` 仍返回最多5个片段，Hit/Recall及证据覆盖@5描述这组直接检索结果。工作流 `kb_analyze` 在一次分析中可使用同次向量召回、归属/ready/软删复核后的最多20个真实候选；候选来源JSON不超过24000 UTF-8字节及上下文四分之一的本地估算额度，完整请求仍保留问题与回答空间。模型只调用一次，最终保留实际引用的最多8个片段并连续编号，来源JSON不超过10000字节，工具整体不超过24000字节；所有来源仍是实际送入模型的片段，不拼接其他分块冒充原chunk。引用过多或输出无效安全失败，不静默删引用或自动追加模型调用。

因此，**直接检索证据@5**与**分析实际引用证据覆盖**需要分开读取。前者未改善不等于分析没有取得更多依据，后者有引用也不证明每句话正确。扩大候选可能增加同一次聊天的输入tokens，需和已知用量、耗时一并衡量。

[派生校准24题](business-evidence-cases.md)保留原问题、split及8篇文档指纹，把原问未要求的产物格式、额外状态码、验证接口等降为补充复核，并记录每项旧值、新值和依据；原24题与历史警报保留。源码与环境变量更新、宿主与Compose配置路径两类真实失败的必答要求没有删改。校准也是AI辅助判读，仍需人工复核。

`eval-kb-cohort.mjs`用于同一完整题集的串行前后基线。与页面批次和原质量CLI不同，它仅在已有合法receipt及明确失败终态时记录失败、继续尚未开始的下一题；网络、超时、结果未知、语料变化或日志写入/关闭失败会停止，禁止自动重试。每题保留独立v1原始日志，整轮另存manifest；所有输出在请求前独占预留，存在旧文件时拒绝运行。

以下命令从仓库根目录执行；正常登录的result保存在被忽略的认证文件，知识库映射填写实际的 `project_ops`、`project_ai` ID，context只放已核对的脱敏运行标签：

```powershell
# 默认与显式plan都只校验素材、展示24题计划，0 HTTP
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --plan --cases ai_document/ai-rag/business-evidence-cases.json

# 真实调用：每题最多一次分析，正常包含直接检索和分析内部检索
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --run --phase baseline --cases ai_document/ai-rag/business-evidence-cases.json --auth-file tmp/auth-private.json --kb-map tmp/project-kb-map.json --context-file tmp/evidence-baseline-context.json --output tmp/evidence-baseline.jsonl
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --run --phase candidate --cases ai_document/ai-rag/business-evidence-cases.json --auth-file tmp/auth-private.json --kb-map tmp/project-kb-map.json --context-file tmp/evidence-candidate-context.json --output tmp/evidence-candidate.jsonl

# 只读复算/比较，JSON结果输出到终端，不调用服务或模型
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --report tmp/evidence-baseline.jsonl --cases ai_document/ai-rag/business-evidence-cases.json
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --compare tmp/evidence-baseline.jsonl tmp/evidence-candidate.jsonl --cases ai_document/ai-rag/business-evidence-cases.json
node --test go-zero-admin/data/scripts/eval-kb-cohort.test.mjs
```

全题都有已知成功/失败终态、每题原始链合法、整轮资料库存/正文SHA/版本/状态一致时，才记为 `complete_cohort=true`。报告分别显示全题执行成功率、成功题质量分母、分析已引用证据与必答事实字符串覆盖，以及共同成功子集；失败不是空召回或错误答案，未知用量不按零费用。`http`仅为此CLI实际HTTP尝试，包含语料核对与轮询，不等于Embedding或供应商尝试数。页面持久批次仍执行失败即停，不因CLI能继续而改变。

比较要求同题集SHA、全部题目顺序、KB映射、语料指纹/库存及模型、Embedding、上下文与输出预算；构建和明确记录的实验变量单列。候选与最终来源限额变化须写入实验说明，不能称作所有检索条件完全相同。共同成功子集或不同轮次的成功记录不能拼成另一轮完整结果；字符串覆盖和小样本P50/P95都不是人工正确率或生产容量。

## 回答一致性失败回归

实际项目场景另见 [32题业务题集](business-cases.md)：8份冻结项目文档、20题开发与12题初始预留，明确AI辅助编题及首次使用边界。知识库分析评价可在“反馈与复核”中补充预期答案和逐字依据，采纳后下载；`merge-kb-feedback.mjs`离线核对KB映射、原文指纹与事实项，写出新题集后复用现有质量评测。聊天原有评价暂不合并为知识库复核记录。

[性能基线](performance-baseline.md)使用独立manifest及每工作原v1日志，显式选择开发题依次观测1/3/5并发，共9次分析。默认plan不发HTTP，真实运行按已配置供应商计费，单列元数据请求、已知聊天用量及不完整证据；不把并发日志伪装为原工具的串行质量记录。

[12题开发回归](answer-regression-cases.md)覆盖原问重复、改写和部分缺资料，[实测夹具](answer-failure-fixtures.json)保留坏答与实际来源。原40题不改；新题是已使用的开发数据，不是新独立验证集。

[14题业务失败回归](business-failure-regression-cases.md)覆盖跨文档漏召回、背景引述的额外不足，以及真正要求回答的引号内问题。复用8篇冻结项目文档，保留任务178/190的原始坏答和实际来源，不补造当时漏掉的引用；修复前后结果续记于[业务基线](business-baseline.md)。

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/answer-regression-cases.json
# 评分规则变化后只离线重算新副本；须同题集SHA，0 HTTP，不重新运行任务
node go-zero-admin/data/scripts/eval-kb-quality.mjs --rescore tmp/answer-original.jsonl --cases ai_document/ai-rag/answer-regression-cases.json --output tmp/answer-rescored.jsonl
```

可选 `answer.absence_policy=forbidden|required` 只提示额外缺口或应有的局部不足措辞；引述、否定或错误范围仍可能误判，必须对照原文。rescore保留任务、UUID、答案、来源、耗时与调用数，记录源文件SHA；非派生事件、语料或计数不合法仍拒绝输出，不能将副本当作新执行。修复结果及边界见[后续基线记录](quality-baseline.md#回答矛盾与问项范围的后续修复)。

完整报告须与实际串行事件顺序一致；失败、取消、拒绝或意外审批的接受状态不能进入成功链，运行器保留原任务标识后停止。`no_matches`仅允许零聊天用量且`complete=false`。分析耗时在GET确认终态时捕获，不包含其后的日志落盘等待，避免100秒边界产生完成记录与离线校验矛盾。

## 边界

- 素材为演示合成数据，不构成投资建议，也不是真实运维规范；请勿对外分发为权威内容。
- 摄取与检索依赖 Embedding 配置（演示使用 DashScope text-embedding-v4）及 Qdrant；未配置时知识库与文档管理仍可用，摄取和检索不可用，详见[配置说明](../dev-guide.md#ai-与邮件)。
- file 模块联动及 PDF/Office/OCR 解析尚未接入；当前文件导入仅支持 UTF-8 TXT/Markdown，CSV/JSON 继续按正文粘贴。
