import type { HostContext, HostHttpClient, HostModalField } from '@liveshop/host-sdk'
import { hostFormModal } from '@liveshop/host-sdk'
import { badge, button, create, dataCard, page, searchCard, searchForm, statusLine, table, ui } from '@liveshop/design-tokens'

type Lifecycle = 'ACTIVE' | 'RETIRED'

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

interface StorageChannel {
  id: number
  code: string
  name: string
  driver: string
  enabled: boolean
  isDefault: boolean
  lifecycle: Lifecycle
  publicConfig: Record<string, string>
  secretMasks: Record<string, string>
  credentialKeyId?: string
  version: number
  createdAt: string
  updatedAt: string
}

const prefix = '/admin/platform/storage'
const channelCodePattern = /^[a-z0-9][a-z0-9_-]{0,31}$/

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

export async function startStorage(root: HTMLElement, client: HostHttpClient, context: HostContext): Promise<void> {
  const state = statusLine()
  const canManage = context.permissions.includes('platform.storage.manage')
  let metadataError = ''
  let driverDefinitions: DriverDefinition[] = []
  try {
    driverDefinitions = await client.request<DriverDefinition[]>(`${prefix}/drivers`)
  } catch (error) {
    metadataError = String(error)
  }
  const driverByCode = new Map(driverDefinitions.map(definition => [definition.code, definition]))
  const driverLabel = (driver: string): string => driverByCode.get(driver)?.name || driver

  const filter = searchForm({
    fields: [
      { name: 'driver', label: '驱动', kind: 'select', options: [{ value: '', label: '全部驱动' }, ...driverDefinitions.map(definition => ({ value: definition.code, label: definition.name }))] },
      { name: 'keyword', label: '编码 / 名称', placeholder: 'local / 本地磁盘' },
      { name: 'lifecycle', label: '生命周期', kind: 'select', options: [{ value: '', label: '全部' }, { value: 'ACTIVE', label: '使用中' }, { value: 'RETIRED', label: '已退役' }] },
    ],
    onSearch: () => void load(),
    onReset: () => void load(),
  })
  const channelsTable = table({ columns: ['通道', '驱动 / 状态', '配置', '时间', '操作'], empty: '还没有存储通道' })
  let channels: StorageChannel[] = []
  const channelCard = dataCard({
    title: '存储通道',
    actions: [button({ label: '刷新', variant: 'secondary', onClick: () => void load() }), ...(canManage ? [button({ label: '新增通道', onClick: () => openChannel() })] : [])],
    body: channelsTable.element,
  })

  function lifecycleBadge(item: StorageChannel): HTMLElement {
    return badge({
      label: item.lifecycle === 'RETIRED' ? '已退役' : item.enabled ? '已启用' : '已停用',
      tone: item.lifecycle === 'RETIRED' ? 'neutral' : item.enabled ? 'success' : 'warning',
    })
  }

  function render(): void {
    channelsTable.setRows(channels.map(item => {
      const secretKeys = Object.keys(item.secretMasks || {})
      const rowActions: Node[] = [
        button({ label: '测试', size: 'sm', variant: 'secondary', onClick: () => openTest(item) }),
      ]
      if (!item.isDefault && item.enabled) {
        rowActions.push(button({ label: '设为默认', size: 'sm', variant: 'secondary', onClick: () => setDefault(item) }))
      }
      rowActions.push(
        button({ label: item.enabled ? '停用' : '启用', size: 'sm', variant: 'secondary', onClick: () => setEnabled(item, !item.enabled) }),
        button({ label: '编辑', size: 'sm', variant: 'secondary', onClick: () => openChannel(item) }),
        button({ label: '退役', size: 'sm', variant: 'danger', onClick: () => openRetire(item) }),
      )
      return [
        details([['ID', item.id], ['编码', item.code], ['名称', item.name], ['版本', item.version]]),
        details([
          ['驱动', driverLabel(item.driver)],
          ['默认', item.isDefault ? badge({ label: '默认通道', tone: 'success' }) : '否'],
          ['状态', lifecycleBadge(item)],
        ]),
        details([
          ['公开配置', Object.entries(item.publicConfig || {}).map(([key, value]) => `${key}=${value}`).join('；') || (item.driver === 'local' ? '服务器本地目录' : '无')],
          ['密钥', secretKeys.length ? secretKeys.map(key => `${key}=${item.secretMasks[key] || '已配置'}`).join('；') : '无'],
          ['密钥 Key ID', item.credentialKeyId || '—'],
        ]),
        details([['创建', displayTime(item.createdAt)], ['更新', displayTime(item.updatedAt)]]),
        canManage && item.lifecycle === 'ACTIVE' ? actions(...rowActions) : '—',
      ]
    }))
    if (metadataError) state.set(`通道列表可读，但驱动元数据加载失败：${metadataError}`, 'danger')
    else state.set(`通道 ${channels.length} 个 · 驱动 ${driverDefinitions.length} 种 · 默认 ${channels.find(item => item.isDefault)?.name || '无'}`)
  }

  async function load(): Promise<void> {
    filter.setBusy(true)
    state.set('正在加载存储通道…')
    try {
      const values = filter.values()
      const query = new URLSearchParams()
      for (const key of Object.keys(values)) if (values[key]?.trim()) query.set(key, values[key].trim())
      channels = await client.request<StorageChannel[]>(`${prefix}/channels${query.size ? `?${query}` : ''}`)
      render()
    } catch (error) {
      channels = []
      channelsTable.setRows([])
      state.set(`加载失败：${String(error)}`, 'danger')
    } finally {
      filter.setBusy(false)
    }
  }

  function openChannel(current?: StorageChannel): void {
    if (!driverDefinitions.length) {
      state.set(`无法创建或编辑：驱动元数据不可用${metadataError ? `（${metadataError}）` : ''}`, 'danger')
      return
    }
    const driverSelector = (): HostModalField => ({
      name: 'driver', label: '存储驱动', kind: 'select', required: true, wide: true,
      options: [{ value: '', label: '请选择存储驱动' }, ...driverDefinitions.map(definition => ({ value: definition.code, label: definition.name }))],
    })
    const editorFields = (definition: DriverDefinition, carried: Record<string, string> = {}): { fields: HostModalField[]; values: Record<string, string | number> } => {
      const fields: HostModalField[] = [
        driverSelector(),
        { name: 'driverDescription', label: '驱动能力说明', disabled: true, wide: true },
        { name: 'code', label: '稳定通道编码', required: true, disabled: Boolean(current), mono: true, placeholder: 'oss-cn' },
        { name: 'name', label: '展示名称', required: true, placeholder: definition.name },
      ]
      const values: Record<string, string | number> = {
        driver: definition.code, driverDescription: definition.description,
        code: carried.code || current?.code || '', name: carried.name || current?.name || '',
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
      title: `${current ? '编辑' : '新增'}存储通道`,
      fields: [driverSelector()],
      submitLabel: current ? '保存' : '创建',
      onChange: (values, field, modal) => {
        if (field !== 'driver') return
        const definition = driverByCode.get(values.driver)
        if (!definition) {
          modal.setFields([driverSelector()], { driver: '' }, `${current ? '编辑' : '新增'}存储通道`)
          return
        }
        const generated = editorFields(definition, values)
        modal.setFields(generated.fields, generated.values, `${current ? '编辑' : '新增'}存储通道 · ${definition.name}`)
      },
      onSubmit: (values, modal) => {
        const definition = driverByCode.get(values.driver)
        if (!definition) { modal.setError('请选择存储驱动。'); return }
        const code = (current?.code || values.code).trim().toLowerCase()
        if (!channelCodePattern.test(code)) { modal.setError('通道编码只能包含小写字母、数字、-、_，长度 1–32。'); return }
        if (!values.name.trim()) { modal.setError('请输入展示名称。'); return }
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
          body: JSON.stringify({ commandKey: crypto.randomUUID(), expectedVersion: current?.version ?? 0, name: values.name.trim(), driver: definition.code, publicConfig, secrets }),
        }).then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    if (current) {
      const definition = driverByCode.get(current.driver)
      if (!definition) { state.set(`无法编辑 ${current.code}：驱动 ${current.driver} 的元数据不存在。`, 'danger'); return }
      const generated = editorFields(definition)
      editor.open(generated.values, `编辑存储通道 · ${definition.name}`)
      editor.setFields(generated.fields, generated.values, `编辑存储通道 · ${definition.name}`)
    } else {
      editor.open({ driver: '' })
    }
  }

  function accessibleUrl(url: string): string {
    if (!url) return ''
    if (/^https?:\/\//i.test(url)) return url
    const base = context.gatewayBaseUrl.replace(/\/$/, '')
    return url.startsWith('/') ? base + url : `${base}/${url}`
  }

  function openTestResult(result: { detail: string; url?: string; driver?: string }): void {
    const url = accessibleUrl(result.url || '')
    if (!url) {
      state.set(result.detail || '写入成功，但未返回访问地址', 'danger')
      return
    }
    const viewer = hostFormModal({
      title: '测试成功',
      fields: [
        { name: 'detail', label: '结果', kind: 'textarea', disabled: true, wide: true, rows: 3 },
        { name: 'url', label: '访问地址', disabled: true, wide: true },
      ],
      submitLabel: '打开地址',
      onSubmit: (values, modal) => {
        window.open(values.url, '_blank', 'noopener,noreferrer')
        modal.close()
      },
    })
    viewer.open({ detail: result.detail, url })
    state.set(`${result.detail} · ${url}`, 'success')
  }

  function openTest(current: StorageChannel): void {
    const editor = hostFormModal({
      title: `测试写入 · ${current.name}`,
      fields: [{ name: 'hint', label: '提示', kind: 'textarea', disabled: true, wide: true, rows: 4 }],
      submitLabel: '测试',
      onSubmit: (_values, modal) => {
        modal.setBusy(true)
        client.request<{ ok: boolean; detail: string; url?: string; driver?: string }>(`${prefix}/channels/${encodeURIComponent(current.code)}/test`, { method: 'POST' })
          .then(result => {
            modal.close()
            if (!result.ok) {
              state.set(`写入未成功：${result.detail}`, 'danger')
              return
            }
            openTestResult(result)
          }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open({
      hint: `将向「${current.name}」写入探测文件 _storage_test/ping-*.txt。测试使用已保存配置，不会提交表单中的未保存密钥。`,
    })
  }

  function setEnabled(current: StorageChannel, enabled: boolean): void {
    const action = enabled ? '启用' : '停用'
    const editor = hostFormModal({
      title: `${action} · ${current.code}`,
      fields: [{ name: 'hint', label: '提示', kind: 'textarea', disabled: true, wide: true, rows: 3 }],
      submitLabel: action,
      onSubmit: (_values, modal) => {
        modal.setBusy(true)
        client.request(`${prefix}/channels/${encodeURIComponent(current.code)}/${enabled ? 'enable' : 'disable'}`, { method: 'POST', body: JSON.stringify({ commandKey: crypto.randomUUID(), expectedVersion: current.version }) })
          .then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open({
      hint: enabled
        ? `确认启用 ${current.code}？启用后该通道可被设为默认并承接新上传。`
        : `确认停用 ${current.code}？停用会同时取消默认标记，配置会保留。`,
    })
  }

  function setDefault(current: StorageChannel): void {
    const editor = hostFormModal({
      title: `设为默认 · ${current.code}`,
      fields: [{ name: 'hint', label: '提示', kind: 'textarea', disabled: true, wide: true, rows: 4 }],
      submitLabel: '设为默认',
      onSubmit: (_values, modal) => {
        modal.setBusy(true)
        client.request(`${prefix}/channels/${encodeURIComponent(current.code)}/default`, { method: 'POST', body: JSON.stringify({ commandKey: crypto.randomUUID(), expectedVersion: current.version }) })
          .then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open({
      hint: `确认将「${current.name}」设为默认通道？之后新上传会立即切到该通道；已有文件不会迁移。目标通道必须已启用。`,
    })
  }

  function openRetire(current: StorageChannel): void {
    const editor = hostFormModal({
      title: `退役通道 · ${current.name}`,
      fields: [{ name: 'confirm', label: '退役保留历史版本，不能再修改。若它是默认通道，新上传将回退到其它启用通道或本地磁盘。', kind: 'select', required: true, options: [{ value: '', label: '请选择' }, { value: current.code, label: `确认退役 ${current.code}` }] }],
      submitLabel: '退役',
      onSubmit: (values, modal) => {
        if (values.confirm !== current.code) { modal.setError('请选择确认项。'); return }
        modal.setBusy(true)
        client.request(`${prefix}/channels/${encodeURIComponent(current.code)}/retire`, { method: 'POST', body: JSON.stringify({ commandKey: crypto.randomUUID(), expectedVersion: current.version }) })
          .then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open()
  }

  root.replaceChildren(page({
    showSummary: false,
    children: [state.element, searchCard(filter.element), channelCard],
  }))
  await load()
}
