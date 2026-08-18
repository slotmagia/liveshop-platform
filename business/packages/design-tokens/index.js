/**
 * The framework-neutral component kit for the platform and merchant consoles.
 *
 * Every factory writes text through `textContent`, so a value that came from a
 * server can never reach an HTML parser. Contributions therefore satisfy the
 * "no innerHTML for server strings" rule by construction rather than by review.
 *
 * `ui` stays exported because a domain layout occasionally needs to hang a
 * class on its own element. Anything beyond layout must be a factory here, not
 * a second stylesheet in a business module.
 */

export {
  NAV_GROUP_ICONS,
  NAV_ICON_NAMES,
  NAV_PAGE_ICONS,
  navIcon,
  resolveGroupIconName,
  resolveNavIconName,
  resolvePageIconName,
} from './nav-icons.js'

export const ui = Object.freeze({
  root: 'ls-ui-root',
  page: 'ls-ui-page',
  pageHeader: 'ls-ui-page-header',
  pageHeading: 'ls-ui-page-heading',
  eyebrow: 'ls-ui-eyebrow',
  title: 'ls-ui-title',
  description: 'ls-ui-description',
  actions: 'ls-ui-actions',
  grid: 'ls-ui-grid',
  card: 'ls-ui-card',
  cardHeader: 'ls-ui-card-header',
  cardTitle: 'ls-ui-card-title',
  cardBody: 'ls-ui-card-body',
  searchCard: 'ls-ui-search-card',
  tabs: 'ls-ui-tabs',
  tab: 'ls-ui-tab',
  dataCard: 'ls-ui-data-card',
  tableToolbar: 'ls-ui-table-toolbar',
  tableToolbarTitle: 'ls-ui-table-toolbar__title',
  tableToolbarActions: 'ls-ui-table-toolbar__actions',
  dataCardStatus: 'ls-ui-data-card__status',
  dataCardFooter: 'ls-ui-data-card__footer',
  pagination: 'ls-ui-pagination',
  paginationSummary: 'ls-ui-pagination__summary',
  paginationControls: 'ls-ui-pagination__controls',
  paginationPageSize: 'ls-ui-pagination__page-size',
  checkboxTree: 'ls-ui-checkbox-tree',
  checkboxTreeNode: 'ls-ui-checkbox-tree__node',
  checkboxTreeRow: 'ls-ui-checkbox-tree__row',
  checkboxTreeToggle: 'ls-ui-checkbox-tree__toggle',
  checkboxTreeTogglePlaceholder: 'ls-ui-checkbox-tree__toggle-placeholder',
  checkboxTreeChoice: 'ls-ui-checkbox-tree__choice',
  checkboxTreeContent: 'ls-ui-checkbox-tree__content',
  checkboxTreeLabel: 'ls-ui-checkbox-tree__label',
  checkboxTreeDescription: 'ls-ui-checkbox-tree__description',
  checkboxTreeChildren: 'ls-ui-checkbox-tree__children',
  form: 'ls-ui-form',
  searchForm: 'ls-ui-search-form',
  searchFormActions: 'ls-ui-search-form__actions',
  searchFormToggle: 'ls-ui-search-form__toggle',
  field: 'ls-ui-field',
  fieldWide: 'ls-ui-field--wide',
  fieldSpan2: 'ls-ui-field--span-2',
  dateRange: 'ls-ui-date-range',
  dateRangeSep: 'ls-ui-date-range__sep',
  input: 'ls-ui-input',
  select: 'ls-ui-select',
  searchSelect: 'ls-ui-search-select',
  searchSelectNative: 'ls-ui-search-select__native',
  searchSelectTrigger: 'ls-ui-search-select__trigger',
  searchSelectValue: 'ls-ui-search-select__value',
  searchSelectMenu: 'ls-ui-search-select__menu',
  searchSelectSearch: 'ls-ui-search-select__search',
  searchSelectList: 'ls-ui-search-select__list',
  searchSelectOption: 'ls-ui-search-select__option',
  searchSelectEmpty: 'ls-ui-search-select__empty',
  textarea: 'ls-ui-textarea',
  button: 'ls-ui-button',
  tableWrap: 'ls-ui-table-wrap',
  table: 'ls-ui-table',
  status: 'ls-ui-status',
  empty: 'ls-ui-empty',
  badge: 'ls-ui-badge',
  statGrid: 'ls-ui-stat-grid',
  stat: 'ls-ui-stat',
  definitionList: 'ls-ui-definition-list',
  code: 'ls-ui-code',
  modalBackdrop: 'ls-ui-modal-backdrop',
  modal: 'ls-ui-modal',
  modalLg: 'ls-ui-modal--lg',
  modalHeader: 'ls-ui-modal-header',
  modalTitle: 'ls-ui-modal-title',
  modalBody: 'ls-ui-modal-body',
  modalFooter: 'ls-ui-modal-footer',
  toastHost: 'ls-ui-toast-host',
  toast: 'ls-ui-toast',
  toastMessage: 'ls-ui-toast__message',
  toastDismiss: 'ls-ui-toast__dismiss',
})

/**
 * Class resolvers for the variant-bearing components. The factories below use
 * them, and so may a renderer that emits its own escaped markup or a React
 * component: the variant vocabulary must have exactly one definition.
 */
export function buttonClass(variant = 'primary', size = 'default') {
  return `${ui.button} ${ui.button}--${variant}${size === 'default' ? '' : ` ${ui.button}--${size}`}`
}

export function badgeClass(tone = 'neutral') {
  return tone === 'neutral' ? ui.badge : `${ui.badge} ${ui.badge}--${tone}`
}

export function statusClass(tone = 'neutral') {
  return tone === 'neutral' ? ui.status : `${ui.status} ${ui.status}--${tone}`
}

/** Creates one element. `text` is always written as text, never as markup. */
export function create(tag, className, text) {
  const node = document.createElement(tag)
  if (className) node.className = className
  if (text !== undefined && text !== null) node.textContent = String(text)
  return node
}

function put(parent, value) {
  if (value === undefined || value === null || value === false) return
  if (value instanceof Node) parent.append(value)
  else parent.textContent = String(value)
}

function list(value) {
  if (value === undefined || value === null) return []
  return Array.isArray(value) ? value.filter(Boolean) : [value]
}

export function button({ label, variant = 'primary', size = 'default', type = 'button', onClick, disabled = false, title } = {}) {
  const node = create('button', buttonClass(variant, size), label)
  node.type = type
  node.disabled = disabled
  if (title) node.title = title
  if (onClick) node.addEventListener('click', onClick)
  return node
}

export function badge({ label, tone = 'neutral' } = {}) {
  return create('span', badgeClass(tone), label)
}

export function emptyState(text) {
  return create('div', ui.empty, text)
}

export function code(text) {
  return create('pre', ui.code, text)
}

const TOAST_LIMIT = 3
const TOAST_DURATION = { danger: 6400, warning: 4800, success: 3200, info: 3200 }

function toastTone(tone) {
  return tone && tone !== 'neutral' ? tone : 'info'
}

function ensureToastHost() {
  let host = document.getElementById('ls-ui-toast-host')
  if (host) return host
  host = create('div', ui.toastHost)
  host.id = 'ls-ui-toast-host'
  host.setAttribute('aria-live', 'polite')
  document.body.append(host)
  return host
}

/**
 * The only page-level message surface. Inside an iframe contribution this
 * posts to the Host, which paints the toast on the top-level document at the
 * viewport top-right. It must never occupy query/data card layout.
 */
export function notify(text, tone = 'info', options = {}) {
  const message = String(text ?? '').trim()
  if (!message) return
  if (typeof window !== 'undefined' && window.parent && window.parent !== window) {
    window.parent.postMessage({
      type: 'LIVESHOP_HOST_NOTIFY',
      protocol: 2,
      text: message,
      tone: toastTone(tone),
    }, '*')
    return
  }
  const host = ensureToastHost()
  while (host.childElementCount >= TOAST_LIMIT) host.firstElementChild?.remove()
  const kind = toastTone(tone)
  const item = create('div', `${ui.toast} ${ui.toast}--${kind}`)
  item.setAttribute('role', kind === 'danger' ? 'alert' : 'status')
  item.append(create('p', ui.toastMessage, message))
  const dismiss = create('button', ui.toastDismiss, '关闭')
  dismiss.type = 'button'
  dismiss.addEventListener('click', () => item.remove())
  item.append(dismiss)
  host.append(item)
  const duration = options.duration ?? TOAST_DURATION[kind]
  if (duration > 0) {
    window.setTimeout(() => { if (item.isConnected) item.remove() }, duration)
  }
}

function shouldNotifyStatus(text, tone) {
  if (tone && tone !== 'neutral') return true
  return /^已/.test(text)
}

/**
 * Compatibility wrapper. New pages call `notify()`. `set` never writes into
 * the page; action outcomes become toasts, progress/count text is discarded.
 */
export function statusLine() {
  const element = create('div', ui.status)
  element.setAttribute('role', 'status')
  return {
    element,
    set(text, tone = 'neutral') {
      const value = text === undefined || text === null ? '' : String(text).trim()
      element.textContent = ''
      element.className = ui.status
      if (!value || !shouldNotifyStatus(value, tone)) return
      notify(value, tone === 'neutral' ? 'success' : tone)
    },
    clear() {
      element.textContent = ''
      element.className = ui.status
    },
  }
}

export function page({ eyebrow, title, description, actions, children, showSummary = true } = {}) {
  const root = create('main', ui.page)
  const header = create('div', ui.pageHeader)
  const heading = create('div', ui.pageHeading)
  if (eyebrow) heading.append(create('p', ui.eyebrow, eyebrow))
  if (title) heading.append(create('h1', ui.title, title))
  if (description) heading.append(create('p', ui.description, description))
  header.append(heading)
  const bar = list(actions)
  if (bar.length) {
    const group = create('div', ui.actions)
    group.append(...bar)
    header.append(group)
  }
  if (showSummary) root.append(header)
  root.append(...list(children))
  return root
}

export function card({ title, headerExtra, body, padded = true } = {}) {
  const root = create('section', ui.card)
  if (title || headerExtra) {
    const header = create('div', ui.cardHeader)
    header.append(create('h2', ui.cardTitle, title ?? ''))
    for (const extra of list(headerExtra)) header.append(extra)
    root.append(header)
  }
  const content = list(body)
  if (!content.length) return root
  if (!padded) {
    root.append(...content)
    return root
  }
  const inner = create('div', ui.cardBody)
  inner.append(...content)
  root.append(inner)
  return root
}

/** A query form is always a standalone card between the Host summary and data. */
export function searchCard(body) {
  const root = card({ body, padded: false })
  root.classList.add(ui.searchCard)
  root.setAttribute('data-page-search', '')
  return root
}

/**
 * Stripe-style chip tabs from the legacy console. They sit between the search
 * card and the data card when one page has several collections.
 */
export function tabs({ items = [], value = '', ariaLabel, onChange } = {}) {
  const root = create('div', ui.tabs)
  root.setAttribute('role', 'tablist')
  if (ariaLabel) root.setAttribute('aria-label', ariaLabel)
  const buttons = new Map()
  let current = String(value ?? '')

  function paint(next) {
    current = String(next ?? '')
    for (const [key, node] of buttons) {
      node.setAttribute('aria-selected', String(key === current))
    }
  }

  for (const item of items) {
    const key = String(item.value)
    const node = create('button', ui.tab, item.label)
    node.type = 'button'
    node.setAttribute('role', 'tab')
    node.addEventListener('click', () => {
      if (key === current) return
      paint(key)
      onChange?.(key)
    })
    buttons.set(key, node)
    root.append(node)
  }
  paint(current)

  return {
    element: root,
    set(next) { paint(next) },
    value() { return current },
  }
}

/**
 * The only collection/list container. Page-level operations live in its
 * toolbar; row-level operations remain in the relevant table row.
 */
export function dataCard({ title, actions, status, body, footer } = {}) {
  const root = create('section', `${ui.card} ${ui.dataCard}`)
  root.setAttribute('data-page-data', '')
  const bar = list(actions)
  if (title || bar.length) {
    const toolbar = create('div', ui.tableToolbar)
    toolbar.append(create('h2', ui.tableToolbarTitle, title ?? ''))
    if (bar.length) {
      const group = create('div', ui.tableToolbarActions)
      group.append(...bar)
      toolbar.append(group)
    }
    root.append(toolbar)
  }
  const statusNodes = list(status)
  if (statusNodes.length) {
    const region = create('div', ui.dataCardStatus)
    region.append(...statusNodes)
    root.append(region)
  }
  root.append(...list(body))
  const footerNodes = list(footer)
  if (footerNodes.length) {
    const region = create('div', ui.dataCardFooter)
    region.append(...footerNodes)
    root.append(region)
  }
  return root
}

/**
 * The single list-pagination control. Pagination state belongs to the data
 * card footer, never to the search form or the table toolbar.
 */
export function pagination({
  page = 1,
  pageSize = 20,
  total = 0,
  itemCount = 0,
  pageSizeOptions = [20, 50, 100],
  previousLabel = '上一页',
  nextLabel = '下一页',
  pageSizeLabel = '每页',
  summary,
  actions,
  onPageChange,
  onPageSizeChange,
} = {}) {
  const element = create('nav', ui.pagination)
  element.setAttribute('aria-label', '分页')
  const summaryNode = create('span', ui.paginationSummary)
  const controls = create('div', ui.paginationControls)
  const sizeField = create('label', ui.paginationPageSize)
  const sizeText = create('span', undefined, pageSizeLabel)
  const sizeSelect = control({
    kind: 'select',
    options: list(pageSizeOptions).map(value => ({ value, label: String(value) })),
    value: pageSize,
  })
  sizeSelect.setAttribute('aria-label', pageSizeLabel)
  const previous = button({ label: previousLabel, variant: 'secondary', size: 'sm' })
  const next = button({ label: nextLabel, variant: 'secondary', size: 'sm' })
  const extraActions = list(actions)
  sizeField.hidden = !list(pageSizeOptions).length
  sizeField.append(sizeText, controlSurface(sizeSelect))
  controls.append(sizeField, previous, next, ...extraActions)
  element.append(summaryNode, controls)

  let currentPage = page
  let currentPageSize = pageSize
  let currentTotal = total
  let currentItemCount = itemCount
  let busy = false

  function render() {
    const knownTotal = Number.isFinite(currentTotal) && currentTotal >= 0
    const pages = knownTotal ? Math.max(1, Math.ceil(currentTotal / currentPageSize)) : undefined
    currentPage = Math.max(1, pages ? Math.min(currentPage, pages) : currentPage)
    const model = { page: currentPage, pageSize: currentPageSize, total: knownTotal ? currentTotal : null, pages }
    summaryNode.textContent = summary
      ? String(summary(model))
      : knownTotal
        ? `共 ${currentTotal} 条 · 第 ${currentPage} / ${pages} 页`
        : `第 ${currentPage} 页`
    sizeSelect.value = String(currentPageSize)
    previous.disabled = busy || currentPage <= 1
    next.disabled = busy || (knownTotal
      ? currentPage * currentPageSize >= currentTotal
      : currentItemCount < currentPageSize)
  }

  previous.addEventListener('click', () => {
    if (currentPage <= 1 || busy) return
    onPageChange?.(currentPage - 1)
  })
  next.addEventListener('click', () => {
    if (next.disabled || busy) return
    onPageChange?.(currentPage + 1)
  })
  sizeSelect.addEventListener('change', () => {
    const value = Number(sizeSelect.value)
    if (!Number.isSafeInteger(value) || value <= 0 || value === currentPageSize) return
    onPageSizeChange?.(value)
  })

  const api = {
    element,
    summary: summaryNode,
    previous,
    next,
    pageSizeSelect: sizeSelect,
    set(nextState = {}) {
      if (nextState.page !== undefined) currentPage = nextState.page
      if (nextState.pageSize !== undefined) currentPageSize = nextState.pageSize
      if (Object.prototype.hasOwnProperty.call(nextState, 'total')) currentTotal = nextState.total
      if (nextState.itemCount !== undefined) currentItemCount = nextState.itemCount
      render()
    },
    setBusy(nextBusy) {
      busy = nextBusy
      sizeSelect.disabled = nextBusy
      render()
    },
  }
  render()
  return api
}

export function grid(children) {
  const root = create('div', ui.grid)
  root.append(...list(children))
  return root
}

export function statGrid(items) {
  const root = create('div', ui.statGrid)
  for (const item of list(items)) {
    const cell = create('div', ui.stat)
    cell.append(create('span', undefined, item.label))
    const value = create('strong')
    value.title = String(item.value ?? '')
    put(value, item.value ?? '')
    cell.append(value)
    root.append(cell)
  }
  return root
}

export function definitionList(items) {
  const root = create('dl', ui.definitionList)
  for (const item of list(items)) {
    root.append(create('dt', undefined, item.label))
    const value = create('dd')
    put(value, item.value ?? '')
    root.append(value)
  }
  return root
}

function control(descriptor) {
  const { kind = 'input', name, value, placeholder, required, disabled, options, rows, type = 'text', mono } = descriptor
  let node
  let base
  if (kind === 'select') {
    base = ui.select
    node = create('select', base)
    for (const option of list(options)) {
      const item = create('option', undefined, option.label ?? option.value ?? option)
      item.value = String(option.value ?? option.label ?? option)
      node.append(item)
    }
    if (value !== undefined) node.value = String(value)
  } else if (kind === 'textarea') {
    base = ui.textarea
    node = create('textarea', base)
    if (rows) node.rows = rows
    if (value !== undefined) node.value = String(value)
  } else {
    base = ui.input
    node = create('input', base)
    node.type = type
    if (value !== undefined) node.value = String(value)
  }
  if (mono) node.classList.add(`${base}--mono`)
  if (name) node.name = name
  if (placeholder) node.placeholder = placeholder
  if (required) node.required = true
  if (disabled) node.disabled = true
  for (const attribute of ['min', 'max', 'step', 'minLength', 'maxLength', 'autocomplete', 'inputMode']) {
    if (descriptor[attribute] !== undefined) node.setAttribute(attribute.toLowerCase(), String(descriptor[attribute]))
  }
  if (kind === 'select') attachSearchSelect(node)
  return node
}

function controlSurface(node) {
  return node?.uiHost || node
}

function attachSearchSelect(select) {
  if (!(select instanceof HTMLSelectElement) || select.multiple || select.uiHost) return select
  const root = create('div', ui.searchSelect)
  const trigger = create('button', ui.searchSelectTrigger)
  const valueNode = create('span', ui.searchSelectValue)
  const menu = create('div', `${ui.searchSelectMenu} ${ui.selectMenu || 'ls-ui-select-menu'}`)
  const search = create('input', ui.searchSelectSearch)
  const list = create('div', ui.searchSelectList)
  const empty = create('div', ui.searchSelectEmpty, '无匹配项')
  const listboxId = `ls-search-select-${Math.random().toString(36).slice(2, 10)}`
  let open = false
  let activeIndex = -1
  select.uiHost = root
  select.classList.add(ui.searchSelectNative)
  select.setAttribute('tabindex', '-1')
  select.setAttribute('aria-hidden', 'true')
  trigger.type = 'button'
  trigger.setAttribute('role', 'combobox')
  trigger.setAttribute('aria-haspopup', 'listbox')
  trigger.setAttribute('aria-controls', listboxId)
  trigger.setAttribute('aria-expanded', 'false')
  search.type = 'search'
  search.placeholder = '搜索'
  search.setAttribute('autocomplete', 'off')
  search.setAttribute('aria-label', '搜索选项')
  list.id = listboxId
  list.setAttribute('role', 'listbox')
  menu.hidden = true
  menu.append(search, list, empty)
  root.append(select, trigger)
  trigger.append(valueNode)

  select.refreshSearchSelect = refreshTrigger
  select.addEventListener('change', refreshTrigger)

  function optionRecords() {
    return [...select.options].map((option, index) => ({
      index,
      value: option.value,
      label: option.textContent || option.label || option.value,
      disabled: option.disabled,
    }))
  }
  function selectedLabel() {
    const selected = select.selectedOptions[0]
    if (selected) return selected.textContent || selected.label || selected.value
    return select.getAttribute('placeholder') || '请选择'
  }
  function refreshTrigger() {
    const selected = select.selectedOptions[0]
    const blank = !selected || selected.value === ''
    valueNode.textContent = selectedLabel()
    valueNode.dataset.placeholder = String(blank && !selected?.textContent)
    trigger.disabled = select.disabled
    root.dataset.disabled = String(select.disabled)
    root.dataset.open = String(open)
  }
  function filteredRecords() {
    const query = search.value.trim().toLowerCase()
    return optionRecords().filter(item => !query || item.label.toLowerCase().includes(query) || item.value.toLowerCase().includes(query))
  }
  function renderMenu() {
    const records = filteredRecords()
    list.replaceChildren()
    empty.hidden = records.length > 0
    records.forEach((item, index) => {
      const option = create('button', ui.searchSelectOption, item.label)
      option.type = 'button'
      option.id = `${listboxId}-option-${item.index}`
      option.setAttribute('role', 'option')
      option.dataset.index = String(item.index)
      option.setAttribute('aria-selected', String(select.options[item.index]?.selected === true))
      if (item.disabled) option.disabled = true
      if (index === activeIndex) option.dataset.active = 'true'
      option.addEventListener('pointerenter', () => { activeIndex = index; paintActive() })
      option.addEventListener('click', event => {
        event.preventDefault()
        event.stopPropagation()
        choose(item.index)
      })
      list.append(option)
    })
    if (activeIndex >= records.length) activeIndex = records.length - 1
    paintActive()
  }
  function paintActive() {
    const options = [...list.children]
    options.forEach((node, index) => {
      node.dataset.active = String(index === activeIndex)
      if (index === activeIndex) trigger.setAttribute('aria-activedescendant', node.id || '')
    })
    options[activeIndex]?.scrollIntoView({ block: 'nearest' })
  }
  function placeMenu() {
    const rect = trigger.getBoundingClientRect()
    const gap = 4
    const maxHeight = 280
    const roomBelow = window.innerHeight - rect.bottom - gap
    const roomAbove = rect.top - gap
    const openAbove = roomBelow < 160 && roomAbove > roomBelow
    const height = Math.max(120, Math.min(maxHeight, openAbove ? roomAbove : roomBelow))
    menu.style.left = `${rect.left}px`
    menu.style.width = `${rect.width}px`
    menu.style.maxHeight = `${height}px`
    menu.style.top = openAbove ? `${Math.max(gap, rect.top - height - gap)}px` : `${rect.bottom + gap}px`
  }
  function closeMenu() {
    if (!open) return
    open = false
    menu.hidden = true
    menu.remove()
    trigger.setAttribute('aria-expanded', 'false')
    trigger.removeAttribute('aria-activedescendant')
    search.value = ''
    root.dataset.open = 'false'
  }
  function openMenu() {
    if (select.disabled || open) return
    open = true
    document.body.append(menu)
    menu.hidden = false
    trigger.setAttribute('aria-expanded', 'true')
    root.dataset.open = 'true'
    const current = optionRecords().findIndex(item => item.value === select.value)
    activeIndex = Math.max(0, current)
    renderMenu()
    placeMenu()
    search.focus()
  }
  function choose(optionIndex) {
    const option = select.options[optionIndex]
    if (!option || option.disabled) return
    const previous = select.value
    select.value = option.value
    closeMenu()
    trigger.focus()
    if (previous !== select.value) select.dispatchEvent(new Event('change', { bubbles: true }))
  }
  function moveActive(step) {
    const records = filteredRecords()
    if (!records.length) return
    activeIndex = (activeIndex + step + records.length) % records.length
    paintActive()
  }
  trigger.addEventListener('click', event => {
    event.preventDefault()
    open ? closeMenu() : openMenu()
  })
  trigger.addEventListener('keydown', event => {
    if (select.disabled) return
    if (!open && (event.key === 'ArrowDown' || event.key === 'ArrowUp' || event.key === 'Enter' || event.key === ' ')) {
      event.preventDefault()
      openMenu()
    }
  })
  search.addEventListener('input', event => {
    event.stopPropagation()
    activeIndex = 0
    renderMenu()
  })
  search.addEventListener('change', event => event.stopPropagation())
  search.addEventListener('keydown', event => {
    if (event.key === 'ArrowDown') { event.preventDefault(); moveActive(1) }
    else if (event.key === 'ArrowUp') { event.preventDefault(); moveActive(-1) }
    else if (event.key === 'Enter') {
      event.preventDefault()
      const records = filteredRecords()
      if (records[activeIndex]) choose(records[activeIndex].index)
    } else if (event.key === 'Escape') {
      event.preventDefault()
      closeMenu()
      trigger.focus()
    }
  })
  function onDocumentPointer(event) {
    if (!open) return
    const target = event.target
    if (root.contains(target) || menu.contains(target)) return
    closeMenu()
  }
  function onViewport() {
    if (open) placeMenu()
  }
  document.addEventListener('pointerdown', onDocumentPointer)
  document.addEventListener('ls-ui-close-popovers', closeMenu)
  window.addEventListener('resize', onViewport)
  window.addEventListener('scroll', onViewport, true)
  const observer = new MutationObserver(() => {
    if (!select.isConnected) {
      destroy()
      return
    }
    refreshTrigger()
    if (open) renderMenu()
  })
  observer.observe(select, { childList: true, subtree: true, attributes: true, attributeFilter: ['disabled'] })
  function destroy() {
    closeMenu()
    observer.disconnect()
    document.removeEventListener('pointerdown', onDocumentPointer)
    document.removeEventListener('ls-ui-close-popovers', closeMenu)
    window.removeEventListener('resize', onViewport)
    window.removeEventListener('scroll', onViewport, true)
  }
  select.addEventListener('ls-ui-destroy', destroy)
  refreshTrigger()
  return select
}

function checkboxTreeValues(value) {
  return new Set(String(value ?? '').split(/[\s,]+/).map(item => item.trim()).filter(Boolean))
}

/**
 * Hierarchical multi-select for permission/resource catalogs. Only leaf values
 * are serialized; group checkboxes select descendants and expose half-selection.
 */
export function checkboxTree({ nodes, value = '', name, disabled = false, empty = '暂无可选项', columns = 1 } = {}) {
  const columnCount = columns === 2 || columns === 3 ? columns : 1
  const element = create('div', ui.checkboxTree)
  element.setAttribute('role', 'tree')
  if (name) element.dataset.name = name
  if (columnCount > 1) element.dataset.columns = String(columnCount)
  const controlNode = create('input')
  controlNode.type = 'hidden'
  if (name) controlNode.name = name
  let selected = checkboxTreeValues(value)
  let currentDisabled = Boolean(disabled)
  const records = []
  const leafValues = new Set()

  function buildNode(item, depth) {
    const children = list(item.children)
    const node = create('div', ui.checkboxTreeNode)
    node.setAttribute('role', 'treeitem')
    node.dataset.depth = String(depth)
    const row = create('div', ui.checkboxTreeRow)
    const compactLeaf = columnCount > 1 && !children.length
    const toggle = children.length
      ? button({ label: '', variant: 'ghost', size: 'sm' })
      : compactLeaf
        ? null
        : create('span', ui.checkboxTreeTogglePlaceholder)
    if (children.length) {
      toggle.classList.add(ui.checkboxTreeToggle)
      toggle.textContent = '▾'
      toggle.setAttribute('aria-label', `折叠 ${item.label}`)
      toggle.setAttribute('aria-expanded', 'true')
    }
    const choice = create('label', ui.checkboxTreeChoice)
    const input = create('input')
    input.type = 'checkbox'
    input.disabled = currentDisabled
    const content = create('span', ui.checkboxTreeContent)
    content.append(create('span', ui.checkboxTreeLabel, item.label ?? item.value ?? '未命名'))
    if (item.description) content.append(create('span', ui.checkboxTreeDescription, item.description))
    choice.append(input, content)
    if (toggle) row.append(toggle, choice)
    else row.append(choice)
    node.append(row)

    const descendants = []
    if (children.length) {
      const group = create('div', ui.checkboxTreeChildren)
      group.setAttribute('role', 'group')
      for (const child of children) {
        const built = buildNode(child, depth + 1)
        descendants.push(...built.values)
        group.append(built.element)
      }
      node.append(group)
      toggle.addEventListener('click', () => {
        const expanded = group.hidden
        group.hidden = !expanded
        toggle.textContent = expanded ? '▾' : '▸'
        toggle.setAttribute('aria-expanded', String(expanded))
        toggle.setAttribute('aria-label', `${expanded ? '折叠' : '展开'} ${item.label}`)
        node.setAttribute('aria-expanded', String(expanded))
      })
      node.setAttribute('aria-expanded', 'true')
    } else if (item.value !== undefined && item.value !== null && String(item.value).trim()) {
      const leafValue = String(item.value).trim()
      descendants.push(leafValue)
      leafValues.add(leafValue)
    }
    records.push({ input, values: descendants })
    input.addEventListener('change', () => {
      for (const leafValue of descendants) {
        if (input.checked) selected.add(leafValue)
        else selected.delete(leafValue)
      }
      update()
    })
    return { element: node, values: descendants }
  }

  function update() {
    for (const record of records) {
      const checked = record.values.filter(item => selected.has(item)).length
      record.input.checked = Boolean(record.values.length) && checked === record.values.length
      record.input.indeterminate = checked > 0 && checked < record.values.length
      record.input.disabled = currentDisabled || !record.values.length
    }
    controlNode.value = api.value
    element.dispatchEvent(new CustomEvent('valuechange', { detail: { value: api.value } }))
  }

  for (const node of list(nodes)) element.append(buildNode(node, 0).element)
  if (!element.childElementCount) element.append(create('div', ui.empty, empty))
  element.append(controlNode)

  const api = {
    element,
    control: controlNode,
    get value() { return [...selected].filter(item => leafValues.has(item)).sort().join('\n') },
    set value(next) { selected = checkboxTreeValues(next); update() },
    get disabled() { return currentDisabled },
    set disabled(next) { currentDisabled = Boolean(next); update() },
    selected() { return [...selected].filter(item => leafValues.has(item)).sort() },
    set(next) { api.value = next },
    reset() { api.value = value },
  }
  update()
  return api
}

function fieldClassName(descriptor) {
  const classes = [ui.field]
  if (descriptor.wide) classes.push(ui.fieldWide)
  else if (descriptor.kind === 'date-range') classes.push(ui.fieldSpan2)
  return classes.join(' ')
}

export function field(descriptor) {
  const wrapper = create(descriptor.kind === 'checkbox-tree' ? 'div' : 'label', fieldClassName(descriptor))
  if (descriptor.label) wrapper.append(create('span', undefined, descriptor.label))
  if (descriptor.kind === 'date-range') {
    const range = create('div', ui.dateRange)
    const from = control({ ...descriptor, kind: 'input', type: 'date', name: descriptor.from || `${descriptor.name}From`, value: descriptor.fromValue })
    const to = control({ ...descriptor, kind: 'input', type: 'date', name: descriptor.to || `${descriptor.name}To`, value: descriptor.toValue })
    range.append(from, create('span', ui.dateRangeSep, descriptor.separator || '至'), to)
    wrapper.append(range)
    return { element: wrapper, control: from, endControl: to, from, to }
  }
  if (descriptor.kind === 'checkbox-tree') {
    const tree = checkboxTree({
      nodes: descriptor.tree,
      value: descriptor.value,
      name: descriptor.name,
      disabled: descriptor.disabled,
      empty: descriptor.empty,
      columns: descriptor.columns,
    })
    wrapper.append(tree.element)
    return {
      element: wrapper,
      control: tree.control,
      tree,
      setValue(value) { tree.value = value },
      resetValue() { tree.reset() },
    }
  }
  const node = control(descriptor)
  wrapper.append(controlSurface(node))
  return { element: wrapper, control: node }
}

/**
 * A form built from field descriptors. Callers read `values()` instead of
 * casting `form.elements`, so a renamed field fails where it is declared.
 */
export function form({ fields, submitLabel, onSubmit, extraActions } = {}) {
  const element = create('form', ui.form)
  const controls = new Map()
  for (const descriptor of list(fields)) {
    const built = field(descriptor)
    controls.set(descriptor.name, built)
    element.append(built.element)
  }
  const submit = submitLabel ? button({ label: submitLabel, type: 'submit' }) : undefined
  const actions = list(extraActions)
  if (submit || actions.length) {
    const bar = create('div', `${ui.actions} ${ui.fieldWide}`)
    if (submit) bar.append(submit)
    bar.append(...actions)
    element.append(bar)
  }
  const api = {
    element,
    submit,
    control(name) { return controls.get(name)?.control },
    values() {
      const result = {}
      for (const [name, built] of controls) result[name] = built.control.value
      return result
    },
    set(values) {
      for (const [name, value] of Object.entries(values)) {
        const built = controls.get(name)
        if (!built) continue
        const next = value === undefined || value === null ? '' : String(value)
        if (built.setValue) built.setValue(next)
        else {
          built.control.value = next
          built.control.refreshSearchSelect?.()
        }
      }
    },
    reset() {
      element.reset()
      for (const built of controls.values()) {
        built.resetValue?.()
        built.control.refreshSearchSelect?.()
      }
    },
    setBusy(busy) { if (submit) submit.disabled = busy },
  }
  if (onSubmit) {
    element.addEventListener('submit', (event) => {
      event.preventDefault()
      onSubmit(api.values(), api)
    })
  }
  return api
}

/**
 * Subject-page search bar: six field columns plus one action column that owns
 * 搜索 / 重置. Date ranges use `kind: 'date-range'` and always span two columns.
 * Field changes auto-search: select/date immediately, text after 300ms.
 */
export function searchForm({
  fields,
  searchLabel = '搜索',
  resetLabel = '重置',
  expandLabel = '展开',
  collapseLabel = '折叠',
  onSearch,
  onReset,
} = {}) {
  const element = create('form', ui.searchForm)
  const controls = new Map()
  const filterFields = []
  let resizeObserver
  let wasConnected = false
  let searchTimer = 0
  for (const descriptor of list(fields)) {
    // Search filters are one equal grid column by contract. A date range is
    // the only compound filter and therefore the only two-column field.
    const built = field({ ...descriptor, wide: false })
    if (descriptor.kind === 'date-range') {
      const fromName = descriptor.from || `${descriptor.name}From`
      const toName = descriptor.to || `${descriptor.name}To`
      controls.set(fromName, built.from)
      controls.set(toName, built.to)
    } else {
      controls.set(descriptor.name, built.control)
    }
    filterFields.push({ element: built.element, span: descriptor.kind === 'date-range' ? 2 : 1 })
    element.append(built.element)
  }
  let collapsed = false
  let hasOverflow = false
  const toggleButton = button({
    label: collapseLabel,
    type: 'button',
    variant: 'secondary',
    onClick: () => {
      if (!hasOverflow) return
      collapsed = !collapsed
      renderCollapseState()
    },
  })
  toggleButton.classList.add(ui.searchFormToggle)
  toggleButton.hidden = true
  const search = button({ label: searchLabel, type: 'submit' })
  const resetButton = button({
    label: resetLabel,
    type: 'button',
    variant: 'secondary',
    onClick: () => {
      window.clearTimeout(searchTimer)
      api.clear()
      if (onReset) onReset(api)
      else if (onSearch) onSearch(api.values(), api)
    },
  })
  const actions = create('div', ui.searchFormActions)
  actions.append(toggleButton, search, resetButton)
  element.append(actions)

  function renderCollapseState() {
    if (!hasOverflow) collapsed = false
    element.dataset.collapsed = String(collapsed)
    toggleButton.hidden = !hasOverflow
    toggleButton.textContent = collapsed ? expandLabel : collapseLabel
    toggleButton.setAttribute('aria-expanded', String(!collapsed))
  }

  function refreshLayout() {
    if (element.isConnected) wasConnected = true
    else if (wasConnected) {
      resizeObserver?.disconnect()
      window.removeEventListener('resize', refreshLayout)
      return
    }
    const declaredColumns = Number.parseInt(
      getComputedStyle(element).getPropertyValue('--ls-search-field-columns'),
      10,
    )
    const columns = Number.isFinite(declaredColumns) && declaredColumns > 0 ? declaredColumns : 6
    let occupied = 0
    let overflowStarted = false
    for (const item of filterFields) {
      const span = Math.min(item.span, columns)
      if (occupied + span > columns) overflowStarted = true
      if (overflowStarted) item.element.dataset.searchOverflow = 'true'
      else delete item.element.dataset.searchOverflow
      if (!overflowStarted) occupied += span
    }
    hasOverflow = overflowStarted
    renderCollapseState()
  }

  refreshLayout()
  if (typeof ResizeObserver === 'function') {
    resizeObserver = new ResizeObserver(refreshLayout)
    resizeObserver.observe(element)
  } else {
    window.addEventListener('resize', refreshLayout)
  }
  const layoutFrame = requestAnimationFrame(refreshLayout)

  const api = {
    element,
    toggleButton,
    search,
    resetButton,
    control(name) { return controls.get(name) },
    values() {
      const result = {}
      for (const [name, node] of controls) result[name] = node.value
      return result
    },
    set(values) {
      for (const [name, value] of Object.entries(values)) {
        const node = controls.get(name)
        if (!node) continue
        node.value = value === undefined || value === null ? '' : String(value)
        node.refreshSearchSelect?.()
      }
    },
    clear() {
      for (const node of controls.values()) {
        node.value = ''
        node.refreshSearchSelect?.()
      }
    },
    reset() { api.clear() },
    isExpanded() { return !collapsed },
    setExpanded(expanded) {
      collapsed = hasOverflow && !expanded
      renderCollapseState()
    },
    destroy() {
      window.clearTimeout(searchTimer)
      cancelAnimationFrame(layoutFrame)
      resizeObserver?.disconnect()
      window.removeEventListener('resize', refreshLayout)
      for (const node of controls.values()) node.dispatchEvent(new Event('ls-ui-destroy'))
    },
    setBusy(busy) {
      toggleButton.disabled = busy
      search.disabled = busy
      resetButton.disabled = busy
    },
  }
  function requestSearch() {
    if (onSearch) onSearch(api.values(), api)
  }
  function scheduleSearch(immediate) {
    if (!onSearch) return
    window.clearTimeout(searchTimer)
    if (immediate) {
      requestSearch()
      return
    }
    searchTimer = window.setTimeout(requestSearch, 300)
  }
  function isSearchControl(target) {
    return target instanceof HTMLInputElement || target instanceof HTMLSelectElement || target instanceof HTMLTextAreaElement
  }
  function isImmediateControl(target) {
    if (target instanceof HTMLSelectElement) return true
    return target instanceof HTMLInputElement && (target.type === 'date' || target.type === 'datetime-local' || target.type === 'time' || target.type === 'checkbox' || target.type === 'radio')
  }
  if (onSearch) {
    element.addEventListener('submit', (event) => {
      event.preventDefault()
      window.clearTimeout(searchTimer)
      requestSearch()
    })
    element.addEventListener('input', (event) => {
      const target = event.target
      if (!isSearchControl(target) || target.type === 'hidden' || isImmediateControl(target)) return
      scheduleSearch(false)
    })
    element.addEventListener('change', (event) => {
      const target = event.target
      if (!isSearchControl(target) || target.type === 'hidden') return
      scheduleSearch(true)
    })
  }
  return api
}

/**
 * A table that owns its own empty state. Cells accept a node so a status badge
 * renders inline without a caller reaching for markup.
 */
export function table({ columns, rows, onRowClick, empty = '暂无数据' } = {}) {
  const element = create('div', ui.tableWrap)
  const node = create('table', ui.table)
  const head = node.createTHead().insertRow()
  for (const column of list(columns)) head.append(create('th', undefined, column))
  const body = node.createTBody()
  element.append(node)
  const placeholder = emptyState(empty)
  const api = {
    element,
    setRows(next) {
      body.replaceChildren()
      const items = list(next)
      placeholder.remove()
      if (!items.length) {
        element.append(placeholder)
        return
      }
      for (const [index, row] of items.entries()) {
        const line = body.insertRow()
        for (const value of row) put(line.insertCell(), value ?? '')
        if (onRowClick) {
          line.dataset.clickable = 'true'
          line.addEventListener('click', () => onRowClick(index))
        }
      }
    },
  }
  api.setRows(rows)
  return api
}

let openModalCount = 0

function focusableElements(root) {
  return [...root.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])')]
    .filter((node) => !node.disabled && !node.hidden)
}

function isPopoverSurface(node) {
  return node instanceof Element && node.classList.contains(ui.searchSelectMenu)
}

function closeOpenPopovers() {
  document.dispatchEvent(new CustomEvent('ls-ui-close-popovers'))
}

function modalFocusables(dialog) {
  const menus = [...document.body.children].filter(isPopoverSurface)
  return [...focusableElements(dialog), ...menus.flatMap((menu) => focusableElements(menu))]
}

/**
 * A modal owned by the caller: `open` attaches it, `close` detaches it.
 * Header, scrollable body and footer are deliberately separate direct children;
 * console.css fixes the outer rows and gives overflow only to the body.
 */
export function modal({ title, body, actions, size = 'md', onClose } = {}) {
  const backdrop = create('div', ui.modalBackdrop)
  const element = create('div', size === 'lg' ? `${ui.modal} ${ui.modalLg}` : ui.modal)
  element.setAttribute('role', 'dialog')
  element.setAttribute('aria-modal', 'true')
  element.tabIndex = -1
  const header = create('div', ui.modalHeader)
  const heading = create('h2', ui.modalTitle, title ?? '')
  heading.id = `ls-modal-title-${Math.random().toString(36).slice(2, 9)}`
  element.setAttribute('aria-labelledby', heading.id)
  header.append(heading)
  const content = create('div', ui.modalBody)
  content.append(...list(body))
  element.append(header, content)
  const bar = list(actions)
  if (bar.length) {
    const footer = create('div', ui.modalFooter)
    footer.append(...bar)
    element.append(footer)
  }
  backdrop.append(element)
  let previousFocus
  let inerted = []
  const close = () => {
    if (!backdrop.isConnected) return
    backdrop.remove()
    document.removeEventListener('keydown', keydown)
    for (const entry of inerted) {
      entry.node.inert = entry.inert
      if (entry.ariaHidden === null) entry.node.removeAttribute('aria-hidden')
      else entry.node.setAttribute('aria-hidden', entry.ariaHidden)
    }
    inerted = []
    openModalCount = Math.max(0, openModalCount - 1)
    if (openModalCount === 0) document.documentElement.classList.remove('ls-ui-modal-open')
    if (previousFocus?.isConnected) previousFocus.focus()
    if (onClose) onClose()
  }
  const keydown = (event) => {
    if (event.key === 'Escape') {
      if (event.defaultPrevented) return
      event.preventDefault()
      close()
      return
    }
    if (event.key !== 'Tab') return
    const items = modalFocusables(element)
    if (!items.length) {
      event.preventDefault()
      element.focus()
      return
    }
    const first = items[0]
    const last = items[items.length - 1]
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault()
      first.focus()
    }
  }
  backdrop.addEventListener('click', (event) => { if (event.target === backdrop) close() })
  return {
    element: backdrop,
    body: content,
    setTitle(next) { heading.textContent = next ?? '' },
    open(host = document.body) {
      if (!backdrop.isConnected) {
        previousFocus = document.activeElement
        closeOpenPopovers()
        host.append(backdrop)
        inerted = [...document.body.children]
          .filter((node) => node !== backdrop && !isPopoverSurface(node))
          .map((node) => ({ node, inert: node.inert, ariaHidden: node.getAttribute('aria-hidden') }))
        for (const entry of inerted) {
          entry.node.inert = true
          entry.node.setAttribute('aria-hidden', 'true')
        }
        openModalCount += 1
        document.documentElement.classList.add('ls-ui-modal-open')
        document.addEventListener('keydown', keydown)
      }
      focusableElements(element)[0]?.focus() || element.focus()
    },
    close,
  }
}

/**
 * Create / edit dialog: two-column `form` in a large modal, footer 取消 / 保存.
 * List pages must not keep an inline save form; open this from 新建 or a row click.
 */
export function formModal({
  title,
  fields,
  submitLabel = '保存',
  cancelLabel = '取消',
  onSubmit,
  onClose,
} = {}) {
  const formId = `ls-form-${Math.random().toString(36).slice(2, 9)}`
  const editor = form({
    fields,
    onSubmit: (values, api) => {
      if (onSubmit) onSubmit(values, { ...api, close, setBusy, setTitle })
    },
  })
  editor.element.id = formId
  const cancel = button({ label: cancelLabel, variant: 'secondary', onClick: () => close() })
  const submit = button({ label: submitLabel, type: 'submit' })
  submit.setAttribute('form', formId)
  const error = create('div', ui.status)
  error.setAttribute('role', 'status')
  const dialog = modal({
    title,
    size: 'lg',
    body: [error, editor.element],
    actions: [cancel, submit],
    onClose,
  })
  function setTitle(next) { dialog.setTitle(next) }
  function setBusy(busy) {
    editor.setBusy(busy)
    submit.disabled = busy
    cancel.disabled = busy
  }
  function setError(message = '') {
    const value = String(message ?? '').trim()
    error.textContent = value
    error.className = value ? statusClass('danger') : ui.status
    error.setAttribute('role', value ? 'alert' : 'status')
  }
  function close() { dialog.close() }
  return {
    element: dialog.element,
    form: editor,
    open(values, nextTitle) {
      editor.reset()
      if (nextTitle) setTitle(nextTitle)
      if (values) editor.set(values)
      dialog.open()
    },
    close,
    set: editor.set.bind(editor),
    reset: editor.reset.bind(editor),
    setTitle,
    setBusy,
    setError,
    control: editor.control.bind(editor),
  }
}
