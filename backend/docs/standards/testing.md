# 分级验证标准

## Fast

运行 `backend/tools/verify-fast.ps1`：格式、vet、单测、架构依赖、Manifest fragments、Buf lint、生成漂移和前端构建。普通逻辑修改最低执行此级。

## Module

运行 `backend/tools/verify-module.ps1`：包含 Fast、race、覆盖率门禁和可选真实 PostgreSQL 集成测试。SQL、事务、鉴权和并发修改必须执行此级。

## Release

运行 `backend/tools/verify-release.ps1`：包含 Module、依赖漏洞、npm audit、容器构建和契约 breaking 门禁。Manifest、Proto、migration、部署或发布依赖修改必须执行此级。

测试不能只断言接口返回值；一致性流程还要断言数据库最终状态、状态版本、审计/事件数量和重复执行后的副作用次数。
