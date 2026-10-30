# 踩坑清单（历史事故记录）

> 规范条目背后的真实事故。规则本体在各 AGENTS.md，本文件只讲事故与成因。

## 生成物相关

1. **pb 字段名凭记忆书写**（事故 ×5+）：`CaptchaId`/`CaptchaID`、`UserId`/`UserID` 反复写错——proto snake_case 生成的 Go 字段大小写不直觉。对策：根规范生成物查证纪律。
2. **正则批量转 proto 字段**：CamelCase→snake_case 正则把 `id`→`i_d`、`IP`→`i_p`，286 个字段逐个修。对策：禁止正则批量替换 desc/ 文件。
3. **proto package 全是 `pb`**：多模块同 package 致 protobuf 全局注册冲突，编译全过、运行时 panic。对策：后端规则 package 唯一性。
4. **goctl 对已有 main 跳过生成**：proto 加 service 后 `pb.go` 不自动注册，运行时 `unknown service`。对策：`gen.ps1 rpc` 注册检查告警。
5. **etc 双 yaml**：goctl 按 package 名生成 `aipb.yaml` 与手写 `pb.yaml` 并存。对策：`--name-from-filename` + 只保留 `<模块>.yaml`。

## 数据层

6. **`QueryRowsCtx` 直扫 pb 类型**（事故 ×7）：pb 无 `db` tag，运行时 `not matching destination to scan`。对策：本地 row struct 过渡。
7. **DATETIME 直扫 int64**：dept/role 报"数据库错误"根因。对策：SQL 层 `UNIX_TIMESTAMP()`。
8. **goctl model 缓存幽灵数据**：直接改库后 `FindOneByAccount` 命中旧缓存。对策：改库后 `FLUSHALL`（dev-env.md §3.2）。

## 环境/工具链

9. **版本漂移**：`go get go-zero@v1.10.3` 静默失败，文档写 1.10.3 实际 1.10.0。对策：验证实际生效版本。
10. **端口被旧项目占用**：旧项目 `gozero-api` 容器占 7001，新服务"启动成功"但请求全 404。对策：dev-env.md §3.1。
11. **跨模块 RPC 复用连接**：`FileRpc` 复用 adminRpc 连接指向 6001，`unknown service`。对策：每个外部模块独立 client 连接。
12. **docker exec 中文乱码**：管道传 SQL 中文乱码；改用 `docker cp`。对策：dev-env.md §3.3。
13. **ps1 无 BOM**：UTF-8 无 BOM 脚本中文注释解析崩溃。对策：dev-env.md §3.4。

## 流程

14. **积攒多模块一次性 e2e**：三模块攒一起测，一次爆 3 类错误。对策：每模块立即验证（后端规则 11）。
15. **API 层直连数据库**：Casbin 初期引入，后重构为 RPC 拉策略。对策：api 层必须经 RPC client。
