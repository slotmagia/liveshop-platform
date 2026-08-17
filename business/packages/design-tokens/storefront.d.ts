export { create } from './index.js'

export type CtaVariant = 'primary' | 'secondary' | 'buy' | 'on-media'

export declare const shopUI: Readonly<{
  page: string
  section: string
  sectionHeader: string
  sectionTitle: string
  sectionLink: string
  hero: string
  grid: string
  product: string
  price: string
  discount: string
  tag: string
  cta: string
  options: string
  option: string
  panel: string
  overlay: string
  overlayBar: string
  overlayPill: string
  messages: string
  message: string
  empty: string
  skeleton: string
  sheetBackdrop: string
  sheet: string
  feature: string
  featureHero: string
  featureGrid: string
}>

export declare function ctaClass(variant?: CtaVariant, block?: boolean): string

export interface CtaOptions {
  label: string
  variant?: CtaVariant
  block?: boolean
  onClick?: (event: MouseEvent) => void
  disabled?: boolean
  type?: 'button' | 'submit' | 'reset'
}
export declare function cta(options: CtaOptions): HTMLButtonElement

export declare function tag(label: string): HTMLSpanElement
export declare function emptyState(text: string): HTMLDivElement
export declare function skeleton(options?: { height?: number | string; width?: number | string }): HTMLDivElement

export interface PriceOptions {
  /** Amount in minor units, so no caller has to round a float. */
  amount: number
  currency?: string
  original?: number
  discount?: string
  minorUnits?: number
}
export declare function price(options: PriceOptions): HTMLSpanElement

export interface HeroOptions {
  eyebrow?: string
  title?: string
  subtitle?: string
  image?: string
  actions?: Node | Node[]
}
export declare function hero(options?: HeroOptions): HTMLElement

export declare function section(options?: { title?: string; link?: Node; children?: Node | Node[] }): HTMLElement

export interface ProductCardOptions {
  title: string
  image?: string
  priceOptions?: PriceOptions
  meta?: string
  tags?: string[]
  onSelect?: (event: MouseEvent) => void
}
export declare function productCard(options: ProductCardOptions): HTMLButtonElement
export declare function productGrid(children?: Node | Node[]): HTMLDivElement

export interface FeatureScaffoldOptions {
  eyebrow?: string
  title?: string
  description?: string
  status?: string
  actions?: Node | Node[]
  metrics?: Array<{ label: string; value?: string | number }>
  sections?: Array<{ icon?: string; title: string; description?: string; items?: string[] }>
}
export declare function featureScaffold(options?: FeatureScaffoldOptions): HTMLElement

export interface OptionItem {
  value: string
  label: string
  hint?: string
  icon?: string
  disabled?: boolean
}
export interface OptionListApi {
  element: HTMLDivElement
  readonly value: string | undefined
  select(value: string): void
}
export declare function optionList(options: { items: OptionItem[]; value?: string; onChange?: (value: string) => void }): OptionListApi

export interface PanelApi {
  element: HTMLElement
  body: HTMLDivElement
}
export declare function panel(options?: { title?: string; actions?: Node | Node[]; body?: Node | Node[]; onMedia?: boolean }): PanelApi

export declare function overlay(options?: { top?: Node | Node[]; bottom?: Node | Node[] }): HTMLDivElement
export declare function livePill(options: { label: string; live?: boolean }): HTMLSpanElement

export interface MessageListApi {
  element: HTMLDivElement
  append(message: { author?: string; text: string }): void
  clear(): void
}
export declare function messageList(options?: { limit?: number }): MessageListApi

export interface SheetApi {
  element: HTMLDivElement
  body: HTMLDivElement
  open(host?: ParentNode): void
  close(): void
}
export declare function sheet(options?: { title?: string; body?: Node | Node[]; footer?: Node | Node[]; onClose?: () => void }): SheetApi
