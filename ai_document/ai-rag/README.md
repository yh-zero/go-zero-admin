# 知识库评测与联调素材

从仓库根目录运行下列命令。初次检查先用 `--plan` 校验本地题集和资料指纹。只有 `--run` 会访问开发服务，其中检索使用 Embedding，回答质量评测还可能调用聊天模型。

| 用途 | 资料与题集 | 结果 |
|---|---|---|
| 合成资料召回、回答可靠性 | [人工问题](test-questions.md)、[14 道直接召回题](recall-cases.json)、[40 道质量题与 12 道回答回归](quality-cases.md) | [质量基线](quality-baseline.md) |
| 项目业务、反馈与跨文档失败回归 | [8 篇冻结摘录、32 道业务题与 14 道失败回归](business-cases.md) | [业务基线](business-baseline.md) |
| 知识库质量验收页、证据覆盖对照 | [24 道原始场景与校准版](business-acceptance-cases.md) | [完整 24 题对照](business-baseline.md#完整对照结果) |
| 小规模并发、资料变更、页面响应与首屏 | [性能测量、原始记录和复现方式](performance-baseline.md) | 同页按日期分别记录 |

## 导入资料

在本人知识库工作区导入资料，等待状态变为“可使用”。合成题使用两个库，分别导入 `kb1-a股投资知识/` 的 6 篇和 `kb2-运维发布SOP/` 的 5 篇。标题使用去掉序号与扩展名的文件名。TXT 和 Markdown 可以从文件导入，CSV 和 JSON 用“粘贴资料”复制完整正文。业务题使用另一组两个库，按[冻结资料清单](business-cases.md#冻结语料与来源)分配 8 篇原文。单篇上限为 65,536 UTF-8 字节。各目录换行按[仓库属性](../../.gitattributes)检出，题集校验使用原始字节。

真实运行先准备被忽略的 `tmp/` 目录。`auth-private.json` 保存正常登录返回的 `result`。`kb-map.json` 将 `stocks`、`ops` 映射到本人实际库 ID，`project-kb-map.json` 对应 `project_ops`、`project_ai`。使用新的输出文件名。认证文件和原始运行日志仅在本地使用。

## 直接召回测试

固定 12 道命中题和 2 道指定库无答案题。真实运行先读取库内资料，再调用直接检索接口，记录每题命中、排序与耗时。生成回答另用质量评测。

```powershell
node go-zero-admin/data/scripts/eval-kb-recall.mjs --plan
node go-zero-admin/data/scripts/eval-kb-recall.mjs --run --auth-file tmp/auth-private.json --kb-map tmp/kb-map.json --output tmp/recall.jsonl
node --test go-zero-admin/data/scripts/eval-kb-recall.test.mjs
```

## 检索与回答质量评测

默认使用 40 题合成题集。自选题集时，给 `--plan`、`--run`、`--report`、`--compare` 和 `--rescore` 显式传同一个 `--cases` 路径。可用 `--split development` 或 `--case-ids` 指定子集。

运行上下文 JSON 由维护者核实后填写。`binary_sha` 为运行产物的 64 位十六进制 SHA256，另填 `source_revision`、`model:{provider,name}`、`embedding:{provider,model,dimension}` 和 `config:{max_context_tokens,analysis_answer_token_cap,recall_top_k,return_top_k}`。可用 `experiment:{label,change}` 记录实验变量。

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan
node go-zero-admin/data/scripts/eval-kb-quality.mjs --run --auth-file tmp/auth-private.json --kb-map tmp/kb-map.json --context-file tmp/quality-baseline-context.json --phase baseline --output tmp/quality-baseline.jsonl
node go-zero-admin/data/scripts/eval-kb-quality.mjs --run --auth-file tmp/auth-private.json --kb-map tmp/kb-map.json --context-file tmp/quality-candidate-context.json --phase candidate --output tmp/quality-candidate.jsonl
node go-zero-admin/data/scripts/eval-kb-quality.mjs --report tmp/quality-baseline.jsonl --output tmp/quality-baseline.md
node go-zero-admin/data/scripts/eval-kb-quality.mjs --compare tmp/quality-baseline.jsonl tmp/quality-candidate.jsonl --output tmp/quality-compare.md
node --test go-zero-admin/data/scripts/eval-kb-quality.test.mjs go-zero-admin/data/scripts/eval-kb-recall.test.mjs
```

每题串行执行检索和草稿分析。完整 40 题最多提交 40 次分析，正常路径每题包括两次逻辑查询 Embedding。开始和结束核对正文 SHA、版本、就绪状态与库存，任务按原 UUID 核查。失败或结果未知时保留已有 JSONL。报告按原始前 K 块计算 Hit、文档 Recall、MRR 和证据组覆盖，分列已知聊天用量、检索耗时和分析提交至终态的耗时。不同构建比较须说明实验变量，并保持题集、语料与模型预算可比。

评分规则变动时，可离线重算派生副本。原日志保留历史答案、实际来源、耗时和调用数。

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --rescore tmp/answer-original.jsonl --cases ai_document/ai-rag/answer-regression-cases.json --output tmp/answer-rescored.jsonl
```

## 知识库质量验收页与24题项目场景

从知识库工作区进入“质量验收”，选题集和同组资料，核对配置与费用后执行。历史批次可逐题查看答案、实际来源、中文检查项和语料校验状态。已采纳反馈可以导出为新题集，维护方式见[业务题说明](business-cases.md#采纳反馈成为可运行回归)。两个 24 题版本须按各自 SHA 查看。当前题集均已用于回归。

## 关键证据覆盖与完整题集基线

`eval-kb-cohort.mjs` 为每个题目保存独立原始日志，并写入整轮 manifest。已知终态失败记入分母后可以继续后续题。未知结果或日志失效会停止。页面批次遇到失败会停止。比较时分别看全题完成情况、成功题质量及共同成功子集，并核查题集 SHA、题序、KB 映射、实际语料和模型预算。

```powershell
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --plan --cases ai_document/ai-rag/business-evidence-cases.json
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --run --phase baseline --cases ai_document/ai-rag/business-evidence-cases.json --auth-file tmp/auth-private.json --kb-map tmp/project-kb-map.json --context-file tmp/evidence-baseline-context.json --output tmp/evidence-baseline.jsonl
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --run --phase candidate --cases ai_document/ai-rag/business-evidence-cases.json --auth-file tmp/auth-private.json --kb-map tmp/project-kb-map.json --context-file tmp/evidence-candidate-context.json --output tmp/evidence-candidate.jsonl
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --report tmp/evidence-baseline.jsonl --cases ai_document/ai-rag/business-evidence-cases.json
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --compare tmp/evidence-baseline.jsonl tmp/evidence-candidate.jsonl --cases ai_document/ai-rag/business-evidence-cases.json
node --test go-zero-admin/data/scripts/eval-kb-cohort.test.mjs
```

完整对照的样本、异常、不同构建与已知用量见[业务基线](business-baseline.md#完整对照结果)。真实模型质量需结合问题和当时实际引用逐题复核。
