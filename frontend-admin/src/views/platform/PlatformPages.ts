import type { HostContext, HostHttpClient } from '@liveshop/host-sdk'
import { adminButtonClass, adminStatusClass, adminUI as ui } from '@liveshop/admin-ui'

interface ReleaseInfo { version: string; digest: string }
interface ModuleInfo { id: string; name: string; activeVersion?: string; releases: ReleaseInfo[] }
interface SettingDocument { namespace: string; value: unknown; version: number; updatedBy?: string; updatedAt?: string }
interface AuditEvent { id:string; occurredAt:string; actorSubject:string; action:string; resourceType:string; resourceId:string; result:string; details:unknown }
interface IdentityAccount { realm:string; appId:number; merchantId:number; subject:string; username:string; status:string; version:number; updatedAt:string }

function message(root: HTMLElement, text: string, error = false) {
  const target = root.querySelector<HTMLElement>('[data-status]')!
  target.textContent = text
  target.className = adminStatusClass(error ? 'danger' : 'neutral')
}

export async function startAccounts(root: HTMLElement, client: HostHttpClient) {
  root.innerHTML = `<main class="${ui.page}"><div class="${ui.pageHeader}"><div class="${ui.pageHeading}"><p class="${ui.eyebrow}">Identity & access</p><h1 class="${ui.title}">后台账户</h1><p class="${ui.description}">统一管理平台端与商户端登录账户；改密或停用会撤销全部刷新会话。</p></div><button data-refresh class="${adminButtonClass('secondary')}">刷新</button></div><div data-status class="${ui.status}"></div><section class="${ui.card}"><form data-account class="${ui.formGrid}"><select name="realm"><option>PLATFORM</option><option>MERCHANT</option></select><input name="subject" placeholder="稳定 Subject，例如 operator-1001" required><input name="username" placeholder="登录账号" required><select name="status"><option>ACTIVE</option><option>DISABLED</option></select><input name="expectedVersion" type="number" min="0" value="0" required><input name="password" type="password" minlength="12" placeholder="新建必填；更新留空则不改密"><button class="${ui.button}">保存账户</button></form></section><section class="${ui.card} ${ui.tableWrap}"><table class="${ui.table}"><thead><tr><th>身份域</th><th>Subject</th><th>账号</th><th>状态</th><th>版本</th><th>更新时间</th></tr></thead><tbody data-accounts></tbody></table></section></main>`
  const form = root.querySelector<HTMLFormElement>('[data-account]')!
  const load = async () => {
    const items = await client.request<IdentityAccount[]>('/admin/platform/identity/accounts')
    const body = root.querySelector<HTMLTableSectionElement>('[data-accounts]')!; body.replaceChildren()
    for (const item of items) {
      const row = body.insertRow()
      for (const value of [item.realm, item.subject, item.username, item.status, item.version, new Date(item.updatedAt).toLocaleString()]) row.insertCell().textContent = String(value)
      row.addEventListener('click', () => {
        ;(form.elements.namedItem('realm') as HTMLSelectElement).value = item.realm
        ;(form.elements.namedItem('subject') as HTMLInputElement).value = item.subject
        ;(form.elements.namedItem('username') as HTMLInputElement).value = item.username
        ;(form.elements.namedItem('status') as HTMLSelectElement).value = item.status
        ;(form.elements.namedItem('expectedVersion') as HTMLInputElement).value = String(item.version)
        ;(form.elements.namedItem('password') as HTMLInputElement).value = ''
      })
    }
    message(root, `已加载 ${items.length} 个后台账户`)
  }
  form.addEventListener('submit', event => {
    event.preventDefault()
    const data = new FormData(form)
    const realm = String(data.get('realm'))
    const subject = String(data.get('subject')).trim()
    const body = {expectedVersion:Number(data.get('expectedVersion')),username:String(data.get('username')).trim(),status:String(data.get('status')),password:String(data.get('password') ?? '')}
    void client.request(`/admin/platform/identity/accounts/${encodeURIComponent(realm)}/${encodeURIComponent(subject)}`, {method:'PUT',body:JSON.stringify(body)}).then(load).catch(error => message(root, String(error), true))
  })
  root.querySelector('[data-refresh]')!.addEventListener('click', () => void load().catch(error => message(root, String(error), true)))
  await load()
}

export async function startRegistry(root: HTMLElement, client: HostHttpClient) {
  root.innerHTML = `<main class="${ui.page}"><div class="${ui.pageHeader}"><div class="${ui.pageHeading}"><p class="${ui.eyebrow}">Module registry</p><h1 class="${ui.title}">模块管理</h1><p class="${ui.description}">查看不可变发布并切换唯一激活版本。</p></div><button data-refresh class="${adminButtonClass('secondary')}">刷新</button></div><div data-status class="${ui.status}"></div><div data-modules class="${ui.grid}"></div></main>`
  const load = async () => {
    const modules = await client.request<ModuleInfo[]>('/admin/platform/registry/modules')
    const grid = root.querySelector<HTMLElement>('[data-modules]')!
    grid.replaceChildren()
    for (const item of modules) {
      const card = document.createElement('section')
      card.className = ui.card
      const protectedModule = item.id === 'platform'
      card.innerHTML = `<div class="${ui.cardHeader}"><h2 class="${ui.cardTitle}" data-module-name></h2><code data-module-id></code></div><div class="${ui.cardBody}"><p>当前版本：<strong data-active-version></strong></p><select class="${ui.select}" data-version></select><div class="${ui.actions}" style="margin-top:12px"><button class="${ui.button}" data-activate>激活版本</button><button data-deactivate class="${adminButtonClass('secondary')}"></button></div></div>`
      card.querySelector<HTMLElement>('[data-module-name]')!.textContent = item.name
      card.querySelector<HTMLElement>('[data-module-id]')!.textContent = item.id
      card.querySelector<HTMLElement>('[data-active-version]')!.textContent = item.activeVersion || '未激活'
      const versionSelect = card.querySelector<HTMLSelectElement>('[data-version]')!
      for (const release of item.releases) {
        const option = document.createElement('option')
        option.value = release.version
        option.textContent = release.version
        option.selected = release.version === item.activeVersion
        versionSelect.append(option)
      }
      const deactivate = card.querySelector<HTMLButtonElement>('[data-deactivate]')!
      deactivate.disabled = !item.activeVersion || protectedModule
      deactivate.textContent = protectedModule ? '控制面不可停用' : '停用'
      card.querySelector<HTMLButtonElement>('[data-activate]')!.addEventListener('click', () => {
        const version = card.querySelector<HTMLSelectElement>('[data-version]')!.value
        void client.request(`/admin/platform/registry/modules/${encodeURIComponent(item.id)}/activate`, { method: 'POST', body: JSON.stringify({ version }) }).then(load).catch(error => message(root, String(error), true))
      })
      card.querySelector<HTMLButtonElement>('[data-deactivate]')!.addEventListener('click', () => {
        void client.request(`/admin/platform/registry/modules/${encodeURIComponent(item.id)}/activation`, { method: 'DELETE' }).then(load).catch(error => message(root, String(error), true))
      })
      grid.append(card)
    }
    message(root, `已加载 ${modules.length} 个模块`)
  }
  root.querySelector('[data-refresh]')!.addEventListener('click', () => void load().catch(error => message(root, String(error), true)))
  await load()
}

export async function startSettings(root: HTMLElement, client: HostHttpClient, context: HostContext) {
  const canWrite = context.permissions.includes('platform.settings.write')
  root.innerHTML = `<main class="${ui.page}"><div class="${ui.pageHeader}"><div class="${ui.pageHeading}"><p class="${ui.eyebrow}">Platform settings</p><h1 class="${ui.title}">平台配置</h1><p class="${ui.description}">仅保存非敏感 JSON；密钥和口令必须进入 Secret Manager。</p></div></div><div data-status class="${ui.status}"></div><section class="${ui.card}"><form data-form class="${ui.formGrid}"><input name="namespace" placeholder="namespace，例如 branding" required><input name="expectedVersion" type="number" min="0" value="0" required><textarea name="value" class="${ui.fieldWide} ${ui.code}" rows="12" required>{"name":"LiveShop","locale":"zh-CN"}</textarea><button class="${ui.button}"${canWrite ? '' : ' disabled'}>保存配置</button></form></section><div data-settings class="${ui.grid}" style="margin-top:12px"></div></main>`
  const load = async () => {
    const items = await client.request<SettingDocument[]>('/admin/platform/settings')
    const target = root.querySelector<HTMLElement>('[data-settings]')!
    target.replaceChildren()
    for (const item of items) {
      const card = document.createElement('section'); card.className = ui.card
      card.innerHTML = `<div class="${ui.cardHeader}"><h2 class="${ui.cardTitle}" data-namespace></h2><span class="${ui.badge}" data-version></span></div><div class="${ui.cardBody}"><p data-updated-by></p><pre data-value></pre></div>`
      card.querySelector<HTMLElement>('[data-namespace]')!.textContent = item.namespace
      card.querySelector<HTMLElement>('[data-version]')!.textContent = `版本 ${item.version}`
      card.querySelector<HTMLElement>('[data-updated-by]')!.textContent = item.updatedBy || ''
      card.querySelector<HTMLElement>('[data-value]')!.textContent = JSON.stringify(item.value, null, 2)
      card.addEventListener('click', () => {
        const form = root.querySelector<HTMLFormElement>('[data-form]')!
        ;(form.elements.namedItem('namespace') as HTMLInputElement).value = item.namespace
        ;(form.elements.namedItem('expectedVersion') as HTMLInputElement).value = String(item.version)
        ;(form.elements.namedItem('value') as HTMLTextAreaElement).value = JSON.stringify(item.value, null, 2)
      })
      target.append(card)
    }
    message(root, `已加载 ${items.length} 个配置命名空间`)
  }
  root.querySelector<HTMLFormElement>('[data-form]')!.addEventListener('submit', event => {
    event.preventDefault()
    const data = new FormData(event.currentTarget as HTMLFormElement)
    let value: unknown
    try { value = JSON.parse(String(data.get('value'))) } catch { message(root, '配置必须是合法 JSON', true); return }
    const namespace = String(data.get('namespace')).trim()
    void client.request(`/admin/platform/settings/${encodeURIComponent(namespace)}`, { method: 'PUT', body: JSON.stringify({ expectedVersion: Number(data.get('expectedVersion')), value }) }).then(load).catch(error => message(root, String(error), true))
  })
  await load()
}

export async function startAudit(root: HTMLElement, client: HostHttpClient) {
  root.innerHTML = `<main class="${ui.page}"><div class="${ui.pageHeader}"><div class="${ui.pageHeading}"><p class="${ui.eyebrow}">Audit trail</p><h1 class="${ui.title}">审计日志</h1><p class="${ui.description}">登录、刷新令牌滥用、IAM、模块生命周期与平台配置变更记录。</p></div><button data-refresh class="${adminButtonClass('secondary')}">刷新</button></div><div data-status class="${ui.status}"></div><section class="${ui.card} ${ui.tableWrap}"><table class="${ui.table}"><thead><tr><th>时间</th><th>操作者</th><th>动作</th><th>资源</th><th>结果</th></tr></thead><tbody data-events></tbody></table></section></main>`
  const load = async () => {
    const items = await client.request<AuditEvent[]>('/admin/platform/audit/events?limit=100')
    const body = root.querySelector<HTMLTableSectionElement>('[data-events]')!; body.replaceChildren()
    for (const item of items) {
      const row = body.insertRow()
      for (const value of [new Date(item.occurredAt).toLocaleString(), item.actorSubject, item.action, `${item.resourceType}:${item.resourceId}`, item.result]) row.insertCell().textContent = value
    }
    message(root, `已加载 ${items.length} 条审计事件`)
  }
  root.querySelector('[data-refresh]')!.addEventListener('click', () => void load().catch(error => message(root, String(error), true)))
  await load()
}
