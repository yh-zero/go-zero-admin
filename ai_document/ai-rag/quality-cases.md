# 知识库质量题集标注说明

[quality-cases.json](quality-cases.json) 是独立于旧 [recall-cases.json](recall-cases.json) 的40题固定题集。11篇来源、标题和原始文件SHA256完整复用旧题集，素材未改动。全部是合成演示资料和人工标注问题，不是真实用户样本、权威运维规范或投资建议。

| split | 有答案 | 指定库无答案 | 合计 |
|---|---:|---:|---:|
| development | 17 | 7 | 24 |
| validation | 11 | 5 | 16 |
| 总计 | 28 | 12 | 40 |

| category | development | validation | 合计 |
|---|---:|---:|---:|
| direct：直接提取条目 | 4 | 3 | 7 |
| paraphrase：场景化改写 | 4 | 2 | 6 |
| multi_fact：同文档多事实 | 4 | 3 | 7 |
| multi_document：跨文档组合 | 3 | 1 | 4 |
| near_miss：主题相近但缺证据 | 3 | 2 | 5 |
| unanswerable：跨库或库外事实 | 3 | 2 | 5 |
| instruction：指令干扰 | 3 | 3 | 6 |

每个split的instruction均含2个有答案题和1个无答案题。stocks共22题，ops共18题；11篇资料均有正例覆盖，4个多文档题均需要2篇文档。direct可以直接问同一条目里的多个字段，multi_fact强调组合多个事实或条目。

标注含义：

- `expected_documents`使用documents的key；正例的`evidence`逐一覆盖全部预期文档。`groups`中每组至少一个候选字符串确实出现在对应原文；组内是候选变体，组间均须覆盖。不能用另一篇文档的同名事实替代本篇证据。
- `answer.required_groups`是答案中可接受的短语变体，用于发现可能遗漏的事实。它们不保证数字归属、条件、否定、因果或公式关系正确；例如两种期限相同、多个告警级别并列时，仍需核对语句关系。
- `answer.forbidden_phrases`只标记特定跨库内容、构造的无依据断言或合成注入标记，没有把“回滚”“FLUSHALL”等普通讨论词设为禁词。命中应进入人工复核：正确拒答也可能先引用再否定该短语或回显干扰标记，不能仅凭命中断定回答错误。列出的虚假断言示例也不能穷尽所有编造内容。
- 无答案题的预期文档、证据与必需答案组均为空，`rationale`说明所选库为何无证据。即使检索返回相近片段或另一库确有答案，也不能据此改成有答案题。

split按目标事实划分，同一篇资料可以出现在两个split，但尽量不让验证题直接改写开发题。开发集的中间件窗口、创业板开户正例与相应跨库负例在同一split；验证集的10转10正例与相应跨库负例也在同一split。validation在固定优化方案前不用于调参；运行基线与复验期间不因模型回答改题或改标注。

本题集只覆盖有限的合成资料、短文本事实与问题侧指令干扰，没有覆盖真实业务语料、文档侧恶意注入、穷举语义等价答案或生产负载。字符串规则是筛查信号，不能称为语义准确率或人工评审结论。

## 12题回答回归

[12 题 JSON](answer-regression-cases.json) 针对备份记录、二线升级和两板块盘中临停三项已观察问题。每个原问各有 2 个逐字重复、1 个改写和 1 个含部分未知的场景，用 `source_case_id` 关联原题。与同一轮 40 题的原问合并看重复表现时，每个原问有 3 次观察。这些回答共用样本，不能按独立题计算置信度。

9 道原问或改写题以 `absence_policy=forbidden` 检查回答是否多报资料缺口。另 3 题的 `required` 要求既回答已知部分，又指出备份保存天数、二线电话号码或规则生效年月等资料未提供的信息。12 题全部属于 development。误报要回看问项、引用和原文。[实际失败夹具](answer-failure-fixtures.json) 保存了当时的坏答及实际来源。

从仓库根目录用以下命令校验题集，运行与重算参数见[评测入口](README.md#检索与回答质量评测)。

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/answer-regression-cases.json --split development
node --test go-zero-admin/data/scripts/eval-kb-quality.test.mjs
```
