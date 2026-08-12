# Platform 业务事实

## 事实所有权

| 事实 | 唯一事实源 | 派生消费者 |
|---|---|---|
| 不可变模块发布及当前激活版本 | Platform Registry PostgreSQL | Gateway 路由、Host contribution、能力目录 |
| 权限定义及是否有效 | 当前活动发布派生的权限目录 | IAM 角色策略、Module Session |
| 用户、角色、部门及数据范围 | Platform IAM PostgreSQL | Host 可见性、Gateway/模块授权 |
| 后台账号和 refresh session | Platform Identity PostgreSQL | Access Identity 签发与撤销 |
| 非敏感平台配置 | Platform Settings PostgreSQL | Platform 管理页面和运行消费者 |
| 安全与配置审计 | `platform_audit_event` | 审计查询、告警和合规导出 |

Manifest 只声明能力，不授予权限。Gateway 只消费路由快照，不拥有模块激活事实。前端状态、缓存和日志都不是业务事实源。
