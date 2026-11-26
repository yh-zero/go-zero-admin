# 后端开发脚本

## 脚本用途

| 文件 | 用途 |
|---|---|
| `gen.ps1` | 使用 goctl 生成 API、RPC、model 和 Swagger。修改 `.api`、`.proto` 或初始化 SQL 后运行。 |
| `verify.ps1` | 后端提交前完整自检：格式、编译、vet、依赖零差异、API 定义、目录白名单和配置扫描。 |
| `verify-network.ps1` | 检查 Docker 网络、代理地址、子网和端口配置。 |
| `build-images.ps1` | 交叉编译 Linux 二进制到 `docker/.build/`，供 Docker 镜像构建使用。 |

## 常用命令

```powershell
# 生成代码
./data/scripts/gen.ps1 api
./data/scripts/gen.ps1 rpc admin
./data/scripts/gen.ps1 model admin sys_user
./data/scripts/gen.ps1 swagger

# 后端自检
./data/scripts/verify.ps1

# 构建镜像所需二进制
./data/scripts/build-images.ps1
```

数据库迁移使用 `pkg/cmd/migrate` 或 Docker `migrate` 服务，不放在这里。
