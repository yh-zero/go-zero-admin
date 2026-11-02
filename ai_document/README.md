# ai_document

长期知识库：架构决策 / 契约 / 开发手册。过程性记录放 git 提交信息，不进本目录。

## 目录约定

- 只放长期知识；文档不复制命令细节（gen.ps1 等脚本为唯一入口）和规范本体（三层 AGENTS.md 为唯一事实源）
- 活待办完成即删行（git 历史可查）

## 文档索引

| 文档 | 内容 | 角色 |
|---|---|---|
| [refactor-plan.md](refactor-plan.md) | 架构决策（多租户/鉴权模型）、AI 平台三阶段演进、废弃项 | 决策与规划 |
| [api-contract.md](api-contract.md) | 前后端共享 API 契约 + 接口实现状态矩阵 | **唯一契约事实源** |
| [dev-guide.md](dev-guide.md) | 环境拓扑与联调 SOP、踩坑清单、登录猫动画维护 | 开发手册 |

数据库初始化唯一入口为 [mysql-all.sql](../go-zero-admin/data/db/init/mysql-all.sql)：新建库导入完整 SQL，Compose 空卷首启自动导入；模型由 `gen.ps1 model` 从已导入的开发库反射生成。

## 后端剩余事项

> 完成即删行；现行功能限制见 [api-contract.md](api-contract.md)。

> 完成即删行（git 历史可查）；现行功能限制见 [api-contract.md](api-contract.md)。

- [ ] 配置：同步宿主/容器 SMTP（465 + 隐式 TLS），确认 AI 容器 env 生效
- [ ] 外部：真实邮箱验证码注册闭环、AI 长调用超时
- [ ] 页面：后台个人头像/普通文件保存闭环、多窗口并发刷新与迟到响应
- [ ] 异常：持续数据库/Redis/存储故障、缓存失效失败、panic/进程中断与文件引用恢复
- [ ] 权限：最后管理员并发移除保护
- [ ] 租户：多实例 DNS 隔离、大桶分页删除、创建中断恢复
