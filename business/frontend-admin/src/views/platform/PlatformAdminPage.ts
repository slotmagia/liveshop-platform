import type { HostContext, HostHttpClient } from '@liveshop/host-sdk'
import { startAudit, startRegistry } from './PlatformPages'
import { startLiveProviders } from './LiveProvidersPage'
import { startSettings } from './SettingsPage'
import { startStorage } from './StoragePage'
import { startI18n } from './I18nPage'
import { startNotifyEvents } from './NotifyEventsPage'
import { startNotifyTemplates } from './NotifyTemplatesPage'
import { startNotifyChannels } from './NotifyChannelsPage'

export function mountPlatformAdmin(root: HTMLElement, client: HostHttpClient, context: HostContext) {
  if (context.contributionId === 'platform.admin.registry') return startRegistry(root, client, context)
  if (context.contributionId === 'platform.admin.settings') return startSettings(root, client, context)
  if (context.contributionId === 'platform.admin.audit') return startAudit(root, client)
  if (context.contributionId === 'platform.admin.live-providers') return startLiveProviders(root, client, context)
  if (context.contributionId === 'platform.admin.notify-channels') return startNotifyChannels(root, client, context)
  if (context.contributionId === 'platform.admin.notify-templates') return startNotifyTemplates(root, client, context)
  if (context.contributionId === 'platform.admin.storage') return startStorage(root, client, context)
  if (context.contributionId === 'platform.admin.notify-events') return startNotifyEvents(root, client, context)
  if (context.contributionId === 'platform.admin.i18n') return startI18n(root, client, context)
  throw new Error(`Unsupported Platform Control Plane contribution: ${context.contributionId}`)
}
