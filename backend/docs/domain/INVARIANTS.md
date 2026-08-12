# Platform 业务不变量

- `(module_id, version)` 一经注册只能对应一个规范化摘要和能力契约。
- 只有已注册发布可以激活；每个模块最多一个活动版本。
- 任意两个活动模块在同一 surface 上不能拥有相同或相互包含的路由前缀。
- 激活失败不能改变活动版本、有效权限目录、revision 或审计结果。
- 重复注册相同内容和重复激活当前版本必须收敛且不能递增 revision。
- 停用后下一快照不再包含对应路由和 contribution，有效授权不再包含其退役权限。
- 一个权限码只能由一个模块拥有；退役权限不级联删除历史角色策略。
- IAM、Identity、Settings 和 Registry 的权威写入与成功审计必须原子提交。
- 租户事实必须由已验证身份 claims 提供，不能由 body、query 或自定义 header 覆盖。
- refresh token 只能成功轮换一次；检测到重用后撤销整个 session family。
