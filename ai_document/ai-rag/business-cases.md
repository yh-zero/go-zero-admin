# 实际项目文档业务题集

这套题集用于检验项目维护者在配置、部署、备份、资料导入、流程运行和权限处理中的实际问题。语料来自本仓库已提交的项目文档，题目和初始答案规则由AI辅助编写，**不是真实用户历史问题，没有真人签署**。规则命中只能帮助找出待复核项，不能称为语义准确率。

[business-cases.json](business-cases.json)共32题：development20题（16个有据问题、4个资料不足问题），validation12题（8个有据问题、4个资料不足问题）。两个部分未知问题分别保留已有依据和本实例实际值的缺口；另有口语改写、多事实、跨文档、冲突建议和跨库无答案问题。分组是首次测量前的初始划分；读取文件、校验SHA和响应桩回归不构成真人验收或真实模型测量。validation一旦用于真实测量、失败分析或调优，应在对应测量记录中标明首次使用时间，此后作为历史回归，不能继续声称独立保留集。

题集SHA256：`f3a4016eabc10a974e3acffc6d7347477976d00923c196c608de55a6db99538b`。后续反馈应另建有审核记录的新版本，保留原始题集与实测日志，不在看到模型结果后原地放宽必需事实或删去失败题。

## 冻结语料与来源

素材冻结自提交[fc226fb](https://github.com/yh-zero/go-zero-admin/commit/fc226fbc8d1cd196453617fa7ff2fcdb548ba945)。[business-corpus-provenance.json](business-corpus-provenance.json)记录每篇素材的原文路径、完整Git blob SHA256、起止行（结束行不含）、章节标记和摘录SHA256。单段是逐字连续摘录，多段只以一个LF连接；后续项目文档修改不覆盖本次素材。素材统一UTF-8/LF，每篇小于65536字节。

| KB映射键 | 导入标题 | 冻结素材 | 来源 |
|---|---|---|---|
| project_ops | 开发环境与配置 | [01-开发环境与配置.md](business-docs/01-开发环境与配置.md) | 开发手册第1节，至数据初始化前 |
| project_ops | 数据初始化与备份 | [02-数据初始化与备份.md](business-docs/02-数据初始化与备份.md) | 开发手册的数据初始化、本地免验证码、备份章节 |
| project_ops | 构建启动与验收 | [03-构建启动与验收.md](business-docs/03-构建启动与验收.md) | 开发手册第2节 |
| project_ops | 临时资源清理 | [04-临时资源清理.md](business-docs/04-临时资源清理.md) | 开发手册第5节 |
| project_ai | 模型与向量存储 | [05-模型与向量存储.md](business-docs/05-模型与向量存储.md) | README的配置模型与检索章节 |
| project_ai | 知识库资料与回答 | [06-知识库资料与回答.md](business-docs/06-知识库资料与回答.md) | README的从资料到有来源的回答章节，至评测说明前 |
| project_ai | 常用流程与运行记录 | [07-常用流程与运行记录.md](business-docs/07-常用流程与运行记录.md) | README的常用流程、运行记录与能力目录章节 |
| project_ai | 身份权限与数据隔离 | [08-身份权限与数据隔离.md](business-docs/08-身份权限与数据隔离.md) | 契约第3节、AI执行约定开头和README的AI入口说明 |

这些文件是复现实验的冻结证据，正文内的相对链接保留原文上下文。核对链接内容时应打开[冻结开发手册](https://github.com/yh-zero/go-zero-admin/blob/fc226fbc8d1cd196453617fa7ff2fcdb548ba945/ai_document/dev-guide.md)、[冻结契约](https://github.com/yh-zero/go-zero-admin/blob/fc226fbc8d1cd196453617fa7ff2fcdb548ba945/ai_document/api-contract.md)或[冻结README](https://github.com/yh-zero/go-zero-admin/blob/fc226fbc8d1cd196453617fa7ff2fcdb548ba945/README.md)，不能把素材目录当作原文链接基准。资料中的演示管理员是明确的本地开发默认账号；题集不包含真实人员账号、生产凭据、会话令牌或AI Key。

## 导入与执行

创建两个仅本人可访问的测试知识库，分别导入对应4篇冻结素材，标题严格使用上表；保留完整正文和换行，等待每篇可使用。不要混入旧合成演示语料、题目、预期答案或来源清单，不复用已经有无关资料的业务库。

原题以 `business-` 开头，逐题问题、证据组、分组与答案规则以[冻结 JSON](business-cases.json) 为准。32 题均已运行并复核，历史结果见[业务基线](business-baseline.md)。从仓库根目录可离线检查题集。真实运行使用[评测入口的命令](README.md#检索与回答质量评测)，将 `--cases` 指向本文件对应 JSON，并使用业务库映射。

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/business-cases.json
node --test go-zero-admin/data/scripts/business-cases.test.mjs
```

## 采纳反馈成为可运行回归

在知识库分析结果提交评价，再到该知识库的待复核列表核对问题、当时答案和实际来源。采纳时填写预期回答、事实检查项、逐字证据摘录及理由；资料不足题不填事实项和摘录。复核记录只表示填写了审核结论，不证明填写者身份或答案语义正确，后续仍需评测和人工复查。

从已采纳记录下载JSON导出，放入忽略目录，例如`tmp/kb-feedback-91.json`。用原始题集和已有KB映射生成新的合并副本：

```powershell
# 只读本地文件，0 HTTP；不调用模型或修改检索参数。
node go-zero-admin/data/scripts/merge-kb-feedback.mjs --feedback tmp/kb-feedback-91.json --cases ai_document/ai-rag/business-cases.json --kb-map tmp/business-kb-map.json --output tmp/business-with-feedback-01.json

# 合并副本沿用原评测入口，默认仅校验。
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases tmp/business-with-feedback-01.json

# 转换器边界回归，不请求服务；其中junction边界用例需要本地文件系统允许创建目录链接。
node --test go-zero-admin/data/scripts/merge-kb-feedback.test.mjs
```

转换只接受已采纳且字段有效的导出。它核对KB映射、问题与实际步骤输入、答案与步骤输出、来源顺序及归属；所有涉及文档都必须是`verified`，并按标题和正文SHA唯一匹配指定题集的冻结素材。`stale`、`unavailable`、未知文档、变更后的正文或不同素材均拒绝，不自动换成当前版本。复核摘录必须逐字属于原历史片段；片段和摘录在同指纹正文核对时，只允许实际分块器在双LF段落边界去首尾空白后重接，保留段内字符、数字及JSON结构。不能用结构压缩或改写原文来替代，无法核对则拒绝。事实检查项还须同时出现在预期回答及所选摘录中，沿用评测器的NFKC和空白规范化。非逐字的解释性事实应先复核成可检查项，工具不代替语义判断。

新题固定为development，ID由知识库ID、反馈ID和revision组成；按文档合并证据组，事实项逐条组成AND规则。预期回答保留为示例，不要求模型逐字复述，也不用额外模型裁判。旧题和原语料保持，新副本的`metadata.feedback_regressions`记录原题集、导出和快照SHA及反馈版本；原有description/annotation描述基础题集范围，当前实际题数以`cases`及plan输出为准。相同反馈已在原题集中出现时，任何revision均拒绝重复并入或静默替换；新复核须明确选择基线、另建新副本。

输出仅允许仓库忽略的`tmp`目录及其真实子目录，符号链接不能越界，已有文件不覆盖；写入后重新执行原题集校验，失败删除本次不完整副本。导出原件、认证、运行日志和合并结果按本地资料保留，不随工具代码直接提交。SHA记录文件一致性，不是导出签名；文档SHA对应反馈采集时的正文核对，不能据此声称取得模型调用时的全库版本快照。

## 14题失败回归

[14 题开发回归](business-failure-regression-cases.json) 复用相同的 8 份冻结资料，本组实际查询限于 `project_ai`。13 题有可回答目标，其中 2 题另有真正未知的问题，1 题全部资料不足。11 题检查额外的不足断言，其余 3 题要求明确指出缺少哪项证据。

| 原任务 | 观测与复核要点 |
|---|---|
| 178 | 资料支持归属、404 与服务端授权。回答把问题引述的错误建议追加成未知问项，旧规则没有报警。复核时区分背景引述与用户明确要求回答的引号内问题。 |
| 190 | 检索与分析引用均缺权限文档，文档 Recall@5 为 0.5。夹具将该文档记为预期缺失，`actual_source_index=null`。 |

[失败夹具](business-failure-fixtures.json) 保存实际回答、来源、直接检索和用量，不包含供应商内部逐问项记录。来源为本机 `tmp/ai-product-20261107/feedback-business-quality.jsonl`，源日志和观测字段的摘要在夹具中。没有本地日志的检出可校验自包含夹具与冻结资料。

失败题的逐字重复、改写、调整问序、三文档题和单文档对照均已用于开发。可用 `--case-ids business-failure-access-repeat-1,business-failure-cross-repeat-1` 单独检查两个原问，完整历史对照见[业务基线](business-baseline.md#跨文档与引文范围的后续修复)。

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/business-failure-regression-cases.json
node --test go-zero-admin/data/scripts/business-failure-regression.test.mjs
```
