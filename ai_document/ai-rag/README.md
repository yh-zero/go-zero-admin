# ai-rag — 知识库联调素材（演示合成数据）

## 内容

| 目录 | 知识库 | 文件类型覆盖 |
|---|---|---|
| kb1-a股投资知识/ | A 股投资知识（交易规则、板块、两融、分红、指标） | md ×2、txt ×2、csv ×1、json ×1 |
| kb2-运维发布SOP/ | 运维与发布 SOP（发布、值班、变更窗口、告警、DB 变更） | md ×2、txt ×1、csv ×1、json ×1 |
| test-questions.md | 检索测试问题集（每库命中题 + 跨库隔离/注入/降级边界题） | — |

## 使用方式（一期：内联文本摄取）

共 11 篇素材（投资知识 6 篇、运维 SOP 5 篇）。一期文档来源为内联文本（≤64KiB/文档，同步摄取），暂不解析文件附件：

1. 登录后台（admin）→ AI 功能 → 知识库，新建两个知识库："A股投资知识"、"运维发布SOP"。
2. 进入各知识库的"文档管理"，把每个文件的正文**全文复制**为一篇文档（标题用文件名去掉序号与扩展名）。
3. 摄取状态变"已就绪"后，按 test-questions.md 在 AI 对话中逐题验证。

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

## 回答一致性失败回归

[12题开发回归](answer-regression-cases.md)覆盖原问重复、改写和部分缺资料，[实测夹具](answer-failure-fixtures.json)保留坏答与实际来源。原40题不改；新题是已使用的开发数据，不是新独立验证集。

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/answer-regression-cases.json
# 评分规则变化后只离线重算新副本；须同题集SHA，0 HTTP，不重新运行任务
node go-zero-admin/data/scripts/eval-kb-quality.mjs --rescore tmp/answer-original.jsonl --cases ai_document/ai-rag/answer-regression-cases.json --output tmp/answer-rescored.jsonl
```

可选 `answer.absence_policy=forbidden|required` 只提示额外缺口或应有的局部不足措辞；引述、否定或错误范围仍可能误判，必须对照原文。rescore保留任务、UUID、答案、来源、耗时与调用数，记录源文件SHA；非派生事件、语料或计数不合法仍拒绝输出，不能将副本当作新执行。修复结果及边界见[后续基线记录](quality-baseline.md#回答矛盾与问项范围的后续修复)。

完整报告须与实际串行事件顺序一致；失败、取消、拒绝或意外审批的接受状态不能进入成功链，运行器保留原任务标识后停止。`no_matches`仅允许零聊天用量且`complete=false`。分析耗时在GET确认终态时捕获，不包含其后的日志落盘等待，避免100秒边界产生完成记录与离线校验矛盾。

## 边界

- 素材为演示合成数据，不构成投资建议，也不是真实运维规范；请勿对外分发为权威内容。
- 检索摄取依赖 AI_QWEN_KEY（Embedding 走 DashScope text-embedding-v4）；未配置时录入返回 503 降级，属设计行为。
- 二期文件上传（file 模块联动 + 解析）落地后，本目录可直接作为上传源。
