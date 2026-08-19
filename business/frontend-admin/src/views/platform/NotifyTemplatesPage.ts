import type { HostContext, HostHttpClient, HostModalField } from '@liveshops/host-sdk'
import { hostFormModal, randomUUID } from '@liveshops/host-sdk'
import { badge, button, create, dataCard, page, searchCard, searchForm, statusLine, table, ui } from '@liveshops/design-tokens'

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

  function channelSelector(disabled: boolean): HostModalField {
    return { name: 'channel', label: '渠道', kind: 'select', required: true, disabled, options: channels }
  }

  function editorFields(channel: string, current?: NotifyTemplate, carried: Record<string, string> = {}): {
    fields: HostModalField[]
    values: Record<string, string>
  } {
    const fields: HostModalField[] = [
      { name: 'code', label: '编码', required: true, disabled: Boolean(current), placeholder: 'identity.auth.otp.requested.sms' },
      channelSelector(Boolean(current)),
    ]
    const values: Record<string, string> = {
      code: carried.code || current?.code || '',
      channel,
      textTemplate: carried.textTemplate || current?.textTemplate || '',
      subject: carried.subject || current?.subject || '',
      bodyHtml: carried.bodyHtml || current?.bodyHtml || '',
      title: carried.title || current?.title || '',
      body: carried.body || current?.body || '',
    }
    if (channel === 'SMS') {
      fields.push({
        name: 'textTemplate', required: true, wide: true, kind: 'textarea', rows: 4,
        label: 'SMS 正文（占位符写成 {{code}}，必须是事件已声明变量）',
        placeholder: '您的验证码是 {{code}}，{{ttlSeconds}} 秒内有效',
      })
    } else if (channel === 'EMAIL') {
      fields.push(
        { name: 'subject', label: 'EMAIL 主题（可用 {{code}}）', required: true, wide: true },
        { name: 'bodyHtml', label: 'EMAIL HTML', required: true, kind: 'textarea', wide: true, rows: 6 },
      )
    } else if (channel === 'IN_APP') {
      fields.push(
        { name: 'title', label: 'IN_APP 标题', required: true, wide: true },
        { name: 'body', label: 'IN_APP 正文', required: true, kind: 'textarea', wide: true, rows: 4 },
      )
    }
    return { fields, values }
  }

  function openEditor(current?: NotifyTemplate): void {
    const draft: Record<string, string> = {
      code: current?.code || '',
      channel: current?.channel || 'SMS',
      textTemplate: current?.textTemplate || '',
      subject: current?.subject || '',
      bodyHtml: current?.bodyHtml || '',
      title: current?.title || '',
      body: current?.body || '',
    }
    const remember = (form: Record<string, string>) => {
      draft.code = form.code ?? draft.code
      draft.channel = form.channel || draft.channel
      if ('textTemplate' in form) draft.textTemplate = form.textTemplate
      if ('subject' in form) draft.subject = form.subject
      if ('bodyHtml' in form) draft.bodyHtml = form.bodyHtml
      if ('title' in form) draft.title = form.title
      if ('body' in form) draft.body = form.body
    }
    const editor = hostFormModal({
      title: current ? `编辑模板 · ${current.code}` : '新增模板',
      fields: editorFields(draft.channel, current, draft).fields,
      submitLabel: '保存',
      onChange: (form, field, modal) => {
        if (field !== 'channel') return
        remember(form)
        const generated = editorFields(form.channel, current, draft)
        modal.setFields(generated.fields, generated.values, current ? `编辑模板 · ${current.code}` : `新增模板 · ${form.channel}`)
      },
      onSubmit: (form, modal) => {
        remember(form)
        const code = (draft.code || '').trim().toLowerCase()
        if (!/^[a-z][a-z0-9._-]{1,63}$/.test(code)) { modal.setError('编码须为小写字母开头的 2–64 位 [a-z0-9._-]。'); return }
        if (draft.channel === 'SMS' && !draft.textTemplate.trim()) { modal.setError('请填写 SMS 正文。'); return }
        if (draft.channel === 'EMAIL' && (!draft.subject.trim() || !draft.bodyHtml.trim())) { modal.setError('请填写 EMAIL 主题和 HTML。'); return }
        if (draft.channel === 'IN_APP' && (!draft.title.trim() || !draft.body.trim())) { modal.setError('请填写 IN_APP 标题和正文。'); return }
        modal.setBusy(true)
        client.request(`${prefix}/${encodeURIComponent(code)}`, {
          method: 'PUT',
          body: JSON.stringify({
            commandKey: randomUUID(), expectedVersion: current?.version || 0, channel: draft.channel,
            textTemplate: draft.channel === 'SMS' ? draft.textTemplate : '',
            subject: draft.channel === 'EMAIL' ? draft.subject : '',
            bodyHtml: draft.channel === 'EMAIL' ? draft.bodyHtml : '',
            title: draft.channel === 'IN_APP' ? draft.title : '',
            body: draft.channel === 'IN_APP' ? draft.body : '',
          }),
        }).then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    const generated = editorFields(draft.channel, current, draft)
    editor.open(generated.values, current ? `编辑模板 · ${current.code}` : `新增模板 · ${draft.channel}`)
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
          body: JSON.stringify({ commandKey: randomUUID(), expectedVersion: item.version }),
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
