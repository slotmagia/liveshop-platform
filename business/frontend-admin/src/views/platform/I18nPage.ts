import type { HostContext, HostHttpClient } from '@liveshop/host-sdk'
import { hostFormModal, randomUUID } from '@liveshop/host-sdk'
import { badge, button, dataCard, notify, page, searchCard, searchForm, table } from '@liveshop/design-tokens'

interface DriverDefinition { code: string; name: string; description: string }
interface LocaleItem { code: string; label: string }
interface EntityItem { entityType: string; label: string; ownerModule: string; field: string }
interface ConfigView { provider: string; apiKeySet: boolean; version: number }
interface TextRow {
  entityId: string
  merchantId: number
  shopId: number
  source: string
  value: string
  status: string
  textSource: string
  stale: boolean
  version: number
}

const prefix = '/admin/platform/i18n'

export async function startI18n(root: HTMLElement, client: HostHttpClient, context: HostContext): Promise<void> {
  const canManage = context.permissions.includes('platform.i18n.manage')
  let entities: EntityItem[] = []
  let locales: LocaleItem[] = []
  try {
    const payload = await client.request<{ items: EntityItem[]; locales: string[] }>(`${prefix}/entities`)
    entities = payload.items || []
    const localePayload = await client.request<{ items: LocaleItem[] }>(`${prefix}/locales`)
    locales = localePayload.items || []
  } catch (error) {
    notify(`无法加载翻译目录：${String(error)}`, 'danger')
  }

  const filter = searchForm({
    fields: [
      { name: 'entityType', label: '内容类型', kind: 'select', options: entities.map(item => ({ value: item.entityType, label: item.label })) },
      { name: 'locale', label: '目标语言', kind: 'select', options: locales.map(item => ({ value: item.code, label: item.label })) },
    ],
    onSearch: () => void load(),
    onReset: () => void load(),
  })
  const rowsTable = table({ columns: ['范围', '源文', '译文', '状态', '操作'], empty: '没有可翻译条目。源文由业务模块变更事件投影，不从业务库直查。' })
  const card = dataCard({
    title: '翻译条目',
    actions: [
      button({ label: '刷新', variant: 'secondary', onClick: () => void load() }),
      ...(canManage ? [
        button({ label: '机器预填', variant: 'secondary', onClick: () => void fill() }),
        button({ label: '翻译配置', variant: 'secondary', onClick: () => void openConfig() }),
      ] : []),
    ],
    body: rowsTable.element,
  })

  async function load(): Promise<void> {
    const values = filter.values()
    const entityType = values.entityType?.trim() || entities[0]?.entityType || ''
    const locale = values.locale?.trim() || locales[0]?.code || ''
    if (!entityType || !locale) {
      rowsTable.setRows([])
      return
    }
    filter.setBusy(true)
    try {
      const payload = await client.request<{ items: TextRow[] }>(`${prefix}/texts?entityType=${encodeURIComponent(entityType)}&locale=${encodeURIComponent(locale)}`)
      const items = payload.items || []
      rowsTable.setRows(items.map(item => {
        const statusLabel = item.status === 'published' ? (item.stale ? '已发布（过期）' : '已发布') : item.status === 'machine' ? '机器草稿' : '未译'
        const tone = item.status === 'published' ? (item.stale ? 'warning' : 'success') : item.status === 'machine' ? 'warning' : 'neutral'
        const actions = canManage ? button({ label: '发布', size: 'sm', onClick: () => openPublish(item, entityType, locale) }) : '—'
        return [
          `${item.merchantId} / ${item.shopId} · ${item.entityId}`,
          item.source || '—',
          item.value || '—',
          badge({ label: statusLabel, tone }),
          actions,
        ]
      }))
    } catch (error) {
      rowsTable.setRows([])
      notify(String(error), 'danger')
    } finally {
      filter.setBusy(false)
    }
  }

  function openPublish(item: TextRow, entityType: string, locale: string): void {
    const editor = hostFormModal({
      title: '发布译文',
      fields: [
        { name: 'source', label: '源文', kind: 'textarea', disabled: true, wide: true, rows: 3 },
        { name: 'value', label: '译文', kind: 'textarea', required: true, wide: true, rows: 4 },
      ],
      submitLabel: '发布',
      onSubmit: (values, modal) => {
        const value = values.value.trim()
        if (!value) { modal.setError('请填写译文'); return }
        modal.setBusy(true)
        client.request(`${prefix}/texts/publish`, {
          method: 'POST',
          body: JSON.stringify({
            commandKey: randomUUID(), expectedVersion: item.version, entityType, entityId: item.entityId,
            locale, value, merchantId: item.merchantId, shopId: item.shopId,
          }),
        }).then(() => { modal.close(); notify('已发布', 'success'); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open({ source: item.source, value: item.value || '' })
  }

  async function fill(): Promise<void> {
    const values = filter.values()
    const entityType = values.entityType?.trim() || entities[0]?.entityType || ''
    const locale = values.locale?.trim() || locales[0]?.code || ''
    if (!entityType || !locale) { notify('请选择内容类型和语言', 'warning'); return }
    try {
      const result = await client.request<{ provider: string; filled: number; skipped: number }>(`${prefix}/texts/fill`, {
        method: 'POST',
        body: JSON.stringify({ commandKey: randomUUID(), entityType, locale }),
      })
      notify(`预填 ${result.filled} 条，跳过 ${result.skipped} 条（${result.provider}）`, 'success')
      await load()
    } catch (error) {
      notify(String(error), 'danger')
    }
  }

  async function openConfig(): Promise<void> {
    let current: ConfigView = { provider: 'noop', apiKeySet: false, version: 0 }
    let drivers: DriverDefinition[] = []
    try {
      current = await client.request<ConfigView>(`${prefix}/config`)
      const payload = await client.request<{ items: DriverDefinition[] }>(`${prefix}/drivers`)
      drivers = payload.items || []
    } catch (error) {
      notify(String(error), 'danger')
      return
    }
    const editor = hostFormModal({
      title: '机器翻译配置',
      fields: [
        { name: 'provider', label: '驱动', kind: 'select', required: true, options: drivers.map(item => ({ value: item.code, label: item.name })) },
        { name: 'apiKey', label: current.apiKeySet ? 'API Key（空=保留）' : 'API Key', type: 'password', wide: true },
        { name: 'apiKeyClear', label: '密钥', kind: 'select', options: [{ value: '', label: '保留已保存密钥' }, { value: 'clear', label: '清除已保存密钥' }] },
      ],
      submitLabel: '保存',
      onSubmit: (values, modal) => {
        modal.setBusy(true)
        client.request(`${prefix}/config`, {
          method: 'PUT',
          body: JSON.stringify({
            commandKey: randomUUID(), expectedVersion: current.version, provider: values.provider,
            apiKey: values.apiKey || '', apiKeyClear: values.apiKeyClear === 'clear',
          }),
        }).then(() => { modal.close(); notify('配置已保存', 'success') }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open({ provider: current.provider || 'noop', apiKey: '', apiKeyClear: '' })
  }

  root.replaceChildren(page({ showSummary: false, children: [searchCard(filter.element), card] }))
  void load()
}
