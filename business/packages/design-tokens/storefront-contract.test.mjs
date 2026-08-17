import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const css = readFileSync(new URL('./storefront.components.css', import.meta.url), 'utf8')
const factory = readFileSync(new URL('./storefront.js', import.meta.url), 'utf8')
const tokens = readFileSync(new URL('./tokens.css', import.meta.url), 'utf8')

test('storefront feature scaffold preserves final responsive geometry without fake business data', () => {
  assert.match(factory, /export function featureScaffold/)
  assert.match(factory, /item\.value \?\? '—'/)
  assert.match(css, /\.ls-shop-feature__metrics\s*\{[^}]*grid-template-columns:\s*repeat\(4,/s)
  assert.match(css, /@media\s*\(max-width:\s*600px\)[\s\S]*?\.ls-shop-feature__metrics\s*\{[^}]*grid-template-columns:\s*repeat\(2,/s)
  assert.match(css, /\.ls-shop-feature__grid\s*\{[^}]*grid-template-columns:\s*repeat\(3,/s)
})

test('storefront controls remain touch sized and sheets respect the safe area', () => {
  assert.match(css, /\.ls-shop-cta\s*\{[^}]*min-height:\s*var\(--ls-touch\)/s)
  assert.match(css, /\.ls-shop-sheet__footer\s*\{[^}]*env\(safe-area-inset-bottom\)/s)
})

test('storefront visual tokens preserve the legacy WOKFOY shop source palette', () => {
  assert.match(tokens, /--ls-shop-page:\s*rgb\(244 246 248\)/)
  assert.match(tokens, /--ls-shop-ink-rgb:\s*17 24 39/)
  assert.match(tokens, /--ls-shop-gold:\s*rgb\(251 209 128\)/)
  assert.match(tokens, /--ls-shop-tabbar:\s*rgb\(67 67 65\)/)
  assert.match(tokens, /--ls-shop-accent-rgb:\s*30 94 255/)
  assert.match(css, /\.ls-shop-product__badge\s*\{[^}]*background:\s*rgb\(17 24 39 \/ \.84\)/s)
})
