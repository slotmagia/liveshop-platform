# `@liveshop/admin-ui`

LiveShop 总后台唯一的共享 UI 契约。视觉基线参考 `D:\liveshop\frontend-admin`，并以框架无关方式提供给 Host 原生页面和各模块的 React iframe。

Admin Host、Platform Admin contribution 以及所有模块的 `frontend-admin` 必须：

1. 依赖 `@liveshop/admin-ui`；
2. 在入口导入 `@liveshop/admin-ui/admin.css`；
3. 从包根导入 `adminUI`、`adminButtonClass`、`adminStatusClass` 或 `adminBadgeClass`；
4. 领域目录只保留特有布局，不能重新定义按钮、表单、卡片、表格、状态或弹窗视觉。

新增通用组件时直接演进本包并迁移调用方，不在业务模块复制一套近似样式。
