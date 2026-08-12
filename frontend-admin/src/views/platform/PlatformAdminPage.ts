import type { HostContext, HostHttpClient } from '@liveshop/host-sdk'
import { adminButtonClass, adminStatusClass, adminUI as ui } from '@liveshop/admin-ui'
import { startAccounts, startAudit, startRegistry, startSettings } from './PlatformPages'

interface Department { id:number; parentId?:number; name:string; status:string; version:number }
interface Role { id:number; name:string; status:string; superAdmin:boolean; version:number }

function field(form: HTMLFormElement, name: string): string {
  return String(new FormData(form).get(name) ?? '').trim()
}
function ids(value:string):number[]{ return value.split(',').map(item=>Number(item.trim())).filter(item=>Number.isInteger(item)&&item>0) }
function table(headers:string[], rows:Array<Array<string|number|boolean|undefined>>):HTMLTableElement{
  const element=document.createElement('table');element.className=ui.table;const head=element.createTHead().insertRow();for(const value of headers){const cell=document.createElement('th');cell.textContent=value;head.append(cell)}
  const body=element.createTBody();for(const row of rows){const line=body.insertRow();for(const value of row){const cell=line.insertCell();cell.textContent=value===undefined?'':String(value)}}return element
}

async function startIAM(root: HTMLElement, client:HostHttpClient){
  root.innerHTML=`<main class="${ui.page}"><div class="${ui.pageHeader}"><div class="${ui.pageHeading}"><p class="${ui.eyebrow}">Identity & access</p><h1 class="${ui.title}">Roles & organization</h1><p class="${ui.description}">Manage departments, roles, policies and subject assignments.</p></div><button id="refresh" class="${adminButtonClass('secondary')}">Refresh</button></div><div id="status" class="${ui.status}"></div><div class="${ui.grid}">
  <section class="${ui.card}"><div class="${ui.cardHeader}"><h2 class="${ui.cardTitle}">Departments</h2></div><form id="department" class="${ui.formGrid}"><input name="id" placeholder="Department ID" required><input name="expected" placeholder="Expected version (0=create)" value="0"><input name="parent" placeholder="Parent ID (optional)"><input name="name" placeholder="Name" required><select name="state"><option>ACTIVE</option><option>DISABLED</option></select><button class="${ui.button}">Save department</button></form><div id="departments" class="${ui.tableWrap}"></div></section>
  <section class="${ui.card}"><div class="${ui.cardHeader}"><h2 class="${ui.cardTitle}">Roles</h2></div><form id="role" class="${ui.formGrid}"><input name="id" placeholder="Role ID" required><input name="expected" placeholder="Expected version (0=create)" value="0"><input name="name" placeholder="Name" required><select name="state"><option>ACTIVE</option><option>DISABLED</option></select><button class="${ui.button}">Save role</button></form><div id="roles" class="${ui.tableWrap}"></div></section>
  <section class="${ui.card}"><div class="${ui.cardHeader}"><h2 class="${ui.cardTitle}">Role policy</h2></div><form id="policy" class="${ui.formGrid}"><input name="id" placeholder="Role ID" required><input name="expected" placeholder="Expected role version" required><textarea class="${ui.fieldWide}" name="permissions" placeholder="Permission codes, comma-separated" required></textarea><input name="resource" placeholder="Resource, e.g. catalog.product" required><select name="scope"><option>ALL</option><option>DEPARTMENT_AND_CHILDREN</option><option>DEPARTMENT</option><option>CUSTOM</option><option>SELF</option></select><input name="departments" placeholder="Custom department IDs"><button class="${ui.button}">Replace policy</button></form></section>
  <section class="${ui.card}"><div class="${ui.cardHeader}"><h2 class="${ui.cardTitle}">User assignment</h2></div><form id="assignment" class="${ui.formGrid}"><input name="subject" placeholder="Authenticated subject" required><input name="roles" placeholder="Role IDs" required><input name="departments" placeholder="Department IDs"><input name="primary" placeholder="Primary department ID"><button class="${ui.button}">Replace assignment</button></form></section></div></main>`
  const status=root.querySelector<HTMLElement>('#status')!;const show=(message:string,error=false)=>{status.textContent=message;status.className=adminStatusClass(error?'danger':'neutral')}
  const refresh=async()=>{const [departments,roles]=await Promise.all([client.request<Department[]>('/admin/platform/iam/departments'),client.request<Role[]>('/admin/platform/iam/roles')]);const departmentRoot=root.querySelector<HTMLElement>('#departments')!;departmentRoot.replaceChildren(table(['ID','Parent','Name','Status','Version'],departments.map(item=>[item.id,item.parentId,item.name,item.status,item.version])));const roleRoot=root.querySelector<HTMLElement>('#roles')!;roleRoot.replaceChildren(table(['ID','Name','Status','Super admin','Version'],roles.map(item=>[item.id,item.name,item.status,item.superAdmin,item.version])));show('Authorization data refreshed')}
  const submit=(selector:string,action:(form:HTMLFormElement)=>Promise<unknown>)=>{root.querySelector<HTMLFormElement>(selector)!.addEventListener('submit',event=>{event.preventDefault();const form=event.currentTarget as HTMLFormElement;void action(form).then(async()=>{show('Saved');await refresh()}).catch(error=>show(error instanceof Error?error.message:String(error),true))})}
  submit('#department',form=>client.request(`/admin/platform/iam/departments/${Number(field(form,'id'))}`,{method:'PUT',body:JSON.stringify({expectedVersion:Number(field(form,'expected')),parentId:field(form,'parent')?Number(field(form,'parent')):null,name:field(form,'name'),status:field(form,'state')})}))
  submit('#role',form=>client.request(`/admin/platform/iam/roles/${Number(field(form,'id'))}`,{method:'PUT',body:JSON.stringify({expectedVersion:Number(field(form,'expected')),name:field(form,'name'),status:field(form,'state')})}))
  submit('#policy',form=>client.request(`/admin/platform/iam/roles/${Number(field(form,'id'))}/policy`,{method:'PUT',body:JSON.stringify({expectedVersion:Number(field(form,'expected')),permissions:field(form,'permissions').split(',').map(item=>item.trim()).filter(Boolean),scopes:[{resource:field(form,'resource'),type:field(form,'scope'),departmentIds:ids(field(form,'departments'))}]})}))
  submit('#assignment', form => {
    const primary = Number(field(form, 'primary'))
    return client.request(`/admin/platform/iam/subjects/${encodeURIComponent(field(form, 'subject'))}/assignment`, {
      method: 'PUT',
      body: JSON.stringify({
        roleIds: ids(field(form, 'roles')),
        departments: ids(field(form, 'departments')).map(departmentId => ({ departmentId, primary: departmentId === primary })),
      }),
    })
  })
  root.querySelector('#refresh')!.addEventListener('click',()=>void refresh().catch(error=>show(error instanceof Error?error.message:String(error),true)));await refresh()
}

export function mountPlatformAdmin(root: HTMLElement, client: HostHttpClient, context: HostContext) {
  if (context.contributionId === 'platform.admin.accounts') return startAccounts(root, client)
  if (context.contributionId === 'platform.admin.registry') return startRegistry(root, client)
  if (context.contributionId === 'platform.admin.settings') return startSettings(root, client, context)
  if (context.contributionId === 'platform.admin.audit') return startAudit(root, client)
  return startIAM(root, client)
}
