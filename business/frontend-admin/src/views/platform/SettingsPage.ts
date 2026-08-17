import type { HostContext, HostHttpClient } from '@liveshop/host-sdk'
import { badge, button, buttonClass, create, dataCard, emptyState, field, page, statusLine, ui } from '@liveshop/design-tokens'

type FieldType = 'TEXT' | 'NUMBER' | 'BOOL' | 'SELECT' | 'TEXTAREA'

interface CatalogField {
  key: string
  label: string
  type: FieldType
  help?: string
  placeholder?: string
  options?: Array<{ value: string; label: string }>
}

interface CatalogCategory {
  key: string
  label: string
  groupKey: string
  groupLabel: string
  fields: CatalogField[]
}

interface CatalogGroup {
  key: string
  label: string
  categories: CatalogCategory[]
}

interface SettingDocument {
  namespace: string
  value: Record<string, unknown>
  version: number
  updatedBy?: string
  updatedAt?: string
}

const prefix = '/admin/platform/settings'

function fieldValue(spec: CatalogField, raw: unknown): string {
  if (spec.type === 'NUMBER') {
    const n = Number(raw)
    return Number.isFinite(n) ? String(n) : ''
  }
  if (spec.type === 'BOOL') return raw === true || raw === 'true' || raw === 1 || raw === '1' ? 'true' : 'false'
  return raw == null ? '' : String(raw)
}

function typedValue(spec: CatalogField, raw: string | boolean): unknown {
  if (spec.type === 'NUMBER') {
    const n = Number(raw)
    return Number.isFinite(n) ? n : 0
  }
  if (spec.type === 'BOOL') return raw === true || raw === 'true'
  return String(raw ?? '')
}

function categoryValues(category: CatalogCategory, document?: SettingDocument): Record<string, string> {
  const values: Record<string, string> = {}
  for (const spec of category.fields) values[spec.key] = fieldValue(spec, document?.value?.[spec.key])
  return values
}

export async function startSettings(root: HTMLElement, client: HostHttpClient, context: HostContext): Promise<void> {
  const state = statusLine()
  const canWrite = context.permissions.includes('platform.settings.write')
  const collapsed = new Set<string>()
  let groups: CatalogGroup[] = []
  let selected = ''
  let documents = new Map<string, SettingDocument>()
  let dirty = false
  let busy = false
  let metadataError = ''

  const titleNode = create('h2', ui.tableToolbarTitle, '配置分类')
  const treeHost = create('div')
  const formHost = create('div')
  formHost.style.minWidth = '0'
  const saveButton = button({
    label: '保存',
    disabled: !canWrite,
    onClick: () => { formHost.querySelector('form')?.requestSubmit() },
  })
  const refreshButton = button({
    label: '刷新',
    variant: 'secondary',
    onClick: () => { void loadDocuments() },
  })

  const treeColumn = create('aside')
  treeColumn.style.minWidth = '0'
  treeColumn.style.padding = '12px 12px 16px'
  treeColumn.style.borderRight = '1px solid var(--ls-border)'
  treeColumn.append(create('div', ui.cardTitle, '配置分类'), treeHost)

  const split = create('div')
  split.style.display = 'grid'
  split.style.gridTemplateColumns = 'minmax(200px, 280px) minmax(0, 1fr)'
  split.style.alignItems = 'start'
  split.append(treeColumn, formHost)

  const card = dataCard({
    title: '配置分类',
    actions: [refreshButton, saveButton],
    status: state.element,
    body: split,
  })
  const toolbarTitle = card.querySelector(`.${ui.tableToolbarTitle}`)
  if (toolbarTitle) toolbarTitle.replaceWith(titleNode)

  root.replaceChildren(page({ showSummary: false, children: [card] }))

  function categories(): CatalogCategory[] {
    return groups.flatMap(group => group.categories || [])
  }

  function selectedCategory(): CatalogCategory | undefined {
    return categories().find(category => category.key === selected)
  }

  function setBusy(next: boolean): void {
    busy = next
    refreshButton.disabled = next
    saveButton.disabled = next || !canWrite || !dirty || !selectedCategory()?.fields.length
  }

  function markDirty(): void {
    if (!canWrite) return
    dirty = true
    saveButton.disabled = busy
  }

  function renderTree(): void {
    treeHost.replaceChildren()
    if (metadataError) {
      const error = create('div', ui.status)
      error.className = `${ui.status} ${ui.status}--danger`
      error.textContent = metadataError
      treeHost.append(error)
    }
    if (!groups.length) {
      treeHost.append(emptyState(metadataError ? '无法加载配置分类' : '暂无配置项'))
      return
    }
    const list = create('ul')
    list.style.cssText = 'list-style:none;margin:10px 0 0;padding:0;display:grid;gap:2px'
    for (const group of groups) {
      const item = create('li')
      const folded = collapsed.has(group.key)
      const toggle = create('button', buttonClass('ghost', 'sm'))
      toggle.type = 'button'
      toggle.style.cssText = 'display:flex;width:100%;justify-content:flex-start;gap:6px'
      toggle.append(create('span', undefined, folded ? '▸' : '▾'), create('span', undefined, group.label))
      toggle.addEventListener('click', () => {
        if (collapsed.has(group.key)) collapsed.delete(group.key)
        else collapsed.add(group.key)
        renderTree()
      })
      item.append(toggle)
      if (!folded) {
        const children = create('ul')
        children.style.cssText = 'list-style:none;margin:0 0 4px 10px;padding:0 0 0 8px;border-left:1px solid var(--ls-border);display:grid;gap:2px'
        for (const category of group.categories || []) {
          const child = create('li')
          const row = create('button')
          row.type = 'button'
          const active = category.key === selected
          row.className = buttonClass(active ? 'secondary' : 'ghost', 'sm')
          row.style.cssText = 'display:flex;width:100%;align-items:center;justify-content:space-between;gap:8px;text-align:left'
          row.append(create('span', undefined, category.label), badge({ label: String(category.fields.length) }))
          row.addEventListener('click', () => { void selectCategory(category.key) })
          child.append(row)
          children.append(child)
        }
        item.append(children)
      }
      list.append(item)
    }
    treeHost.append(list)
  }

  function renderForm(): void {
    const category = selectedCategory()
    titleNode.textContent = category ? category.label : '配置分类'
    if (!category) {
      formHost.replaceChildren(emptyState('请选择左侧的配置分类。'))
      dirty = false
      setBusy(busy)
      return
    }
    if (!category.fields.length) {
      formHost.replaceChildren(emptyState('该分类暂无可配置参数。'))
      dirty = false
      setBusy(busy)
      return
    }
    const document = documents.get(category.key)
    const values = categoryValues(category, document)
    const editor = create('form', ui.form)
    editor.style.paddingTop = '12px'
    for (const spec of category.fields) {
      if (spec.type === 'BOOL') {
        const wrapper = create('label', `${ui.field} ${ui.fieldWide}`)
        const row = create('span')
        row.style.cssText = 'display:flex;align-items:center;gap:8px;color:var(--ls-ink);font-size:13px'
        const input = create('input')
        input.type = 'checkbox'
        input.name = spec.key
        input.checked = values[spec.key] === 'true'
        input.disabled = !canWrite
        row.append(input, create('span', undefined, spec.label))
        wrapper.append(row)
        if (spec.help) wrapper.append(create('span', undefined, spec.help))
        editor.append(wrapper)
        continue
      }
      const built = field({
        name: spec.key,
        label: spec.label,
        kind: spec.type === 'SELECT' ? 'select' : spec.type === 'TEXTAREA' ? 'textarea' : 'input',
        type: spec.type === 'NUMBER' ? 'number' : 'text',
        wide: spec.type === 'TEXTAREA',
        rows: spec.type === 'TEXTAREA' ? 4 : undefined,
        placeholder: spec.placeholder,
        options: spec.options,
        value: values[spec.key],
        disabled: !canWrite,
      })
      if (spec.help) built.element.append(create('span', undefined, spec.help))
      editor.append(built.element)
    }
    editor.addEventListener('input', markDirty)
    editor.addEventListener('change', markDirty)
    editor.addEventListener('submit', event => {
      event.preventDefault()
      void save(editor, category)
    })
    formHost.replaceChildren(editor)
    dirty = false
    setBusy(false)
  }

  async function selectCategory(key: string): Promise<void> {
    selected = key
    renderTree()
    renderForm()
  }

  async function save(editor: HTMLFormElement, category: CatalogCategory): Promise<void> {
    if (!canWrite || busy) return
    const data = new FormData(editor)
    const value: Record<string, unknown> = {}
    for (const spec of category.fields) {
      if (spec.type === 'BOOL') {
        const input = editor.querySelector<HTMLInputElement>(`input[name="${spec.key}"]`)
        value[spec.key] = typedValue(spec, Boolean(input?.checked))
        continue
      }
      value[spec.key] = typedValue(spec, String(data.get(spec.key) ?? ''))
    }
    setBusy(true)
    try {
      await client.request(`${prefix}/${encodeURIComponent(category.key)}`, {
        method: 'PUT',
        body: JSON.stringify({ expectedVersion: documents.get(category.key)?.version || 0, value }),
      })
      await loadDocuments()
      state.set('已保存', 'success')
    } catch (error) {
      state.set(String(error), 'danger')
      setBusy(false)
    }
  }

  async function loadDocuments(): Promise<void> {
    setBusy(true)
    try {
      const items = await client.request<SettingDocument[]>(prefix)
      documents = new Map((items || []).map(item => [item.namespace, {
        ...item,
        value: item.value && typeof item.value === 'object' ? item.value as Record<string, unknown> : {},
      }]))
      if (!metadataError) state.set(selectedCategory() ? '' : `共 ${categories().length} 个配置分类`)
      renderForm()
    } catch (error) {
      documents = new Map()
      state.set(`加载失败：${String(error)}`, 'danger')
      renderForm()
    } finally {
      setBusy(false)
    }
  }

  try {
    groups = await client.request<CatalogGroup[]>(`${prefix}/catalog`)
  } catch (error) {
    metadataError = String(error)
  }
  if (!selected) selected = categories()[0]?.key || ''
  renderTree()
  if (metadataError && !groups.length) {
    titleNode.textContent = '配置分类'
    formHost.replaceChildren(emptyState('无法加载配置分类'))
    state.set(`无法加载配置分类：${metadataError}`, 'danger')
    return
  }
  await loadDocuments()
}
