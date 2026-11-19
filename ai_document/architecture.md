# 系统架构

## 服务与数据边界

平台由一个 HTTP API 和 admin、app、file、ai、tenant 五个 RPC 服务组成。API 验证请求、建立身份上下文并调用 RPC。RPC 编排业务，由 model 访问 MySQL。服务使用配置中的 Endpoints 直连。建库以完整初始化 SQL 为来源，已有库按实际差异维护。[平台 Compose](../go-zero-admin/docker-compose.yml) · [数据初始化](dev-guide.md#数据初始化)

后台与 App 使用独立的 JWT、签发者和 Redis 会话空间。后台接口按身份、部门与操作授权；用户数据范围另在服务端查询和修改时校验。文件服务管理 RustFS 对象与业务引用，跨服务修改头像后按当前数据库值对账。响应、会话生命周期和引用失败语义见[共享契约](api-contract.md)。

## 部门授权

Casbin 按用户、部门域、路由和 HTTP 方法作接口授权。部门域写作 `dept:<id>`，全局角色使用 `*`。平台 API 读取 admin-rpc 的完整策略，在内存中用普通 Casbin Enforcer 鉴权；收到 Redis 策略更新通知后构建新实例并原子替换。[鉴权模型](../go-zero-admin/pkg/casbinauth/enforcer.go) · [API 策略加载](../go-zero-admin/application/admin/api/internal/svc/service_context.go)

请求可通过 `X-Dept-Id` 选择当前用户有权使用的部门。RPC/model 按有效角色合并 `ALL`、`DEPT_AND_CHILDREN`、`DEPT`、`SELF` 范围，利用部门物化路径查下级。菜单可见、接口授权和行级访问分别检查。租户隔离由各自的服务栈及数据库承担。[用户数据范围](../go-zero-admin/application/admin/rpc/internal/model/sys_user_model.go)

## 平台与租户部署

平台业务库保存 `tenant_registry`。只有平台栈运行可操作 Docker socket 和 Nginx 配置的 tenant-rpc；租户后台操作限定平台超级管理员。每个租户独立运行 API、admin/app/file/ai 四个 RPC、MySQL 和 Redis，使用专属持久卷。RustFS 为平台共享，租户使用各自的公开桶与私有桶。[平台 Compose](../go-zero-admin/docker-compose.yml) · [租户模板](../go-zero-admin/docker/tenant-stack/docker-compose.tenant.yml)

| 网络 | 服务与用途 |
|---|---|
| 平台默认网络 | Nginx 按专属别名 `platform-api` 访问平台 API，平台服务通过内网连接数据库与 RPC。开发 Compose 发布 Nginx 的 80 端口，API、MySQL、Redis、Qdrant 与 RustFS 的调试端口绑定宿主回环地址。 |
| 租户默认网络 | 租户应用、数据库和 Redis 内部通信。admin-rpc、file-rpc 有租户专属别名，租户模板没有发布宿主端口。 |
| 共享边缘网络 | 平台 Nginx、RustFS 和租户的 API、file-rpc 接入。Nginx 通过 `<slug>-api` 转发租户子域名，file-rpc 经此网络访问对象存储。平台 Nginx 同时接入平台默认网络。 |

创建时先核对租户标识、资源冲突和本机镜像，写入 `creating` 登记及专属运行配置。控制面先以 `--no-build --pull never` 启动 MySQL/Redis，初始化数据库和管理员，创建桶后置 `admin_account_ready`。随后以同一镜像策略启动全栈，等待租户 API 的 `/ready`，校验并重载 Nginx 路由后才置 `active`。失败保留阶段与安全错误信息。[创建实现](../go-zero-admin/application/tenant/rpc/internal/logic/tenantservice/create_tenant_logic.go)

再次启用要求管理员已初始化，并复用镜像启动、探测和恢复路由。停用撤除路由并停止容器。重置管理员密码后同时失效旧会话和缓存；删除须确认 slug，再撤除路由、清理该租户的容器与卷、桶及配置，最后软删除登记。[启用与路由](../go-zero-admin/application/tenant/rpc/internal/logic/tenantservice/start_tenant_logic.go) · [删除](../go-zero-admin/application/tenant/rpc/internal/logic/tenantservice/delete_tenant_logic.go)

## AI、检索与执行

- 聊天通过 Eino 和已注册工具调用模型。服务端绑定操作者，并在模型、工具和步骤之间保留请求身份；每次真实供应商调用尝试用独立账本计量。消息与步骤保存的用量供结果展示。[共享契约](api-contract.md#用量与额度)
- 知识库正文、归属、分块与摄取作业以 MySQL 为准。Qdrant 保存可重建的向量索引，查询先按归属筛选，再回 MySQL 复核。租户模板未部署独立向量服务，现行平台集合仅对应一个业务库。
- 文档摄取使用持久批次与作业，未领取的作业可跨重启继续。工作流按冻结定义顺序执行，审批以任务断点为前置条件，进程中断后不自动重放副作用操作。质量评测按冻结题集、资料及模型配置核对，未知结果保留原标识供重查。[执行契约](api-contract.md#工作流执行定义与运行记录)
