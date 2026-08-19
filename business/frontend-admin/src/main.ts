import { connectToHost, iframeHttpClient } from '@liveshops/host-sdk'
import '@liveshops/design-tokens/console.css'
import './style.css'
import { mountPlatformAdmin } from './views/platform/PlatformAdminPage'
import { renderPlaceholder } from '../../ui/placeholder'

const root = document.querySelector<HTMLElement>('#app')!

void connectToHost()
  .then(context => {
    if (context.contributionId.startsWith('platform.admin.') && !context.contributionId.includes('placeholder')) {
      return mountPlatformAdmin(root, iframeHttpClient(context), context)
    }
    return renderPlaceholder(root, context)
  })
  .catch(error => { root.textContent = error instanceof Error ? error.message : String(error) })
