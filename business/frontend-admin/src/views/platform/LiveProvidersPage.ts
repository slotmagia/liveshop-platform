import type { HostContext, HostHttpClient, HostModalField } from '@liveshop/host-sdk'
import { hostFormModal } from '@liveshop/host-sdk'
import { badge, button, create, dataCard, page, searchCard, searchForm, statusLine, table, ui } from '@liveshop/design-tokens'

type Kind = 'RTMP' | 'RTC'
type Driver = 'STATIC' | 'SRS' | 'CLOUD' | 'AGORA' | 'AGORA_MEDIA_GATEWAY'
type Lifecycle = 'ACTIVE' | 'RETIRED'
type CredentialGroup = 'SECRET' | 'APP_CERTIFICATE' | 'CUSTOMER_CREDENTIAL'

interface DriverField {
  key: string
  label: string
  type: 'TEXT' | 'PASSWORD' | 'NUMBER' | 'SELECT'
  group?: string
  required: boolean
  secret: boolean
  credential?: CredentialGroup
  default?: string
  placeholder?: string
  help?: string
  options?: Array<{ value: string; label: string }>
  min?: number
  max?: number
  advanced: boolean
}

interface DriverDefinition {
  code: Driver
  name: string
  kind: Kind
  pushTransport: 'OBS_RTMP' | 'BROWSER_SDK'
  description: string
  fields: DriverField[]
}

interface LiveProvider {
  id: number
  code: string
  name: string
  kind: Kind
  driver: Driver
  app: string
  pushDomain: string
  pullDomain: string
  agoraAppId: string
  codec: string
  ingestDomain: string
  region: string
  ttlSeconds: number
  enabled: boolean
  isDefault: boolean
  lifecycle: Lifecycle
  healthStatus: 'UNKNOWN' | 'HEALTHY' | 'UNHEALTHY'
  healthMessage?: string
  healthCheckedAt?: string
  secretSet: boolean
  secretMask?: string
  appCertificateSet: boolean
  appCertificateMask?: string
  customerCredentialSet: boolean
  customerKeyMask?: string
  customerSecretMask?: string
  credentialKeyId?: string
  version: number
  createdAt: string
  updatedAt: string
}

const prefix = '/admin/platform/live-providers'
const codePattern = /^[a-z0-9][a-z0-9_-]{0,31}$/

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

function secretChange(mode: string, value = '', secondaryValue = {}) {
  return { mode: value.trim() ? 'REPLACE' : mode, value, ...secondaryValue }
}

const credentialModeName = (credential: CredentialGroup): string => ({
  SECRET: 'secretMode', APP_CERTIFICATE: 'certificateMode', CUSTOMER_CREDENTIAL: 'customerMode',
})[credential]

function credentialSet(current: LiveProvider | undefined, credential: CredentialGroup): boolean {
  if (!current) return false
  if (credential === 'SECRET') return current.secretSet
  if (credential === 'APP_CERTIFICATE') return current.appCertificateSet
  return current.customerCredentialSet
}

function credentialStatus(current: LiveProvider | undefined, credential: CredentialGroup): string {
  if (!current || !credentialSet(current, credential)) return '未配置'
  if (credential === 'SECRET') return current.secretMask || '已配置'
  if (credential === 'APP_CERTIFICATE') return current.appCertificateMask || '已配置'
  return `${current.customerKeyMask || '***'} / ${current.customerSecretMask || '***'}`
}

export async function startLiveProviders(root: HTMLElement, client: HostHttpClient, context: HostContext): Promise<void> {
  const state = statusLine()
  const canManage = context.permissions.includes('platform.live-provider.manage')
  let metadataError = ''
  let driverDefinitions: DriverDefinition[] = []
  try {
    driverDefinitions = await client.request<DriverDefinition[]>(`${prefix}/drivers`)
  } catch (error) {
    metadataError = String(error)
  }
  const driverByCode = new Map(driverDefinitions.map(definition => [definition.code, definition]))
  const driverLabel = (driver: Driver): string => driverByCode.get(driver)?.name || driver
  const providers = table({
    columns: ['方式信息', '类型 / 驱动', '连接配置', '凭据状态', '运行设置', '时间', '操作'],
    empty: '没有符合条件的流媒体方式',
  })
  let items: LiveProvider[] = []

  const filter = searchForm({
    fields: [
      { name: 'keyword', label: '编码 / 名称', placeholder: 'srs-main / 主线路' },
      { name: 'kind', label: '流媒体类型', kind: 'select', options: [{ value: '', label: '全部类型' }, 'RTMP', 'RTC'] },
      { name: 'driver', label: '驱动', kind: 'select', options: [
        { value: '', label: '全部驱动' },
        ...driverDefinitions.map(definition => ({ value: definition.code, label: definition.name })),
      ] },
      { name: 'lifecycle', label: '生命周期', kind: 'select', options: [{ value: '', label: '全部' }, { value: 'ACTIVE', label: '使用中' }, { value: 'RETIRED', label: '已退役' }] },
    ],
    onSearch: () => void load(),
    onReset: () => void load(),
  })

  function renderRows(): void {
    providers.setRows(items.map(item => {
      const connection = item.kind === 'RTMP'
        ? details([['App', item.app], ['推流', item.pushDomain], ['拉流', item.pullDomain]])
        : details([['App ID', item.agoraAppId], ['编码', item.codec.toUpperCase()], ['区域', item.region || '—'], ['Ingest', item.ingestDomain || '—']])
      const credentials = item.kind === 'RTMP'
        ? details([[item.driver === 'STATIC' ? '鉴权' : 'Secret', item.driver === 'STATIC' ? '无需' : item.secretSet ? item.secretMask || '已配置' : '未配置'], ['密钥 Key ID', item.credentialKeyId || '—']])
        : details([['App Certificate', item.appCertificateSet ? item.appCertificateMask || '已配置' : '未配置'], ['REST Customer', item.customerCredentialSet ? `${item.customerKeyMask || '***'} / ${item.customerSecretMask || '***'}` : '未配置'], ['密钥 Key ID', item.credentialKeyId || '—']])
      const healthTone = item.healthStatus === 'HEALTHY' ? 'success' : item.healthStatus === 'UNHEALTHY' ? 'danger' : 'neutral'
      return [
        details([['ID', item.id], ['编码', item.code], ['名称', item.name], ['版本', item.version]]),
        details([['类型', badge({ label: item.kind, tone: item.kind === 'RTC' ? 'info' : 'neutral' })], ['驱动', driverLabel(item.driver)]]),
        connection,
        credentials,
        details([
          ['状态', badge({ label: item.lifecycle === 'RETIRED' ? '已退役' : item.enabled ? '已启用' : '已停用', tone: item.lifecycle === 'RETIRED' ? 'neutral' : item.enabled ? 'success' : 'warning' })],
          ['默认', item.isDefault ? '是' : '否'], ['凭据 TTL', `${item.ttlSeconds} 秒`],
          ['健康', badge({ label: item.healthStatus === 'UNKNOWN' ? '未检查' : item.healthStatus === 'HEALTHY' ? '正常' : '异常', tone: healthTone })],
        ]),
        details([['创建', displayTime(item.createdAt)], ['更新', displayTime(item.updatedAt)], ['健康检查', displayTime(item.healthCheckedAt)]]),
        canManage && item.lifecycle === 'ACTIVE' ? actions(
          button({ label: '编辑', size: 'sm', variant: 'secondary', onClick: () => openEditor(item) }),
          button({ label: '退役', size: 'sm', variant: 'danger', onClick: () => openRetire(item) }),
        ) : '—',
      ]
    }))
    const active = items.filter(item => item.lifecycle === 'ACTIVE').length
    const enabled = items.filter(item => item.lifecycle === 'ACTIVE' && item.enabled).length
    const defaults = items.filter(item => item.lifecycle === 'ACTIVE' && item.isDefault).length
    if (metadataError) state.set(`Provider 列表可读，但驱动元数据加载失败：${metadataError}`, 'danger')
    else state.set(`流媒体方式 ${items.length} 个 · 驱动 ${driverDefinitions.length} 种 · 使用中 ${active} 个 · 已启用 ${enabled} 个 · 默认 ${defaults} 个`)
  }

  async function load(): Promise<void> {
    filter.setBusy(true)
    state.set('正在加载 Platform 流媒体 Provider 目录…')
    try {
      const values = filter.values()
      const query = new URLSearchParams()
      for (const key of ['keyword', 'kind', 'driver', 'lifecycle']) if (values[key]?.trim()) query.set(key, values[key].trim())
      items = await client.request<LiveProvider[]>(`${prefix}${query.size ? `?${query}` : ''}`)
      renderRows()
    } catch (error) {
      items = []
      providers.setRows([])
      state.set(`流媒体方式加载失败：${String(error)}`, 'danger')
    } finally { filter.setBusy(false) }
  }

  function driverSelector(): HostModalField {
    return {
      name: 'driver', label: '流媒体方式', kind: 'select', required: true, wide: true,
      options: [
        { value: '', label: '请选择流媒体方式' },
        ...driverDefinitions.map(definition => ({
          value: definition.code,
          label: `${definition.name} · ${definition.kind} · ${definition.pushTransport === 'OBS_RTMP' ? 'OBS 推流' : '浏览器 SDK 推流'}`,
        })),
      ],
    }
  }

  function editorFields(definition: DriverDefinition, current?: LiveProvider, carried: Record<string, string> = {}): {
    fields: HostModalField[]
    values: Record<string, string | number>
  } {
    const fields: HostModalField[] = [
      driverSelector(),
      { name: 'driverDescription', label: '驱动能力说明', disabled: true, wide: true },
      { name: 'code', label: '稳定方式编码', required: true, disabled: Boolean(current), mono: true, placeholder: 'srs-main' },
      { name: 'name', label: '展示名称', required: true, placeholder: definition.name },
    ]
    const values: Record<string, string | number> = {
      driver: definition.code, driverDescription: definition.description,
      code: carried.code || current?.code || '', name: carried.name || current?.name || '',
      isDefault: carried.isDefault || String(current?.isDefault ?? false),
    }
    const currentConfig: Record<string, string | number> = {
      app: current?.app || '', pushDomain: current?.pushDomain || '', pullDomain: current?.pullDomain || '',
      agoraAppId: current?.agoraAppId || '', codec: current?.codec || '', region: current?.region || '',
      ingestDomain: current?.ingestDomain || '', ttlSeconds: current?.ttlSeconds || 7200,
    }
    const emittedCredentials = new Set<CredentialGroup>()
    for (const field of definition.fields) {
      const label = `${field.group ? `${field.group} · ` : ''}${field.label}${field.advanced ? '（高级）' : ''}`
      if (field.secret && field.credential) {
        if (!emittedCredentials.has(field.credential)) {
          emittedCredentials.add(field.credential)
          const configured = credentialSet(current, field.credential)
          const required = definition.fields.some(item => item.credential === field.credential && item.required)
          const modeName = credentialModeName(field.credential)
          fields.push({
            name: modeName,
            label: `${field.group || '凭据'}处理（当前：${credentialStatus(current, field.credential)}）`, kind: 'select', required: true,
            options: [
              ...(configured ? [{ value: 'KEEP', label: '保留当前凭据' }] : []),
              { value: 'CLEAR', label: '不配置 / 清除' }, { value: 'REPLACE', label: '填写新凭据' },
            ],
          })
          const carriedMode = carried[modeName]
          values[modeName] = carriedMode === 'KEEP' || carriedMode === 'CLEAR' || carriedMode === 'REPLACE'
            ? carriedMode : configured ? 'KEEP' : required ? 'REPLACE' : 'CLEAR'
        }
        fields.push({
          name: field.key, label: `${label}${field.required ? '（必需）' : ''}`, type: 'password',
          placeholder: field.help || field.placeholder, autocomplete: 'new-password', wide: field.advanced,
        })
        values[field.key] = carried[field.key] || ''
        continue
      }
      const modalField: HostModalField = {
        name: field.key, label, required: field.required, placeholder: field.help || field.placeholder,
        wide: field.advanced, type: field.type === 'NUMBER' ? 'number' : 'text',
      }
      if (field.type === 'SELECT') {
        modalField.kind = 'select'
        modalField.options = field.options || []
      }
      if (field.type === 'NUMBER') {
        if (field.min) modalField.min = field.min
        if (field.max) modalField.max = field.max
      }
      fields.push(modalField)
      const carriedValue = carried[field.key]
      const carriedAllowed = !field.options?.length || field.options.some(option => option.value === carriedValue)
      values[field.key] = carriedValue && carriedAllowed ? carriedValue : currentConfig[field.key] || field.default || ''
    }
    fields.push(
      { name: 'isDefault', label: '默认方式', kind: 'select', required: true, options: [{ value: 'false', label: '否' }, { value: 'true', label: '是' }] },
    )
    return { fields, values }
  }

  function openEditor(current?: LiveProvider): void {
    if (!driverDefinitions.length) {
      state.set(`无法创建或编辑：驱动元数据不可用${metadataError ? `（${metadataError}）` : ''}`, 'danger')
      return
    }
    const editor = hostFormModal({
      title: `${current ? '编辑' : '新增'}流媒体方式`,
      fields: [driverSelector()],
      submitLabel: current ? '保存' : '创建',
      onChange: (values, field, modal) => {
        if (field !== 'driver') return
        const definition = driverByCode.get(values.driver as Driver)
        if (!definition) {
          modal.setFields([driverSelector()], { driver: '' }, `${current ? '编辑' : '新增'}流媒体方式`)
          return
        }
        const generated = editorFields(definition, current, values)
        modal.setFields(generated.fields, generated.values, `${current ? '编辑' : '新增'}流媒体方式 · ${definition.name}`)
      },
      onSubmit: (values, modal) => {
        const definition = driverByCode.get(values.driver as Driver)
        if (!definition) { modal.setError('请选择流媒体方式。'); return }
        const code = (current?.code || values.code).trim().toLowerCase()
        const enabled = current?.enabled ?? true
        const isDefault = values.isDefault === 'true'
        if (!codePattern.test(code)) { modal.setError('方式编码只能包含小写字母、数字、-、_，长度 1–32。'); return }
        if (!values.name.trim()) { modal.setError('请输入展示名称。'); return }
        if (isDefault && !enabled) { modal.setError('默认方式必须处于启用状态。'); return }
        for (const field of definition.fields.filter(item => !item.secret)) {
          const value = values[field.key]?.trim() || ''
          if (field.required && !value) { modal.setError(`请填写${field.label}。`); return }
          if (field.type === 'NUMBER' && value) {
            const number = Number(value)
            if (!Number.isSafeInteger(number) || (field.min && number < field.min) || (field.max && number > field.max)) {
              modal.setError(`${field.label}不符合驱动元数据约束。`); return
            }
          }
          if (field.options?.length && value && !field.options.some(option => option.value === value)) {
            modal.setError(`${field.label}不是该驱动支持的选项。`); return
          }
        }
        const emittedCredentials = new Set(definition.fields.flatMap(field => field.credential ? [field.credential] : []))
        for (const credential of emittedCredentials) {
          const credentialFields = definition.fields.filter(field => field.credential === credential)
          const mode = values[credentialModeName(credential)]
          const supplied = credentialFields.map(field => values[field.key]?.trim() || '')
          const complete = supplied.every(Boolean)
          if (supplied.some(Boolean) && !complete) { modal.setError(`${credentialFields.map(field => field.label).join('、')}必须完整填写。`); return }
          if (mode === 'REPLACE' && !complete) { modal.setError(`请选择保留/清除，或完整填写新的${credentialFields.map(field => field.label).join('、')}。`); return }
          const required = enabled && credentialFields.some(field => field.required)
          const ready = mode === 'REPLACE' ? complete : mode === 'KEEP' ? credentialSet(current, credential) : false
          if (required && !ready) { modal.setError(`${definition.name}启用时必须配置${credentialFields.map(field => field.label).join('、')}。`); return }
        }
        const fieldKeys = new Set(definition.fields.map(field => field.key))
        const value = (key: string, fallback = '') => fieldKeys.has(key) ? values[key]?.trim() || fallback : fallback
        const credentialChange = (credential: CredentialGroup) => {
          const credentialFields = definition.fields.filter(field => field.credential === credential)
          if (!credentialFields.length) return { mode: 'CLEAR', value: '' }
          const mode = values[credentialModeName(credential)]
          const primary = values[credentialFields[0].key]?.trim() || ''
          const secondary = credentialFields[1] ? values[credentialFields[1].key] || '' : ''
          return secretChange(mode, primary, credential === 'CUSTOMER_CREDENTIAL' ? { secondaryValue: secondary } : {})
        }
        const ttlSeconds = Number(value('ttlSeconds', String(current?.ttlSeconds || 7200)))
        modal.setBusy(true)
        client.request(`${prefix}/${encodeURIComponent(code)}`, {
          method: 'PUT',
          body: JSON.stringify({
            commandKey: crypto.randomUUID(), expectedVersion: current?.version ?? 0,
            name: values.name.trim(), driver: definition.code,
            app: value('app'), pushDomain: value('pushDomain'), pullDomain: value('pullDomain'),
            agoraAppId: value('agoraAppId'), codec: value('codec'), region: value('region'), ingestDomain: value('ingestDomain'),
            ttlSeconds, isDefault,
            secret: credentialChange('SECRET'), appCertificate: credentialChange('APP_CERTIFICATE'),
            customerCredential: credentialChange('CUSTOMER_CREDENTIAL'),
          }),
        }).then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    if (current) {
      const definition = driverByCode.get(current.driver)
      if (!definition) { state.set(`无法编辑 ${current.code}：驱动 ${current.driver} 的元数据不存在。`, 'danger'); return }
      const generated = editorFields(definition, current)
      editor.open(generated.values, `${current ? '编辑' : '新增'}流媒体方式 · ${definition.name}`)
      editor.setFields(generated.fields, generated.values, `编辑流媒体方式 · ${definition.name}`)
    } else {
      editor.open({ driver: '' })
    }
  }

  function openRetire(current: LiveProvider): void {
    const retire = hostFormModal({
      title: `退役流媒体方式 · ${current.name}`,
      fields: [{ name: 'confirm', label: '退役只阻止新分配和新场次；历史场次继续引用不可变版本。', kind: 'select', required: true, options: [{ value: '', label: '请选择' }, { value: current.code, label: `确认退役 ${current.code}` }] }],
      submitLabel: '退役',
      onSubmit: (values, modal) => {
        if (values.confirm !== current.code) { modal.setError('请选择确认项。'); return }
        modal.setBusy(true)
        client.request(`${prefix}/${encodeURIComponent(current.code)}/retire`, { method: 'POST', body: JSON.stringify({ commandKey: crypto.randomUUID(), expectedVersion: current.version }) })
          .then(() => { modal.close(); return load() }).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
      },
    })
    retire.open()
  }

  root.replaceChildren(page({
    showSummary: false,
    children: [
      searchCard(filter.element),
      dataCard({
        title: '流媒体方式目录',
        actions: [button({ label: '刷新', variant: 'secondary', onClick: () => void load() }), ...(canManage ? [button({ label: '新增方式', onClick: () => openEditor() })] : [])],
        status: state.element,
        body: providers.element,
      }),
    ],
  }))
  await load()
}
