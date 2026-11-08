# 24题关键证据覆盖验收：校准派生版

[business-evidence-cases.json](business-evidence-cases.json)由[原24题场景集](business-acceptance-cases.md)派生，供本轮关键证据覆盖修复的完整前后基线使用。校准只依据原问范围和冻结项目文档，属于**AI辅助校准，非真人签署**；不是把已有回答改判为语义正确。原题集、旧评测日志与历史警报均保留，派生版使用独立文件和SHA，不能拿它与旧SHA的分数直接计算提升。

- 父题集SHA256：`176e20b4563d8c892b375fe45af89a4a27b78ee1d62c9686e9b49b3956696760`。
- 本派生版SHA256：`1b51cb93a575876002bd9877c4ffa2ef3ab95f43785bd5ea4a20292c8b6eb5ca`，UTF-8原字节97572。
- 原24题的`id/query/kb/split/category/rationale`逐项保持；8篇素材的全部字段、正文和SHA保持，来源仍为[冻结项目资料](business-corpus-provenance.json)。
- 16个development、8个validation标识保留历史分组。24个场景均已执行和阅读，派生校准不会恢复独立holdout身份。
- 20个有据题、4个无答案题及真正的实例未知范围保持；`checks`继续逐一对应必需事实组，没有新增评分器或模型裁判。

## 六项字段校准

JSON的`annotation.calibration_changes`逐项记录原问依据、补充复核项及每个字段的`before/after`。下表只摘要差异，审查时应以完整字段记录为准。

| 场景 | 派生版调整 | 原问依据与保留边界 |
|---|---|---|
| `dev-backup-checksum-gap` | “完整原始SQL”由必答降为补充复核；对应必引证据子组同步调整 | 原问要求记录来源与校验信息、恢复环境、验证影响及实例SHA未知范围，没有询问产物格式。资料中的完整原始SQL要求仍真实存在，没有宣称已在旧答案补齐。其余4组必答与真正未知保持。 |
| `dev-refresh-advice` | 越权/不存在均404由必答降为补充接口行为 | 原问要求访问隔离、刷新令牌使用与退出状态，未询问错误码。按创建者隔离、服务端独立授权、一次性轮换、旧双令牌失效仍必答，不能以省略404为由抹去主体隔离错误。 |
| `val-captcha-runtime-gap` | 生产/租户覆盖边界、captcha核验接口降为补充复核；false设置增加有字段和值约束的自然表述 | 原问限定本机AI开发，明确要求本地副本、关闭字段及保留的校验；当前实例实际开关仍须标未知，不能用文档推定当前值。补充部署边界仍应单独核查，不能据此授权生产免验证码。 |
| `dev-isolated-cleanup` | 不再要求重述题设的独立端口/调试前隔离数据；仅支持此前置建议的`ops-build`从必需文档移除 | 所问是验证结束后的精确清理、项目/测试/数据/备份保留及Git改动保护，这7组仍必答。该题`category`继承原跨文档场景标签，但本版必要证据来自1篇文档；覆盖率以实际`expected_documents`计，不据历史分类标签声称仍需两文档。 |
| `dev-text-limits` | 超字节拒绝增加与正文、UTF-8字节或65536上限绑定的同义表达 | 仍需正文按UTF-8字节而非字数判断，并明确超限不可提交；裸“不能”不作充分命中。标题/正文上限、批次篇数及两类并发要求均保持。规则仍不能完整识别否定和目标归属，需人工核读。 |
| `dev-capability-config` | 移除该题全局`absence_policy=forbidden`词法警报 | “配置状态未知”是资料中的合法状态标签，条件性的“未配置”也不等于模型声称整个问题无依据。5组必答、全部来源/引用要求不变；缺少必答仍报警，真正无依据拒答仍由人工核对，不将取消一条词法警报当作语义通过。 |

每项校准的对应证据子组或必需文档同步调整，避免把已移出必答范围的事实继续当成“必须引用”的事实。没有重写冻结正文、替换供应商答案或删除真实资料要求；补充观察仍记录在JSON审计字段及人工复核中。

## 保留的真实失败

`dev-code-and-env`与`val-config-selection`两题整条记录与父题集深等：源码产物、环境变量重建、运行版本核验、宿主及Compose两套路径等事实和证据没有减项。初始化SQL的DROP/重建、授权与备份风险、权限隔离和真正实例未知也没有放宽。

本轮应先在旧运行构建完成这份派生版的24题基线，再以同一SHA、题目集合、实际语料指纹、模型及预算执行修复后24题。完整运行才能作对照；失败或结果未知保持原记录，不自动重试，也不把多份续跑拼成一次完整运行。应同时看实际关键片段覆盖、问项完整性和引用范围，规则警报数只定位复核项。

原题的既有结果可使用本版在离线副本中重新筛查，以说明“标注范围变化”影响；这不构成检索或回答的质量提升，也不能覆写原JSONL或与新运行混合。完整基线、实际构建和残余问题由[业务基线](business-baseline.md)记录。

## 离线回归与内嵌同步

从仓库根执行，以下操作不调用HTTP、数据库或模型：

```powershell
node --test go-zero-admin/data/scripts/business-evidence-cases.test.mjs go-zero-admin/data/scripts/business-acceptance-cases.test.mjs
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/business-evidence-cases.json
node go-zero-admin/data/scripts/sync-kb-evaluation-data.mjs
node go-zero-admin/data/scripts/sync-kb-evaluation-data.mjs --check
```

同步入口验证四份题集后逐字复制到固定`evaluation_data/`目录；旧24/32/14及新派生版均核对冻结SHA，不手改内嵌副本。`--check`只读，内容相同不重写。原24题的同步回归相应从3份更新为4份，原问题、原断言和历史文件没有覆盖。

本地回归包括：逐字段审计重建、18条未调整题与两条重点失败深等、实际来源校验、必答与未知负例、4类有范围的超字节否定同义及裸否定/允许超限反例、配置状态标签误报边界、24题0HTTP计划、四份同步及幂等。答案夹具是明确构造的离线边界，不冒充供应商原始回复，不证明所有自然语言表达或真实模型都已通过。

## 完整基线

完整24题前后基线使用 `eval-kb-cohort.mjs`，默认或 `--plan` 为0 HTTP；显式 `--run --phase baseline|candidate` 后，每题各保存v1原始日志，另存整轮manifest。已知失败单列后可继续未运行题，未知结果或日志失效立即停止，不自动重试。页面批次仍失败即停。

```powershell
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --plan --cases ai_document/ai-rag/business-evidence-cases.json
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --report tmp/evidence-baseline.jsonl --cases ai_document/ai-rag/business-evidence-cases.json
node go-zero-admin/data/scripts/eval-kb-cohort.mjs --compare tmp/evidence-baseline.jsonl tmp/evidence-candidate.jsonl --cases ai_document/ai-rag/business-evidence-cases.json
```

真实运行的认证、KB映射、context参数和分母说明见[使用说明](README.md#关键证据覆盖与完整题集基线)。比较须用同一派生SHA、全部24题、同语料及模型预算；直接检索Top5指标与分析实际引用证据覆盖分别读取。不能用校准前后两套标注计算模型质量提升，也不能拼接中间成功记录替代完整运行。
