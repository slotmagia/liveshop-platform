# Platform Admin Contribution 规则

- 前端只通过 `HostContext.api`/Host SDK 经 Gateway 调用已声明路由，不能直连模块后端。
- 服务端返回的任何字符串都必须通过 `textContent`、`value` 或安全 DOM API 写入；禁止拼入 `innerHTML`。
- 不保存 access token 或 Module Session，不从 URL、localStorage 或日志读取/输出令牌。
- contribution 必须支持独立 mount/unmount，不修改 Host 私有 DOM、Store 或全局样式。
- 权限只控制可见性和交互提示，后端仍是最终授权边界。
- 修改页面、事件或 action 时同步更新 Manifest fragment，并运行前端构建与 release 验证。
