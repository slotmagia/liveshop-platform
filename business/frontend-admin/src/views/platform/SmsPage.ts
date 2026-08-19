import type { HostContext, HostHttpClient, HostModalField, HostModalTreeNode } from '@liveshops/host-sdk'
import { hostFormModal, randomUUID } from '@liveshops/host-sdk'
import { badge, button, create, dataCard, page, searchCard, searchForm, statusLine, table, tabs, ui } from '@liveshops/design-tokens'

export interface NotifyChannelMounts {
  search?: HTMLElement
  data: HTMLElement
}

type Lifecycle = 'ACTIVE' | 'RETIRED'
type Tab = 'channels' | 'regions' | 'grants'

interface DriverField {
  key: string
  label: string
  type: 'TEXT' | 'PASSWORD'
  required: boolean
  secret: boolean
  placeholder?: string
  help?: string
}

interface DriverDefinition {
  code: string
  name: string
  description: string
  fields: DriverField[]
}

interface SMSChannel {
  id: number
  code: string
  name: string
  driver: string
  region: string
  priority: number
  enabled: boolean
  lifecycle: Lifecycle
  publicConfig: Record<string, string>
  secretMasks: Record<string, string>
  credentialKeyId?: string
  version: number
  createdAt: string
  updatedAt: string
}

interface SMSRegion {
  id: number
  code: string
  dialCode: string
  name: string
  iso2: string
  emoji: string
  sort: number
  enabled: boolean
  lifecycle: Lifecycle
  version: number
  createdAt: string
  updatedAt: string
}

interface MerchantGrant {
  id: number
  merchantId: number
  shopId: number
  dialCodes: string[]
  unrestricted: boolean
  version: number
  createdAt?: string
  updatedAt?: string
}

const prefix = '/admin/platform/sms'
const channelCodePattern = /^[a-z0-9][a-z0-9_-]{0,31}$/
const dialCodePattern = /^\+[1-9][0-9]{0,6}$/

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

function selectedCodes(value: string): string[] {
  return [...new Set(value.split(/[\s,]+/).map(code => code.trim()).filter(Boolean))]
}

export async function startSMS(root: HTMLElement, client: HostHttpClient, context: HostContext, options?: { embedded?: boolean; mounts?: NotifyChannelMounts }): Promise<void> {
  const state = statusLine()
  const canManage = context.permissions.includes('platform.sms.manage')
  let tab: Tab = 'channels'
  let metadataError = ''
  let driverDefinitions: DriverDefinition[] = []
  try {
    driverDefinitions = await client.request<DriverDefinition[]>(`${prefix}/drivers`)
  } catch (error) {
    metadataError = String(error)
  }
  const driverByCode = new Map(driverDefinitions.map(definition => [definition.code, definition]))
  const driverLabel = (driver: string): string => driverByCode.get(driver)?.name || driver

  const channelFilter = searchForm({
    fields: [
      { name: 'keyword', label: '编码 / 名称', placeholder: 'aliyun-cn / 中国通道' },
      { name: 'driver', label: '驱动', kind: 'select', options: [{ value: '', label: '全部驱动' }, ...driverDefinitions.map(definition => ({ value: definition.code, label: definition.name }))] },
      { name: 'lifecycle', label: '生命周期', kind: 'select', options: [{ value: '', label: '全部' }, { value: 'ACTIVE', label: '使用中' }, { value: 'RETIRED', label: '已退役' }] },
    ],
    onSearch: () => void load(),
    onReset: () => void load(),
  })
  const regionFilter = searchForm({
    fields: [
      { name: 'keyword', label: '区号 / 名称 / ISO', placeholder: '+86 / 中国 / CN' },
      { name: 'lifecycle', label: '生命周期', kind: 'select', options: [{ value: '', label: '全部' }, { value: 'ACTIVE', label: '使用中' }, { value: 'RETIRED', label: '已退役' }] },
    ],
    onSearch: () => void load(),
    onReset: () => void load(),
  })
  const grantFilter = searchForm({
    fields: [
      { name: 'merchantId', label: '商户 ID', placeholder: '2001', type: 'number' },
      { name: 'shopId', label: '店铺 ID', placeholder: '3001', type: 'number' },
    ],
    onSearch: () => void loadGrant(),
    onReset: () => { grant = null; renderGrant(); state.set('输入 merchant_id 与 shop_id 后查询开通范围。') },
  })

  const channelsTable = table({ columns: ['通道', '驱动 / 路由', '配置状态', '时间', '操作'], empty: '还没有短信通道' })
  const regionsTable = table({ columns: ['区域', '标识', '排序 / 状态', '时间', '操作'], empty: '还没有短信区域' })
  const grantsTable = table({ columns: ['租户', '开通范围', '版本', '操作'], empty: '输入商户 ID 与店铺 ID 后查询' })

  let channels: SMSChannel[] = []
  let regions: SMSRegion[] = []
  let grant: MerchantGrant | null = null

  const channelSearch = searchCard(channelFilter.element)
  const regionSearch = searchCard(regionFilter.element)
  const grantSearch = searchCard(grantFilter.element)
  const searchHost = options?.mounts?.search ?? create('div')
  const dataHost = options?.mounts?.data ?? create('div')
  let dataCardNode: HTMLElement | undefined

  const tableTabs = tabs({
    items: [
      { value: 'channels', label: '通道' },
      { value: 'regions', label: '区域' },
      { value: 'grants', label: '商户开通' },
    ],
    value: 'channels',
    ariaLabel: '短信表格',
    onChange: value => switchTab(value as Tab),
  })

  function currentTable(): HTMLElement {
    if (tab === 'regions') return regionsTable.element
    if (tab === 'grants') return grantsTable.element
    return channelsTable.element
  }

  function currentActions(): HTMLElement[] {
    const refresh = button({
      label: '刷新',
      variant: 'secondary',
      onClick: () => { if (tab === 'grants') void loadGrant(); else void load() },
    })
    if (!canManage) return [refresh]
    if (tab === 'regions') return [refresh, button({ label: '新增区域', onClick: () => openRegion() })]
    if (tab === 'grants') return [refresh, button({ label: '保存开通范围', onClick: () => openGrant() })]
    return [refresh, button({ label: '新增通道', onClick: () => openChannel() })]
  }

  function syncSearch(): void {
    searchHost.replaceChildren(tab === 'regions' ? regionSearch : tab === 'grants' ? grantSearch : channelSearch)
  }

  function syncCard(): void {
    const next = dataCard({
      title: tab === 'regions' ? '短信区域' : tab === 'grants' ? '商户区域开通' : '短信通道',
      actions: currentActions(),
      body: currentTable(),
    })
    next.querySelector(`.${ui.tableToolbarTitle}`)?.replaceWith(tableTabs.element)
    if (dataCardNode) dataCardNode.replaceWith(next)
    else dataHost.replaceChildren(next)
    dataCardNode = next
  }

  function switchTab(next: Tab): void {
    tab = next
    tableTabs.set(next)
    syncSearch()
    syncCard()
    if (next === 'grants' && !grant) return
    void load()
  }

  function lifecycleBadge(item: { lifecycle: Lifecycle; enabled: boolean }): HTMLElement {
    return badge({
      label: item.lifecycle === 'RETIRED' ? '已退役' : item.enabled ? '已启用' : '已停用',
      tone: item.lifecycle === 'RETIRED' ? 'neutral' : item.enabled ? 'success' : 'warning',
    })
  }

  function renderChannels(): void {
    channelsTable.setRows(channels.map(item => {
      const secretKeys = Object.keys(item.secretMasks || {})
      return [
        details([['ID', item.id], ['编码', item.code], ['名称', item.name], ['版本', item.version]]),
        details([['驱动', driverLabel(item.driver)], ['区域', item.region], ['优先级', item.priority], ['状态', lifecycleBadge(item)]]),
        details([
          ['公开配置', Object.entries(item.publicConfig || {}).map(([key, value]) => `${key}=${value}`).join('；') || '无'],
          ['密钥', secretKeys.length ? secretKeys.map(key => `${key}=${item.secretMasks[key] || '已配置'}`).join('；') : '无'],
          ['密钥 Key ID', item.credentialKeyId || '—'],
        ]),
        details([['创建', displayTime(item.createdAt)], ['更新', displayTime(item.updatedAt)]]),
        canManage && item.lifecycle === 'ACTIVE' ? actions(
          button({ label: '编辑', size: 'sm', variant: 'secondary', onClick: () => openChannel(item) }),
          button({ label: item.enabled ? '停用' : '启用', size: 'sm', variant: 'secondary', onClick: () => setEnabled('channels', item.code, item.version, !item.enabled) }),
          button({ label: '测试发送', size: 'sm', variant: 'secondary', onClick: () => openTest(item) }),
          button({ label: '退役', size: 'sm', variant: 'danger', onClick: () => openRetire('channels', item.code, item.name, item.version) }),
        ) : '—',
      ]
    }))
    if (tab !== 'channels') return
    if (metadataError) state.set(`通道列表可读，但驱动元数据加载失败：${metadataError}`, 'danger')
    else state.set(`通道 ${channels.length} 个 · 驱动 ${driverDefinitions.length} 种 · 使用中 ${channels.filter(item => item.lifecycle === 'ACTIVE').length} 个`)
  }

  function renderRegions(): void {
    regionsTable.setRows(regions.map(item => [
      details([['ID', item.id], ['区号', item.dialCode], ['名称', `${item.emoji} ${item.name}`], ['版本', item.version]]),
      details([['编码', item.code], ['ISO2', item.iso2]]),
      details([['排序', item.sort], ['状态', lifecycleBadge(item)]]),
      details([['创建', displayTime(item.createdAt)], ['更新', displayTime(item.updatedAt)]]),
      canManage && item.lifecycle === 'ACTIVE' ? actions(
        button({ label: '编辑', size: 'sm', variant: 'secondary', onClick: () => openRegion(item) }),
        button({ label: item.enabled ? '停用' : '启用', size: 'sm', variant: 'secondary', onClick: () => setEnabled('regions', item.code, item.version, !item.enabled) }),
        button({ label: '退役', size: 'sm', variant: 'danger', onClick: () => openRetire('regions', item.code, item.name, item.version) }),
      ) : '—',
    ]))
    if (tab === 'regions') state.set(`区域 ${regions.length} 个 · 使用中 ${regions.filter(item => item.lifecycle === 'ACTIVE').length} 个`)
  }

  function renderGrant(): void {
    if (!grant) {
      grantsTable.setRows([])
      return
    }
    grantsTable.setRows([[
      details([['商户', grant.merchantId], ['店铺', grant.shopId]]),
      details([['范围', grant.unrestricted ? '不限制' : grant.dialCodes.join('、') || '—']]),
      grant.version || 0,
      canManage ? button({ label: '编辑开通', size: 'sm', variant: 'secondary', onClick: () => openGrant() }) : '—',
    ]])
  }

  async function load(): Promise<void> {
    if (tab === 'grants') return
    const filter = tab === 'channels' ? channelFilter : regionFilter
    filter.setBusy(true)
    state.set(tab === 'channels' ? '正在加载短信通道…' : '正在加载短信区域…')
    try {
      const values = filter.values()
      const query = new URLSearchParams()
      for (const key of Object.keys(values)) if (values[key]?.trim()) query.set(key, values[key].trim())
      if (tab === 'channels') {
        channels = await client.request<SMSChannel[]>(`${prefix}/channels${query.size ? `?${query}` : ''}`)
        try { regions = await client.request<SMSRegion[]>(`${prefix}/regions?lifecycle=ACTIVE`) } catch { /* region options stay as last known */ }
        renderChannels()
      } else {
        regions = await client.request<SMSRegion[]>(`${prefix}/regions${query.size ? `?${query}` : ''}`)
        renderRegions()
      }
    } catch (error) {
      if (tab === 'channels') { channels = []; channelsTable.setRows([]) }
      else { regions = []; regionsTable.setRows([]) }
      state.set(`加载失败：${String(error)}`, 'danger')
    } finally { filter.setBusy(false) }
  }

  async function loadGrant(): Promise<void> {
    const values = grantFilter.values()
    const merchantId = Number(values.merchantId)
    const shopId = Number(values.shopId)
    if (!Number.isInteger(merchantId) || merchantId <= 0 || !Number.isInteger(shopId) || shopId <= 0) {
      state.set('请输入有效的 merchant_id 与 shop_id。', 'danger')
      return
    }
    grantFilter.setBusy(true)
    state.set('正在查询商户区域开通…')
    try {
      const query = new URLSearchParams({ merchantId: String(merchantId), shopId: String(shopId) })
      grant = await client.request<MerchantGrant>(`${prefix}/merchant-grants?${query}`)
      renderGrant()
      state.set(grant.unrestricted ? `店铺 ${merchantId}/${shopId} 当前不限制区域。` : `店铺 ${merchantId}/${shopId} 已开通 ${grant.dialCodes.length} 个区域。`)
    } catch (error) {
      grant = null
      grantsTable.setRows([])
      state.set(`开通范围加载失败：${String(error)}`, 'danger')
    } finally { grantFilter.setBusy(false) }
  }

  function activeRegions(): SMSRegion[] {
    return regions.filter(item => item.lifecycle === 'ACTIVE' && item.enabled)
  }

  function regionOptions(): Array<{ value: string; label: string }> {
    return [{ value: '*', label: '* 全部区域' }, ...activeRegions().map(item => ({ value: item.dialCode, label: `${item.emoji} ${item.dialCode} ${item.name}` }))]
  }

  function regionTree(): HostModalTreeNode[] {
    return [{
      id: 'regions',
      label: '可开通区域',
      children: activeRegions().map(item => ({ id: item.dialCode, label: `${item.emoji} ${item.dialCode} ${item.name}`, value: item.dialCode })),
    }]
  }

  function openChannel(current?: SMSChannel): void {
    if (!driverDefinitions.length) {
      state.set(`无法创建或编辑：驱动元数据不可用${metadataError ? `（${metadataError}）` : ''}`, 'danger')
      return
    }
    const driverSelector = (): HostModalField => ({
      name: 'driver', label: '短信驱动', kind: 'select', required: true, wide: true,
      options: [{ value: '', label: '请选择短信驱动' }, ...driverDefinitions.map(definition => ({ value: definition.code, label: definition.name }))],
    })
    const editorFields = (definition: DriverDefinition, carried: Record<string, string> = {}): { fields: HostModalField[]; values: Record<string, string | number> } => {
      const fields: HostModalField[] = [
        driverSelector(),
        { name: 'driverDescription', label: '驱动能力说明', disabled: true, wide: true },
        { name: 'code', label: '稳定通道编码', required: true, disabled: Boolean(current), mono: true, placeholder: 'aliyun-cn' },
        { name: 'name', label: '展示名称', required: true, placeholder: definition.name },
        { name: 'region', label: '路由区域', kind: 'select', required: true, options: regionOptions() },
        { name: 'priority', label: '优先级（越大越优先）', type: 'number', required: true, min: 0, max: 1000 },
      ]
      const values: Record<string, string | number> = {
        driver: definition.code, driverDescription: definition.description,
        code: carried.code || current?.code || '', name: carried.name || current?.name || '',
        region: carried.region || current?.region || '*', priority: carried.priority || current?.priority || 0,
      }
      for (const field of definition.fields) {
        if (field.secret) {
          const configured = Boolean(current?.secretMasks?.[field.key])
          fields.push({
            name: `${field.key}Mode`,
            label: `${field.label}处理（当前：${configured ? current?.secretMasks?.[field.key] || '已配置' : '未配置'}）`,
            kind: 'select', required: true,
            options: [...(configured ? [{ value: 'KEEP', label: '保留当前密钥' }] : []), { value: 'CLEAR', label: '不配置 / 清除' }, { value: 'REPLACE', label: '填写新密钥' }],
          })
          fields.push({ name: field.key, label: field.label, type: 'password', placeholder: field.help || field.placeholder, autocomplete: 'new-password', wide: true })
          const carriedMode = carried[`${field.key}Mode`]
          values[`${field.key}Mode`] = carriedMode === 'KEEP' || carriedMode === 'CLEAR' || carriedMode === 'REPLACE' ? carriedMode : configured ? 'KEEP' : field.required ? 'REPLACE' : 'CLEAR'
          values[field.key] = carried[field.key] || ''
          continue
        }
        fields.push({ name: field.key, label: field.label, required: field.required, placeholder: field.help || field.placeholder, wide: true })
        values[field.key] = carried[field.key] || current?.publicConfig?.[field.key] || ''
      }
      return { fields, values }
    }
    const editor = hostFormModal({
      title: `${current ? '编辑' : '新增'}短信通道`,
      fields: [driverSelector()],
      submitLabel: current ? '保存' : '创建',
      onChange: (values, field, modal) => {
        if (field !== 'driver') return
        const definition = driverByCode.get(values.driver)
        if (!definition) {
          modal.setFields([driverSelector()], { driver: '' }, `${current ? '编辑' : '新增'}短信通道`)
          return
        }
        const generated = editorFields(definition, values)
        modal.setFields(generated.fields, generated.values, `${current ? '编辑' : '新增'}短信通道 · ${definition.name}`)
      },
      onSubmit: (values, modal) => {
        const definition = driverByCode.get(values.driver)
        if (!definition) { modal.setError('请选择短信驱动。'); return }
        const code = (current?.code || values.code).trim().toLowerCase()
        if (!channelCodePattern.test(code)) { modal.setError('通道编码只能包含小写字母、数字、-、_，长度 1–32。'); return }
        if (!values.name.trim()) { modal.setError('请输入展示名称。'); return }
        const region = values.region.trim() || '*'
        if (region !== '*' && !dialCodePattern.test(region)) { modal.setError('路由区域必须是 * 或已启用区号。'); return }
        const priority = Number(values.priority)
        if (!Number.isInteger(priority) || priority < 0 || priority > 1000) { modal.setError('优先级必须是 0–1000 的整数。'); return }
        const publicConfig: Record<string, string> = {}
        const secrets: Record<string, { mode: string; value?: string }> = {}
        for (const field of definition.fields) {
          if (field.secret) {
            const mode = values[`${field.key}Mode`]
            const value = values[field.key]?.trim() || ''
            if (mode === 'REPLACE' && !value) { modal.setError(`请填写${field.label}。`); return }
            if ((mode === 'KEEP' || mode === 'CLEAR') && value) { modal.setError(`${field.label}选择保留或清除时不要填写新值。`); return }
            secrets[field.key] = { mode, value: mode === 'REPLACE' ? value : '' }
            continue
          }
          const value = values[field.key]?.trim() || ''
          if (field.required && !value) { modal.setError(`请填写${field.label}。`); return }
          if (value) publicConfig[field.key] = value
        }
        modal.setBusy(true)
        client.request(`${prefix}/channels/${encodeURIComponent(code)}`, {
          method: 'PUT',
          body: JSON.stringify({ commandKey: randomUUID(), expectedVersion: current?.version ?? 0, name: values.name.trim(), driver: definition.code, region, priority, publicConfig, secrets }),
        }).then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    if (current) {
      const definition = driverByCode.get(current.driver)
      if (!definition) { state.set(`无法编辑 ${current.code}：驱动 ${current.driver} 的元数据不存在。`, 'danger'); return }
      const generated = editorFields(definition)
      editor.open(generated.values, `编辑短信通道 · ${definition.name}`)
      editor.setFields(generated.fields, generated.values, `编辑短信通道 · ${definition.name}`)
    } else {
      editor.open({ driver: '' })
    }
  }

  function openRegion(current?: SMSRegion): void {
    const editor = hostFormModal({
      title: `${current ? '编辑' : '新增'}短信区域`,
      fields: [
        { name: 'dialCode', label: '区号', required: true, disabled: Boolean(current), mono: true, placeholder: '+86' },
        { name: 'name', label: '名称', required: true, placeholder: '中国大陆' },
        { name: 'iso2', label: 'ISO2', required: true, placeholder: 'CN', maxLength: 2 },
        { name: 'emoji', label: '旗帜', placeholder: '🇨🇳' },
        { name: 'sort', label: '排序', type: 'number', required: true, min: 0, max: 10000 },
      ],
      submitLabel: current ? '保存' : '创建',
      onSubmit: (values, modal) => {
        const dialCode = (current?.dialCode || values.dialCode).trim()
        if (!dialCodePattern.test(dialCode)) { modal.setError('区号必须是 + 开头的数字，例如 +86。'); return }
        const code = current?.code || dialCode.slice(1)
        if (!values.name.trim()) { modal.setError('请输入名称。'); return }
        const iso2 = values.iso2.trim().toUpperCase()
        if (!/^[A-Z]{2}$/.test(iso2)) { modal.setError('ISO2 必须是两位大写字母。'); return }
        const sort = Number(values.sort)
        if (!Number.isInteger(sort) || sort < 0 || sort > 10000) { modal.setError('排序必须是 0–10000 的整数。'); return }
        modal.setBusy(true)
        client.request(`${prefix}/regions/${encodeURIComponent(code)}`, {
          method: 'PUT',
          body: JSON.stringify({ commandKey: randomUUID(), expectedVersion: current?.version ?? 0, dialCode, name: values.name.trim(), iso2, emoji: values.emoji.trim(), sort }),
        }).then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open(current ? { dialCode: current.dialCode, name: current.name, iso2: current.iso2, emoji: current.emoji, sort: current.sort } : { sort: 100 })
  }

  function openGrant(): void {
    const values = grantFilter.values()
    const merchantId = Number(grant?.merchantId || values.merchantId)
    const shopId = Number(grant?.shopId || values.shopId)
    if (!Number.isInteger(merchantId) || merchantId <= 0 || !Number.isInteger(shopId) || shopId <= 0) {
      state.set('请先查询有效的 merchant_id 与 shop_id。', 'danger')
      return
    }
    const editor = hostFormModal({
      title: `商户区域开通 · ${merchantId}/${shopId}`,
      fields: [
        { name: 'mode', label: '开通方式', kind: 'select', required: true, options: [{ value: 'unrestricted', label: '不限制（空选择）' }, { value: 'restricted', label: '仅开通所选区域' }] },
        { name: 'dialCodes', label: '开通区域', kind: 'checkbox-tree', tree: regionTree(), wide: true, empty: '没有可开通的启用区域' },
      ],
      onSubmit: (form, modal) => {
        const dialCodes = form.mode === 'restricted' ? selectedCodes(String(form.dialCodes || '')) : []
        if (form.mode === 'restricted' && !dialCodes.length) { modal.setError('限制开通时至少选择一个区域。'); return }
        modal.setBusy(true)
        client.request(`${prefix}/merchant-grants`, {
          method: 'PUT',
          body: JSON.stringify({ commandKey: randomUUID(), expectedVersion: grant?.version ?? 0, merchantId, shopId, dialCodes }),
        }).then(async () => { modal.close(); await loadGrant() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open({
      mode: grant && !grant.unrestricted ? 'restricted' : 'unrestricted',
      dialCodes: (grant?.dialCodes || []).join(','),
    })
  }

  function openTest(current: SMSChannel): void {
    const editor = hostFormModal({
      title: `测试发送 · ${current.name}`,
      fields: [{ name: 'phone', label: 'E.164 手机号', required: true, placeholder: '+8613800138000' }],
      submitLabel: '发送',
      onSubmit: (values, modal) => {
        const phone = values.phone.trim()
        if (!/^\+[1-9][0-9]{7,}$/.test(phone)) { modal.setError('请输入 E.164 手机号，例如 +8613800138000。'); return }
        modal.setBusy(true)
        client.request<{ ok: boolean; detail: string; mock?: boolean; code?: string }>(`${prefix}/test`, {
          method: 'POST',
          body: JSON.stringify({ channelCode: current.code, phone }),
        }).then(result => {
          modal.close()
          state.set(result.ok ? `${result.detail}${result.code ? ` · 验证码 ${result.code}` : ''}` : `发送未成功：${result.detail}`, result.ok ? 'success' : 'danger')
        }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open()
  }

  function setEnabled(kind: 'channels' | 'regions', code: string, version: number, enabled: boolean): void {
    const action = enabled ? '启用' : '停用'
    const subject = kind === 'channels' ? '通道' : '区域'
    const editor = hostFormModal({
      title: `${action} · ${code}`,
      fields: [{ name: 'hint', label: '提示', kind: 'textarea', disabled: true, wide: true, rows: 3 }],
      submitLabel: action,
      onSubmit: (_values, modal) => {
        modal.setBusy(true)
        client.request(`${prefix}/${kind}/${encodeURIComponent(code)}/${enabled ? 'enable' : 'disable'}`, { method: 'POST', body: JSON.stringify({ commandKey: randomUUID(), expectedVersion: version }) })
          .then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open({
      hint: enabled
        ? `确认启用 ${code}？启用后该${subject}将重新参与短信路由。`
        : `确认停用 ${code}？停用后该${subject}不再参与短信路由，配置会保留。`,
    })
  }

  function openRetire(kind: 'channels' | 'regions', code: string, name: string, version: number): void {
    const editor = hostFormModal({
      title: `退役${kind === 'channels' ? '通道' : '区域'} · ${name}`,
      fields: [{ name: 'confirm', label: '退役保留历史版本，不能再修改。', kind: 'select', required: true, options: [{ value: '', label: '请选择' }, { value: code, label: `确认退役 ${code}` }] }],
      submitLabel: '退役',
      onSubmit: (values, modal) => {
        if (values.confirm !== code) { modal.setError('请选择确认项。'); return }
        modal.setBusy(true)
        client.request(`${prefix}/${kind}/${encodeURIComponent(code)}/retire`, { method: 'POST', body: JSON.stringify({ commandKey: randomUUID(), expectedVersion: version }) })
          .then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open()
  }

  if (options?.mounts) {
    searchHost.replaceChildren()
    dataHost.replaceChildren()
  } else if (options?.embedded) {
    root.replaceChildren(searchHost, dataHost)
  } else {
    root.replaceChildren(page({ showSummary: false, children: [searchHost, dataHost] }))
  }
  switchTab('channels')
  try {
    regions = await client.request<SMSRegion[]>(`${prefix}/regions?lifecycle=ACTIVE`)
  } catch {
    regions = []
  }
  await load()
}
