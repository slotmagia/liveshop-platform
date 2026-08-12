# 运维命令

该目录只放可独立发布的迁移、校验或运维命令。Platform 主进程入口保持在 `internal/platform/cmd`，避免根级命令重复装配服务依赖。
