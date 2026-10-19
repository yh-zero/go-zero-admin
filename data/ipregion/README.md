# 离线 IP 归属地数据

本目录的 IPv4、IPv6 数据来自 [lionsoul2014/ip2region 官方仓库](https://github.com/lionsoul2014/ip2region)，固定使用 [v3.18.0 发布 commit](https://github.com/lionsoul2014/ip2region/tree/c1a1fc7d5941760db3f8431dc05c48cf7f0e30a1)。文件原样保存，下载日期为 2026-10-20；不在启动或查询时自动更新，也不调用外部 IP 查询服务。

对应 Go 查询模块：`github.com/lionsoul2014/ip2region/binding/golang v0.0.0-20260901011515-c1a1fc7d5941`。这些上游文件按 `Apache-2.0 OR MIT` 双许可证分发，原文保存在 [LICENSE.ip2region.md](LICENSE.ip2region.md)。

| 文件 | 字节数 | SHA-256 |
| --- | ---: | --- |
| `ip2region_v4.xdb` | 11122036 | `8e31bbdccb5bf21028af10592d4312ec975da0bffa108c0c5d862a12190f9ad3` |
| `ip2region_v6.xdb` | 37277813 | `939f6b46bd2b8bec3cf7c5ceb8ba782266ae9b1f35b5ba7916700dec0b7506ed` |

固定来源：[IPv4 数据](https://raw.githubusercontent.com/lionsoul2014/ip2region/c1a1fc7d5941760db3f8431dc05c48cf7f0e30a1/data/ip2region_v4.xdb)、[IPv6 数据](https://raw.githubusercontent.com/lionsoul2014/ip2region/c1a1fc7d5941760db3f8431dc05c48cf7f0e30a1/data/ip2region_v6.xdb)、[许可证](https://raw.githubusercontent.com/lionsoul2014/ip2region/c1a1fc7d5941760db3f8431dc05c48cf7f0e30a1/LICENSE.md)。

RPC 启动时读取两个文件到内存，合计约 46.2 MiB。默认路径分别是 `data/ipregion/ip2region_v4.xdb`、`data/ipregion/ip2region_v6.xdb`；Docker 将本目录复制到 `/app/data/ipregion`。文件缺失、地址版本不匹配或结构损坏会返回明确的初始化错误。

数据格式为 `国家|省|市|运营商|国家代码`，页面只显示国家、省、市并移除空项与重复项。私网、回环和本地链路地址显示“内网”，无效地址或未命中显示“未知”。归属地是该离线数据版本给出的网络地址区域，不能据此判断用户实时位置。

需要升级时，从明确的官方发布 commit 替换两个 `.xdb`，同步 Go 依赖、许可证、上表 SHA-256 和 `pkg/iplocation` 的数据校验测试，再运行 `go test ./pkg/iplocation`。数据更新不需要增加脚本。
