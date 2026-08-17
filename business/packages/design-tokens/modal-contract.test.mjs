import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const css = readFileSync(new URL('./console.components.css', import.meta.url), 'utf8')
const factory = readFileSync(new URL('./index.js', import.meta.url), 'utf8')

test('console modal owns the full viewport and scrolls only its body', () => {
  assert.match(css, /\.ls-ui-modal-backdrop\s*\{[^}]*position:\s*fixed[^}]*inset:\s*0[^}]*width:\s*100vw[^}]*height:\s*100dvh[^}]*overflow:\s*hidden/s)
  assert.match(css, /\.ls-ui-modal\s*\{[^}]*overflow:\s*hidden[^}]*display:\s*grid[^}]*grid-template-rows:\s*auto minmax\(0, 1fr\) auto/s)
  assert.match(css, /\.ls-ui-modal-header\s*\{[^}]*flex:\s*none/s)
  assert.match(css, /\.ls-ui-modal-body\s*\{[^}]*min-height:\s*0[^}]*overflow-y:\s*auto/s)
  assert.match(css, /\.ls-ui-modal-footer\s*\{[^}]*flex:\s*none/s)
  assert.doesNotMatch(css, /\.ls-ui-modal\s*\{[^}]*overflow:\s*auto/s)
})

test('console page summary is a standalone card', () => {
  assert.match(css, /\.ls-ui-page-header\s*\{[^}]*border:\s*1px solid var\(--ls-border\)[^}]*border-radius:\s*var\(--ls-radius-md\)[^}]*background:\s*var\(--ls-surface\)[^}]*box-shadow:\s*var\(--ls-shadow-card\)/s)
})

test('search and data regions are independent cards with a data toolbar', () => {
  const pageRules = [...css.matchAll(/\.ls-ui-page\s*\{(?<body>[^}]*)\}/gs)]
  assert.ok(pageRules.length > 0)
  for (const rule of pageRules) assert.match(rule.groups.body, /padding:\s*0\s*;/)
  assert.match(css, /\.ls-ui-search-card\s*\{[^}]*margin-bottom:\s*12px/s)
  assert.match(css, /\.ls-ui-table-toolbar\s*\{[^}]*justify-content:\s*space-between[^}]*border-bottom:\s*1px solid var\(--ls-border\)[^}]*background:\s*var\(--ls-surface\)/s)
  assert.match(css, /\.ls-ui-table-toolbar__actions\s*\{[^}]*justify-content:\s*flex-end/s)
})

test('list pagination belongs to the data-card footer and owns page size', () => {
  assert.match(css, /\.ls-ui-pagination\s*\{[^}]*justify-content:\s*space-between[^}]*width:\s*100%/s)
  assert.match(css, /\.ls-ui-pagination__controls\s*\{[^}]*justify-content:\s*flex-end/s)
  assert.match(css, /\.ls-ui-pagination__page-size\s*\{[^}]*display:\s*inline-flex/s)
  assert.match(factory, /export function pagination\(\{[\s\S]*?pageSizeOptions = \[20, 50, 100\]/)
  assert.match(factory, /controls\.append\(sizeField, previous, next, \.\.\.extraActions\)/)
  assert.match(factory, /onPageSizeChange\?\.\(value\)/)
})

test('permission selection uses one shared hierarchical checkbox tree', () => {
  assert.match(css, /\.ls-ui-checkbox-tree\s*\{[^}]*max-height:\s*360px[^}]*overflow:\s*auto/s)
  assert.match(css, /\.ls-ui-checkbox-tree__children\s*\{[^}]*border-top:\s*1px solid var\(--ls-border\)/s)
  assert.match(css, /\.ls-ui-checkbox-tree\[data-columns="3"\]\s*\{[^}]*grid-template-columns:\s*repeat\(3,\s*minmax\(0,\s*1fr\)\)/s)
  assert.match(factory, /export function checkboxTree\(\{[\s\S]*?selected = checkboxTreeValues\(value\)/)
  assert.match(factory, /record\.input\.indeterminate = checked > 0 && checked < record\.values\.length/)
  assert.match(factory, /controlNode\.type = 'hidden'[\s\S]*?controlNode\.value = api\.value/)
  assert.match(factory, /if \(columnCount > 1\) element\.dataset\.columns = String\(columnCount\)/)
  assert.match(factory, /if \(descriptor\.kind === 'checkbox-tree'\)/)
})

test('search actions stay at the card right edge on the first row', () => {
  assert.match(css, /\.ls-ui-search-form__actions\s*\{[^}]*justify-content:\s*flex-end[^}]*justify-self:\s*end[^}]*grid-column:\s*-2\s*\/\s*-1[^}]*grid-row:\s*1/s)
  assert.match(css, /@media\s*\(max-width:\s*820px\)[^{]*\{[\s\S]*?\.ls-ui-search-form\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)\s*max-content[^}]*\}[\s\S]*?\.ls-ui-search-form__actions\s*\{[^}]*grid-column:\s*2[^}]*grid-row:\s*1/s)
})

test('multi-row search forms expose one responsive expand and collapse control before search', () => {
  assert.match(css, /\.ls-ui-search-form\s*\{[^}]*--ls-search-field-columns:\s*6/s)
  assert.match(css, /@media\s*\(max-width:\s*1100px\)[^{]*\{[\s\S]*?--ls-search-field-columns:\s*3/s)
  assert.match(css, /@media\s*\(max-width:\s*820px\)[^{]*\{[\s\S]*?--ls-search-field-columns:\s*1/s)
  assert.match(css, /\.ls-ui-search-form\[data-collapsed="true"\]\s*>\s*\[data-search-overflow="true"\]\s*\{\s*display:\s*none/s)
  assert.match(css, /\.ls-ui-search-form__toggle\[hidden\]\s*\{\s*display:\s*none/s)
  assert.match(factory, /actions\.append\(toggleButton,\s*search,\s*resetButton\)/)
  assert.match(factory, /toggleButton\.textContent\s*=\s*collapsed\s*\?\s*expandLabel\s*:\s*collapseLabel/)
  assert.match(factory, /if \(overflowStarted\) item\.element\.dataset\.searchOverflow = 'true'/)
  assert.match(factory, /toggleButton\.setAttribute\('aria-expanded',\s*String\(!collapsed\)\)/)
})

test('search fields use equal columns and only date ranges span two', () => {
  assert.match(css, /\.ls-ui-search-form\s*\{[^}]*grid-template-columns:\s*repeat\(6,\s*minmax\(0,\s*1fr\)\)/s)
  assert.match(factory, /else if \(descriptor\.kind === 'date-range'\) classes\.push\(ui\.fieldSpan2\)/)
  assert.doesNotMatch(factory, /descriptor\.span/)
  assert.match(factory, /export function searchForm[\s\S]*?field\(\{ \.\.\.descriptor, wide: false \}\)/)
})

test('search forms auto-search when fields change', () => {
  assert.match(factory, /searchTimer = window\.setTimeout\(requestSearch, 300\)/)
  assert.match(factory, /element\.addEventListener\('input'/)
  assert.match(factory, /element\.addEventListener\('change'/)
  assert.match(factory, /isImmediateControl\(target\)/)
})

test('selects are a shared searchable combobox', () => {
  assert.match(factory, /function attachSearchSelect\(/)
  assert.match(factory, /trigger\.setAttribute\('role', 'combobox'\)/)
  assert.match(factory, /search\.addEventListener\('input', event => \{\s*event\.stopPropagation\(\)/)
  assert.match(css, /\.ls-ui-search-select__trigger\s*\{[^}]*height:\s*36px/s)
  assert.match(css, /\.ls-ui-search-select__search\s*\{[^}]*height:\s*32px/s)
  assert.match(css, /\.ls-ui-search-select__option\s*\{[^}]*min-height:\s*32px/s)
  assert.match(css, /\.ls-ui-search-select__menu\s*\{[^}]*position:\s*fixed/s)
  assert.match(factory, /document\.body\.append\(menu\)/)
  assert.doesNotMatch(factory, /searchSelectSpacer|spacer\.style\.height/)
})

test('searchable select menus stack above host form modals', () => {
  const tokens = readFileSync(new URL('./tokens.css', import.meta.url), 'utf8')
  const token = (name) => {
    const match = tokens.match(new RegExp(`--ls-z-${name}:\\s*(\\d+)`))
    assert.ok(match, `missing --ls-z-${name}`)
    return Number(match[1])
  }
  const modal = token('modal')
  const hostOverlay = token('host-overlay')
  const hostModal = token('host-modal')
  const toast = token('toast')
  const popover = token('popover')
  assert.ok(modal < hostOverlay, 'iframe modal must sit below Host overlay')
  assert.ok(hostOverlay < hostModal, 'Host overlay must sit below Host form modal')
  assert.ok(hostModal < toast, 'toasts must sit above Host form modal')
  assert.ok(toast < popover, 'select menus must sit above toasts')
  assert.match(css, /\.ls-ui-search-select__menu\s*\{[^}]*z-index:\s*var\(--ls-z-popover\)/s)
  assert.match(css, /\.ls-ui-modal-backdrop\s*\{[^}]*z-index:\s*var\(--ls-z-modal\)/s)
  assert.match(css, /\.ls-ui-toast-host\s*\{[^}]*position:\s*fixed[^}]*z-index:\s*var\(--ls-z-toast\)/s)
  assert.match(factory, /export function notify\(/)
  assert.match(factory, /LIVESHOP_HOST_NOTIFY/)
  assert.match(factory, /shouldNotifyStatus/)
  assert.match(factory, /function setError\(message = ''\) \{\s*const value = String\(message/)
  assert.doesNotMatch(factory, /const error = statusLine\(\)/)
  assert.match(factory, /document\.body\.append\(menu\)/)
  assert.match(factory, /ls-ui-close-popovers/)
  assert.match(factory, /event\.defaultPrevented/)
  assert.match(factory, /!isPopoverSurface\(node\)/)
})
