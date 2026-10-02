# 前端模块菜单路径兼容迁移

更新日期：2026-10-03。

`20261003_frontend_routes_compat.sql` 是第14条迁移，不修改已应用的 `20261003_frontend_modules.sql` 或其校验和。旧迁移只去掉路径边缘的斜线，可能把已有 `/admin//audit` 与新种子的 `/admin/audit` 当作不同路径，登录后生成菜单时发生规范路径冲突。

新迁移从根目录和绝对路径解析所有有效菜单的完整路径，压缩连续斜线，并检查大小写规范后的路径冲突。它只调整五个模块中仍与原始种子一致、没有定制记录的菜单：名称、组件、相对路径、标题、排序和展示属性保持原值，创建时间等于更新时间，父目录仍为原系统管理目录，且没有子菜单或菜单参数。

发生冲突时，保留其他自定义菜单的原始数据，仅将可确认的原始种子改为 `<原路径>-module-<菜单ID>`。如果候选路径被占用，依次尝试 `-1` 至 `-99`；同时检查原始相对路径和规范完整路径。菜单ID、角色菜单授权、按钮及其授权均保留，修复与 `repairFrontendRoute` 审计在同一事务提交。无冲突菜单不变，重复执行不会重复修复或重复写入审计。

迁移输出 `UNCHANGED`、`REPAIRED`、`MISSING` 或 `SKIPPED`。种子已定制、路径无法确认，或100个候选都被占用时输出 `SKIPPED`，保留原数据，需人工处理既有冲突。它不会自动修复两个自定义菜单之间的冲突。

`test/sh/db-regression.ps1` 在隔离 MySQL 库验证：旧13条迁移产生重复规范路径；新迁移修复冲突；首候选被占用时选择下一候选；自定义菜单原数据和管理员授权保留；其他模块路径不变；重复执行幂等；已定制的种子输出 `SKIPPED`。原有迁移失败停止、重试和校验和保护回归也保留。

本地开发库已先备份至 `bin/db-backups/gozero-development-20261003-031600-853.sql`，再应用新迁移。五个模块均输出 `UNCHANGED`，最终14条迁移全部为 `APPLIED`。默认路径仍为 `/admin/audit`、`/admin/files`、`/admin/sessions`、`/admin/organization/departments`、`/admin/organization/positions`。

`test/sh/dev.ps1` 的 `Start`、`Stop`、`Restart`、`Status` 现在使用项目运行目录 `bin/dev/managed/operation.lock` 的独占文件句柄，覆盖完整操作和失败清理。并发调用立即报告 `Another backend operation is in progress`；`finally` 释放句柄，进程异常退出由操作系统释放，锁文件保留。已用独立隐藏 PowerShell 持锁验证另一个 `Status` 被拒绝，释放后 `Status` 成功；此测试没有启动第二套服务。记录位于 `bin/dev/lock-regression-f1071b9c9bf047a19b4b478db403ed01`。
