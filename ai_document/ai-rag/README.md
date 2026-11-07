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

## 边界

- 素材为演示合成数据，不构成投资建议，也不是真实运维规范；请勿对外分发为权威内容。
- 检索摄取依赖 AI_QWEN_KEY（Embedding 走 DashScope text-embedding-v4）；未配置时录入返回 503 降级，属设计行为。
- 二期文件上传（file 模块联动 + 解析）落地后，本目录可直接作为上传源。
