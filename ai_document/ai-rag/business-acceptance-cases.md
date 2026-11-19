# 知识库质量验收的24题项目场景集

本题集是基于仓库实际文档的 **AI辅助场景编题**，不是真实用户历史问题，没有真人签署。它用于知识库质量验收页、关键事实与风险检查、失败样本复测；字符串命中、0规则警报及AI辅助复核不能证明语义准确率。

[business-acceptance-cases.json](business-acceptance-cases.json)沿用既有version 1和评测入口，复用[原业务题集](business-cases.md)的8篇冻结摘录、标题、逻辑知识库标识和SHA。[来源记录](business-corpus-provenance.json)指向提交`fc226fbc8d1cd196453617fa7ff2fcdb548ba945`的原始文档，不因当前README或功能继续修改而覆盖旧素材。首版不需要新建库或重新导入；逻辑KB`project_ops`、`project_ai`须映射到本人且正文一致的实际知识库。

24题中运维、AI各12题，每组8开发、4初始预留，共16题development、8题validation。20个有据题包含4个部分未知，另外4题在指定库无答案。分类为8跨文档、6多事实、2背景指令、4部分未知、4无答案；所有跨文档问题均限定同一个逻辑知识库，不要求或允许跨用户检索。

## 中文检查项

每题`checks`与`answer.required_groups`一一对应，`required_group`为0起始索引，`label`用于页面展示，`kind`为`fact`、`condition`、`risk`或`unknown`。当前题目的检查项使用事实、条件和风险分类，不另建评分规则；关键风险本身也放进原有事实字符串组，而非仅写在说明中。例如初始化题分别检查删除/DROP、重建、非升级脚本及重置前置条件。

负例没有必需事实组，`checks=[]`。部分未知题仍为`answerable`，检查项仅映射有据事实；真正请求的实例SHA、开关、task_id或job_id不在文档中，使用既有`absence_policy=required`单独提示不足范围待复核，不能为了新增“unknown”标签把不存在于原文的短语塞进事实组。字段`checks_version=1`表示这份题集要求完整检查项映射；旧题无`checks`格式保持兼容。

事实组给出来源中的至少一种表述及常见回答备选，证据组只在对应文档查找。它们仍是NFKC与空白规范化后的词法筛查，不能识别所有否定、指代、条件归属和同义改写。不能因为答案是另一种正确措辞就改旧记录求绿，也不能把有据事实缺失全归为词法误报；需对照问题、当时答案及实际引用复核。

24 道题都已执行并复核。每题 ID、原始提问、事实组与各题的证据见[原 24 题](business-acceptance-cases.json)。页面与 CLI 使用同一份冻结题集。

## 24题校准版

[证据覆盖校准版](business-evidence-cases.json)保留父版 24 题的问题、分组与 8 篇资料指纹，共 20 道有据题、4 道无答案题。旧 SHA 为 `176e20b4563d8c892b375fe45af89a4a27b78ee1d62c9686e9b49b3956696760`，新 SHA 为 `1b51cb93a575876002bd9877c4ffa2ef3ab95f43785bd5ea4a20292c8b6eb5ca`。以下六项校准依据原问和冻结资料。逐字段差异见新 JSON 的 `annotation.calibration_changes`。

| 场景 | 调整与依据 |
|---|---|
| `dev-backup-checksum-gap` | 原问关注校验、恢复环境及实例 SHA，“完整原始 SQL”移为补充复核。 |
| `dev-refresh-advice` | 原问关注访问隔离、刷新与退出，404 移为补充复核，归属、授权和令牌轮换仍计入必答。 |
| `val-captcha-runtime-gap` | 原问限定本机开发，生产/租户边界及核验接口移为补充，并纳入有字段和值约束的自然表述。当前实例实际开关仍须标为未知。 |
| `dev-isolated-cleanup` | 删除对题设所给隔离端口的复述要求，移除仅支持该前提的必需文档。七组清理与保护要求保留。 |
| `dev-text-limits` | 增加与正文 UTF-8 字节绑定的超限拒绝同义表述，仍须明确指出超限后无法提交。 |
| `dev-capability-config` | 移除会把合法配置状态“未知”误报成无依据的全局词法警报，五组必答事实与来源要求保留。 |

源码与环境变量同时更新、宿主与 Compose 配置选择两道实际失败的完整题目与父版相同。对比实验前后使用同一版题集和语料，校准前后的词法得分不可直接换算为质量提升。AI 辅助判断没有真人签署，两个版本均已用于回归。[同版完整测量](business-baseline.md#完整对照结果) · [实际来源与构造边界](business-evidence-fixtures.json)

## 离线校验与内嵌同步

从仓库根目录运行。四份源 JSON 为原 24 题、校准 24 题、32 道业务题和 14 道失败回归。服务端 `evaluation_data/` 内的副本按源文件逐字复制，原始字节参与 SHA 校验。

```powershell
node --test go-zero-admin/data/scripts/business-acceptance-cases.test.mjs go-zero-admin/data/scripts/business-evidence-cases.test.mjs
node go-zero-admin/data/scripts/sync-kb-evaluation-data.mjs --check
```

有意更新内嵌副本时运行 `node go-zero-admin/data/scripts/sync-kb-evaluation-data.mjs`，再用 `--check` 核对。完整运行、分母与对照要求见[评测入口](README.md#关键证据覆盖与完整题集基线)。
