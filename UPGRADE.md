# go-zero / goctl 升级操作清单（1.6.0 → 1.10.3）

> 升级日期：2026-08-16
> 适用范围：本项目（go-zero-admin）
> 结论：go-zero v1.10.3 要求 Go ≥ 1.24，本项目只使用核心包，升级后编译零错误，业务代码无需改动。

---

## 一、升级前状态

| 项目 | 升级前版本 | 升级后版本 | 说明 |
| --- | --- | --- | --- |
| Go 工具链 | 1.26.6 | 1.26.6（不变） | 本地已是最新，满足 go-zero v1.10 要求（≥ 1.24） |
| go.mod `go` 指令 | 1.21 | 1.24.0 | go-zero v1.10 强制要求，`go get` 时自动提升 |
| go-zero | v1.6.0 | v1.10.3 | 官方最新（2026-07-31 发布） |
| goctl | 1.10.2 | 1.10.2（即最新） | 官方最新，自报版本号未 bump，已包含 v1.10.3 时点的修复 |

---

## 二、操作步骤

### 步骤 1：升级 go-zero 依赖

```powershell
# 项目根目录执行
go get github.com/zeromicro/go-zero@v1.10.3
go mod tidy
```

自动完成的变更：

| 依赖 | 升级前 | 升级后 |
| --- | --- | --- |
| go 指令 | 1.21 | 1.24.0 |
| github.com/zeromicro/go-zero | v1.6.0 | v1.10.3 |
| github.com/redis/go-redis/v9 | v9.0.3 | v9.21.0 |
| google.golang.org/grpc | v1.59.0 | v1.80.0 |
| google.golang.org/protobuf | v1.31.0 | v1.36.11 |
| go.opentelemetry.io/otel | v1.19.0 | v1.40.0 |
| github.com/golang-jwt/jwt/v4 | v4.5.0 | v4.5.2（含安全修复） |
| github.com/go-sql-driver/mysql | v1.7.1 | v1.10.0 |
| go.etcd.io/etcd/client/v3 | v3.5.10 | v3.5.21 |
| golang.org/x/crypto | v0.14.0 | v0.48.0 |

### 步骤 2：编译验证

```powershell
go build ./...
```

本项目只使用 go-zero 核心包（logx / conf / rest / rest/httpx / zrpc / core/stores/redis / trace / metric / service），
这些包 1.6 → 1.10 API 完全兼容，**业务代码零改动**。

### 步骤 3：升级 goctl 工具

> 注意：goctl 是 go-zero 仓库的嵌套模块（`tools/goctl` 有独立 go.mod），**没有独立的版本 tag**，
> `go install ...@v1.10.3` 会报错 `does not contain package`，必须用 `@latest`。

```powershell
go install github.com/zeromicro/go-zero/tools/goctl@latest

# 验证（自报 1.10.2 即为官方最新，版本常量未 bump，已包含 v1.10.3 时点的 goctl 修复）
goctl --version
```

### 步骤 4：导出新版代码生成模板

```powershell
# 项目根目录执行，导出 1.10.3 原生模板
goctl template init --home test/goctl/1.10.3
```

### 步骤 5：合并自定义模板改动

排查方法：克隆 v1.6.0 源码导出原生模板，与 `test/goctl/1.6.0` 逐文件 diff：

```powershell
git clone --depth 1 --branch v1.6.0 https://github.com/zeromicro/go-zero.git $env:TEMP\gozero-v1.6.0
cd $env:TEMP\gozero-v1.6.0\tools\goctl
go run . template init --home $env:TEMP\goctl-tpl-1.6.0-native

# 项目根目录对比
git diff --no-index --ignore-cr-at-eol $env:TEMP\goctl-tpl-1.6.0-native test\goctl\1.6.0
```

排查结论：仅 2 个文件被自定义过，需手工合并到 1.10.3 模板（不能整文件覆盖，官方在 1.10.3 也改了这两个文件）：

| 文件 | 自定义内容 |
| --- | --- |
| `test/goctl/1.10.3/api/handler.tpl` | 统一响应封装：`result.HttpResult(r, w, resp, err)` 替换默认的 httpx.ErrorCtx/OkJsonCtx 分支；import 增加 `go-zero-admin/pkg/result` |
| `test/goctl/1.10.3/api/main.tpl` | JWT 过期统一响应：`rest.WithUnauthorizedCallback` 返回 `xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)`；import 增加 `net/http`、`go-zero-admin/pkg/result`、`go-zero-admin/pkg/result/xerr` |

### 步骤 6：验证模板可用

```powershell
# 用一个小 api 文件实际生成一次，确认模板语法与变量在新版 goctl 下正常渲染
goctl api go -api test.api -dir . --home test\goctl\1.10.3
# 检查生成的 handler 是否带 result.HttpResult、main 是否带 WithUnauthorizedCallback
```

### 步骤 7：冒烟验证

```text
1. 启动依赖：docker compose up -d（MySQL / Redis / Etcd）
2. 启动 rpc：cd application\applet\rpc && go run .\applet.go
3. 启动 api：cd application\applet\api && go run .\applet.go
4. 验证登录、casbin 鉴权、JWT 过期返回格式等核心链路
```

---

## 三、需要修改的文件清单

| 文件 | 操作 | 说明 |
| --- | --- | --- |
| `go.mod` | 自动 | `go get` + `go mod tidy` 自动更新（go 指令与依赖版本） |
| `go.sum` | 自动 | 随 `go mod tidy` 更新 |
| `test/goctl/1.10.3/`（整套新增） | 新增 | `goctl template init` 导出的新版原生模板 |
| `test/goctl/1.10.3/api/handler.tpl` | 手工 | 合并统一响应封装自定义 |
| `test/goctl/1.10.3/api/main.tpl` | 手工 | 合并 JWT 过期统一响应自定义 |
| `README.md` | 手工 | 末尾追加「goctl 模板说明」 |
| `test/sh/api.bat` | 可选 | `--home` 由 `test/goctl/1.6.0` 改为 `test/goctl/1.10.3` |
| `test/sh/rpc.bat` | 可选 | 同上 |
| 业务代码（application/、pkg/） | 无需改动 | 核心包 API 兼容，编译零错误 |

---

## 四、1.6.0 → 1.10.3 官方模板差异（供参考）

新增模板：

- `api/handler_test.tpl`、`api/logic_test.tpl`、`api/svc_test.tpl`、`api/integration_test.tpl`（单元/集成测试）
- `api/sse_handler.tpl`、`api/sse_logic.tpl`（SSE 支持）
- `model/customized.tpl`（自定义方法模板）

主要修改：

- `api/handler.tpl`、`api/main.tpl`（头部增加 `// Code scaffolded by goctl` 注释、`{{.version}}`、`{{.Doc}}`）
- `docker/docker.tpl`、`mongo/model.tpl`、`rpc/call.tpl`、`model/import.tpl` 等

---

## 五、遗留事项

1. `go vet ./...` 存在 4 个历史遗留警告（非本次升级引入，升级前就存在）：
   - `application/applet/api/internal/types/types.go:393` — struct tag `form:"id":"id"` 格式错误
   - `application/applet/rpc/internal/logic/menu/get_menu_authority_logic.go:63` — `fmt.Println` 直接打印 protobuf 消息（拷贝内部锁）
   - `application/applet/rpc/internal/logic/authority/update_authority_logic.go:39` — 同上
   - `application/applet/rpc/internal/logic/menu/get_menu_tree_logic.go:78` — `%s` 格式化 `*[]SysAuthorityBtn` 类型不匹配
2. 可选升级（本次未做）：
   - gorm v1.25.5 → v1.30.x
   - golang-jwt/jwt v4 → v5（官方主推 v5，涉及 API 变更）
3. 临时对比产物（可删除）：`%TEMP%\gozero-v1.6.0`、`%TEMP%\goctl-tpl-1.6.0-native`、`%TEMP%\goctl-tpl-1.10.3-native`、`%TEMP%\goctl-tpl-verify`
