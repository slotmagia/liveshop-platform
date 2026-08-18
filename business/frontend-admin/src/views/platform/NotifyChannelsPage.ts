import type { HostContext, HostHttpClient } from '@liveshop/host-sdk'
import { hostFormModal } from '@liveshop/host-sdk'
import { badge, button, create, dataCard, page, statusLine, table, ui } from '@liveshop/design-tokens'
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
  const state = statusLine()
  const canManage = context.permissions.includes('platform.notify-channel.manage')
  let kind: Kind = 'sms'
  const body = create('div')
  const tabs = create('div', ui.actions)
  const buttons: Record<Kind, HTMLElement> = {
    sms: button({ label: '短信', variant: 'secondary', onClick: () => void switchKind('sms') }),
    email: button({ label: '邮件', variant: 'secondary', onClick: () => void switchKind('email') }),
    'in-app': button({ label: '站内信', variant: 'secondary', onClick: () => void switchKind('in-app') }),
  }
  tabs.append(buttons.sms, buttons.email, buttons['in-app'])

  async function switchKind(next: Kind): Promise<void> {
    kind = next
    for (const [key, control] of Object.entries(buttons)) {
      control.setAttribute('data-active', key === next ? 'true' : 'false')
    }
    body.replaceChildren()
    if (next === 'sms') {
      await startSMS(body, client, context, { embedded: true })
      return
    }
    if (next === 'email') {
      await startEmail(body, client, context, { embedded: true })
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
    body.replaceChildren(card)
    state.set('正在加载站内信驱动…')
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
                  body: JSON.stringify({ commandKey: crypto.randomUUID(), expectedVersion: config.version || 0, enabled: !config.enabled }),
                }).then(() => { modal.close(); return renderInApp() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
              },
            })
            editor.open()
          },
        }) : '',
      ]])
      state.set(`站内信驱动 inbox · ${config.enabled ? '已启用' : '已停用'}`)
    } catch (error) {
      state.set(`加载站内信失败：${String(error)}`, 'danger')
    }
  }

  root.replaceChildren(page({
    showSummary: false,
    children: [
      dataCard({ title: '通知方式', body: create('p', undefined, '短信、邮件、站内信均为驱动。模板在「通知模板」，事件规则在菜单目录或「通知事件」。') }),
      tabs,
      state.element,
      body,
    ],
  }))
  await switchKind('sms')
}
