/**
 * The framework-neutral component kit for the storefront and live-room
 * surfaces. It mirrors the console kit: same `create` helper, same rule that
 * every value is written through `textContent`, and the same prohibition on a
 * business module declaring its own visuals.
 *
 * The two kits stay separate because the experiences genuinely differ. A
 * storefront control is touch-sized and may sit on video; a back-office control
 * is dense and always sits on a neutral surface. Sharing one component here
 * would force one of the two to compromise.
 */

import { create } from './index.js'

export { create } from './index.js'

export const shopUI = Object.freeze({
  page: 'ls-shop-page',
  section: 'ls-shop-section',
  sectionHeader: 'ls-shop-section-header',
  sectionTitle: 'ls-shop-section-title',
  sectionLink: 'ls-shop-section-link',
  hero: 'ls-shop-hero',
  grid: 'ls-shop-grid',
  product: 'ls-shop-product',
  price: 'ls-shop-price',
  discount: 'ls-shop-discount',
  tag: 'ls-shop-tag',
  cta: 'ls-shop-cta',
  options: 'ls-shop-options',
  option: 'ls-shop-option',
  panel: 'ls-shop-panel',
  overlay: 'ls-shop-overlay',
  overlayBar: 'ls-shop-overlay__bar',
  overlayPill: 'ls-shop-overlay__pill',
  messages: 'ls-shop-messages',
  message: 'ls-shop-message',
  empty: 'ls-shop-empty',
  skeleton: 'ls-shop-skeleton',
  sheetBackdrop: 'ls-shop-sheet-backdrop',
  sheet: 'ls-shop-sheet',
  feature: 'ls-shop-feature',
  featureHero: 'ls-shop-feature__hero',
  featureGrid: 'ls-shop-feature__grid',
})

function list(value) {
  if (value === undefined || value === null) return []
  return Array.isArray(value) ? value.filter(Boolean) : [value]
}

export function ctaClass(variant = 'primary', block = false) {
  const base = variant === 'primary' ? shopUI.cta : `${shopUI.cta} ${shopUI.cta}--${variant}`
  return block ? `${base} ${shopUI.cta}--block` : base
}

export function cta({ label, variant = 'primary', block = false, onClick, disabled = false, type = 'button' } = {}) {
  const node = create('button', ctaClass(variant, block), label)
  node.type = type
  node.disabled = disabled
  if (onClick) node.addEventListener('click', onClick)
  return node
}

export function tag(label) {
  return create('span', shopUI.tag, label)
}

export function emptyState(text) {
  return create('div', shopUI.empty, text)
}

/** A shimmering placeholder sized by the caller while data is in flight. */
export function skeleton({ height = 160, width } = {}) {
  const node = create('div', shopUI.skeleton)
  node.style.height = typeof height === 'number' ? `${height}px` : height
  if (width !== undefined) node.style.width = typeof width === 'number' ? `${width}px` : width
  return node
}

/**
 * Formats a minor-unit amount. Money never reaches the DOM as a raw number:
 * a storefront that renders "1999" instead of "¥19.99" is a pricing incident.
 */
export function price({ amount, currency = '¥', original, discount, minorUnits = 2 } = {}) {
  const node = create('span', shopUI.price)
  node.append(create('span', 'ls-shop-price__symbol', currency))
  node.append(create('span', undefined, formatAmount(amount, minorUnits)))
  if (original !== undefined && original !== null) {
    node.append(create('span', 'ls-shop-price__original', `${currency}${formatAmount(original, minorUnits)}`))
  }
  if (discount) node.append(create('span', shopUI.discount, discount))
  return node
}

function formatAmount(amount, minorUnits) {
  const value = Number(amount)
  if (!Number.isFinite(value)) return '—'
  return (value / 10 ** minorUnits).toFixed(minorUnits)
}

export function hero({ eyebrow, title, subtitle, image, actions } = {}) {
  const node = create('section', shopUI.hero)
  if (image) node.style.backgroundImage = `linear-gradient(180deg, transparent 30%, var(--ls-overlay) 100%), url("${encodeURI(image)}")`
  if (eyebrow) node.append(create('p', 'ls-shop-hero__eyebrow', eyebrow))
  if (title) node.append(create('h1', 'ls-shop-hero__title', title))
  if (subtitle) node.append(create('p', 'ls-shop-hero__subtitle', subtitle))
  for (const action of list(actions)) node.append(action)
  return node
}

export function section({ title, link, children } = {}) {
  const node = create('section', shopUI.section)
  if (title || link) {
    const header = create('div', shopUI.sectionHeader)
    header.append(create('h2', shopUI.sectionTitle, title ?? ''))
    if (link) header.append(link)
    node.append(header)
  }
  node.append(...list(children))
  return node
}

export function productCard({ title, image, priceOptions, meta, tags, onSelect } = {}) {
  const node = create('button', shopUI.product)
  node.type = 'button'
  const tagItems = list(tags)
  if (tagItems.length) node.append(create('span', 'ls-shop-product__badge', tagItems[0]))
  const media = create('img', 'ls-shop-product__media')
  media.loading = 'lazy'
  media.alt = title ?? ''
  if (image) media.src = image
  node.append(media)
  const body = create('div', 'ls-shop-product__body')
  body.append(create('p', 'ls-shop-product__title', title))
  if (meta) body.append(create('p', 'ls-shop-product__meta', meta))
  if (priceOptions) {
    const footer = create('div', 'ls-shop-product__footer')
    footer.append(price(priceOptions), create('span', 'ls-shop-product__add', '+'))
    body.append(footer)
  }
  node.append(body)
  if (onSelect) node.addEventListener('click', onSelect)
  return node
}

export function productGrid(children) {
  const node = create('div', shopUI.grid)
  node.append(...list(children))
  return node
}

/**
 * A truthful feature scaffold for a route whose domain API is not published
 * yet. It preserves the final page geometry and interaction map without
 * inventing balances, orders, products or any other business fact.
 */
export function featureScaffold({ eyebrow, title, description, status = '功能迁移中', actions, metrics, sections } = {}) {
  const root = create('main', shopUI.feature)
  const banner = create('section', shopUI.featureHero)
  const copy = create('div', 'ls-shop-feature__copy')
  if (eyebrow) copy.append(create('p', 'ls-shop-feature__eyebrow', eyebrow))
  copy.append(create('h2', undefined, title ?? ''))
  if (description) copy.append(create('p', undefined, description))
  const state = create('span', 'ls-shop-feature__status', status)
  const actionBar = create('div', 'ls-shop-feature__actions')
  for (const action of list(actions)) actionBar.append(action)
  banner.append(copy, state)
  if (actionBar.childElementCount) banner.append(actionBar)
  root.append(banner)

  const metricItems = list(metrics)
  if (metricItems.length) {
    const row = create('section', 'ls-shop-feature__metrics')
    for (const item of metricItems) {
      const metric = create('div', 'ls-shop-feature__metric')
      metric.append(create('span', undefined, item.label), create('strong', undefined, item.value ?? '—'))
      row.append(metric)
    }
    root.append(row)
  }

  const cards = create('section', shopUI.featureGrid)
  for (const item of list(sections)) {
    const card = create('article', 'ls-shop-feature__card')
    const icon = create('span', 'ls-shop-feature__icon', item.icon ?? '•')
    const body = create('div')
    body.append(create('h3', undefined, item.title ?? ''))
    if (item.description) body.append(create('p', undefined, item.description))
    const steps = list(item.items)
    if (steps.length) {
      const checklist = create('ul')
      for (const value of steps) checklist.append(create('li', undefined, value))
      body.append(checklist)
    }
    card.append(icon, body)
    cards.append(card)
  }
  if (cards.childElementCount) root.append(cards)
  return root
}

/**
 * A single-select list. It owns the selected state so a checkout page cannot
 * end up with two payment methods visually selected at once.
 */
export function optionList({ items, value, onChange } = {}) {
  const element = create('div', shopUI.options)
  element.setAttribute('role', 'radiogroup')
  let selected = value
  const nodes = new Map()
  for (const item of list(items)) {
    const node = create('div', shopUI.option)
    node.setAttribute('role', 'radio')
    node.tabIndex = 0
    if (item.icon) {
      const icon = create('img', 'ls-shop-option__icon')
      icon.src = item.icon
      icon.alt = ''
      node.append(icon)
    }
    const body = create('div', 'ls-shop-option__body')
    body.append(create('span', 'ls-shop-option__label', item.label))
    if (item.hint) body.append(create('span', 'ls-shop-option__hint', item.hint))
    node.append(body, create('span', 'ls-shop-option__mark'))
    const choose = () => {
      if (item.disabled) return
      selected = item.value
      apply()
      if (onChange) onChange(selected)
    }
    node.addEventListener('click', choose)
    node.addEventListener('keydown', (event) => {
      if (event.key === ' ' || event.key === 'Enter') { event.preventDefault(); choose() }
    })
    nodes.set(item.value, node)
    element.append(node)
  }
  const apply = () => {
    for (const [key, node] of nodes) {
      const active = key === selected
      node.dataset.selected = String(active)
      node.setAttribute('aria-checked', String(active))
    }
  }
  apply()
  return {
    element,
    get value() { return selected },
    select(next) { selected = next; apply() },
  }
}

export function panel({ title, actions, body, onMedia = false } = {}) {
  const element = create('section', onMedia ? `${shopUI.panel} ${shopUI.panel}--on-media` : shopUI.panel)
  if (title || actions) {
    const header = create('div', 'ls-shop-panel__header')
    header.append(create('h2', 'ls-shop-panel__title', title ?? ''))
    for (const action of list(actions)) header.append(action)
    element.append(header)
  }
  const content = create('div', 'ls-shop-panel__body')
  content.append(...list(body))
  element.append(content)
  return { element, body: content }
}

/** The control layer over a player. Only its children take pointer events. */
export function overlay({ top, bottom } = {}) {
  const element = create('div', shopUI.overlay)
  const topBar = create('div', shopUI.overlayBar)
  topBar.append(...list(top))
  const bottomBar = create('div', shopUI.overlayBar)
  bottomBar.append(...list(bottom))
  element.append(topBar, bottomBar)
  return element
}

export function livePill({ label, live = false } = {}) {
  const node = create('span', shopUI.overlayPill)
  if (live) node.append(create('i'))
  node.append(create('span', undefined, label))
  return node
}

/** An append-only feed that keeps the newest message in view. */
export function messageList({ limit = 200 } = {}) {
  const element = create('div', shopUI.messages)
  return {
    element,
    append({ author, text }) {
      const node = create('div', shopUI.message)
      if (author) node.append(create('span', 'ls-shop-message__author', author))
      node.append(create('span', 'ls-shop-message__text', text))
      element.append(node)
      while (element.childElementCount > limit) element.firstElementChild.remove()
      node.scrollIntoView({ block: 'nearest' })
    },
    clear() { element.replaceChildren() },
  }
}

/** A bottom sheet on phones, a centred dialog from 760px up. */
export function sheet({ title, body, footer, onClose } = {}) {
  const backdrop = create('div', shopUI.sheetBackdrop)
  const element = create('div', shopUI.sheet)
  element.setAttribute('role', 'dialog')
  element.setAttribute('aria-modal', 'true')
  element.append(create('div', 'ls-shop-sheet__handle'))
  const header = create('div', 'ls-shop-sheet__header')
  header.append(create('h2', 'ls-shop-sheet__title', title ?? ''))
  const content = create('div', 'ls-shop-sheet__body')
  content.append(...list(body))
  element.append(header, content)
  const bar = list(footer)
  if (bar.length) {
    const actions = create('div', 'ls-shop-sheet__footer')
    actions.append(...bar)
    element.append(actions)
  }
  backdrop.append(element)
  const close = () => {
    backdrop.remove()
    document.removeEventListener('keydown', escape)
    if (onClose) onClose()
  }
  const escape = (event) => { if (event.key === 'Escape') close() }
  backdrop.addEventListener('click', (event) => { if (event.target === backdrop) close() })
  return {
    element: backdrop,
    body: content,
    open(host = document.body) {
      host.append(backdrop)
      document.addEventListener('keydown', escape)
      element.querySelector('button, input, select, textarea')?.focus()
    },
    close,
  }
}
