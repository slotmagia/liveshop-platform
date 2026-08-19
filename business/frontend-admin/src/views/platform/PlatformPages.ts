import type { HostContext, HostHttpClient } from '@liveshops/host-sdk'
import { hostFormModal, randomUUID } from '@liveshops/host-sdk'
import { badge, button, card, create, dataCard, field, grid, navIcon, page, resolveGroupIconName, resolvePageIconName, searchCard, searchForm, statusLine, table, ui } from '@liveshops/design-tokens'

interface ReleaseInfo { version: string; digest: string }
interface ModuleInfo { id: string; name: string; activeVersion?: string; releases: ReleaseInfo[] }
type Surface = 'admin' | 'merch' | 'shop' | 'live'
interface PermissionDefinition { code: string; name: string; resource: string; action: string; description?: string }
interface AllowedRoute { methods: string[]; prefix: string; requiredPermissions: string[] }
interface ManifestField { name: string; label?: string; type: string; required?: boolean; description?: string }
interface HTTPResponse { status: number; description: string; fields: ManifestField[] }
interface NotificationDeclaration {
  eventKey: string
  title: string
  variables: string[]
  allowedChannels: string[]
  defaultDispatch: string
}
interface HTTPOperation {
  id: string
  method: string
  path: string
  summary: string
  description: string
  authentication: string
  idempotency: string
  requiredPermissions: string[]
  requestFields?: ManifestField[]
  responses?: HTTPResponse[]
  notifications?: NotificationDeclaration[]
}
interface HTTPRoute { surface: Surface | 'internal'; prefix: string; operations: HTTPOperation[] }
interface FrontendAction { id: string; label: string; description: string; invocation: string; target: string; requiredPermissions: string[] }
interface Contribution {
  id: string
  surface: Surface
  kind: string
  route?: string
  title: string
  description: string
  icon?: string
  sort?: number
  navigation?: { groupId: string; groupTitle: string; groupIcon?: string; groupSort: number }
  requiredPermissions: string[]
  allowedRoutes: AllowedRoute[]
  artifact: { type: string; name: string; version: string; entry: string; exportName?: string; integrity: string }
  frontend: { component: string; actions: FrontendAction[] }
}
interface CapabilityRelease {
  version: string
  digest: string
  active: boolean
  backend: { service: string; origin: string; httpRoutes: HTTPRoute[] }
  permissions: PermissionDefinition[]
  contributions: Contribution[]
}
interface ModuleCapabilityCatalog { id: string; name: string; activeVersion?: string; releases: CapabilityRelease[] }
interface CapabilityCatalog { revision: number; items: ModuleCapabilityCatalog[] }
interface ActivePage {
  module: ModuleCapabilityCatalog
  release: CapabilityRelease
  contribution: Contribution
  group: { groupId: string; groupTitle: string; groupIcon?: string; groupSort: number }
}
interface ActiveNavigationGroup {
  group: ActivePage['group']
  pages: ActivePage[]
}
interface AuditEvent { id: string; occurredAt: string; actorSubject: string; action: string; resourceType: string; resourceId: string; result: string; details: unknown }

const surfaces: Array<{ value: Surface; label: string }> = [
  { value: 'admin', label: '总后台' },
  { value: 'merch', label: '商户后台' },
  { value: 'shop', label: '店铺端' },
  { value: 'live', label: '直播端' },
]

const workbenchGroup = { groupId: 'host-workbench', groupTitle: '工作台', groupSort: 0 }

function pathWithinPrefix(path: string, prefix: string): boolean {
  const normalized = prefix.replace(/\/$/, '') || '/'
  return normalized === '/' || path === normalized || path.startsWith(`${normalized}/`)
}

function activePages(catalog: CapabilityCatalog, surface: Surface): ActivePage[] {
  const pages: ActivePage[] = []
  for (const module of catalog.items) {
    if (!module.activeVersion) continue
    const release = module.releases.find((item) => item.version === module.activeVersion)
    if (!release) continue
    for (const contribution of release.contributions) {
      if (contribution.surface !== surface || contribution.kind !== 'page') continue
      pages.push({ module, release, contribution, group: contribution.navigation || workbenchGroup })
    }
  }
  return pages.sort((left, right) => (
    left.group.groupSort - right.group.groupSort
    || left.group.groupId.localeCompare(right.group.groupId)
    || (left.contribution.sort || 0) - (right.contribution.sort || 0)
    || left.contribution.id.localeCompare(right.contribution.id)
  ))
}

function linkedOperations(item: ActivePage): HTTPOperation[] {
  const allowed = item.contribution.allowedRoutes
  return item.release.backend.httpRoutes
    .filter((route) => route.surface === item.contribution.surface)
    .flatMap((route) => route.operations)
    .filter((operation) => allowed.some((route) => (
      route.methods.some((method) => method.toUpperCase() === operation.method.toUpperCase())
      && pathWithinPrefix(operation.path, route.prefix)
    )))
}

function activeNavigationGroups(items: ActivePage[]): ActiveNavigationGroup[] {
  const groups = new Map<string, ActiveNavigationGroup>()
  for (const item of items) {
    const key = item.group.groupId
    const current = groups.get(key)
    if (current) current.pages.push(item)
    else groups.set(key, { group: item.group, pages: [item] })
  }
  return [...groups.values()].sort((left, right) => (
    left.group.groupSort - right.group.groupSort
    || left.group.groupId.localeCompare(right.group.groupId)
  ))
}

interface NotifyChannelPolicy { enabled: boolean; templateCode?: string }
interface NotifyEventRow {
  eventKey: string
  title: string
  variables: string[]
  allowedChannels: string[]
  defaultDispatch: string
  dispatchMode: string
  delaySeconds: number
  channels: Record<string, NotifyChannelPolicy>
  policyVersion: number
}
interface NotifyTemplateOption { code: string; channel: string; lifecycle: string; variables?: string[] }

type RegistryNotify = { client: HostHttpClient; canManage: boolean; fail: (error: unknown) => void }
type RegistryActionIcon = 'plus' | 'refresh-cw' | 'pencil' | 'trash-2' | 'chevron-down' | 'chevron-right' | 'corner-down-right'

function registryIcon(name: string, className = '', fallback = 'layout-grid'): SVGElement {
  return navIcon(name, className, fallback)
}

function compactAction(label: string, iconName: RegistryActionIcon, onClick: () => void): HTMLButtonElement {
  const control = create('button', 'registry-tree__compact-action')
  control.type = 'button'
  control.append(registryIcon(iconName), create('span', undefined, label))
  control.addEventListener('click', onClick)
  return control
}

function iconAction(label: string, iconName: RegistryActionIcon, onClick: () => void): HTMLButtonElement {
  const control = create('button', 'registry-tree__icon-action')
  control.type = 'button'
  control.title = label
  control.setAttribute('aria-label', label)
  control.append(registryIcon(iconName))
  control.addEventListener('click', onClick)
  return control
}

function enabledAction(onClick: () => void): HTMLButtonElement {
  const control = create('button', 'registry-tree__enabled', '启用')
  control.type = 'button'
  control.title = '查看当前启用状态'
  control.addEventListener('click', onClick)
  return control
}

function fieldTable(title: string, fields: ManifestField[]): HTMLElement {
  const section = create('section', 'registry-tree__field-section')
  section.append(create('p', 'registry-tree__field-title', title))
  if (!fields.length) {
    section.append(create('p', 'registry-tree__field-empty', '未声明'))
    return section
  }
  const body = create('div', 'registry-tree__field-table')
  const head = create('div', 'registry-tree__field-row registry-tree__field-row--head')
  head.append(create('span', undefined, '字段名'), create('span', undefined, '显示名'), create('span', undefined, '类型'))
  body.append(head)
  for (const field of fields) {
    const row = create('div', 'registry-tree__field-row')
    row.append(
      create('code', undefined, field.name),
      create('span', undefined, field.label || field.description || '—'),
      badge({ label: field.type || 'string' }),
    )
    body.append(row)
  }
  section.append(body)
  return section
}

function operationDetail(operation: HTTPOperation): HTMLElement {
  const detail = create('div', 'registry-tree__operation-detail')
  const responseFields = (operation.responses || []).flatMap((response) => response.fields || [])
  detail.append(
    fieldTable('请求参数', operation.requestFields || []),
    fieldTable('返回值字段', responseFields),
  )
  return detail
}

function navigationTree(
  items: ActivePage[],
  showDetail: (item: ActivePage) => void,
  showManifestAction: (action: string, target: string) => void,
  notify?: RegistryNotify,
): HTMLElement {
  const tree = create('div', 'registry-tree')
  const groups = activeNavigationGroups(items)
  if (!groups.length) {
    tree.append(create('div', 'registry-tree__empty', '当前交付端没有活动菜单。请先检查模块 Manifest 是否已注册并激活。'))
    return tree
  }

  for (const entry of groups) {
    const section = create('section', 'registry-tree__group')
    const groupHeader = create('div', 'registry-tree__group-header')
    const groupIdentity = create('div', 'registry-tree__identity')
    groupIdentity.append(
      registryIcon(resolveGroupIconName(entry.group.groupId, entry.group.groupIcon), 'registry-tree__nav-icon', 'folder-kanban'),
      create('strong', undefined, entry.group.groupTitle),
      badge({ label: '目录' }),
      create('code', 'registry-tree__code registry-tree__code--subtle', entry.group.groupId),
    )
    const groupMeta = create('div', 'registry-tree__meta')
    groupMeta.append(
      compactAction('新增菜单', 'plus', () => showManifestAction('新增菜单', entry.group.groupTitle)),
      enabledAction(() => showManifestAction('启停目录', entry.group.groupTitle)),
      iconAction(`编辑目录 ${entry.group.groupTitle}`, 'pencil', () => showManifestAction('编辑目录', entry.group.groupTitle)),
      iconAction(`删除目录 ${entry.group.groupTitle}`, 'trash-2', () => showManifestAction('删除目录', entry.group.groupTitle)),
    )
    groupHeader.append(groupIdentity, groupMeta)

    const groupBody = create('div', 'registry-tree__group-body')
    for (const item of entry.pages) {
      const menu = create('article', 'registry-tree__menu')
      const menuRow = create('div', 'registry-tree__menu-row')
      const menuIdentity = create('div', 'registry-tree__identity')
      menuIdentity.append(
        registryIcon(resolvePageIconName(item.contribution.id, item.contribution.icon), 'registry-tree__nav-icon'),
        create('strong', undefined, item.contribution.title),
        badge({ label: '菜单', tone: 'info' }),
        create('code', 'registry-tree__code', item.contribution.route || '—'),
      )
      const menuMeta = create('div', 'registry-tree__meta')
      menuMeta.append(
        create('code', 'registry-tree__code registry-tree__code--subtle', item.contribution.id),
        enabledAction(() => showManifestAction('启停菜单', item.contribution.title)),
        iconAction(`编辑菜单 ${item.contribution.title}`, 'pencil', () => showDetail(item)),
        iconAction(`删除菜单 ${item.contribution.title}`, 'trash-2', () => showManifestAction('删除菜单', item.contribution.title)),
      )
      menuRow.append(menuIdentity, menuMeta)
      menu.append(menuRow)

      const operations = linkedOperations(item)
      const operationList = create('div', 'registry-tree__operations')
      if (!operations.length) {
        operationList.append(create('div', 'registry-tree__operation registry-tree__operation--empty', '该菜单没有关联 HTTP 接口'))
      } else {
        for (const operation of operations) {
          let expanded = false
          const operationRow = create('div', 'registry-tree__operation')
          const operationLine = create('div', 'registry-tree__operation-line')
          const detail = operationDetail(operation)
          detail.hidden = true
          const expand = iconAction(`展开接口 ${operation.summary || operation.id}`, 'chevron-right', () => {
            expanded = !expanded
            detail.hidden = !expanded
            expand.replaceChildren(registryIcon(expanded ? 'chevron-down' : 'chevron-right'))
            expand.setAttribute('aria-expanded', String(expanded))
            expand.setAttribute('aria-label', `${expanded ? '收起' : '展开'}接口 ${operation.summary || operation.id}`)
          })
          expand.classList.add('registry-tree__expand')
          expand.setAttribute('aria-expanded', 'false')
          const operationIdentity = create('div', 'registry-tree__operation-identity')
          operationIdentity.append(
            expand,
            registryIcon('corner-down-right', 'registry-tree__branch'),
            create('strong', undefined, operation.summary || operation.id),
            create('code', 'registry-tree__code', operation.path),
          )
          const operationMeta = create('div', 'registry-tree__meta')
          operationMeta.append(
            create('code', 'registry-tree__code registry-tree__code--subtle', operation.id),
            enabledAction(() => showManifestAction('启停接口', operation.summary || operation.id)),
            iconAction(`编辑接口 ${operation.summary || operation.id}`, 'pencil', () => showManifestAction('编辑接口', operation.summary || operation.id)),
            iconAction(`删除接口 ${operation.summary || operation.id}`, 'trash-2', () => showManifestAction('删除接口', operation.summary || operation.id)),
          )
          operationLine.append(operationIdentity, operationMeta)
          operationRow.append(operationLine, detail)
          const declarations = operation.notifications || []
          if (declarations.length) {
            for (const declaration of declarations) {
              const eventRow = create('div', 'registry-tree__operation registry-tree__operation--event')
              const eventLine = create('div', 'registry-tree__operation-line')
              const eventIdentity = create('div', 'registry-tree__operation-identity')
              eventIdentity.append(
                registryIcon('corner-down-right', 'registry-tree__branch'),
                create('strong', undefined, declaration.title || declaration.eventKey),
                badge({ label: '事件', tone: 'warning' }),
                create('code', 'registry-tree__code', declaration.eventKey),
              )
              const eventMeta = create('div', 'registry-tree__meta')
              eventMeta.append(create('code', 'registry-tree__code registry-tree__code--subtle', `${declaration.defaultDispatch} · ${(declaration.allowedChannels || []).join('/')}`))
              if (notify?.canManage) {
                eventMeta.append(compactAction('配置规则', 'pencil', () => void openCatalogEventPolicy(notify, declaration)))
              }
              eventLine.append(eventIdentity, eventMeta)
              eventRow.append(eventLine)
              operationRow.append(eventRow)
            }
          }
          operationList.append(operationRow)
        }
      }
      operationList.append(compactAction('新增接口', 'plus', () => showManifestAction('新增接口', item.contribution.title)))
      menu.append(operationList)
      groupBody.append(menu)
    }
    section.append(groupHeader, groupBody)
    tree.append(section)
  }
  return tree
}

function inDateRange(value: string, from: string, to: string): boolean {
  if (!from && !to) return true
  const time = new Date(value).getTime()
  if (Number.isNaN(time)) return false
  if (from) {
    const start = new Date(`${from}T00:00:00`).getTime()
    if (time < start) return false
  }
  if (to) {
    const end = new Date(`${to}T23:59:59.999`).getTime()
    if (time > end) return false
  }
  return true
}

function includesText(haystack: string, needle: string): boolean {
  if (!needle.trim()) return true
  return haystack.toLowerCase().includes(needle.trim().toLowerCase())
}

async function openCatalogEventPolicy(notify: RegistryNotify, declaration: NotificationDeclaration): Promise<void> {
  const modes = [
    { value: 'SYNC', label: '同步' },
    { value: 'ASYNC', label: '异步' },
    { value: 'SCHEDULED', label: '定时' },
  ]
  let item: NotifyEventRow
  let templates: NotifyTemplateOption[] = []
  try {
    item = await notify.client.request<NotifyEventRow>(`/admin/platform/notify-events/${encodeURIComponent(declaration.eventKey)}`)
    templates = await notify.client.request<NotifyTemplateOption[]>('/admin/platform/notify-templates')
  } catch (error) {
    notify.fail(error)
    return
  }
  const eventVariables = item.variables || declaration.variables || []
  const compatible = (channel: string) => templates.filter(template => (
    template.channel === channel
    && template.lifecycle !== 'RETIRED'
    && (template.variables || []).every(name => eventVariables.includes(name))
  ))
  const fields = [
    { name: 'eventVariables', label: '事件变量（模板占位符必须是其子集）', disabled: true, wide: true },
    { name: 'dispatchMode', label: '投递模式', kind: 'select' as const, required: true, options: modes },
    { name: 'delaySeconds', label: '定时延迟（秒）', type: 'number', placeholder: '仅 SCHEDULED' },
    ...declaration.allowedChannels.flatMap(channel => ([
      { name: `channel_${channel}`, label: `${channel} 渠道`, kind: 'select' as const, required: true, options: [{ value: 'true', label: '开启' }, { value: 'false', label: '关闭' }] },
      {
        name: `template_${channel}`,
        label: `${channel} 模板`,
        kind: 'select' as const,
        options: [
          { value: '', label: '不选择模板' },
          ...compatible(channel).map(template => ({ value: template.code, label: template.code })),
        ],
      },
    ])),
  ]
  const values: Record<string, string> = {
    eventVariables: eventVariables.map(name => `{{${name}}}`).join(' ') || '（无）',
    dispatchMode: item.dispatchMode || declaration.defaultDispatch,
    delaySeconds: String(item.delaySeconds || 0),
  }
  for (const channel of declaration.allowedChannels) {
    values[`channel_${channel}`] = String(Boolean(item.channels?.[channel]?.enabled))
    values[`template_${channel}`] = item.channels?.[channel]?.templateCode || ''
  }
  const editor = hostFormModal({
    title: `通知规则 · ${declaration.title}`,
    fields,
    submitLabel: '保存',
    onSubmit: (form, modal) => {
      const dispatchMode = form.dispatchMode
      const delaySeconds = Number(form.delaySeconds || 0)
      if (dispatchMode === 'SCHEDULED' && (!Number.isInteger(delaySeconds) || delaySeconds < 0 || delaySeconds > 2592000)) {
        modal.setError('定时延迟须为 0 到 2592000 的整数秒。'); return
      }
      const channels: Record<string, NotifyChannelPolicy> = {}
      for (const channel of declaration.allowedChannels) {
        const enabled = form[`channel_${channel}`] === 'true'
        const templateCode = (form[`template_${channel}`] || '').trim()
        if (enabled && !templateCode) { modal.setError(`${channel} 开启时必须选择模板。`); return }
        channels[channel] = { enabled, templateCode }
      }
      modal.setBusy(true)
      notify.client.request(`/admin/platform/notify-events/${encodeURIComponent(declaration.eventKey)}/policy`, {
        method: 'PUT',
        body: JSON.stringify({
          commandKey: randomUUID(), expectedVersion: item.policyVersion, dispatchMode,
          delaySeconds: dispatchMode === 'SCHEDULED' ? delaySeconds : 0, channels,
        }),
      }).then(() => modal.close()).catch(error => modal.setError(String(error))).finally(() => modal.setBusy(false))
    },
  })
  editor.open(values)
}

export async function startRegistry(root: HTMLElement, client: HostHttpClient, context?: HostContext) {
  const status = statusLine()
  const fail = (error: unknown) => status.set(error instanceof Error ? error.message : String(error), 'danger')
  let modules: ModuleInfo[] = []
  let catalog: CapabilityCatalog = { revision: 0, items: [] }
  let filters: Record<string, string> = {}
  let surface: Surface = 'admin'
  const moduleList = grid()
  const navigation = create('div')
  const manifestGuidance = '当前菜单目录来自活动 Module Manifest，不能在运行时直接覆盖。请在所属模块修改导航或接口定义，发布新版本后由 Registry 激活。'
  const detailModal = hostFormModal({
    title: '页面能力详情',
    fields: [
      { name: 'overview', label: '能力摘要', kind: 'textarea', rows: 5, wide: true, mono: true, disabled: true },
      { name: 'page', label: '页面贡献', kind: 'textarea', rows: 8, wide: true, mono: true, disabled: true },
      { name: 'routes', label: 'Module Capability 路由范围', kind: 'textarea', rows: 6, wide: true, mono: true, disabled: true },
      { name: 'operations', label: '关联 HTTP Operations', kind: 'textarea', rows: 12, wide: true, mono: true, disabled: true },
      { name: 'actions', label: 'Frontend Actions', kind: 'textarea', rows: 7, wide: true, mono: true, disabled: true },
    ],
    submitLabel: '确定',
    cancelLabel: '关闭',
    onSubmit: (_values, current) => current.close(),
  })
  const manifestActionModal = hostFormModal({
    title: '活动 Module Manifest',
    fields: [
      { name: 'target', label: '操作对象', wide: true, disabled: true },
      {
        name: 'guidance',
        label: '处理方式',
        kind: 'textarea',
        rows: 6,
        wide: true,
        disabled: true,
        value: manifestGuidance,
      },
    ],
    submitLabel: '知道了',
    cancelLabel: '关闭',
    onSubmit: (_values, current) => current.close(),
  })

  const showDetail = (item: ActivePage) => {
    const contribution = item.contribution
    const operations = linkedOperations(item)
    detailModal.open({
      overview: [
        `Surface: ${contribution.surface}`,
        `活动模块: ${item.module.id}@${item.release.version}`,
        `导航分组: ${item.group.groupTitle} (${item.group.groupId})`,
        `目录图标: ${resolveGroupIconName(item.group.groupId, item.group.groupIcon)}`,
        `菜单图标: ${resolvePageIconName(item.contribution.id, item.contribution.icon)}`,
        `关联 HTTP 能力: ${operations.length}`,
      ].join('\n'),
      page: [
        `Contribution ID: ${contribution.id}`,
        `页面路由: ${contribution.route || '—'}`,
        `说明: ${contribution.description || '—'}`,
        `组件: ${contribution.frontend.component}`,
        `制品: ${contribution.artifact.name}@${contribution.artifact.version} (${contribution.artifact.type})`,
        `页面权限: ${contribution.requiredPermissions.join(', ') || '—'}`,
      ].join('\n'),
      routes: contribution.allowedRoutes.map((route) => (
        `${route.methods.join(', ')} ${route.prefix}\n权限: ${route.requiredPermissions.join(', ') || '—'}`
      )).join('\n\n') || '—',
      operations: operations.map((operation) => (
        `${operation.method} ${operation.path}\n${operation.summary} (${operation.id})\n认证 / 幂等: ${operation.authentication} / ${operation.idempotency}\n权限: ${operation.requiredPermissions.join(', ') || '—'}`
      )).join('\n\n') || '—',
      actions: contribution.frontend.actions.map((action) => (
        `${action.label} (${action.id})\n${action.invocation} → ${action.target}\n权限: ${action.requiredPermissions.join(', ') || '—'}`
      )).join('\n\n') || '—',
    }, contribution.title)
  }

  const showManifestAction = (action: string, target: string) => {
    manifestActionModal.open({ target, guidance: manifestGuidance }, action)
  }

  const visibleModules = () => modules.filter((item) => (
    includesText(item.id, filters.keyword || '')
    || includesText(item.name, filters.keyword || '')
  ) && (
    !filters.active
    || (filters.active === 'ACTIVE' ? Boolean(item.activeVersion) : !item.activeVersion)
  ))

  const moduleCard = (item: ModuleInfo) => {
    const controlPlane = item.id === 'platform'
    const release = field({
      name: 'version',
      label: '发布版本',
      kind: 'select',
      options: item.releases.map((entry) => entry.version),
      value: item.activeVersion,
      wide: true,
    })
    const activate = button({
      label: '激活版本',
      onClick: () => void client.request(`/admin/platform/registry/modules/${encodeURIComponent(item.id)}/activate`, {
        method: 'POST',
        body: JSON.stringify({ version: release.control.value }),
      }).then(load).catch(fail),
    })
    const deactivate = button({
      label: controlPlane ? '控制面不可停用' : '停用',
      variant: 'secondary',
      disabled: !item.activeVersion || controlPlane,
      onClick: () => void client.request(`/admin/platform/registry/modules/${encodeURIComponent(item.id)}/deactivate`, { method: 'DELETE' })
        .then(load).catch(fail),
    })
    const actions = create('div', ui.actions)
    actions.append(activate, deactivate)
    return card({
      title: item.name,
      headerExtra: create('code', undefined, item.id),
      body: [
        create('p', undefined, `当前版本：${item.activeVersion || '未激活'}`),
        release.element,
        actions,
      ],
    })
  }

  const render = () => {
    const items = visibleModules()
    moduleList.replaceChildren(...items.map((item) => moduleCard(item)))
    const pages = activePages(catalog, surface)
    navigation.replaceChildren(navigationTree(pages, showDetail, showManifestAction, {
      client,
      canManage: Boolean(context?.permissions?.includes('platform.notify-event.manage')),
      fail,
    }))
    const surfaceLabel = surfaces.find((item) => item.value === surface)?.label || surface
    status.set(`Registry revision ${catalog.revision}；${surfaceLabel}活动页面 ${pages.length} 个；显示模块 ${items.length} / ${modules.length} 个`)
  }

  const load = async () => {
    ;[modules, catalog] = await Promise.all([
      client.request<ModuleInfo[]>('/admin/platform/registry/modules'),
      client.request<CapabilityCatalog>('/admin/platform/registry/capabilities'),
    ])
    render()
  }

  const filtersForm = searchForm({
    fields: [
      { name: 'surface', label: '交付端', kind: 'select', options: surfaces, value: surface },
      { name: 'keyword', label: '模块', placeholder: 'module id / name' },
      { name: 'active', label: '激活状态', kind: 'select', options: [{ value: '', label: '全部' }, { value: 'ACTIVE', label: '已激活' }, { value: 'INACTIVE', label: '未激活' }] },
    ],
    onSearch: (values) => {
      if (surfaces.some((item) => item.value === values.surface)) surface = values.surface as Surface
      filters = values
      render()
    },
    onReset: () => {
      surface = 'admin'
      filters = {}
      render()
    },
  })

  const catalogPanel = create('div', 'registry-catalog')
  catalogPanel.append(
    create('p', 'registry-catalog__description', '四级管理：目录 → 菜单 → 接口 → 事件。事件来自模块 Manifest 声明，不能在此新建。有权限时可配置投递规则并选择模板。'),
    navigation,
  )
  const moduleBody = create('div', ui.cardBody)
  moduleBody.append(moduleList)

  root.replaceChildren(page({
    showSummary: false,
    children: [
      searchCard(filtersForm.element),
      dataCard({
        title: '菜单与接口目录',
        actions: [compactAction('刷新', 'refresh-cw', () => void load().catch(fail)), compactAction('新增目录', 'plus', () => showManifestAction('新增目录', '活动菜单目录'))],
        status: status.element,
        body: catalogPanel,
      }),
      dataCard({ title: '模块版本', body: moduleBody }),
    ],
  }))
  await load()
}

export async function startAudit(root: HTMLElement, client: HostHttpClient) {
  const status = statusLine()
  const fail = (error: unknown) => status.set(error instanceof Error ? error.message : String(error), 'danger')
  const events = table({ columns: ['时间', '操作者', '动作', '资源', '结果'] })
  let items: AuditEvent[] = []
  let filters: Record<string, string> = {}

  const render = () => {
    const visible = items.filter((item) => (
      includesText(item.actorSubject, filters.actorSubject || '')
      && includesText(item.action, filters.action || '')
      && includesText(`${item.resourceType}:${item.resourceId}`, filters.resource || '')
      && (!filters.result || item.result === filters.result)
      && inDateRange(item.occurredAt, filters.occurredAtFrom || '', filters.occurredAtTo || '')
    ))
    events.setRows(visible.map((item) => [
      new Date(item.occurredAt).toLocaleString(),
      item.actorSubject,
      item.action,
      `${item.resourceType}:${item.resourceId}`,
      badge({ label: item.result, tone: item.result === 'SUCCESS' || item.result === 'SUCCEEDED' ? 'success' : 'danger' }),
    ]))
    status.set(`显示 ${visible.length} / ${items.length} 条审计事件`)
  }

  const load = async () => {
    items = await client.request<AuditEvent[]>('/admin/platform/audit/events?limit=100')
    render()
  }

  const filtersForm = searchForm({
    fields: [
      { name: 'actorSubject', label: '操作者' },
      { name: 'action', label: '动作', placeholder: 'identity.login' },
      { name: 'resource', label: '资源', placeholder: 'identity.account:…' },
      { name: 'result', label: '结果', kind: 'select', options: [{ value: '', label: '全部' }, 'SUCCEEDED', 'DENIED', 'FAILED'] },
      { name: 'occurredAt', label: '发生时间', kind: 'date-range' },
    ],
    onSearch: (values) => {
      filters = values
      render()
    },
    onReset: () => {
      filters = {}
      render()
    },
  })

  root.replaceChildren(page({
    showSummary: false,
    children: [
      searchCard(filtersForm.element),
      dataCard({ title: '审计事件', actions: button({ label: '刷新', variant: 'secondary', onClick: () => void load().catch(fail) }), status: status.element, body: events.element }),
    ],
  }))
  await load()
}
