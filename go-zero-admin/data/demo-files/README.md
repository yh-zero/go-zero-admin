# 演示文件

三张 128×128 PNG 由 Go 标准库 `image` / `image/png` 绘制，不含个人信息，也不依赖外部图片服务。它们用于全新安装的文件资源、头像和公开/私有下载示例。

| 素材 | Bucket 配置 | 对象 key | MIME | 字节数 |
|---|---|---|---|---:|
| `public-landscape.png` | `STORAGE_PUBLIC_BUCKET` | `go-zero-admin/00000000-0000-4000-8000-000000000101.png` | `image/png` | 703 |
| `public-shapes.png` | `STORAGE_PUBLIC_BUCKET` | `go-zero-admin/00000000-0000-4000-8000-000000000102.png` | `image/png` | 611 |
| `private-lock.png` | `STORAGE_PRIVATE_BUCKET` | `go-zero-admin/00000000-0000-4000-8000-000000000103.png` | `image/png` | 525 |

`docker/Dockerfile.storage-init` 将素材打包到镜像，`docker/storage-init.sh` 在创建 Bucket 和应用身份后自动补齐这些固定 key。本地和服务器部署 Compose 共用该镜像及脚本，无需手工上传。写入使用 `If-None-Match: *`，已有对象保持原样，其他上传错误会使初始化失败。脚本不修改其他业务对象，也不删除文件。

数据库中的演示文件大小与上述素材一致，文件引用由 `data/db/seeds/demo.sql` 设置。修改素材时需同步检查 fresh seed 中的大小；已有同名对象不会因素材更新被覆盖。

SHA-256：

```text
7772989143bba49faa462c18219809dae69486c947caa6f19f9c39355df6fa02  public-landscape.png
ee3f8806ae20a293d2b98ca744461ef5a800c647a8607cb743f5f6c9147cd8ca  public-shapes.png
80ddafabb3cfe17b9ec4300bd7a6ca3d743b4a35036a215b017352b92c4d3b2a  private-lock.png
```
