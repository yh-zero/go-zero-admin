# 回答问题开发回归题集

[answer-regression-cases.json](answer-regression-cases.json)只用于修复已发现的备份记录、二线升级、两板块盘中临停回答问题，全部12题均为`development`。题目已经用于开发或回归，**不是独立holdout**；原40题中已看过的validation临停题也不能重新宣称独立验证。原[quality-cases.json](quality-cases.json)保持冻结，11篇合成资料的来源、标题与原始文件SHA256完整复用，不改写素材。

| 目标 | 原题source_case_id | 原query额外重复 | 同目标改写 | 部分已知与明确缺口 |
|---|---|---:|---:|---:|
| 备份记录 | `quality-dev-runtime-backup` | 2 | 1 | 1 |
| 二线升级 | `quality-dev-incident-second-line` | 2 | 1 | 1 |
| 两板块盘中临停 | `quality-val-intraday-halt` | 2 | 1 | 1 |

所有题保留`source_case_id`关联原题。6个repeat题的query逐字复制原题，各用不同ID；与**同一轮新运行**原40题中的那一次组成每个原query三次观察。旧轮结果不能冒充同一构建/配置下的新重复。重复样本不独立，3个改写也不是3个新事实，不据此计算独立样本置信度。

12题的`answer.kind`均为`answerable`：已知目标仍须回答。新增可选字段`answer.absence_policy`只辅助评测器提示复核，不传给模型：

- `forbidden`共9题：6次原query重复与3个同目标改写。原文完整覆盖所问目标，不应在回答正确事实后再声称同一目标缺失，也不应附加无关缺口。
- `required`共3题：回答原文已知要求后，还须对问题明确新增的缺失目标说明“资料未提供”“无法确定”等。备份保存天数、二线具体联系电话、该两板块临停规则正式生效年月日在全部11篇资料中均无事实。其他观察/响应时限不是保留天数，7x24电话可达不是电话号码，沪市2018年后的收盘集合竞价也不是临停规则生效日期。

部分缺失题的`expected_documents`、`evidence`与`required_groups`只标已知事实，未给缺口编造“标准答案”。引用同一知识库的相近主题也不能补足缺口。负例和部分未知目标仍要表达证据不足，不能为了消除“额外缺口”提示而压掉正确拒答。

词法标记不能证明语义：例如否定“资料不足”、引用错误说法，可能触发额外缺口提示；包含“无法确定”也不能证明它指向真正未知的目标。每题需核对已知事实、缺口所属目标、条件、引用和来源。全部素材是演示合成数据，不是真实用户样本、真实联系人或投资建议。

从仓库根目录只校验题集并查看计划，0 HTTP请求：

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/answer-regression-cases.json --split development
```

实跑、认证、上下文和新输出路径沿用[评测入口](README.md#检索与回答质量评测)。`--run`、`--report`、`--compare`涉及本题集时均须显式传同一个`--cases`路径，不能默认回到原40题。真实调用需保持本轮已授权预算、构建与配置；本说明及题集校验不调用服务或模型。
