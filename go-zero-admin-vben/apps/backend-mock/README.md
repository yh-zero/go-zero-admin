# @vben/backend-mock

## Description

Vben Admin 数据 mock 服务，不连接数据库，所有数据均为模拟值，用于独立示例和测试。当前业务应用 web-antdv-next 已关闭 Mock，登录、菜单及业务接口使用 go-zero-admin；该服务不会随当前业务应用启动，也不能替代真实后端或验收结果。

## Running the app

```bash
# development
$ pnpm run start

# production mode
$ pnpm run build
```
