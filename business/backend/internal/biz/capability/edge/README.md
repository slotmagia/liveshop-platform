# edge

负责平台入口主机、CNAME 目标、Caddy on-demand TLS 准入、按 Host 选择上游，以及可选的 Caddyfile 生成与热加载。

自定义主机和店铺短码/子域名由 Identity 拥有。本 capability 实时查询 Identity 内部解析口，不复制绑定行，不保存证书私钥。直播房间码站点不在本切片。

Admin 配置页继续写入 `domain-base`。`edge.enabled=false`（本地默认）时设置仍保存，Ask/Route 仍可解析；只有启用后才向 Caddy admin 投递生成文件。Identity 查询与 Caddy `/load` 适配器在 `internal/common/edgehttp`，不进入 `biz`。
