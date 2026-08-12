# 编码 Agent 驱动的业务模块开发

> 适用对象：Codex 等编码 Agent、工程师和评审者
> 范围：基于 LiveShop 架构创建或演进业务模块
> 非目标：在 Platform 中实现 Agent Run、Task、Lease、Checkpoint、Approval 或调度运行时

## 1. Agent 的输入

每个业务任务开始前，编码 Agent 必须读取当前目录向上的 `AGENTS.md`，再读取模块 `backend/docs/domain/` 中五份业务契约：

- `FACTS.md`：唯一事实源和数据所有权；
- `INVARIANTS.md`：任何故障与并发下都不能破坏的条件；
- `STATE-MACHINE.md`：合法状态、转换前置条件和终态；
- `TRANSACTIONS.md`：事务边界、Outbox/Inbox、失败窗口与恢复；
- `EXTERNAL-CONTRACTS.md`：HTTP、gRPC、事件及外部副作用。

如果关键事实、不变量或状态机没有业务依据，Agent 应停止实现并提出最小澄清问题，不能用技术假设替代业务决策。

## 2. 创建新模块

```powershell
./backend/tools/new-business-module.ps1 `
  -Destination ../liveshop-order `
  -ModuleId order `
  -ModuleName "Order" `
  -GoModule github.com/liveshop/order
```

生成结果采用独立仓库、独立数据库和独立发布单元。依赖方向固定为：

```text
HTTP / gRPC transport
          ↓
application use cases
          ↓
domain facts + ports
          ↑
PostgreSQL / broker / external adapters
```

GoFrame 只存在于 HTTP 契约和 transport；Proto 只存在于 gRPC transport；domain 不依赖框架。HTTP 与 gRPC 复用同一个 application service。

## 3. 单次业务变更闭环

1. 将需求写成事实、不变量、状态转换和验收场景。
2. 搜索现有调用方、契约、migration 和事件消费者，确定影响面。
3. 先变更机器契约：Manifest operation、Proto、事件 schema 或 migration。
4. 实现唯一主路径，同时迁移仓库内调用方；不创建新旧并行实现。
5. 对一致性敏感写入实现条件更新、稳定幂等键、Outbox/Inbox、有限重试和结果未知恢复。
6. 验证正常、非法转换、重复、并发、乱序、超时、提交后发布失败和进程崩溃窗口。
7. 依次运行快速、模块和发布门禁，将可复现命令与未决外部阻塞写入交付说明。

## 4. 并行协作边界

Agent 可以按 `ownership.yaml` 并行处理互不重叠的纵向切片。`module.json`、Proto、migration 序号、组合根和发布脚本在一个变更周期内各保持一个写入者。跨切片通过公开 application port、Proto 或事件契约协作，禁止通过其他模块的 `internal`、DAO 或数据库联调。

## 5. 验证层级

```powershell
./backend/tools/verify-fast.ps1
./backend/tools/verify-module.ps1
./backend/tools/verify-release.ps1
```

- Fast：格式、静态检查、单测、架构边界、Manifest/Proto 漂移和前端构建。
- Module：race、覆盖率和真实 PostgreSQL migration/一致性集成测试。
- Release：漏洞审计、Proto breaking baseline、依赖独立性和容器构建。

完成的含义是业务不变量在故障后仍能收敛，而不只是接口返回成功。
