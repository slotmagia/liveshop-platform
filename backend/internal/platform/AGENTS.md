# Platform 领域规则

- Platform 是身份、IAM、模块 Registry、Platform Settings 和安全审计的事实所有者，不拥有订单、商品、支付等业务领域。
- Registry 的活动发布是路由、contribution 和有效权限目录的唯一来源。
- 激活、权限目录变更、Registry revision 和操作审计必须位于同一串行化事务。
- 停用模块只退役权限定义，不级联删除已有角色策略；有效授权必须忽略退役权限。
- Platform 控制面模块不能停用自身。
- Identity 修改密码或停用账号必须在同一事务撤销全部 refresh session。
- IAM 写入必须按租户隔离、校验版本，并在同一事务更新 revision 和审计。
- 变更这些不变量时必须同步更新 `backend/docs/domain/` 和真实 PostgreSQL 集成测试。
