import type { HostContext, HostHttpClient, HostModalField } from '@liveshop/host-sdk'
import { hostFormModal, randomUUID } from '@liveshop/host-sdk'
import { badge, button, create, dataCard, page, searchCard, searchForm, statusLine, table, ui } from '@liveshop/design-tokens'

interface NotifyChannelPolicy {
  enabled: boolean
  templateCode?: string
}

interface NotifyEvent {
  eventKey: string
  moduleId: string
  moduleName: string
  operationId: string
  title: string
  variables: string[]
  allowedChannels: string[]
  defaultDispatch: string
  dispatchMode: string
  delaySeconds: number
  channels: Record<string, NotifyChannelPolicy>
  policyVersion: number
  updatedAt: string
}

interface NotifyTemplate {
  code: string
  channel: string
  lifecycle: string
  variables?: string[]
}

interface NotifyDelivery {
  deliveryId: string
  deliveryKey: string
  channel: string
  status: string
  recipient: string
  lastError?: string
  attemptCount: number
  createdAt: string
  updatedAt: string
}

const prefix = '/admin/platform/notify-events'
const modes = [
  { value: 'SYNC', label: '同步' },
  { value: 'ASYNC', label: '异步' },
  { value: 'SCHEDULED', label: '定时' },
]

function actions(...children: Node[]): HTMLElement {
  const node = create('div', ui.actions)
  node.style.flexDirection = 'column'
  node.style.alignItems = 'stretch'
  node.style.flexWrap = 'nowrap'
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

function displayTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
  }).format(date)
}

function modeLabel(mode: string): string {
  return modes.find(item => item.value === mode)?.label || mode || '—'
}

function channelBadge(policy?: NotifyChannelPolicy): HTMLElement {
  return badge({ label: policy?.enabled ? (policy.templateCode || '开') : '关', tone: policy?.enabled ? 'success' : 'neutral' })
}

export async function startNotifyEvents(root: HTMLElement, client: HostHttpClient, context: HostContext): Promise<void> {
  const state = statusLine()
  const canManage = context.permissions.includes('platform.notify-event.manage')
  const events = table({
    columns: ['模块', '接口', '事件', '投递模式', 'SMS', 'EMAIL', 'IN_APP', '更新时间', '操作'],
    empty: '还没有已激活模块声明的通知事件',
  })
  let items: NotifyEvent[] = []

  const filter = searchForm({
    fields: [
      { name: 'module', label: '模块', placeholder: 'identity / trade' },
      { name: 'channel', label: '已开启渠道', kind: 'select', options: [
        { value: '', label: '全部渠道' },
        { value: 'SMS', label: 'SMS' },
        { value: 'EMAIL', label: 'EMAIL' },
        { value: 'IN_APP', label: 'IN_APP' },
      ] },
      { name: 'keyword', label: '关键字', placeholder: 'title / eventKey / operationId' },
    ],
    onSearch: () => void load(),
    onReset: () => void load(),
  })

  function renderRows(): void {
    events.setRows(items.map(item => [
      details([['模块', item.moduleName || item.moduleId], ['ID', item.moduleId]]),
      details([['接口', item.operationId]]),
      details([['名称', item.title], ['事件', item.eventKey]]),
      details([['现行', modeLabel(item.dispatchMode)], ['声明默认', modeLabel(item.defaultDispatch)], ...(item.dispatchMode === 'SCHEDULED' ? [['延迟', `${item.delaySeconds} 秒`] as [string, string]] : [])]),
      channelBadge(item.channels?.SMS),
      channelBadge(item.channels?.EMAIL),
      channelBadge(item.channels?.IN_APP),
      displayTime(item.updatedAt),
      actions(
        ...(canManage ? [button({ label: '渠道配置', size: 'sm', onClick: () => void openPolicy(item) })] : []),
        button({ label: '投递记录', size: 'sm', variant: 'secondary', onClick: () => void openDeliveries(item) }),
      ),
    ]))
    state.set(`通知事件 ${items.length} 个`)
  }

  async function load(): Promise<void> {
    filter.setBusy(true)
    state.set('正在加载通知事件目录…')
    try {
      const values = filter.values()
      const query = new URLSearchParams()
      for (const key of ['module', 'channel', 'keyword']) if (values[key]?.trim()) query.set(key, values[key].trim())
      items = await client.request<NotifyEvent[]>(`${prefix}${query.size ? `?${query}` : ''}`)
      renderRows()
    } catch (error) {
      items = []
      events.setRows([])
      state.set(`通知事件加载失败：${String(error)}`, 'danger')
    } finally { filter.setBusy(false) }
  }

  async function openPolicy(item: NotifyEvent): Promise<void> {
    let templates: NotifyTemplate[] = []
    try {
      templates = await client.request<NotifyTemplate[]>('/admin/platform/notify-templates')
    } catch (error) {
      state.set(`加载模板库失败：${String(error)}`, 'danger')
      return
    }
    const eventVariables = item.variables || []
    const compatible = (channel: string) => templates.filter(template => (
      template.channel === channel
      && template.lifecycle !== 'RETIRED'
      && (template.variables || []).every(name => eventVariables.includes(name))
    ))
    const fields: HostModalField[] = [
      { name: 'eventVariables', label: '事件变量（模板占位符必须是其子集）', disabled: true, wide: true },
      { name: 'dispatchMode', label: '投递模式', kind: 'select', required: true, options: modes },
      { name: 'delaySeconds', label: '定时延迟（秒）', type: 'number', placeholder: '仅 SCHEDULED' },
      ...item.allowedChannels.flatMap(channel => {
        const options = [
          { value: '', label: '不选择模板' },
          ...compatible(channel).map(template => ({ value: template.code, label: template.code })),
        ]
        return [
          { name: `channel_${channel}`, label: `${channel} 渠道`, kind: 'select' as const, required: true, options: [{ value: 'true', label: '开启' }, { value: 'false', label: '关闭' }] },
          { name: `template_${channel}`, label: `${channel} 模板`, kind: 'select' as const, options },
        ]
      }),
    ]
    const values: Record<string, string> = {
      eventVariables: eventVariables.map(name => `{{${name}}}`).join(' ') || '（无）',
      dispatchMode: item.dispatchMode || item.defaultDispatch,
      delaySeconds: String(item.delaySeconds || 0),
    }
    for (const channel of item.allowedChannels) {
      values[`channel_${channel}`] = String(Boolean(item.channels?.[channel]?.enabled))
      values[`template_${channel}`] = item.channels?.[channel]?.templateCode || ''
    }
    const editor = hostFormModal({
      title: `渠道策略 · ${item.title}`,
      fields,
      submitLabel: '保存',
      onSubmit: (form, modal) => {
        const dispatchMode = form.dispatchMode
        const delaySeconds = Number(form.delaySeconds || 0)
        if (dispatchMode === 'SCHEDULED' && (!Number.isInteger(delaySeconds) || delaySeconds < 0 || delaySeconds > 2592000)) {
          modal.setError('定时延迟须为 0 到 2592000 的整数秒。'); return
        }
        const channels: Record<string, NotifyChannelPolicy> = {}
        for (const channel of item.allowedChannels) {
          const enabled = form[`channel_${channel}`] === 'true'
          const templateCode = (form[`template_${channel}`] || '').trim()
          if (enabled && !templateCode) { modal.setError(`${channel} 开启时必须选择模板。`); return }
          channels[channel] = { enabled, templateCode }
        }
        modal.setBusy(true)
        client.request(`${prefix}/${encodeURIComponent(item.eventKey)}/policy`, {
          method: 'PUT',
          body: JSON.stringify({
            commandKey: randomUUID(), expectedVersion: item.policyVersion, dispatchMode,
            delaySeconds: dispatchMode === 'SCHEDULED' ? delaySeconds : 0, channels,
          }),
        }).then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open(values)
  }

  async function openDeliveries(item: NotifyEvent): Promise<void> {
    let rows: NotifyDelivery[] = []
    try {
      rows = await client.request<NotifyDelivery[]>(`${prefix}/${encodeURIComponent(item.eventKey)}/deliveries`)
    } catch (error) {
      state.set(`加载投递记录失败：${String(error)}`, 'danger')
      return
    }
    const lines = rows.length
      ? rows.map(row => `${displayTime(row.createdAt)}  ${row.channel}  ${row.status}  ${row.deliveryKey}  ${row.recipient || ''}  ${row.lastError || ''}`).join('\n')
      : '暂无投递记录'
    const viewer = hostFormModal({
      title: `投递记录 · ${item.title}`,
      fields: [{ name: 'records', label: '只读证据', kind: 'textarea', disabled: true, wide: true, rows: 12 }],
      submitLabel: '关闭',
      onSubmit: (_values, modal) => { modal.close() },
    })
    viewer.open({ records: lines })
  }

  root.replaceChildren(page({
    showSummary: false,
    children: [
      searchCard(filter.element),
      dataCard({
        title: '通知事件目录',
        actions: [button({ label: '刷新', variant: 'secondary', onClick: () => void load() })],
        status: state.element,
        body: events.element,
      }),
    ],
  }))
  await load()
}
