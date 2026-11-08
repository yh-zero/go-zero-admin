# 跨文档与引用范围失败回归

这套14题用于修复业务首轮的两项已观测问题：任务178把被引用的越权建议追加成未知问项，任务190没有召回已有的权限文档。题目由AI辅助从失败原问及必要控制场景编写，全部是development，**不是真实用户历史题集、没有真人签署，也不是独立保留集**。原问重复、同义改写和同题再次执行均不算独立样本。

[题集JSON](business-failure-regression-cases.json)SHA256：`6f6fbf2a4881751d1a9a77f2cde4571c2176cd3cca26fdab310f3a6eed0c9b87`。13题包含有据目标，1题全部资料不足；其中两个有据问题另外包含真实未知目标。11题使用`absence_policy=forbidden`检查额外不足措辞，两个部分未知和一个全未知使用`required`保留具体未知边界。词法警报只供人工复核，不能证明语义正确或全部问项都已覆盖。

## 资料与失败来源

documents逐项复用[business-cases.json](business-cases.json)原8份冻结素材，标题、路径和原文SHA不变，未增加新资料、改动原32/40/12题或调整它们的分组。实际执行本题集只涉及`project_ai`的模型配置、知识库操作、流程运行与权限四份资料；其余原资料条目保留为同源记录，不因此产生跨库检索。

[失败夹具](business-failure-fixtures.json)来自忽略目录中的`feedback-business-quality.jsonl`，原日志SHA为`57571c3b7baa28f2d0e23ec5e5714edf380c7f629b15559a80810a26ca41fe7c`，当时AI RPC构建SHA为`86698e4d8a9d8d0cb93cf706fed7317e7c136fa8e6673f3042d75d54056551c2`。夹具完整保留已观测外层analysis、实际sources、usage、直接检索片段和旧评分，只选取非认证字段；没有保存或推测未观测的供应商内部逐问项JSON。

- 178原规则没有标记，但来源已支持归属、404和服务端授权，回答仍把被引述建议列为“本次片段未能确定”。新原问回归启用额外不足警报，旧回答仍保留原样。
- 190原search与analysis.sources均没有权限文档，文档Recall@5为0.5。夹具把预期缺失的`ai-access`另列，`actual_source_index=null`；它不是当时的引用，不能补进实际sources伪造已回答。

source_record_sha256针对`JSON.stringify`后的原result对象，observed_sha256针对夹具所选观测字段；二者配合原日志摘要用于一致性核对，不是来源数字签名。CI未携带本地原日志时，仅复验自包含夹具和冻结资料，不宣称重新读取原始运行。更多首轮结果见[业务基线](business-baseline.md)。

## 题目与边界

| ID末段（均以business-failure-开头） | 范围 | 必须保留的行为 |
|---|---|---|
| access-repeat-1 / access-repeat-2 | 178同一原问两次 | 按资料回答归属、404和服务端授权，不把错误建议当缺证据的独立目标 |
| access-advice-ascii | ASCII双引号建议 | 真实请求在引号外；引述不是要执行或另行回答的指令 |
| access-advice-single | 中文成对单引号建议 | 同一范围规则覆盖其他常见引号形式 |
| cross-repeat-1 / cross-repeat-2 | 190同一原问两次 | 模型降级和跨用户隔离都需实际来源 |
| cross-reordered | 先问隔离、后问模型 | 问序变化不能丢失任一已有依据 |
| cross-three-docs | 模型、隔离、试运行三文档 | 多文档覆盖不等于只保留每篇一个片段，也不能牺牲问项证据 |
| partial-quoted-task | 归属与404＋未知task_id | 被“并回答”请求、含问号的引号内问题是真实目标，未知范围须保留 |
| partial-quoted-job | 模型降级＋未知job_id | 有据部分回答，实例实际值单独说明不足 |
| single-import | 单篇的四项导入限制 | 正文字节、批次、浏览器并发和后台槽不能因文档覆盖策略而丢失 |
| single-embedding | 单篇的Embedding与存储职责 | 维持既有明确单文档能力 |
| whole-quoted-question | 整句仅引号包围的真实问句 | 不能把整句引号一律解释成应忽略的外部建议 |
| quoted-request-unknown | 明确请求回答引号内实例问题 | 资料没有的实际任务ID仍须表达不足，不跳过问题或编造数字 |

每个有据题使用与预期文档对应的证据组，部分未知题仍标为answerable并要求不足措辞。负例不标注预期文档、证据或事实。`source_case_id`记录衍生关系，不表示新题曾在原日志中运行；只有夹具中的178和190是原始实测结果。

## 执行与复核

从仓库根执行，默认和plan只读本地题集与素材，不请求HTTP：

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/business-failure-regression-cases.json
node --test go-zero-admin/data/scripts/business-failure-regression.test.mjs
```

真实运行使用正常登录结果、已经核对的KB映射和实际构建/模型配置。以下`tmp/<本轮目录>/`须替换为存在的被忽略目录，原报告不覆盖：

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --run --phase candidate --cases ai_document/ai-rag/business-failure-regression-cases.json --auth-file tmp/<本轮目录>/auth-private.json --kb-map tmp/<本轮目录>/kb-map.json --context-file tmp/<本轮目录>/context.json --output tmp/<本轮目录>/business-failure-candidate.jsonl
node go-zero-admin/data/scripts/eval-kb-quality.mjs --report tmp/<本轮目录>/business-failure-candidate.jsonl --cases ai_document/ai-rag/business-failure-regression-cases.json --output tmp/<本轮目录>/business-failure-candidate.md
```

报告也必须显式指定本题集；省略会使用旧默认40题，不能把异题集报告的“不完整”当成原运行失败。只复验两个原问时，可加`--case-ids business-failure-access-repeat-1,business-failure-cross-repeat-1`。当前没有validation题，不能把空分组说成新保留集通过。

全部14题正常路径最多提交14次分析、28次逻辑查询Embedding和14次聊天；资料不足题若仍有候选片段，也可能调用聊天。文档核对与任务轮询的HTTP另计，实际供应商尝试/重试和费用不能按逻辑上限直接推算。失败或未知结果保留原UUID与任务ID，不自动重试、换ID重发或调用额外模型裁判。

本地四项回归覆盖题集/原文冻结、实际旧坏答的来源与规则、阻断fetch的三组plan，以及14题全响应桩编排；桩验证接口、引用和边界传递，不代表修复后的模型效果。真实候选运行、逐题复核和清理证据由本轮主任务另记，不将题集存在或桩通过写成真实修复成功。

2026-11-07 最终构建已实际完成本14题、原32题业务与原12题旧失败回归，共58次分析。14题0规则警报且逐答未再发现目标问题；中间曾有失败和0警报仍有歧义的候选，均保留原记录。具体构建、同题对比、32题剩余细节省略、耗时/用量及清理边界见[业务后续基线](business-baseline.md#跨文档与引文范围的后续修复)，此结果仍为开发回归和AI辅助复核。
