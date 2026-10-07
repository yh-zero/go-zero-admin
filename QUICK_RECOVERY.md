# 演示系统快速恢复

部署只在 `/yanghao/gozeroadmin/` 下操作。区分三种情况：服务异常先重启；数据库或演示数据被破坏时新建独立环境；服务器或磁盘丢失时使用本机保留的发布包重新部署。

当前公开账号为 `admin / 123456`，固定只读角色9527。私有维护账号 `demo-maintainer` 的随机密码在各环境的 `runtime/ADMIN_CREDENTIALS.txt`（0600）中。公开弱密码账号的全部写权限变更被自动审批拒绝；本工具保留只读限制，不实施该扩权。AI 不配置 Key，保持禁用。

## 1. 服务异常：先恢复原系统

进入当前发布版本的 `backend`，保留数据库与配置：

```sh
cd /yanghao/gozeroadmin/releases/当前版本/backend
sh docker/demo.sh start
sh docker/demo.sh status
```

检查 `https://yh9527.top/`、验证码和正常登录。数据库损坏、迁移状态错误时不要重复初始化旧目录，改用下一节的新环境。

## 2. 数据被破坏：准备全新空数据库

本机提前执行 `powershell -File test/sh/package-demo.ps1`，保留发布包和 `package-summary.json` 中的 SHA256。新发布包包含 Linux 二进制、前端、`data/db/gozero-admin.sql`、增量迁移和恢复工具，不含 `.env`、AI Key、业务数据和证书。数据库流程调整后须重新打包；历史发布包仍使用包内原流程。

把已验证的发布包上传到 `/yanghao/gozeroadmin/uploads/`。从当前发布版本的后端执行：

```sh
cd /yanghao/gozeroadmin/releases/当前版本/backend
python3 docker/demo-rebuild.py \
  /yanghao/gozeroadmin \
  /yanghao/gozeroadmin/uploads/新发布包.tar.gz \
  发布包SHA256 \
  yh9527.top 175.178.67.80 \
  --previous-backend /yanghao/gozeroadmin/releases/当前版本/backend \
  --reuse-tls
```

SHA256 必须与本机可信打包结果一致，不使用远端待验证文件临时生成的值代替。工具拒绝越界路径、符号链接、硬链接、特殊文件及发布包中的私有配置。

工具只创建 `/yanghao/gozeroadmin/rebuilds/唯一编号/`，先验证、暂存解压、生成独立配置，再输出 `RECOVERY_COMMANDS.md`；不会运行 Docker、数据库迁移或停止旧系统，也不会读取旧 `.env`。每套环境使用独立 Compose 项目名、独立 `runtime` 和独立数据库。

`--reuse-tls` 只将旧环境 Caddy 证书复制到新环境私有目录，减少重复申请证书。旧证书目录不存在或需要从零申请时移除此参数。证书不能进入公开发布包。

准备成功后按生成文档操作：旧 MySQL 可读时先停止写入服务并备份；数据库已损坏时不强制要求备份成功。停止旧项目全部七个服务（保留数据），新环境执行 `sh docker/demo.sh init`、`start`、`status`。两套系统都使用 `80/443` 和本机 `127.0.0.1:7001`，2 GB 服务器不要同时启动两套。服务器现有 Docker 镜像层可以复用，无需重新安装 Go 和 Node；正常准备后切换需要几分钟，首次下载镜像更久。

新版演示专用 `init` 先启动 MySQL、Redis、etcd，将同一 `data/db/gozero-admin.sql` 显式导入独立空库，再处理演示账号；全量 SQL 已含当前结构与迁移记录。Compose 不自动导入 SQL，通用 `db.sh` / `db.ps1` 仅提供状态、备份和迁移。该步骤只准备数据库，后续 `start` 才启动应用与站点。

登录验证新系统成功后记录新后端路径。后续恢复时 `--previous-backend` 使用实际运行环境的 `.../rebuilds/编号/releases/编号/backend`；工具仍以 `/yanghao/gozeroadmin` 为授权根目录。不要自动删除旧系统，确认稳定后自行规划保留周期。

新系统失败时使用生成文档中的回退命令：先 `stop` 新项目全部七个服务，再在旧后端运行 Compose `up -d mysql redis etcd rpc ai-rpc api web`。旧业务数据仍在原 `runtime/mysql`。不要执行 `down -v`、删除、搬移或复用旧数据库目录。

## 3. 整台服务器或磁盘丢失

同一服务器的 `rebuilds`、`runtime/backups` 都不是异地灾备。需要在本机保留无密钥发布包及 SHA256；若需要恢复业务数据，另行保存经过校验的数据库备份到服务器之外。`sh docker/demo.sh backup` 只生成本机服务器内备份，随后需要人工安全转存。

新服务器已有 Docker 和 Python 3.11/3.12 时，将发布包上传到新服务器授权项目目录，按 `docker/部署说明.md` 的 `prepare → init → start` 重新部署；首次库由演示专用 `init` 导入全量 SQL。恢复旧业务数据需要使用匹配的迁移版本与专门的恢复步骤；本快速工具只准备空数据库，不自动导入业务备份。

本轮不创建定时任务、异地账号连接、自动删库、旧目录清理或主机配置修改。
