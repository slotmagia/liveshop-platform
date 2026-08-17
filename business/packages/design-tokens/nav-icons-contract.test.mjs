import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'
import { NAV_GROUP_ICONS, NAV_PAGE_ICONS, resolveGroupIconName, resolvePageIconName } from './nav-icons.js'

const root = join(dirname(fileURLToPath(import.meta.url)), '../../../..')
const manifests = [
  'liveshop-identity/business/module.json',
  'liveshop-trade/business/module.json',
  'liveshop-catalog/business/module.json',
  'liveshop-live/business/module.json',
  'liveshop-platform/business/module.json',
]

test('nav icon catalog resolves registered names and stable fallbacks', () => {
  assert.equal(resolvePageIconName('identity.admin.authorization', 'shield-check'), 'shield-check')
  assert.equal(resolvePageIconName('identity.admin.authorization'), 'shield-check')
  assert.equal(resolvePageIconName('unknown.page', undefined, 'layout-grid'), 'layout-grid')
  assert.equal(resolveGroupIconName('legacy-admin-system', 'shield'), 'shield')
  assert.equal(resolveGroupIconName('legacy-admin-mall'), 'store')
  assert.notEqual(NAV_GROUP_ICONS['legacy-admin-system'], NAV_GROUP_ICONS['legacy-admin-mall'])
  assert.notEqual(NAV_PAGE_ICONS['identity.admin.authorization'], NAV_PAGE_ICONS['platform.admin.registry'])
})

test('every registered page and directory has a distinct Lucide icon', { skip: !existsSync(join(root, manifests[0])) }, () => {
  const groups = new Map()
  for (const relative of manifests) {
    const manifest = JSON.parse(readFileSync(join(root, relative), 'utf8'))
    for (const contribution of manifest.spec.contributions || []) {
      if (contribution.kind !== 'page') continue
      assert.equal(contribution.icon, NAV_PAGE_ICONS[contribution.id], contribution.id)
      assert.equal(resolvePageIconName(contribution.id, contribution.icon), contribution.icon)
      const navigation = contribution.navigation || { groupId: 'host-workbench', groupIcon: NAV_GROUP_ICONS['host-workbench'] }
      if (contribution.navigation) {
        assert.equal(navigation.groupIcon, NAV_GROUP_ICONS[navigation.groupId], navigation.groupId)
      }
      const key = `${contribution.surface}|${navigation.groupId}`
      const icons = groups.get(key) || new Set()
      assert.equal(icons.has(contribution.icon), false, `${key} reused ${contribution.icon}`)
      icons.add(contribution.icon)
      groups.set(key, icons)
      assert.equal(resolveGroupIconName(navigation.groupId, navigation.groupIcon), NAV_GROUP_ICONS[navigation.groupId])
    }
  }
  assert.ok(groups.size > 0)
})
