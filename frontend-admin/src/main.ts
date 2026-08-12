import { connectToHost, iframeHttpClient } from '@liveshop/host-sdk'
import './style.css'
import { mountPlatformAdmin } from './views/platform/PlatformAdminPage'

const root = document.querySelector<HTMLElement>('#app')!

void connectToHost()
  .then(context => mountPlatformAdmin(root, iframeHttpClient(context), context))
  .catch(error => { root.textContent = error instanceof Error ? error.message : String(error) })
