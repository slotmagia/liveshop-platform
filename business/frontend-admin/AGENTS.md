# Platform Admin Contribution 规则

- 前端只通过 `HostContext.api`/Host SDK 经 Gateway 调用已声明路由，不能直连模块后端。
- 服务端返回的任何字符串都必须通过 `textContent`、`value` 或安全 DOM API 写入；禁止拼入 `innerHTML`。
- 不保存 access token 或 Module Capability，不从 URL、localStorage 或日志读取/输出令牌。
- contribution 必须支持独立 mount/unmount，不修改 Host 私有 DOM、Store 或全局样式。
- 权限只控制可见性和交互提示，后端仍是最终授权边界。
- 修改页面、事件或 action 时同步更新 Manifest fragment，并运行前端构建与 release 验证。

## 模态框与遮罩

- 后台 iframe 的普通表单、提示、确认框和可序列化只读详情只能使用 `hostFormModal()`，由 Host 顶层渲染，打开前后 iframe 尺寸及 Host 布局不得变化。只有确实无法表达为 Host 字段的富交互内容才允许共享 `modal()` + `hostOverlay()`；不得用 `hostOverlay()` 实现菜单管理、普通详情或确认提示。禁止自建 modal/backdrop 组件。
- 遮罩必须由 Host 覆盖整个应用视口（`100vw × 100dvh`）。禁止用 HTML5 Fullscreen、`window.top` 或向父页面注入 DOM 绕过 Host 协议。
- 模态框结构固定为 Header / Body / Footer：Header 和 Footer永不参与滚动，只有 Body 可以 `overflow-y: auto`；页面和对话框外壳不得出现第二条滚动条。

## 菜单说明卡片

- 页面标题和描述只在活动 Manifest contribution 中维护，由 Host 在业务内容上方统一渲染卡片。
- 本 contribution 使用共享 `page()` 时必须设置 `showSummary: false`，不得复制页面级标题和描述。
- 管理型页面有查询条件时必须使用独立 `searchCard()`；表格、树或集合必须使用 `dataCard()`。
- 查询字段变化后由共享 `searchForm()` 自动搜索，不得为查询字段自写 `change`/`input` 监听。级联下拉必须在 `onSearch` 内同步选项，替换 option 后调用 `refreshSearchSelect()`。`kind: 'select'` 使用共享可搜索下拉，点选即选中；空值表示全部。菜单 portal 到当前文档 `body`，`--ls-z-popover` 必须高于 Host 表单模态遮罩。
- 刷新、新增、导入、导出和批量操作只能放在 `dataCard()` 的表格工具栏；单行编辑/启停操作留在行内。禁止 `ls-ui-page-toolbar` 悬空工具栏。
