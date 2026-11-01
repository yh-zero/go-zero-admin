# 后端剩余事项

更新：2026-11-01。以下为待配置或补验事项。

- [ ] 配置：同步宿主/容器 SMTP（465 + 隐式 TLS），确认 AI 容器 env 生效。
- [ ] 外部：真实邮箱验证码注册闭环、AI 长调用超时。
- [ ] 页面：后台个人头像/普通文件保存闭环、多窗口并发刷新与迟到响应。
- [ ] 异常：持续数据库/Redis/存储故障、缓存失效失败、panic/进程中断与文件引用恢复。
- [ ] 权限：最后管理员并发移除保护。
- [ ] 租户：多实例 DNS 隔离、大桶分页删除、创建中断恢复。

已完成结果及现行功能限制见 [API 契约](api-contract.md)，早期记录见 [Git 历史](https://github.com/yh-zero/go-zero-admin/blob/d1e128ea9bbf595c72d341281ab0975e78248945/ai_document/backend-api-integration-issues.md)。
