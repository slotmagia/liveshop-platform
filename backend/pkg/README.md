# 公共 Go 包

该目录仅允许放可被仓库外消费者安全导入、具备稳定版本契约的域中立 Go 包。Platform 私有实现必须留在 `internal`；当前公开 Manifest 契约位于 `contracts` 独立 Go module。
