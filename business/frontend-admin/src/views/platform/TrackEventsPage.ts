import type { HostContext, HostHttpClient } from '@liveshops/host-sdk'
import { hostFormModal } from '@liveshops/host-sdk'
import { badge, button, dataCard, notify, page, pagination, searchCard, searchForm, table } from '@liveshops/design-tokens'

interface TrackEvent {
  merchantId: number
  shopId: number
  surface: string
  eventId: string
  eventType: string
  eventName: string
  page: string
  component: string
  action: string
  bizType: string
  bizId: string
  sessionId: string
  anonymousId: string
  subject: string
  clientTs: number
  occurredAt: string
  receivedAt: string
  schemaVersion: number
  liveContext: unknown
  props: unknown
  state: unknown
  extra: unknown
  userAgent: string
  ip: string
  referer: string
  adTouchId: number
  clickIdType: string
  createdAt: string
}

const prefix = '/admin/platform/track-events'

export async function startTrackEvents(root: HTMLElement, client: HostHttpClient, _context: HostContext): Promise<void> {
  let pageNo = 1
  let pageSize = 20
  const rowsTable = table({
    columns: ['时间', '商户 / 店铺', '端', '事件', '类型', '主体', '操作'],
    empty: '没有匹配的上报事件。',
  })
  const pager = pagination({
    page: pageNo,
    pageSize,
    onPageChange: next => { pageNo = next; void load() },
    onPageSizeChange: next => { pageSize = next; pageNo = 1; void load() },
  })
  const filter = searchForm({
    fields: [
      { name: 'merchantId', label: '商户 ID', type: 'number', placeholder: '2001' },
      { name: 'shopId', label: '店铺 ID', type: 'number', placeholder: '3001' },
      { name: 'surface', label: '端', kind: 'select', options: [
        { value: '', label: '全部' },
        { value: 'shop', label: 'shop' },
        { value: 'merch', label: 'merch' },
        { value: 'admin', label: 'admin' },
        { value: 'live', label: 'live' },
      ] },
      { name: 'eventName', label: '事件名', placeholder: 'page_view' },
      { name: 'eventType', label: '事件类型', placeholder: 'page' },
      { name: 'subject', label: '主体', placeholder: 'subject' },
      { name: 'anonymousId', label: '匿名 ID', placeholder: 'anonymousId' },
      { name: 'occurredAt', label: '发生时间', kind: 'date-range' },
    ],
    onSearch: () => { pageNo = 1; void load() },
    onReset: () => { pageNo = 1; void load() },
  })

  async function load(): Promise<void> {
    const values = filter.values()
    const query = new URLSearchParams()
    query.set('page', String(pageNo))
    query.set('pageSize', String(pageSize))
    for (const name of ['merchantId', 'shopId', 'surface', 'eventName', 'eventType', 'subject', 'anonymousId']) {
      const value = values[name]?.trim()
      if (value) query.set(name, value)
    }
    const from = dayStartMs(values.occurredAtFrom || '')
    const to = dayEndMs(values.occurredAtTo || '')
    if (from) query.set('startMs', String(from))
    if (to) query.set('endMs', String(to))
    filter.setBusy(true)
    pager.setBusy(true)
    try {
      const payload = await client.request<{ items: TrackEvent[]; total: number }>(`${prefix}?${query.toString()}`)
      const items = payload.items || []
      rowsTable.setRows(items.map(item => [
        displayTime(item.occurredAt || item.clientTs),
        `${item.merchantId} / ${item.shopId}`,
        item.surface || '—',
        item.eventName || '—',
        badge({ label: item.eventType || '—', tone: 'neutral' }),
        item.subject || item.anonymousId || '—',
        button({ label: '详情', size: 'sm', onClick: () => openDetail(item) }),
      ]))
      pager.set({ page: pageNo, pageSize, total: payload.total || 0, itemCount: items.length })
    } catch (error) {
      rowsTable.setRows([])
      pager.set({ page: pageNo, pageSize, total: 0, itemCount: 0 })
      notify(String(error), 'danger')
    } finally {
      filter.setBusy(false)
      pager.setBusy(false)
    }
  }

  root.replaceChildren(page({
    showSummary: false,
    children: [
      searchCard(filter.element),
      dataCard({
        title: '上报事件',
        actions: [button({ label: '刷新', variant: 'secondary', onClick: () => void load() })],
        body: rowsTable.element,
        footer: pager.element,
      }),
    ],
  }))
  await load()
}

function openDetail(item: TrackEvent): void {
  const viewer = hostFormModal({
    title: `事件详情 · ${item.eventName}`,
    fields: [
      { name: 'eventId', label: '事件 ID', disabled: true, mono: true },
      { name: 'surface', label: '端', disabled: true },
      { name: 'merchantId', label: '商户 ID', disabled: true },
      { name: 'shopId', label: '店铺 ID', disabled: true },
      { name: 'eventName', label: '事件名', disabled: true },
      { name: 'eventType', label: '事件类型', disabled: true },
      { name: 'subject', label: '主体', disabled: true },
      { name: 'anonymousId', label: '匿名 ID', disabled: true },
      { name: 'sessionId', label: '会话 ID', disabled: true, wide: true },
      { name: 'page', label: '页面', disabled: true, wide: true },
      { name: 'component', label: '组件', disabled: true },
      { name: 'action', label: '动作', disabled: true },
      { name: 'bizType', label: '业务类型', disabled: true },
      { name: 'bizId', label: '业务 ID', disabled: true },
      { name: 'occurredAt', label: '发生时间', disabled: true },
      { name: 'receivedAt', label: '接收时间', disabled: true },
      { name: 'ip', label: 'IP', disabled: true },
      { name: 'adTouchId', label: '广告触点', disabled: true },
      { name: 'userAgent', label: 'User-Agent', kind: 'textarea', disabled: true, wide: true, rows: 2 },
      { name: 'referer', label: 'Referer', kind: 'textarea', disabled: true, wide: true, rows: 2 },
      { name: 'liveContext', label: 'liveContext', kind: 'textarea', disabled: true, wide: true, rows: 6, mono: true },
      { name: 'props', label: 'props', kind: 'textarea', disabled: true, wide: true, rows: 6, mono: true },
      { name: 'state', label: 'state', kind: 'textarea', disabled: true, wide: true, rows: 4, mono: true },
      { name: 'extra', label: 'extra', kind: 'textarea', disabled: true, wide: true, rows: 4, mono: true },
    ],
    submitLabel: '关闭',
    onSubmit: (_values, modal) => { modal.close() },
  })
  viewer.open({
    eventId: item.eventId,
    surface: item.surface,
    merchantId: String(item.merchantId),
    shopId: String(item.shopId),
    eventName: item.eventName,
    eventType: item.eventType,
    subject: item.subject,
    anonymousId: item.anonymousId,
    sessionId: item.sessionId,
    page: item.page,
    component: item.component,
    action: item.action,
    bizType: item.bizType,
    bizId: item.bizId,
    occurredAt: displayTime(item.occurredAt || item.clientTs),
    receivedAt: displayTime(item.receivedAt),
    ip: item.ip,
    adTouchId: item.adTouchId ? `${item.adTouchId} ${item.clickIdType || ''}`.trim() : '—',
    userAgent: item.userAgent,
    referer: item.referer,
    liveContext: formatJSON(item.liveContext),
    props: formatJSON(item.props),
    state: formatJSON(item.state),
    extra: formatJSON(item.extra),
  })
}

function displayTime(value: string | number): string {
  if (!value) return '—'
  const date = typeof value === 'number' ? new Date(value) : new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  return date.toLocaleString()
}

function dayStartMs(value: string): number {
  if (!value) return 0
  const date = new Date(`${value}T00:00:00`)
  return Number.isNaN(date.getTime()) ? 0 : date.getTime()
}

function dayEndMs(value: string): number {
  if (!value) return 0
  const date = new Date(`${value}T23:59:59.999`)
  return Number.isNaN(date.getTime()) ? 0 : date.getTime()
}

function formatJSON(value: unknown): string {
  if (value == null || value === '') return '{}'
  if (typeof value === 'string') {
    try { return JSON.stringify(JSON.parse(value), null, 2) } catch { return value }
  }
  try { return JSON.stringify(value, null, 2) } catch { return String(value) }
}
