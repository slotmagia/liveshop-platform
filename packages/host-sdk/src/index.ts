export const HOST_PROTOCOL = 1 as const
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
  icon?: string
  sort?: number
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
  tenant?: { appId: number; merchantId: number }
  theme: { mode: 'light' | 'dark' }
}

export interface HostHttpClient {
  request<T>(path: string, init?: RequestInit): Promise<T>
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

export async function connectToHost(timeoutMs = 10_000): Promise<HostContext> {
  if (currentContext) return currentContext
  if (window.parent === window) throw new Error('module contribution must run inside a Liveshop Host')
  const hostOrigin = document.referrer ? new URL(document.referrer).origin : ''
  if (!hostOrigin) throw new Error('host origin cannot be determined')
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => reject(new Error('host handshake timed out')), timeoutMs)
    const receive = (event: MessageEvent) => {
      if (event.source !== window.parent || event.origin !== hostOrigin) return
      if (event.data?.type !== 'LIVESHOP_HOST_CONTEXT' || event.data?.context?.protocol !== HOST_PROTOCOL) return
      window.clearTimeout(timer)
      currentContext = event.data.context as HostContext
      resolve(currentContext)
    }
    window.addEventListener('message', receive)
    window.parent.postMessage({ type: 'LIVESHOP_MODULE_READY', protocol: HOST_PROTOCOL }, hostOrigin)
  })
}

export function createHttpClient(context: HostContext): HostHttpClient {
  return {
    async request<T>(path: string, init: RequestInit = {}): Promise<T> {
      if (!path.startsWith('/')) throw new Error('module API path must be absolute')
      let response: Response
      try {
        response = await fetch(context.gatewayBaseUrl + path, {
          ...init,
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${context.moduleToken}`,
            'X-Liveshop-Surface': context.surface,
            ...init.headers,
          },
        })
      } catch (error) {
        throw new Error(`gateway request failed: ${error instanceof Error ? error.message : String(error)}`)
      }
      const body = await response.json().catch(() => null) as { code?: number; message?: string; data?: T } | T | null
      if (!response.ok) throw new Error((body as { message?: string } | null)?.message || response.statusText)
      return ((body as { data?: T } | null)?.data ?? body) as T
    },
  }
}

export function iframeHttpClient(context: HostContext): HostHttpClient {
  return createHttpClient(context)
}
