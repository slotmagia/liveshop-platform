import type { HostContext, HostHttpClient } from '@liveshop/host-sdk'
import { startAudit, startRegistry } from './PlatformPages'
import { startLiveProviders } from './LiveProvidersPage'
import { startEmail } from './EmailPage'
import { startSettings } from './SettingsPage'
import { startSMS } from './SmsPage'
import { startStorage } from './StoragePage'

export function mountPlatformAdmin(root: HTMLElement, client: HostHttpClient, context: HostContext) {
  if (context.contributionId === 'platform.admin.registry') return startRegistry(root, client)
  if (context.contributionId === 'platform.admin.settings') return startSettings(root, client, context)
  if (context.contributionId === 'platform.admin.audit') return startAudit(root, client)
  if (context.contributionId === 'platform.admin.live-providers') return startLiveProviders(root, client, context)
  if (context.contributionId === 'platform.admin.sms') return startSMS(root, client, context)
  if (context.contributionId === 'platform.admin.email') return startEmail(root, client, context)
  if (context.contributionId === 'platform.admin.storage') return startStorage(root, client, context)
  throw new Error(`Unsupported Platform Control Plane contribution: ${context.contributionId}`)
}
