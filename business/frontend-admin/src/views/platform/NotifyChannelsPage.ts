import type { HostContext, HostHttpClient } from '@liveshop/host-sdk'
import { hostFormModal, randomUUID } from '@liveshop/host-sdk'
import { badge, button, create, dataCard, notify, page, table, tabs } from '@liveshop/design-tokens'
import { startEmail } from './EmailPage'
import { startSMS } from './SmsPage'

type Kind = 'sms' | 'email' | 'in-app'

interface InAppConfig {
  driver: string
  enabled: boolean
  version: number
  updatedAt?: string
}

export async function startNotifyChannels(root: HTMLElement, client: HostHttpClient, context: HostContext): Promise<void> {
  const canManage = context.permissions.includes('platform.notify-channel.manage')
  const searchSlot = create('div')
  const dataSlot = create('div')
  const kindTabs = tabs({
    items: [
      { value: 'sms', label: '短信' },
      { value: 'email', label: '邮件' },
      { value: 'in-app', label: '站内信' },
    ],
    value: 'sms',
    ariaLabel: '通知方式',
    onChange: value => { void switchKind(value as Kind) },
  })

  async function switchKind(next: Kind): Promise<void> {
    kindTabs.set(next)
    searchSlot.hidden = next !== 'sms'
    if (next !== 'sms') searchSlot.replaceChildren()
    dataSlot.replaceChildren()
    if (next === 'sms') {
      await startSMS(dataSlot, client, context, { mounts: { search: searchSlot, data: dataSlot } })
      return
    }
    if (next === 'email') {
      await startEmail(dataSlot, client, context, { mounts: { data: dataSlot } })
      return
    }
    await renderInApp()
  }

  async function renderInApp(): Promise<void> {
    const rows = table({ columns: ['驱动', '状态', '版本', '操作'], empty: '尚未初始化站内信驱动' })
    const card = dataCard({
      title: '站内信 inbox',
      actions: [button({ label: '刷新', variant: 'secondary', onClick: () => void renderInApp() })],
      body: rows.element,
    })
    dataSlot.replaceChildren(card)
    try {
      const config = await client.request<InAppConfig>('/admin/platform/notify-channels/in-app')
      rows.setRows([[
        'inbox（写入站内信，无外部密钥）',
        badge({ label: config.enabled ? '已启用' : '已停用', tone: config.enabled ? 'success' : 'neutral' }),
        `v${config.version || 0}`,
        canManage ? button({
          label: config.enabled ? '停用' : '启用',
          size: 'sm',
          variant: 'secondary',
          onClick: () => {
            const editor = hostFormModal({
              title: config.enabled ? '停用站内信' : '启用站内信',
              fields: [{ name: 'confirm', label: '确认', kind: 'select', required: true, options: [{ value: 'yes', label: '确认变更' }] }],
              submitLabel: '保存',
              onSubmit: (values, modal) => {
                if (values.confirm !== 'yes') { modal.setError('请确认变更。'); return }
                modal.setBusy(true)
                client.request('/admin/platform/notify-channels/in-app', {
                  method: 'PUT',
                  body: JSON.stringify({ commandKey: randomUUID(), expectedVersion: config.version || 0, enabled: !config.enabled }),
                }).then(() => { modal.close(); return renderInApp() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
              },
            })
            editor.open()
          },
        }) : '',
      ]])
    } catch (error) {
      rows.setRows([])
      notify(`加载站内信失败：${String(error)}`, 'danger')
    }
  }

  root.replaceChildren(page({
    showSummary: false,
    children: [searchSlot, kindTabs.element, dataSlot],
  }))
  await switchKind('sms')
}
