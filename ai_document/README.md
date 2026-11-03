# ai_document

长期知识库：架构决策 / 契约 / 开发手册。过程与验证记录随对应任务写入 Git 提交说明或 PR，不另维护只追加的流水账；当前限制与待办保留在本目录。

## 目录约定

- 只放长期知识；文档不复制命令细节（gen.ps1 等脚本为唯一入口）和规范本体（三层 AGENTS.md 为唯一事实源）
- 活待办完成即删行（git 历史可查）

## 文档索引

| 文档 | 内容 | 角色 |
|---|---|---|
| [refactor-plan.md](refactor-plan.md) | 架构决策（多租户/鉴权模型）、AI 平台三阶段演进、废弃项 | 决策与规划 |
| [ai-agent-roadmap.md](ai-agent-roadmap.md) | AI Agent 阶段建议、长期功能与首版边界 | 待开发规划 |
| [api-contract.md](api-contract.md) | 当前响应、字段、权限、模块行为与能力边界；不维护历史验收矩阵 | **唯一契约事实源** |
| [dev-guide.md](dev-guide.md) | 环境配置、构建启动、开发验证、排障清理及专项维护入口 | 开发手册 |

数据库初始化唯一入口为 [mysql-all.sql](../go-zero-admin/data/db/init/mysql-all.sql)：新建库导入完整 SQL，Compose 空卷首启自动导入；模型由 `gen.ps1 model` 从已导入的开发库反射生成。

## 后端剩余事项

> 完成并验证后删行（git 历史可查）；现行功能限制见 [api-contract.md](api-contract.md)。规范修订不代表下列实现问题已修复。

- [ ] 配置：对齐宿主/容器 SMTP Host、Port、ImplicitTLS（当前容器 Host 仍依赖模板未列的 env，宿主 465 与 false 不匹配）；隐式 TLS 分支已存在，需配置与真实邮件验证，不重复开发
- [ ] 外部：真实邮箱验证码注册完整闭环、AI 工具调用与长调用超时；历史单次 SMTP 发信或模型回复不等于这些链路已覆盖
- [ ] 页面：浏览器实际上传并保存后台个人头像/普通文件闭环、多窗口并发刷新与迟到响应；接口正常路径或模拟页面响应不能替代真实闭环
- [ ] 异常：持续数据库/Redis/存储故障及故障注入、缓存失效失败、panic/进程中断与跨服务连续失败时的头像/文件引用恢复；对账正常路径不证明跨库原子或全部故障恢复
- [ ] 权限：最后管理员并发移除保护
- [ ] 租户：当前版本按需复验实际生命周期、多实例 DNS 隔离、大桶分页删除、创建中断恢复；历史生命周期验收不能推定中断/大数据场景已覆盖，菜单读取也不能替代生命周期验证

## 规范与自动化待对齐

> 2026-11-03 静态核对；本次仅优化规范，未改业务、CI 或本地 hook，以下均未完成。

- [ ] 配置覆盖：当前只有 AI/App 主入口启用 `conf.UseEnv()`；评估并统一 [admin-api](../go-zero-admin/application/admin/api/admin.go)、[admin-rpc](../go-zero-admin/application/admin/rpc/admin.go)、[file-rpc](../go-zero-admin/application/file/rpc/file.go)、[tenant-rpc](../go-zero-admin/application/tenant/rpc/tenant.go) 及 [tenantStorage 二次加载](../go-zero-admin/application/tenant/rpc/internal/logic/tenantservice/create_tenant_logic.go)，补直填/占位符/缺失变量回归，不以文档声明替代实现
- [ ] AI 初始化一致性：[service_context.go](../go-zero-admin/application/ai/rpc/internal/svc/service_context.go) 只更新局部 `ai.Enabled`，保存的 `Config.AI.Enabled` 未同步；辅助函数注释声称“配置无效返回 nil”，代码实际 panic。按已有契约核对运行状态与错误语义，不能借纯重构静默改变行为
- [ ] CI：[前端任务](../.github/workflows/ci.yml) 缺少本地门槛 `pnpm lint`；补齐并验证，本地继续执行类型/lint/构建，不删规范迁就 CI
- [ ] 本地 hook：现有 `.git/hooks/pre-commit` 报错拼入密钥值且读取工作区而非暂存 blob，需停止回显值、扫描实际暂存内容；只在独立临时仓库验证“暂存/工作区不一致、合法占位符、拒绝凭据”等场景。该 hook 未纳入版本控制，本轮不运行或修改
- [ ] 自检可信性：[verify.ps1](../go-zero-admin/data/scripts/verify.ps1) 的 gofmt 仅看输出且丢弃 stderr，需补非零退出断言；其余 go list/vet/build/tidy/API 已有退出码检查，不笼统记作全部缺失。YAML 字段扫描不覆盖 DSN 内密码或所有文件，不能视为全仓密钥防护
- [ ] 回归沉淀：当前自检不运行 Go 测试，尚无保留的 `*_test.go`；沿用现有 [登录 HTTP 验证](../go-zero-admin/test/login_e2e.ps1)，按受影响功能补持久回归与入口，外部环境需求单列，不以空测试命令成功冒充行为覆盖
