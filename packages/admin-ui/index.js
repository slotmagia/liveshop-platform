/**
 * Stable, framework-neutral component class contract for every Admin surface.
 * React and DOM contributions intentionally share this exact vocabulary.
 */
export const adminUI = Object.freeze({
  root: 'ls-admin-root',
  page: 'ls-admin-page',
  pageHeader: 'ls-admin-page-header',
  pageHeading: 'ls-admin-page-heading',
  eyebrow: 'ls-admin-eyebrow',
  title: 'ls-admin-title',
  description: 'ls-admin-description',
  actions: 'ls-admin-actions',
  grid: 'ls-admin-grid',
  card: 'ls-admin-card',
  cardHeader: 'ls-admin-card-header',
  cardTitle: 'ls-admin-card-title',
  cardBody: 'ls-admin-card-body',
  formGrid: 'ls-admin-form-grid',
  field: 'ls-admin-field',
  fieldWide: 'ls-admin-field--wide',
  input: 'ls-admin-input',
  select: 'ls-admin-select',
  textarea: 'ls-admin-textarea',
  button: 'ls-admin-button',
  tableWrap: 'ls-admin-table-wrap',
  table: 'ls-admin-table',
  status: 'ls-admin-status',
  empty: 'ls-admin-empty',
  badge: 'ls-admin-badge',
  statGrid: 'ls-admin-stat-grid',
  stat: 'ls-admin-stat',
  definitionList: 'ls-admin-definition-list',
  code: 'ls-admin-code',
  modalBackdrop: 'ls-admin-modal-backdrop',
  modal: 'ls-admin-modal',
  modalHeader: 'ls-admin-modal-header',
  modalBody: 'ls-admin-modal-body',
  modalFooter: 'ls-admin-modal-footer',
})

export function adminButtonClass(variant = 'primary', size = 'default') {
  return `${adminUI.button} ${adminUI.button}--${variant}${size === 'default' ? '' : ` ${adminUI.button}--${size}`}`
}

export function adminStatusClass(tone = 'neutral') {
  return `${adminUI.status} ${adminUI.status}--${tone}`
}

export function adminBadgeClass(tone = 'neutral') {
  return `${adminUI.badge} ${adminUI.badge}--${tone}`
}
