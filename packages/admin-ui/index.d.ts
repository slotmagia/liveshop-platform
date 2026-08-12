export type AdminButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
export type AdminButtonSize = 'default' | 'sm' | 'icon'
export type AdminTone = 'neutral' | 'success' | 'warning' | 'danger' | 'info'

export declare const adminUI: Readonly<{
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
  formGrid: string
  field: string
  fieldWide: string
  input: string
  select: string
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
  modalHeader: string
  modalBody: string
  modalFooter: string
}>

export declare function adminButtonClass(variant?: AdminButtonVariant, size?: AdminButtonSize): string
export declare function adminStatusClass(tone?: AdminTone): string
export declare function adminBadgeClass(tone?: AdminTone): string
