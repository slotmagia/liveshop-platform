export declare const NAV_GROUP_ICONS: Readonly<Record<string, string>>
export declare const NAV_PAGE_ICONS: Readonly<Record<string, string>>
export declare const NAV_ICON_NAMES: readonly string[]
export declare function resolveNavIconName(name?: string, fallback?: string): string
export declare function resolvePageIconName(pageId?: string, explicit?: string, fallback?: string): string
export declare function resolveGroupIconName(groupId?: string, explicit?: string, fallback?: string): string
export declare function navIcon(name?: string, className?: string, fallback?: string): SVGSVGElement

export type Tone = 'neutral' | 'success' | 'warning' | 'danger' | 'info'
export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
export type ButtonSize = 'default' | 'sm'

/** A value a component may render: plain text, or a node built by this kit. */
export type Content = string | number | boolean | null | undefined | Node

export declare const ui: Readonly<{
  root: string
  page: string
  pageHeader: string
  pageHeading: string
  eyebrow: string
  title: string
  description: string
  actions: string
  grid: string
  card: string
  cardHeader: string
  cardTitle: string
  cardBody: string
  searchCard: string
  tabs: string
  tab: string
  dataCard: string
  tableToolbar: string
  tableToolbarTitle: string
  tableToolbarActions: string
  dataCardStatus: string
  dataCardFooter: string
  pagination: string
  paginationSummary: string
  paginationControls: string
  paginationPageSize: string
  checkboxTree: string
  checkboxTreeNode: string
  checkboxTreeRow: string
  checkboxTreeToggle: string
  checkboxTreeTogglePlaceholder: string
  checkboxTreeChoice: string
  checkboxTreeContent: string
  checkboxTreeLabel: string
  checkboxTreeDescription: string
  checkboxTreeChildren: string
  form: string
  searchForm: string
  searchFormActions: string
  searchFormToggle: string
  field: string
  fieldWide: string
  fieldSpan2: string
  dateRange: string
  dateRangeSep: string
  input: string
  select: string
  searchSelect: string
  searchSelectNative: string
  searchSelectTrigger: string
  searchSelectValue: string
  searchSelectMenu: string
  searchSelectSearch: string
  searchSelectList: string
  searchSelectOption: string
  searchSelectEmpty: string
  textarea: string
  button: string
  tableWrap: string
  table: string
  status: string
  empty: string
  badge: string
  statGrid: string
  stat: string
  definitionList: string
  code: string
  modalBackdrop: string
  modal: string
  modalLg: string
  modalHeader: string
  modalTitle: string
  modalBody: string
  toastHost: string
  toast: string
  toastMessage: string
  toastDismiss: string
}>

export declare function buttonClass(variant?: ButtonVariant, size?: ButtonSize): string
export declare function badgeClass(tone?: Tone): string
export declare function statusClass(tone?: Tone): string

export declare function create<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  text?: string | number | boolean | null,
): HTMLElementTagNameMap[K]

export interface ButtonOptions {
  label: string
  variant?: ButtonVariant
  size?: ButtonSize
  type?: 'button' | 'submit' | 'reset'
  onClick?: (event: MouseEvent) => void
  disabled?: boolean
  title?: string
}
export declare function button(options: ButtonOptions): HTMLButtonElement

export declare function badge(options: { label: string; tone?: Tone }): HTMLSpanElement
export declare function emptyState(text: string): HTMLDivElement
export declare function code(text: string): HTMLPreElement

export interface NotifyOptions {
  duration?: number
}
export declare function notify(text: string, tone?: Tone, options?: NotifyOptions): void

export interface StatusLine {
  element: HTMLDivElement
  set(text: string | null | undefined, tone?: Tone): void
  clear(): void
}
export declare function statusLine(): StatusLine

export interface PageOptions {
  eyebrow?: string
  title?: string
  description?: string
  actions?: Node | Node[]
  children?: Node | Node[]
  /** Host-mounted menu pages set false because the Host owns the Manifest-backed summary card. */
  showSummary?: boolean
}
export declare function page(options?: PageOptions): HTMLElement

export interface CardOptions {
  title?: string
  headerExtra?: Node | Node[]
  body?: Node | Node[]
  /** Set false when the body owns its own padding, such as a table. */
  padded?: boolean
}
export declare function card(options?: CardOptions): HTMLElement
export declare function searchCard(body: Node | Node[]): HTMLElement
export interface TabItem {
  value: string
  label: string
}
export interface TabsOptions {
  items: TabItem[]
  value?: string
  ariaLabel?: string
  onChange?: (value: string) => void
}
export interface TabsApi {
  element: HTMLElement
  set(value: string): void
  value(): string
}
export declare function tabs(options: TabsOptions): TabsApi
export interface DataCardOptions {
  title?: string
  actions?: Node | Node[]
  status?: Node | Node[]
  body?: Node | Node[]
  footer?: Node | Node[]
}
export declare function dataCard(options?: DataCardOptions): HTMLElement

export interface PaginationState {
  page: number
  pageSize: number
  total: number | null
  pages?: number
}
export interface PaginationApi {
  element: HTMLElement
  summary: HTMLSpanElement
  previous: HTMLButtonElement
  next: HTMLButtonElement
  pageSizeSelect: HTMLSelectElement
  set(state: Partial<{ page: number; pageSize: number; total: number | null; itemCount: number }>): void
  setBusy(busy: boolean): void
}
export interface PaginationOptions {
  page?: number
  pageSize?: number
  total?: number | null
  itemCount?: number
  pageSizeOptions?: number[]
  previousLabel?: string
  nextLabel?: string
  pageSizeLabel?: string
  summary?: (state: PaginationState) => Content
  actions?: Node | Node[]
  onPageChange?: (page: number) => void
  onPageSizeChange?: (pageSize: number) => void
}
export declare function pagination(options?: PaginationOptions): PaginationApi

export declare function grid(children?: Node | Node[]): HTMLDivElement
export declare function statGrid(items: Array<{ label: string; value: Content }>): HTMLDivElement
export declare function definitionList(items: Array<{ label: string; value: Content }>): HTMLDListElement

export type FormControl = HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
export interface SelectOption { value: string | number; label?: string }
export interface CheckboxTreeNode {
  id: string
  label: string
  value?: string | number
  description?: string
  children?: CheckboxTreeNode[]
}
export interface CheckboxTreeApi {
  element: HTMLDivElement
  control: HTMLInputElement
  value: string
  disabled: boolean
  selected(): string[]
  set(value: string | string[]): void
  reset(): void
}
export declare function checkboxTree(options?: {
  nodes?: CheckboxTreeNode[]
  value?: string | string[]
  name?: string
  disabled?: boolean
  empty?: string
  columns?: 1 | 2 | 3
}): CheckboxTreeApi

export interface FieldDescriptor {
  name: string
  label?: string
  kind?: 'input' | 'select' | 'textarea' | 'date-range' | 'checkbox-tree'
  type?: string
  value?: string | number
  placeholder?: string
  inputMode?: string
  required?: boolean
  disabled?: boolean
  /** Spans the full form width. */
  wide?: boolean
  /** Renders the value in the monospace token, for JSON and identifiers. */
  mono?: boolean
  rows?: number
  options?: Array<SelectOption | string>
  min?: number | string
  max?: number | string
  step?: number | string
  minLength?: number
  maxLength?: number
  autocomplete?: string
  /** date-range: start control name; defaults to `${name}From`. */
  from?: string
  /** date-range: end control name; defaults to `${name}To`. */
  to?: string
  fromValue?: string | number
  toValue?: string | number
  /** date-range separator text; defaults to 至. */
  separator?: string
  /** checkbox-tree: hierarchical groups whose leaf values are submitted. */
  tree?: CheckboxTreeNode[]
  /** checkbox-tree: leaf catalog columns; default 1 keeps the hierarchical list. */
  columns?: 1 | 2 | 3
  empty?: string
}
export declare function field(descriptor: FieldDescriptor): {
  element: HTMLElement
  control: FormControl
  endControl?: FormControl
  from?: FormControl
  to?: FormControl
}

export interface FormApi {
  element: HTMLFormElement
  submit?: HTMLButtonElement
  control(name: string): FormControl | undefined
  values(): Record<string, string>
  set(values: Record<string, string | number | null | undefined>): void
  reset(): void
  setBusy(busy: boolean): void
}
export interface FormOptions {
  fields: FieldDescriptor[]
  submitLabel?: string
  onSubmit?: (values: Record<string, string>, api: FormApi) => void
  extraActions?: Node | Node[]
}
export declare function form(options: FormOptions): FormApi

export interface SearchFormApi {
  element: HTMLFormElement
  toggleButton: HTMLButtonElement
  search: HTMLButtonElement
  resetButton: HTMLButtonElement
  control(name: string): FormControl | undefined
  values(): Record<string, string>
  set(values: Record<string, string | number | null | undefined>): void
  clear(): void
  reset(): void
  isExpanded(): boolean
  setExpanded(expanded: boolean): void
  destroy(): void
  setBusy(busy: boolean): void
}
export interface SearchFormOptions {
  fields: Array<Omit<FieldDescriptor, 'wide'>>
  searchLabel?: string
  resetLabel?: string
  expandLabel?: string
  collapseLabel?: string
  onSearch?: (values: Record<string, string>, api: SearchFormApi) => void
  onReset?: (api: SearchFormApi) => void
}
/** Six equal field columns + one action column. Field changes auto-invoke onSearch. */
export declare function searchForm(options: SearchFormOptions): SearchFormApi

export interface TableApi {
  element: HTMLDivElement
  setRows(rows: Content[][]): void
}
export interface TableOptions {
  columns: string[]
  rows?: Content[][]
  onRowClick?: (index: number) => void
  empty?: string
}
export declare function table(options: TableOptions): TableApi

export interface ModalApi {
  element: HTMLDivElement
  body: HTMLDivElement
  setTitle(title: string): void
  open(host?: ParentNode): void
  close(): void
}
export interface ModalOptions {
  title?: string
  body?: Node | Node[]
  actions?: Node | Node[]
  size?: 'md' | 'lg'
  onClose?: () => void
}
export declare function modal(options?: ModalOptions): ModalApi

export interface FormModalApi {
  element: HTMLDivElement
  form: FormApi
  open(values?: Record<string, string | number | null | undefined>, title?: string): void
  close(): void
  set(values: Record<string, string | number | null | undefined>): void
  reset(): void
  setTitle(title: string): void
  setBusy(busy: boolean): void
  setError(message?: string): void
  control(name: string): FormControl | undefined
}
export interface FormModalSubmitApi extends FormApi {
  close(): void
  setTitle(title: string): void
}
export interface FormModalOptions {
  title?: string
  fields: FieldDescriptor[]
  submitLabel?: string
  cancelLabel?: string
  onSubmit?: (values: Record<string, string>, api: FormModalSubmitApi) => void
  onClose?: () => void
}
/** Create/edit dialog: two-column form, footer 取消 / 保存. */
export declare function formModal(options: FormModalOptions): FormModalApi
