# Platform 状态机

## 模块发布

```text
REGISTERED ──activate──> ACTIVE ──replace/deactivate──> REGISTERED
```

- `REGISTERED → ACTIVE`：发布存在、摘要不变、路由无冲突；同步有效权限并递增 revision。
- `ACTIVE → REGISTERED`：停用或被另一版本替换；移除运行快照并退役对应权限。
- Platform 自身不允许通过管理 API 执行 `ACTIVE → REGISTERED`。

## 权限定义

```text
ACTIVE ──release removed/deactivated──> RETIRED ──future release declares──> ACTIVE
```

`RETIRED` 记录可保留历史策略引用，但不能用于新授权或有效权限计算。物理清理由独立数据迁移完成。

## Refresh token

```text
ACTIVE ──rotate──> USED
ACTIVE ──logout/account change──> REVOKED
USED ──reuse detected──> session REVOKED
```
