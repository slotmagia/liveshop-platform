export const HOST_PROTOCOL = 2 as const
export type Surface = 'admin' | 'merch' | 'shop' | 'live'
export type ContributionKind = 'page' | 'slot' | 'widget' | 'action'
export interface AllowedRoute {
  methods: Array<'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'>
  prefix: string
  requiredPermissions: string[]
}

export interface Artifact {
  type: 'iframe' | 'remote-esm'
  name: string
  version: string
  entry: string
  exportName?: string
  integrity: string
}

export interface Contribution {
  id: string
  surface: Surface
  kind: ContributionKind
  route?: string
  outlet?: string
  title: string
  description: string
  icon?: string
  sort?: number
  navigation?: {
    groupId: string
    groupTitle: string
    groupIcon?: string
    groupSort: number
  }
  requiredPermissions: string[]
  allowedRoutes: AllowedRoute[]
  artifact: Artifact
}

export interface RuntimeContribution {
  moduleId: string
  moduleVersion: string
  contribution: Contribution
}

export interface HostContext {
  protocol: typeof HOST_PROTOCOL
  surface: Surface
  moduleId: string
  moduleVersion: string
  contributionId: string
  moduleToken: string
  gatewayBaseUrl: string
  locale: string
  permissions: string[]
  tenant?: { merchantId: number; shopId: number }
  theme: { mode: 'light' | 'dark' }
}

export interface HostModalSelectOption {
  value: string | number
  label?: string
}

export interface HostModalTreeNode {
  id: string
  label: string
  value?: string | number
  description?: string
  children?: HostModalTreeNode[]
}

/** Serializable form field rendered by the Host in its top-level document. */
export interface HostModalField {
  name: string
  label?: string
  kind?: 'input' | 'select' | 'textarea' | 'date-range' | 'checkbox-tree'
  type?: string
  value?: string | number
  placeholder?: string
  required?: boolean
  disabled?: boolean
  wide?: boolean
  span?: 1 | 2
  mono?: boolean
  rows?: number
  options?: Array<HostModalSelectOption | string>
  min?: number | string
  max?: number | string
  step?: number | string
  minLength?: number
  maxLength?: number
  autocomplete?: string
  from?: string
  to?: string
  fromValue?: string | number
  toValue?: string | number
  separator?: string
  tree?: HostModalTreeNode[]
  columns?: 1 | 2 | 3
  empty?: string
}

export interface HostFormModalOpenMessage {
  type: 'LIVESHOP_HOST_FORM_MODAL_OPEN'
  protocol: typeof HOST_PROTOCOL
  requestId: string
  title: string
  fields: HostModalField[]
  values: Record<string, string | number | null | undefined>
  submitLabel: string
  cancelLabel: string
  busy: boolean
}

export interface HostFormModalCommandMessage {
  type: 'LIVESHOP_HOST_FORM_MODAL_COMMAND'
  protocol: typeof HOST_PROTOCOL
  requestId: string
  command: 'close' | 'set-busy' | 'set-title' | 'set-error' | 'set-fields'
  busy?: boolean
  title?: string
  message?: string
  fields?: HostModalField[]
  values?: Record<string, string | number | null | undefined>
}

export interface HostFormModalSubmitMessage {
  type: 'LIVESHOP_HOST_FORM_MODAL_SUBMIT'
  protocol: typeof HOST_PROTOCOL
  requestId: string
  values: Record<string, string>
}

export interface HostFormModalChangeMessage {
  type: 'LIVESHOP_HOST_FORM_MODAL_CHANGE'
  protocol: typeof HOST_PROTOCOL
  requestId: string
  field: string
  values: Record<string, string>
}

export interface HostFormModalClosedMessage {
  type: 'LIVESHOP_HOST_FORM_MODAL_CLOSED'
  protocol: typeof HOST_PROTOCOL
  requestId: string
  reason: 'dismissed' | 'programmatic' | 'replaced' | 'owner-unmounted'
}

export type HostFormModalRequestMessage = HostFormModalOpenMessage | HostFormModalCommandMessage
export type HostFormModalResponseMessage = HostFormModalSubmitMessage | HostFormModalChangeMessage | HostFormModalClosedMessage

export interface HostOverlayMessage {
  type: 'LIVESHOP_HOST_OVERLAY_OPEN' | 'LIVESHOP_HOST_OVERLAY_CLOSE'
  protocol: typeof HOST_PROTOCOL
  requestId: string
}

export interface HostModuleUploadRequestMessage {
  type: 'LIVESHOP_HOST_MODULE_UPLOAD'
  protocol: typeof HOST_PROTOCOL
  requestId: string
  moduleId: string
  contributionId: string
  path: string
  file: File
  fields: Record<string, string>
}

export interface HostModuleUploadResponseMessage {
  type: 'LIVESHOP_HOST_MODULE_UPLOAD_RESULT'
  protocol: typeof HOST_PROTOCOL
  requestId: string
  ok: boolean
  status: number
  data?: unknown
  message?: string
}

export interface HostModuleSizeMessage {
  type: 'LIVESHOP_MODULE_SIZE'
  protocol: typeof HOST_PROTOCOL
  height: number
}

export interface HostNotifyMessage {
  type: 'LIVESHOP_HOST_NOTIFY'
  protocol: typeof HOST_PROTOCOL
  text: string
  tone: 'success' | 'warning' | 'danger' | 'info'
}

export interface HostHttpClient {
  request<T>(path: string, init?: RequestInit): Promise<T>
  requestRaw(path: string, init?: RequestInit): Promise<Response>
}

export interface RemoteModuleContext extends HostContext {
  api: HostHttpClient
  navigate(path: string): void
  events: EventTarget
}

export interface RemoteModule {
  mount(container: HTMLElement, context: RemoteModuleContext): void | Promise<void>
  unmount?(container: HTMLElement): void | Promise<void>
}

let currentContext: HostContext | undefined
let hostOrigin = ''
let modalBridgeInstalled = false
let moduleUploadBridgeInstalled = false
let moduleSizeBridgeInstalled = false

interface HostFormModalHandler {
  submit(values: Record<string, string>): void
  change(values: Record<string, string>, field: string): void
  closed(reason: HostFormModalClosedMessage['reason']): void
}

const hostFormModalHandlers = new Map<string, HostFormModalHandler>()
const hostModuleUploadHandlers = new Map<string, { resolve(value: unknown): void; reject(reason: Error): void }>()

function installHostModuleSizeBridge(): void {
  if (moduleSizeBridgeInstalled || window.parent === window || !hostOrigin) return
  moduleSizeBridgeInstalled = true
  let scheduled = 0
  let lastHeight = 0
  const report = () => {
    scheduled = 0
    const height = Math.ceil(Math.max(document.body?.scrollHeight || 0, document.documentElement.scrollHeight || 0))
    if (height <= 0 || height === lastHeight) return
    lastHeight = height
    window.parent.postMessage({ type: 'LIVESHOP_MODULE_SIZE', protocol: HOST_PROTOCOL, height } satisfies HostModuleSizeMessage, hostOrigin)
  }
  const schedule = () => {
    if (!scheduled) scheduled = window.requestAnimationFrame(report)
  }
  const observer = new ResizeObserver(schedule)
  observer.observe(document.documentElement)
  if (document.body) observer.observe(document.body)
  window.addEventListener('load', schedule, { once: true })
  schedule()
}

function installHostModuleUploadBridge(): void {
  if (moduleUploadBridgeInstalled) return
  moduleUploadBridgeInstalled = true
  window.addEventListener('message', (event: MessageEvent) => {
    if (event.source !== window.parent || event.origin !== hostOrigin) return
    const message = event.data as HostModuleUploadResponseMessage
    if (message?.protocol !== HOST_PROTOCOL || message.type !== 'LIVESHOP_HOST_MODULE_UPLOAD_RESULT' || typeof message.requestId !== 'string') return
    const handler = hostModuleUploadHandlers.get(message.requestId)
    if (!handler) return
    hostModuleUploadHandlers.delete(message.requestId)
    if (message.ok) handler.resolve(message.data)
    else handler.reject(new Error(message.message || `module upload failed with HTTP ${message.status}`))
  })
}

/**
 * Ask the top-level Host to invoke another module's action contribution.
 * The Host obtains a separate Identity-issued Module Capability for that contribution; the
 * caller's Catalog token is never widened or replayed against Media.
 */
export function hostModuleUpload<T>(options: {
  moduleId: string
  contributionId: string
  path: string
  file: File
  fields?: Record<string, string>
}): Promise<T> {
  if (window.parent === window || !hostOrigin) return Promise.reject(new Error('module upload requires an active iframe Host connection'))
  if (!options.path.startsWith('/')) return Promise.reject(new Error('module upload path must be absolute'))
  installHostModuleUploadBridge()
  const requestId = crypto.randomUUID()
  return new Promise<T>((resolve, reject) => {
    const timer = window.setTimeout(() => {
      hostModuleUploadHandlers.delete(requestId)
      reject(new Error('module upload timed out'))
    }, 30_000)
    hostModuleUploadHandlers.set(requestId, {
      resolve(value) { window.clearTimeout(timer); resolve(value as T) },
      reject(error) { window.clearTimeout(timer); reject(error) },
    })
    window.parent.postMessage({
      type: 'LIVESHOP_HOST_MODULE_UPLOAD', protocol: HOST_PROTOCOL, requestId,
      moduleId: options.moduleId, contributionId: options.contributionId,
      path: options.path, file: options.file, fields: options.fields || {},
    } satisfies HostModuleUploadRequestMessage, hostOrigin)
  })
}

function installHostModalBridge(): void {
  if (modalBridgeInstalled) return
  modalBridgeInstalled = true
  window.addEventListener('message', (event: MessageEvent) => {
    if (event.source !== window.parent || event.origin !== hostOrigin) return
    if (event.data?.protocol !== HOST_PROTOCOL || typeof event.data?.requestId !== 'string') return
    const handler = hostFormModalHandlers.get(event.data.requestId)
    if (!handler) return
    if (event.data.type === 'LIVESHOP_HOST_FORM_MODAL_SUBMIT' && event.data.values && typeof event.data.values === 'object') {
      handler.submit(event.data.values as Record<string, string>)
      return
    }
    if (event.data.type === 'LIVESHOP_HOST_FORM_MODAL_CHANGE' && event.data.values && typeof event.data.values === 'object') {
      handler.change(event.data.values as Record<string, string>, typeof event.data.field === 'string' ? event.data.field : '')
      return
    }
    if (event.data.type === 'LIVESHOP_HOST_FORM_MODAL_CLOSED') {
      handler.closed(event.data.reason as HostFormModalClosedMessage['reason'])
      hostFormModalHandlers.delete(event.data.requestId)
    }
  })
}

function postModalMessage(message: HostFormModalRequestMessage): void {
  if (window.parent === window || !hostOrigin) throw new Error('Host modal requires an active iframe Host connection')
  window.parent.postMessage(message, hostOrigin)
}

/**
 * Promote the owning iframe above the Host shell. The module keeps rendering
 * its own rich dialog while the Host supplies the cross-frame stacking layer.
 * This deliberately does not use the browser Fullscreen API: it remains an
 * application modal, preserves browser chrome, and needs no fullscreen grant.
 */
export function hostOverlay(): { open(): void; close(): void } {
  let requestId = ''
  const post = (type: HostOverlayMessage['type']) => {
    if (window.parent === window || !hostOrigin) throw new Error('Host overlay requires an active iframe Host connection')
    window.parent.postMessage({ type, protocol: HOST_PROTOCOL, requestId } satisfies HostOverlayMessage, hostOrigin)
  }
  return {
    open() {
      if (requestId) return
      requestId = crypto.randomUUID()
      post('LIVESHOP_HOST_OVERLAY_OPEN')
    },
    close() {
      if (!requestId) return
      post('LIVESHOP_HOST_OVERLAY_CLOSE')
      requestId = ''
    },
  }
}

export interface HostFormModalSubmitApi {
  close(): void
  setBusy(busy: boolean): void
  setTitle(title: string): void
  setError(message?: string): void
  setFields(fields: HostModalField[], values?: Record<string, string | number | null | undefined>, title?: string): void
}

export interface HostFormModalOptions {
  title?: string
  fields: HostModalField[]
  submitLabel?: string
  cancelLabel?: string
  onSubmit?: (values: Record<string, string>, api: HostFormModalSubmitApi) => void
  onChange?: (values: Record<string, string>, field: string, api: HostFormModalSubmitApi) => void
  onClose?: (reason: HostFormModalClosedMessage['reason']) => void
}

export interface HostFormModalApi extends HostFormModalSubmitApi {
  open(values?: Record<string, string | number | null | undefined>, title?: string): void
}

/** Create a form modal that is rendered by the top-level Host, outside the iframe boundary. */
export function hostFormModal(options: HostFormModalOptions): HostFormModalApi {
  const baseTitle = options.title || ''
  let title = baseTitle
  let busy = false
  let requestId = ''

  const command = (message: Omit<HostFormModalCommandMessage, 'type' | 'protocol' | 'requestId'>) => {
    if (!requestId) return
    postModalMessage({ type: 'LIVESHOP_HOST_FORM_MODAL_COMMAND', protocol: HOST_PROTOCOL, requestId, ...message })
  }
  const api: HostFormModalApi = {
    open(values = {}, nextTitle) {
      if (requestId) api.close()
      requestId = crypto.randomUUID()
      title = nextTitle ?? baseTitle
      const activeRequestId = requestId
      hostFormModalHandlers.set(activeRequestId, {
        submit(submittedValues) {
          if (requestId !== activeRequestId) return
          options.onSubmit?.(submittedValues, api)
        },
        change(changedValues, field) {
          if (requestId !== activeRequestId) return
          options.onChange?.(changedValues, field, api)
        },
        closed(reason) {
          if (requestId === activeRequestId) requestId = ''
          options.onClose?.(reason)
        },
      })
      postModalMessage({
        type: 'LIVESHOP_HOST_FORM_MODAL_OPEN',
        protocol: HOST_PROTOCOL,
        requestId: activeRequestId,
        title,
        fields: options.fields,
        values,
        submitLabel: options.submitLabel || '保存',
        cancelLabel: options.cancelLabel || '取消',
        busy,
      })
    },
    close() {
      command({ command: 'close' })
    },
    setBusy(nextBusy) {
      busy = Boolean(nextBusy)
      command({ command: 'set-busy', busy })
    },
    setTitle(nextTitle) {
      title = nextTitle
      command({ command: 'set-title', title })
    },
    setError(message = '') {
      command({ command: 'set-error', message })
    },
    setFields(fields, values = {}, nextTitle) {
      command({ command: 'set-fields', fields, values, title: nextTitle })
    },
  }
  return api
}

export async function connectToHost(timeoutMs = 10_000): Promise<HostContext> {
  if (currentContext) return currentContext
  if (window.parent === window) throw new Error('module contribution must run inside a Liveshop Host')
  hostOrigin = document.referrer ? new URL(document.referrer).origin : ''
  if (!hostOrigin) throw new Error('host origin cannot be determined')
  installHostModuleSizeBridge()
  installHostModalBridge()
  installHostModuleUploadBridge()
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => reject(new Error('host handshake timed out')), timeoutMs)
    const receive = (event: MessageEvent) => {
      if (event.source !== window.parent || event.origin !== hostOrigin) return
      if (event.data?.type !== 'LIVESHOP_HOST_CONTEXT' || event.data?.context?.protocol !== HOST_PROTOCOL) return
      window.clearTimeout(timer)
      const incoming = event.data.context as HostContext
      if (currentContext) Object.assign(currentContext, incoming)
      else currentContext = incoming
      resolve(currentContext)
    }
    window.addEventListener('message', receive)
    window.parent.postMessage({ type: 'LIVESHOP_MODULE_READY', protocol: HOST_PROTOCOL }, hostOrigin)
  })
}

export function createHttpClient(context: HostContext): HostHttpClient {
  const requestRaw = async (path: string, init: RequestInit = {}): Promise<Response> => {
      if (!path.startsWith('/')) throw new Error('module API path must be absolute')
      let response: Response
      const headers = new Headers(init.headers)
      if (!(init.body instanceof FormData) && init.body !== undefined && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
      headers.set('Authorization', `Bearer ${context.moduleToken}`)
      headers.set('X-Liveshop-Surface', context.surface)
      try {
        response = await fetch(context.gatewayBaseUrl + path, {
          ...init,
          headers,
        })
      } catch (error) {
        throw new Error(`gateway request failed: ${error instanceof Error ? error.message : String(error)}`)
      }
      if (!response.ok) {
        const body = await response.clone().json().catch(() => null) as { message?: string } | null
        throw new Error(body?.message || response.statusText)
      }
      return response
  }
  return {
    requestRaw,
    async request<T>(path: string, init: RequestInit = {}): Promise<T> {
      const response = await requestRaw(path, init)
      const body = await response.json().catch(() => null) as { code?: number; message?: string; data?: T } | T | null
      return ((body as { data?: T } | null)?.data ?? body) as T
    },
  }
}

export function iframeHttpClient(context: HostContext): HostHttpClient {
  return createHttpClient(context)
}
