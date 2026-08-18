import type { HostContext, HostHttpClient } from '@liveshop/host-sdk'
import { hostFormModal } from '@liveshop/host-sdk'
import { badge, button, create, dataCard, page, searchCard, searchForm, statusLine, table, ui } from '@liveshop/design-tokens'

interface NotifyTemplate {
  code: string
  channel: string
  textTemplate?: string
  subject?: string
  bodyHtml?: string
  title?: string
  body?: string
  variables: string[]
  lifecycle: string
  version: number
  updatedAt?: string
}

const prefix = '/admin/platform/notify-templates'
const channels = [
  { value: 'SMS', label: 'SMS' },
  { value: 'EMAIL', label: 'EMAIL' },
  { value: 'IN_APP', label: 'IN_APP' },
]

function actions(...children: Node[]): HTMLElement {
  const node = create('div', ui.actions)
  node.style.flexDirection = 'column'
  node.style.alignItems = 'stretch'
  node.append(...children)
  return node
}

function details(lines: Array<[string, string | number | Node]>): HTMLElement {
  const node = create('small')
  node.style.fontSize = '12px'
  node.style.lineHeight = '1.55'
  for (const [index, [label, value]] of lines.entries()) {
    node.append(create('strong', undefined, `${label}：`), value instanceof Node ? value : document.createTextNode(String(value || '—')))
    if (index < lines.length - 1) node.append(document.createElement('br'))
  }
  return node
}

export async function startNotifyTemplates(root: HTMLElement, client: HostHttpClient, context: HostContext): Promise<void> {
  const state = statusLine()
  const canManage = context.permissions.includes('platform.notify-template.manage')
  const rows = table({ columns: ['编码', '渠道', '内容摘要', '变量', '状态', '操作'], empty: '还没有通知模板' })
  let items: NotifyTemplate[] = []
  const filter = searchForm({
    fields: [
      { name: 'channel', label: '渠道', kind: 'select', options: [{ value: '', label: '全部渠道' }, ...channels] },
      { name: 'keyword', label: '关键字', placeholder: 'code / title / subject' },
    ],
    onSearch: () => void load(),
    onReset: () => void load(),
  })

  function summary(item: NotifyTemplate): string {
    if (item.channel === 'SMS') return item.textTemplate || '—'
    if (item.channel === 'EMAIL') return item.subject || '—'
    return item.title || '—'
  }

  function render(): void {
    rows.setRows(items.map(item => [
      details([['编码', item.code], ['版本', `v${item.version}`]]),
      item.channel,
      summary(item),
      (item.variables || []).map(name => `{{${name}}}`).join(' ') || '—',
      badge({ label: item.lifecycle === 'RETIRED' ? '已退役' : '使用中', tone: item.lifecycle === 'RETIRED' ? 'neutral' : 'success' }),
      actions(
        ...(canManage && item.lifecycle !== 'RETIRED' ? [
          button({ label: '编辑', size: 'sm', onClick: () => openEditor(item) }),
          button({ label: '退役', size: 'sm', variant: 'danger', onClick: () => retire(item) }),
        ] : []),
      ),
    ]))
    state.set(`模板 ${items.length} 份`)
  }

  async function load(): Promise<void> {
    filter.setBusy(true)
    state.set('正在加载模板库…')
    try {
      const values = filter.values()
      const query = new URLSearchParams()
      if (values.channel?.trim()) query.set('channel', values.channel.trim())
      if (values.keyword?.trim()) query.set('keyword', values.keyword.trim())
      items = await client.request<NotifyTemplate[]>(`${prefix}${query.size ? `?${query}` : ''}`)
      render()
    } catch (error) {
      items = []
      rows.setRows([])
      state.set(`模板加载失败：${String(error)}`, 'danger')
    } finally { filter.setBusy(false) }
  }

  function openEditor(current?: NotifyTemplate): void {
    const channel = current?.channel || 'SMS'
    const editor = hostFormModal({
      title: current ? `编辑模板 · ${current.code}` : '新增模板',
      fields: [
        { name: 'code', label: '编码', required: true, disabled: Boolean(current), placeholder: 'identity.auth.otp.requested.sms' },
        { name: 'channel', label: '渠道', kind: 'select', required: true, disabled: Boolean(current), options: channels },
        { name: 'textTemplate', label: 'SMS 正文（{{variable}}）', kind: 'textarea', wide: true, rows: 4 },
        { name: 'subject', label: 'EMAIL 主题', wide: true },
        { name: 'bodyHtml', label: 'EMAIL HTML', kind: 'textarea', wide: true, rows: 6 },
        { name: 'title', label: 'IN_APP 标题', wide: true },
        { name: 'body', label: 'IN_APP 正文', kind: 'textarea', wide: true, rows: 4 },
      ],
      submitLabel: '保存',
      onSubmit: (form, modal) => {
        const code = (form.code || '').trim().toLowerCase()
        if (!/^[a-z][a-z0-9._-]{1,63}$/.test(code)) { modal.setError('编码须为小写字母开头的 2–64 位 [a-z0-9._-]。'); return }
        modal.setBusy(true)
        client.request(`${prefix}/${encodeURIComponent(code)}`, {
          method: 'PUT',
          body: JSON.stringify({
            commandKey: crypto.randomUUID(), expectedVersion: current?.version || 0, channel: form.channel,
            textTemplate: form.textTemplate || '', subject: form.subject || '', bodyHtml: form.bodyHtml || '',
            title: form.title || '', body: form.body || '',
          }),
        }).then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open({
      code: current?.code || '',
      channel,
      textTemplate: current?.textTemplate || '',
      subject: current?.subject || '',
      bodyHtml: current?.bodyHtml || '',
      title: current?.title || '',
      body: current?.body || '',
    })
  }

  function retire(item: NotifyTemplate): void {
    const editor = hostFormModal({
      title: `退役 ${item.code}`,
      fields: [{ name: 'confirm', label: '确认', kind: 'select', required: true, options: [{ value: item.code, label: item.code }] }],
      submitLabel: '退役',
      onSubmit: (values, modal) => {
        if (values.confirm !== item.code) { modal.setError('请选择确认项。'); return }
        modal.setBusy(true)
        client.request(`${prefix}/${encodeURIComponent(item.code)}/retire`, {
          method: 'POST',
          body: JSON.stringify({ commandKey: crypto.randomUUID(), expectedVersion: item.version }),
        }).then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open()
  }

  root.replaceChildren(page({
    showSummary: false,
    children: [
      searchCard(filter.element),
      dataCard({
        title: '通知模板',
        actions: [
          button({ label: '刷新', variant: 'secondary', onClick: () => void load() }),
          ...(canManage ? [button({ label: '新增模板', onClick: () => openEditor() })] : []),
        ],
        status: state.element,
        body: rows.element,
      }),
    ],
  }))
  await load()
}
