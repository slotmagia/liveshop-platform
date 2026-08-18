import type { HostContext, HostHttpClient, HostModalField } from '@liveshop/host-sdk'
import { hostFormModal } from '@liveshop/host-sdk'
import { badge, button, create, dataCard, page, statusLine, table, ui } from '@liveshop/design-tokens'

interface DriverFieldOption {
  value: string
  label: string
}

interface DriverField {
  key: string
  label: string
  type: 'TEXT' | 'PASSWORD' | 'NUMBER' | 'SELECT'
  required: boolean
  secret: boolean
  placeholder?: string
  help?: string
  options?: DriverFieldOption[]
}

interface DriverDefinition {
  code: string
  name: string
  description: string
  fields: DriverField[]
}

interface EmailConfig {
  id: number
  driver: string
  enabled: boolean
  publicConfig: Record<string, string>
  secretMasks: Record<string, string>
  credentialKeyId?: string
  version: number
  createdAt: string
  updatedAt: string
}

const prefix = '/admin/platform/email'
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

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

export async function startEmail(root: HTMLElement, client: HostHttpClient, context: HostContext, options?: { embedded?: boolean }): Promise<void> {
  const state = statusLine()
  const canManage = context.permissions.includes('platform.email.manage')
  let metadataError = ''
  let driverDefinitions: DriverDefinition[] = []
  try {
    driverDefinitions = await client.request<DriverDefinition[]>(`${prefix}/drivers`)
  } catch (error) {
    metadataError = String(error)
  }
  const driverByCode = new Map(driverDefinitions.map(definition => [definition.code, definition]))
  const driverLabel = (driver: string): string => driverByCode.get(driver)?.name || driver

  const configTable = table({ columns: ['驱动', '发信配置', '状态', '时间', '操作'], empty: '还没有邮件发信配置' })
  let config: EmailConfig | null = null

  const card = dataCard({
    title: '发信配置',
    actions: [
      button({ label: '刷新', variant: 'secondary', onClick: () => void load() }),
      ...(canManage ? [button({ label: '编辑配置', onClick: () => openConfig() })] : []),
    ],
    body: configTable.element,
  })

  function render(): void {
    if (!config || !config.version) {
      configTable.setRows([])
      return
    }
    const definition = driverByCode.get(config.driver)
    const publicLines: Array<[string, string]> = []
    for (const field of definition?.fields ?? []) {
      if (field.secret) continue
      const value = config.publicConfig?.[field.key] || ''
      if (!value) continue
      const option = field.options?.find(item => item.value === value)
      publicLines.push([field.label, option?.label || value])
    }
    for (const [key, mask] of Object.entries(config.secretMasks || {})) {
      const field = definition?.fields.find(item => item.key === key)
      publicLines.push([field?.label || key, mask])
    }
    configTable.setRows([[
      details([['驱动', driverLabel(config.driver)], ['版本', `v${config.version}`]]),
      publicLines.length ? details(publicLines) : '该驱动无需额外配置',
      badge({ label: config.enabled ? '已启用' : '已停用', tone: config.enabled ? 'success' : 'neutral' }),
      details([['更新', displayTime(config.updatedAt)], ['创建', displayTime(config.createdAt)]]),
      canManage ? actions(
        button({ label: '测试发送', size: 'sm', variant: 'secondary', onClick: () => openTest() }),
        button({ label: config.enabled ? '停用' : '启用', size: 'sm', variant: 'secondary', onClick: () => setEnabled(!config!.enabled) }),
        button({ label: '编辑', size: 'sm', variant: 'secondary', onClick: () => openConfig() }),
      ) : '',
    ]])
  }

  async function load(): Promise<void> {
    state.set('正在加载邮件发信配置…')
    try {
      const item = await client.request<EmailConfig>(`${prefix}/config`)
      config = item?.version ? item : null
      render()
      state.set(config ? `已加载发信配置 · ${driverLabel(config.driver)}` : '尚未保存发信配置。模板请到「通知模板」维护。')
    } catch (error) {
      state.set(`加载失败：${String(error)}`, 'danger')
    }
  }

  function openConfig(): void {
    if (!driverDefinitions.length) {
      state.set(`无法编辑：驱动元数据不可用${metadataError ? `（${metadataError}）` : ''}`, 'danger')
      return
    }
    const current = config
    const driverSelector = (): HostModalField => ({
      name: 'driver', label: '邮件驱动', kind: 'select', required: true, wide: true,
      options: [{ value: '', label: '请选择邮件驱动' }, ...driverDefinitions.map(definition => ({ value: definition.code, label: definition.name }))],
    })
    const editorFields = (definition: DriverDefinition, carried: Record<string, string> = {}): { fields: HostModalField[]; values: Record<string, string | number> } => {
      const fields: HostModalField[] = [driverSelector(), { name: 'driverDescription', label: '驱动能力说明', disabled: true, wide: true }]
      const values: Record<string, string | number> = { driver: definition.code, driverDescription: definition.description }
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
        if (field.type === 'SELECT') {
          fields.push({
            name: field.key, label: field.label, kind: 'select', required: field.required, wide: true,
            options: [{ value: '', label: field.placeholder || '请选择' }, ...(field.options || []).map(option => ({ value: option.value, label: option.label }))],
          })
        } else {
          fields.push({
            name: field.key, label: field.label, type: field.type === 'NUMBER' ? 'number' : 'text',
            required: field.required, placeholder: field.help || field.placeholder, wide: true,
          })
        }
        values[field.key] = carried[field.key] || current?.publicConfig?.[field.key] || ''
      }
      return { fields, values }
    }
    const editor = hostFormModal({
      title: current ? '编辑发信配置' : '新建发信配置',
      fields: [driverSelector()],
      submitLabel: current ? '保存' : '创建',
      onChange: (values, field, modal) => {
        if (field !== 'driver') return
        const definition = driverByCode.get(values.driver)
        if (!definition) {
          modal.setFields([driverSelector()], { driver: '' }, current ? '编辑发信配置' : '新建发信配置')
          return
        }
        const generated = editorFields(definition, values)
        modal.setFields(generated.fields, generated.values, `${current ? '编辑' : '新建'}发信配置 · ${definition.name}`)
      },
      onSubmit: (values, modal) => {
        const definition = driverByCode.get(values.driver)
        if (!definition) { modal.setError('请选择邮件驱动。'); return }
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
        client.request(`${prefix}/config`, {
          method: 'PUT',
          body: JSON.stringify({ commandKey: crypto.randomUUID(), expectedVersion: current?.version ?? 0, driver: definition.code, publicConfig, secrets }),
        }).then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    if (current) {
      const definition = driverByCode.get(current.driver)
      if (!definition) { state.set(`无法编辑：驱动 ${current.driver} 的元数据不存在。`, 'danger'); return }
      const generated = editorFields(definition)
      editor.open(generated.values, `编辑发信配置 · ${definition.name}`)
      editor.setFields(generated.fields, generated.values, `编辑发信配置 · ${definition.name}`)
    } else {
      editor.open({ driver: '' })
    }
  }

  function setEnabled(enabled: boolean): void {
    if (!config) return
    const action = enabled ? '启用' : '停用'
    const version = config.version
    const editor = hostFormModal({
      title: `${action}发信`,
      fields: [{ name: 'hint', label: '提示', kind: 'textarea', disabled: true, wide: true, rows: 3 }],
      submitLabel: action,
      onSubmit: (_values, modal) => {
        modal.setBusy(true)
        client.request(`${prefix}/config/${enabled ? 'enable' : 'disable'}`, { method: 'POST', body: JSON.stringify({ commandKey: crypto.randomUUID(), expectedVersion: version }) })
          .then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open({
      hint: enabled
        ? '确认启用发信？启用后通知事件将按已保存的 SMTP 配置发送邮件。'
        : '确认停用发信？停用后不再对外发信，配置会保留。',
    })
  }

  function openTest(): void {
    if (!config) { state.set('请先保存发信配置。', 'danger'); return }
    const editor = hostFormModal({
      title: '发送测试邮件',
      fields: [
        { name: 'to', label: '收件人', required: true, placeholder: 'someone@example.com', wide: true },
        { name: 'subject', label: '主题', placeholder: '不填则用默认测试主题', wide: true },
      ],
      submitLabel: '发送',
      onSubmit: (values, modal) => {
        const to = values.to.trim()
        if (!emailPattern.test(to)) { modal.setError('请输入有效的收件人邮箱。'); return }
        modal.setBusy(true)
        client.request<{ ok: boolean; detail: string; mock?: boolean }>(`${prefix}/config/test`, {
          method: 'POST',
          body: JSON.stringify({ to, subject: values.subject.trim() }),
        }).then(result => {
          modal.close()
          state.set(result.ok ? `${result.mock ? 'Mock 已接受' : '发送成功'}：${result.detail}` : `发送失败：${result.detail}`, result.ok ? 'success' : 'danger')
        }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    editor.open()
  }

  const children = [state.element, card]
  if (options?.embedded) root.replaceChildren(...children)
  else root.replaceChildren(page({ showSummary: false, children }))
  void load()
}
