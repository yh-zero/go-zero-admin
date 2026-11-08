# 知识库质量验收的24题项目场景集

本题集是基于仓库实际文档的 **AI辅助场景编题**，不是真实用户历史问题，没有真人签署。它用于知识库质量验收页、关键事实与风险检查、失败样本复测；字符串命中、0规则警报及AI辅助复核不能证明语义准确率。

[business-acceptance-cases.json](business-acceptance-cases.json)沿用既有version 1和评测入口，复用[原业务题集](business-cases.md)的8篇冻结摘录、标题、逻辑知识库标识和SHA。[来源记录](business-corpus-provenance.json)指向提交`fc226fbc8d1cd196453617fa7ff2fcdb548ba945`的原始文档，不因当前README或功能继续修改而覆盖旧素材。首版不需要新建库或重新导入；逻辑KB`project_ops`、`project_ai`须映射到本人且正文一致的实际知识库。

24题中运维、AI各12题，每组8开发、4初始预留，共16题development、8题validation。20个有据题包含4个部分未知，另外4题在指定库无答案。分类为8跨文档、6多事实、2背景指令、4部分未知、4无答案；所有跨文档问题均限定同一个逻辑知识库，不要求或允许跨用户检索。

validation仅表示初始预留编排。先用development核对旧构建基线及开发；最终验收时才首次执行validation，并在测量记录中记下首次使用时间。任何题一旦执行或据其调参，就属于已使用数据，不能继续宣称独立holdout。编题者能够查看本文件，分组本身也不证明统计独立性。旧32/40/12/14题及失败夹具保持原样。

## 中文检查项

每题`checks`与`answer.required_groups`一一对应，`required_group`为0起始索引，`label`用于页面展示，`kind`为`fact`、`condition`、`risk`或`unknown`。当前题目的检查项使用事实、条件和风险分类，不另建评分规则；关键风险本身也放进原有事实字符串组，而非仅写在说明中。例如初始化题分别检查删除/DROP、重建、非升级脚本及重置前置条件。

负例没有必需事实组，`checks=[]`。部分未知题仍为`answerable`，检查项仅映射有据事实；真正请求的实例SHA、开关、task_id或job_id不在文档中，使用既有`absence_policy=required`单独提示不足范围待复核，不能为了新增“unknown”标签把不存在于原文的短语塞进事实组。字段`checks_version=1`表示这份题集要求完整检查项映射；旧题无`checks`格式保持兼容。

事实组给出来源中的至少一种表述及常见回答备选，证据组只在对应文档查找。它们仍是NFKC与空白规范化后的词法筛查，不能识别所有否定、指代、条件归属和同义改写。不能因为答案是另一种正确措辞就改旧记录求绿，也不能把有据事实缺失全归为词法误报；需对照问题、当时答案及实际引用复核。

## 场景范围

题目ID均以`acceptance-`开头。

| 分组 | 后缀 | 主要检查内容 |
|---|---|---|
| development | dev-init-destructive | 已有库生成model，明确DROP/重建风险及授权/确认/备份条件 |
| development | dev-code-and-env | 同时改Go源码与环境，编译、镜像、重建及运行产物验收 |
| development | dev-smtp-proof | SMTP服务商切换、From回退及假SMTP与真实发信边界 |
| development | dev-host-network | 宿主API的RPC依赖、地址、容器内部端口与宿主映射 |
| development | dev-reset-advice | 背景错误重置建议，实际要求是数据风险与重置边界 |
| development | dev-isolated-cleanup | 隔离调试后的精确清理、既有数据/测试/备份/Git改动保留 |
| development | dev-backup-checksum-gap | 备份校验及隔离恢复有据，实例实际SHA未知 |
| development | dev-oncall-unknown | 指定文档未收录的实际告警阈值与值班电话 |
| validation | val-backup-sensitive | 原始备份保留与公开提交受阻、替代文件和历史快照边界 |
| validation | val-config-selection | 两套配置、字面值优先、env重建及ready不代替业务验收 |
| validation | val-captcha-runtime-gap | 本地验证码配置规则有据，本实例当前实际开关未知 |
| validation | val-runtime-build-unknown | 文档未收录的当前运行二进制实际SHA |
| development | dev-import-pending | 接收/排队/处理/可用、停止批次、离页与待确认不重传 |
| development | dev-history-and-owner | 修改后的版本/处理、历史来源与当前原文、创建者隔离 |
| development | dev-text-limits | 标题字符、正文UTF-8字节、批次和两种并发范围 |
| development | dev-capability-config | 配置状态非实时健康，缺聊天密钥降级及分析依赖 |
| development | dev-refresh-advice | 背景错误权限/令牌建议，实际隔离、轮换及退出规则 |
| development | dev-models-and-draft | 模型/向量未配置的管理与分析区别、未保存流程试跑 |
| development | dev-cancel-task-gap | 取消不回滚和未知不重跑有据，实际历史任务ID未知 |
| development | dev-sales-unknown | 指定AI资料没有公司真实回款或客户明细 |
| validation | val-query-and-file | 2000与65536字节范围、两类无聊天场景、相似度含义 |
| validation | val-task-owner-cancel | 他人任务越权、自己的任务取消及未知结果不重发 |
| validation | val-edit-job-gap | 编辑、版本冲突和历史摘录有据，实际摄取作业ID未知 |
| validation | val-invoice-unknown | 实际AI账单和发票未收录，不能用tokens或计费说明编造 |

## 离线验证与内嵌数据

从仓库根执行，以下命令不会调用HTTP、模型或数据库：

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/business-acceptance-cases.json --split development
node --test go-zero-admin/data/scripts/business-acceptance-cases.test.mjs

# 校验题集、原文SHA、证据和检查项后，只复制四份源JSON的原始字节。
node go-zero-admin/data/scripts/sync-kb-evaluation-data.mjs
# 仅校验，不创建目录、不写文件。
node go-zero-admin/data/scripts/sync-kb-evaluation-data.mjs --check
```

同步目标固定为`go-zero-admin/application/ai/rpc/internal/logic/aiservice/evaluation_data/`中的`business-acceptance-cases.json`、`business-cases.json`、`business-failure-regression-cases.json`及新增的`business-evidence-cases.json`，供服务端`go:embed`读取。源文件是事实源，内嵌副本只作逐字复制，不手改；旧24/32/14题及新校准版分别核对冻结SHA。同步拒绝任意输出路径参数，`--check`遇到缺失或字节差异即失败；重复同步的相同文件不重写。

本题集首次规划使用16开发题基线、8个validation最终首次验收；目前24题均已真实运行和阅读，不能再称为独立保留集。后续关键证据覆盖使用[派生校准版](business-evidence-cases.md)建立同SHA完整前后基线，原题和历史记录保留。认证、KB映射、运行配置及输出放入被忽略的`tmp`目录，报告/比较必须显式指定对应题集；不覆盖旧日志，不自动重试失败或未知任务。不能把本文件、离线桩或内嵌数据同步成功当作真实回答已通过。

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --run --phase baseline --cases ai_document/ai-rag/business-acceptance-cases.json --split development --auth-file tmp/<本轮目录>/auth-private.json --kb-map tmp/<本轮目录>/kb-map.json --context-file tmp/<本轮目录>/context.json --output tmp/<本轮目录>/acceptance-development-before.jsonl
```

正常路径16开发题最多提交16次分析、32次逻辑查询Embedding及16次聊天；最终24题全部执行则对应24/48/24。未知题若仍有候选也可能调用聊天，具体供应商尝试数和费用需另记，失败用量未知不能按零。耗时按同题/同语料/同配置记录；原始引用、漏答、关键限制、未知范围及词法误报都应逐条复核。
