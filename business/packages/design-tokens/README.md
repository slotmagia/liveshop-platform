# `@liveshop/design-tokens`

LiveShop 所有前端界面唯一的视觉与组件契约。视觉基线是既有后台 UI（Stripe 风格 indigo `#635bff` + 中性画布，Inter / JetBrains Mono）。

包内有两套组件层，共用同一份令牌：

- **console**：总后台与商户后台。密集布局，34px 控件，中性表面。两个后台渲染的是同一套组件，因此类名里不带 `admin` 或 `merch`。
- **storefront**：商城与直播间。触屏优先，44px 控件，大圆角，带"压在视频/图片上"的变体。

两套刻意分开：后台控件要密、C 端控件要能用拇指点，硬合成一套只会让其中一边妥协。品牌色和中性色阶是共用的，所以两端仍然是一个产品。

## 入口

| 入口 | 内容 | 谁用 |
| --- | --- | --- |
| `@liveshop/design-tokens/console.css` | 令牌 + 后台组件层 | 总后台、商户后台及其 contribution |
| `@liveshop/design-tokens/storefront.css` | 令牌 + C 端组件层 | 商城 Host、直播 Host |
| `@liveshop/design-tokens/tokens.css` | 只有 CSS 变量和基础 reset | 需要自带组件层的场景 |
| `@liveshop/design-tokens/tailwind-preset` | Tailwind theme：色阶、圆角、阴影、字体全部指回令牌 | 用 React + Tailwind 写的后台界面 |
| `@liveshop/design-tokens/tailwind.css` | `@tailwind` 三段指令 + 基线层与密集工具条按钮 | 同上，在 `console.css` 之后导入 |
| `@liveshop/design-tokens/*.components.css` | 只有组件层，不含令牌 | remote-esm contribution：Host 文档里已有令牌，重复导入会放大产物 |
| `@liveshop/design-tokens` | `ui` 类名契约 + 后台组件工厂 | 写后台页面的地方 |
| `@liveshop/design-tokens/storefront` | `shopUI` 类名契约 + C 端组件工厂 | 写商城/直播页面的地方 |

## 规则

1. 后台入口导入 `console.css`，C 端入口导入 `storefront.css`，不再单独引其它基础样式表。用 Tailwind 的后台再按顺序追加 `tailwind.css`——放在 `console.css` 之后，工具类才压得住 `.ls-ui-*`，而 Preflight 只匹配裸元素，压不动它。
2. 页面用组件工厂构建 DOM，不拼 `innerHTML`。工厂全部走 `textContent`，服务端返回的字符串不可能进入 HTML 解析器。React 页面用 JSX，同样不碰 `dangerouslySetInnerHTML`。
3. 颜色、圆角、阴影、字体一律取 `tokens.css` 里的变量。禁止写 `var(--ls-x, #fallback)`：令牌缺失必须直接暴露，而不是悄悄退回第二套配色。
4. Tailwind 配置只允许 `presets: [tailwindPreset]` 加一份 `content`。禁止在 `theme.extend` 里写颜色、圆角、阴影或字体：那等于绕开令牌复制一套配色，正是本包要消灭的东西。需要新色阶就加令牌，preset 会自动带出对应工具类。
5. 需要新的通用组件时演进本包并迁移调用方，不在业务模块里复制一套近似样式。
6. 菜单标题和描述由 Host 从活动 Manifest 渲染为独立说明卡片。Host 内的 contribution 调用 `page()` 时必须设置 `showSummary: false`，页面源码不得复制 Manifest 文案；完整契约见本仓库 [`docs/开发规范.md`](../../../docs/开发规范.md) 第 5 章。

## 令牌为什么是 RGB 通道

每个颜色只声明一次，写成空格分隔的 sRGB 通道（`--ls-primary-rgb: 99 91 255`），`--ls-primary` 再包一层 `rgb()` 给普通 CSS 用。preset 消费的是通道那份，于是 Tailwind 能自己塞透明度，`bg-primary/10` 编译成 `rgb(var(--ls-primary-rgb) / .1)`，不需要为半透明再硬编码一遍同一个颜色。

## 后台组件工厂

`page` `card` `grid` `form` `formModal` `searchForm` `pagination` `checkboxTree` `field` `table` `button` `badge` `notify` `statusLine` `statGrid` `definitionList` `modal` `emptyState` `code` `create`，以及 `buttonClass` / `badgeClass` / `statusClass` 三个类名解析函数（供自带转义的字符串渲染器和将来的 React 组件使用）。

主体列表页的**查询区**必须用 `searchForm`，不要用 `form` 冒充：

- 布局固定为 **6 列字段 + 1 列操作**（`折叠/展开` / `搜索` / `重置`）；操作列固定在查询卡第一行最右侧，筛选字段换行时按钮不得跟随字段流到第二行。查询条件默认展开，只有形成多行时才显示折叠按钮，折叠后只保留第一行字段。
- 普通输入、下拉等查询字段统一占 **1 个等宽列**；只有 `kind: 'date-range'` 的日期范围固定占 **2 个等宽列**，查询字段禁止使用 `wide` 或自定义跨列。
- 查询字段变化后由 `searchForm()` 自动搜索：下拉和日期立即触发，文本输入防抖 300ms。页面不得再为查询字段手写 `change`/`input` 监听；搜索按钮仍保留。
- 商户/店铺、视图/状态等从属选项必须在 `onSearch` 内同步：用 `api.set()` 清空从属字段，替换 `<option>` 后调用 `control.refreshSearchSelect()`。
- `kind: 'select'` 渲染共享可搜索下拉：打开后面板可输入过滤，点选选项即写入值并触发 `change`。空值表示全部。页面不得自建 combobox。菜单必须 portal 到当前文档 `body`，使用 `position: fixed` 与 `--ls-z-popover`，层级高于 Host 表单模态 `--ls-z-host-modal`。禁止把菜单留在模态框 Body 内或写局部 z-index。
- Host 菜单说明卡、独立查询卡和数据卡必须使用同一左右边界；Host 负责外层留白，contribution 的 `.ls-ui-page` 不得再添加水平 padding。
- 日期筛选用 `kind: 'date-range'`，文案为「开始 至 结束」，**固定占两列**；`values()` 得到 `${name}From` / `${name}To`（可用 `from` / `to` 覆盖键名）。
- 分页列表把 `pagination().element` 传给 `dataCard.footer`；每页条数、总数、页码和翻页操作全部位于数据卡底部。成功、失败、警告、复制结果只允许 `notify()`；iframe 内调用时由 Host 画在顶层文档右上角。禁止把 `statusLine().element` 放进数据卡或查询卡。加载中和条数统计不得作为消息提示。
- 层级权限集合使用 `checkboxTree()` 或表单字段 `kind: 'checkbox-tree'`；只有叶子 `value` 会提交，父节点负责整组选择并自动展示半选状态。Host 模态框协议会校验和透传树节点，业务模块不得退回自由文本权限码。扁平目录可用 `columns: 2 | 3` 在同一容器内分列，默认仍是单列层级列表。

**新建 / 编辑**必须用模态表单（大号模态框，两列表单，页脚「取消 / 保存」），禁止把保存表单嵌在列表页卡片里：

- 页头或卡片头放「新建」；点表格行（或「编辑」）打开同一套对话框并 `open(values, '编辑…')`。
- iframe contribution 的简单表单使用 Host SDK 的 `hostFormModal`；Host 在顶层文档渲染遮罩和表单，保存成功后调用 `api.close()`，失败调用 `api.setError(message)`。选择项决定后续字段时，在 `onChange` 中调用 `api.setFields(fields, values, title)` 原地重建同一个弹窗，不要串联两个弹窗。富内容编辑器使用本包的 `modal`，但打开和关闭时必须成对调用 Host SDK `hostOverlay()`，由 Host 将所属 iframe 提升到全视口层。
- native page 和 remote ESM 可直接使用本包的 `formModal`；确认类对话框使用 `modal`。所有后台模态框共享固定的 Header / Body / Footer DOM：遮罩覆盖 `100vw × 100dvh`，对话框整体不滚动，Header 和 Footer 固定，只有 Body 可以纵向滚动。
- iframe 不得使用 HTML Fullscreen API、读取父窗口 DOM、自行拼接外围遮罩，或直接创建第二套 modal/backdrop CSS。Host 校验 source、origin、protocol 和 `requestId`，同一时刻只保留一个顶层模态框，并在 iframe 卸载时强制清理。

```js
import { hostFormModal } from '@liveshop/host-sdk'
import { button, notify, page, searchCard, searchForm, table, dataCard } from '@liveshop/design-tokens'

const editor = hostFormModal({
  title: '新建账户',
  fields: [
    { name: 'username', label: '登录账号', required: true },
    { name: 'status', label: '状态', kind: 'select', options: ['ACTIVE', 'DISABLED'] },
  ],
  submitLabel: '保存',
  onSubmit: (values, api) => {
    api.setBusy(true)
    save(values).then(() => { api.close(); notify('已保存', 'success') }).catch((error) => api.setError(String(error))).finally(() => api.setBusy(false))
  },
})
const accounts = table({
  columns: ['账号', '状态'],
  onRowClick: (index) => editor.open(items[index], '编辑账户'),
})
const filters = searchForm({
  fields: [
    { name: 'username', label: '登录账号' },
    { name: 'status', label: '状态', kind: 'select', options: [{ value: '', label: '全部' }, 'ACTIVE', 'DISABLED'] },
    { name: 'updatedAt', label: '更新时间', kind: 'date-range' },
  ],
  onSearch: (values) => applyFilters(values),
})

root.replaceChildren(page({
  showSummary: false,
  children: [
    searchCard(filters.element),
    dataCard({
      title: '账户列表',
      actions: [
        button({ label: '刷新', variant: 'secondary', onClick: load }),
        button({ label: '新建账户', onClick: () => editor.open({ status: 'ACTIVE' }, '新建账户') }),
      ],
      body: accounts.element,
    }),
  ],
}))
```

`form.values()` / `searchForm.values()` 返回按字段名索引的对象，调用方不需要再对 `form.elements` 做类型断言。`table.setRows` 的单元格既接受字符串，也接受 `badge()` 这类节点。

## C 端组件工厂

`hero` `section` `productCard` `productGrid` `price` `cta` `tag` `optionList` `panel` `overlay` `livePill` `messageList` `sheet` `emptyState` `skeleton` `create`。

```js
import { cta, price, productCard, productGrid } from '@liveshop/design-tokens/storefront'

productGrid(items.map((item) => productCard({
  title: item.title,
  image: item.cover,
  priceOptions: { amount: item.priceMinor, original: item.listPriceMinor, discount: item.discountLabel },
  onSelect: () => open(item.id),
})))
```

`price` 接收**最小货币单位**的整数并自己做格式化——把 `1999` 直接渲染成价格是线上事故，所以这一步不留给调用方。`optionList` 自己持有选中态，结算页不可能出现两个支付方式同时高亮。`overlay` 只让子节点接收指针事件，播放器的手势不会被整层挡住。
