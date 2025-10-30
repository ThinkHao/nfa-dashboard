<template>
  <div class="rates-view">
    <h1 class="page-title">客户业务费率</h1>
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">客户业务费率筛选</span>
          <div>
            <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
            <el-button @click="onReset">重置</el-button>
            <el-button v-if="canManageSyncRules" @click="goSyncRules">同步规则管理</el-button>
            <el-button v-if="canSync" type="warning" :loading="syncing" @click="onExecuteSync">执行规则同步</el-button>
            <el-button v-if="canWrite" type="success" @click="openDialog()">新增/更新</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="区域">
          <el-select v-model="query.region" clearable filterable placeholder="选择区域" style="width: 180px">
            <el-option v-for="r in regionOptions" :key="r" :label="r" :value="r" />
          </el-select>
        </el-form-item>
        <el-form-item label="运营商">
          <el-select v-model="query.cp" clearable filterable placeholder="选择运营商" style="width: 180px">
            <el-option v-for="c in cpOptions" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="学校">
          <el-select
            v-model="query.school_name"
            clearable
            filterable
            remote
            :remote-method="remoteSearchSchoolsFilter"
            :loading="schoolsLoading"
            placeholder="搜索学校"
            style="width: 240px"
          >
            <el-option v-for="s in schoolOptions" :key="s" :label="s" :value="s" />
          </el-select>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card" style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <span class="card-title">费率列表</span>
          <div>
            <el-radio-group v-model="settlementTab" size="small" @change="onSettlementTabChange">
              <el-radio-button label="all">全部</el-radio-button>
              <el-radio-button label="ready">参与</el-radio-button>
              <el-radio-button label="not_ready">不参与</el-radio-button>
            </el-radio-group>
          </div>
        </div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="region" label="区域" width="120" />
        <el-table-column prop="cp" label="运营商" width="120" />
        <el-table-column prop="school_name" label="学校" min-width="160" show-overflow-tooltip />
        <el-table-column label="模式" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.fee_mode === 'configed' ? 'warning' : 'info'">
              {{ formatMode(row.fee_mode) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="参与结算" width="120">
          <template #default="{ row }">
            <el-tooltip v-if="!row.settlement_ready" placement="top" :content="formatMissingFields(row.missing_fields)">
              <el-tag size="small" type="danger">否</el-tag>
            </el-tooltip>
            <el-tag v-else size="small" type="success">是</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最近同步" width="180">
          <template #default="{ row }">
            {{ formatTime(row.last_sync_time) }}
          </template>
        </el-table-column>
        <el-table-column prop="last_sync_rule_id" label="规则ID" width="100" />
        <el-table-column label="扩展" min-width="120">
          <template #default="{ row }">
            <el-button v-if="row.extra" type="primary" link @click="openExtra(row)">
              查看 ({{ extraCount(row.extra) }})
            </el-button>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="customer_fee" label="客户费" width="120" />
        <el-table-column prop="network_line_fee" label="线路费" width="120" />
        <el-table-column prop="general_fee" label="节点通用费" width="120" />
        <el-table-column label="节点通用费归属" width="200">
          <template #default="{ row }">
            <span v-if="row.general_fee_owner_id">{{ displayOwner(row.general_fee_owner_id) }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="客户费归属" width="200">
          <template #default="{ row }">
            <span v-if="row.customer_fee_owner_id">{{ displayOwner(row.customer_fee_owner_id) }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="线路费归属" width="200">
          <template #default="{ row }">
            <span v-if="row.network_line_fee_owner_id">{{ displayOwner(row.network_line_fee_owner_id) }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
        <el-table-column v-if="canWrite" label="操作" fixed="right" width="100">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDialog(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          background
          layout="prev, pager, next, sizes, total"
          :total="total"
          :page-size="pageSize"
          :current-page="page"
          :page-sizes="[10,20,50,100]"
          @size-change="onPageSizeChange"
          @current-change="onPageChange"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" title="新增/更新 客户业务费率" width="560px">
      <el-form :model="form" label-width="140px">
        <el-form-item label="区域" required>
          <el-select v-model="form.region" filterable placeholder="选择区域" style="width: 240px">
            <el-option v-for="r in regionOptions" :key="r" :label="r" :value="r" />
          </el-select>
        </el-form-item>
        <el-form-item label="运营商" required>
          <el-select v-model="form.cp" filterable placeholder="选择运营商" style="width: 240px">
            <el-option v-for="c in cpOptions" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="学校">
          <el-select
            v-model="form.school_name"
            clearable
            filterable
            remote
            :remote-method="remoteSearchSchoolsDialog"
            :loading="schoolsLoading"
            placeholder="搜索学校"
            style="width: 300px"
          >
            <el-option v-for="s in schoolOptions" :key="s" :label="s" :value="s" />
          </el-select>
        </el-form-item>
        <el-form-item label="客户费">
          <el-input-number v-model="form.customer_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="线路费">
          <el-input-number v-model="form.network_line_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="节点通用费">
          <el-input-number v-model="form.general_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="节点通用费归属（节点供应商）">
          <el-select
            v-model="form.general_fee_owner_id"
            filterable
            remote
            clearable
            :remote-method="remoteSearchSystemUsersNode"
            :loading="ownerUserNodeLoading"
            placeholder="搜索节点供应商（按配置的节点角色过滤）"
            style="width: 300px"
            @visible-change="(v) => v && remoteSearchSystemUsersNode('')"
          >
            <el-option
              v-for="u in ownerUserNodeOptions"
              :key="u.id"
              :label="u.label"
              :value="u.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="客户费归属（销售）">
          <el-select
            v-model="form.customer_fee_owner_id"
            filterable
            remote
            clearable
            :remote-method="remoteSearchSystemUsers"
            :loading="ownerUserLoading"
            placeholder="搜索销售用户（系统用户，受角色配置过滤）"
            style="width: 300px"
            @visible-change="(v) => v && remoteSearchSystemUsers('')"
          >
            <el-option
              v-for="u in ownerUserOptions"
              :key="u.id"
              :label="u.label"
              :value="u.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="线路费归属（线路用户）">
          <el-select
            v-model="form.network_line_fee_owner_id"
            filterable
            remote
            clearable
            :remote-method="remoteSearchSystemUsersLine"
            :loading="ownerUserLineLoading"
            placeholder="搜索线路相关用户（按配置的线路角色过滤）"
            style="width: 300px"
            @visible-change="(v) => v && remoteSearchSystemUsersLine('')"
          >
            <el-option
              v-for="u in ownerUserLineOptions"
              :key="u.id"
              :label="u.label"
              :value="u.id"
            />
          </el-select>
        </el-form-item>
        
        <el-form-item label="扩展JSON">
          <el-input
            v-model="extraEditorText"
            type="textarea"
            :rows="6"
            placeholder='可选，填写合法 JSON（如 {"remark":"..."}）'
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>

    <!-- 扩展字段查看弹窗 -->
    <el-dialog v-model="extraDialogVisible" title="扩展字段 JSON" width="600px">
      <pre style="max-height: 400px; overflow: auto; white-space: pre-wrap;">{{ stringify(extraDialogContent) }}</pre>
      <template #footer>
        <el-button @click="extraDialogVisible=false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import type { RateCustomer, PaginatedData, UpsertRateCustomerRequest, BusinessEntity } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const canWrite = computed(() => auth.hasPermission('rates.customer.write'))
const canSync = computed(() => auth.hasPermission('rates.sync.execute'))
const canManageSyncRules = computed(() => auth.hasPermission('rates.sync_rules.read'))

const loading = ref(false)
const syncing = ref(false)
const items = ref<RateCustomer[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const query = reactive<{ region?: string; cp?: string; school_name?: string; settlement_ready?: string | '' }>({})
// 表头分类标签：全部/参与/不参与
const settlementTab = ref<'all' | 'ready' | 'not_ready'>('all')
// 下拉与远程搜索状态
const regionOptions = ref<string[]>([])
const cpOptions = ref<string[]>([])
const schoolOptions = ref<string[]>([])
const schoolsLoading = ref(false)
const entitiesLoading = ref(false)
// 客户费归属（销售）由系统用户提供，不再使用实体列表
const entityOptionsCustomer = ref<BusinessEntity[]>([])
// ID -> 实体 映射，用于兼容旧数据的列表展示（历史可能存了业务对象ID）
const entityMap = ref<Record<number, BusinessEntity>>({})
// ID -> 用户映射，用于显示“客户费/线路费归属”为系统用户时的别名
const userMap = ref<Record<number, { id: number; alias?: string; display_name?: string; username: string }>>({})

// 系统用户（销售）下拉：用于“客户费归属（销售）”与绑定
const ownerUserOptions = ref<{ id: number; label: string }[]>([])
const ownerUserLoading = ref(false)
// 从后端获取：允许绑定的角色（如 ['sales']），可配置
const allowedBindRoles = ref<string[]>([])
// 系统用户（线路）下拉：用于“线路费归属（线路用户）”
const ownerUserLineOptions = ref<{ id: number; label: string }[]>([])
const ownerUserLineLoading = ref(false)
const allowedLineRoles = ref<string[]>([])
// 系统用户（节点）下拉：用于“节点通用费归属（节点供应商）”
const ownerUserNodeOptions = ref<{ id: number; label: string }[]>([])
const ownerUserNodeLoading = ref(false)
const allowedNodeRoles = ref<string[]>([])

// 扩展字段弹窗
const extraDialogVisible = ref(false)
const extraDialogContent = ref<any>(null)

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.region) p.region = query.region
  if (query.cp) p.cp = query.cp
  if (query.school_name) p.school_name = query.school_name
  if (query.settlement_ready === 'true' || query.settlement_ready === 'false') {
    p.settlement_ready = query.settlement_ready
  }
  return p
}

async function fetchData() {
  loading.value = true
  try {
    const res: PaginatedData<RateCustomer> = await api.settlementRates.customer.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
    // 批量加载归属对象信息，构建映射
    await Promise.all([loadEntitiesForItems(), loadUsersForItems()])
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

// 根据当前 items 收集 owner_id，批量按ids获取实体并缓存映射
async function loadEntitiesForItems() {
  const ids = new Set<number>()
  for (const r of items.value) {
    if (r?.customer_fee_owner_id) ids.add(r.customer_fee_owner_id)
    if (r?.network_line_fee_owner_id) ids.add(r.network_line_fee_owner_id)
    if (r?.general_fee_owner_id) ids.add(r.general_fee_owner_id)
  }
  if (ids.size === 0) { entityMap.value = {}; return }
  try {
    const params: any = { ids: Array.from(ids).join(',') }
    const res = await api.settlementEntities.list(params)
    const list: BusinessEntity[] = Array.isArray((res as any)?.items) ? (res as any).items as BusinessEntity[] : []
    const m: Record<number, BusinessEntity> = {}
    for (const e of list) { if (e && typeof e.id === 'number') m[e.id] = e }
    entityMap.value = m
  } catch {}
}

// 根据当前 items 收集 owner_id，批量按 ids 获取系统用户并缓存映射（优先用于显示别名）
async function loadUsersForItems() {
  const ids = new Set<number>()
  for (const r of items.value) {
    if (r?.customer_fee_owner_id != null) {
      const n = Number(r.customer_fee_owner_id)
      if (!Number.isNaN(n) && n > 0) ids.add(n)
    }
    if (r?.network_line_fee_owner_id != null) {
      const n = Number(r.network_line_fee_owner_id)
      if (!Number.isNaN(n) && n > 0) ids.add(n)
    }
    if (r?.general_fee_owner_id != null) {
      const n = Number(r.general_fee_owner_id)
      if (!Number.isNaN(n) && n > 0) ids.add(n)
    }
  }
  if (ids.size === 0) { userMap.value = {}; return }
  try {
    const res: any = await api.system.users.list({ ids: Array.from(ids).join(',') })
    const list: any[] = Array.isArray(res?.items) ? res.items : []
    const m: Record<number, { id: number; alias?: string; display_name?: string; username: string }> = {}
    for (const u of list) { if (u && typeof u.id === 'number') m[u.id] = { id: u.id, alias: u.alias, display_name: u.display_name, username: u.username } }
    userMap.value = m
  } catch { userMap.value = {} }
}

function displayOwner(id?: number | null): string {
  if (!id) return '-'
  const key = Number(id)
  const u = userMap.value[key]
  if (u) {
    const alias = (u.alias && String(u.alias).trim()) ? String(u.alias).trim() : ''
    const dn = (u.display_name && String(u.display_name).trim()) ? String(u.display_name).trim() : ''
    const un = (u.username && String(u.username).trim()) ? String(u.username).trim() : ''
    return alias || dn || un || `用户#${key}`
  }
  const e = entityMap.value[key]
  if (e) return `${e.entity_name}`
  return String(key)
}

// 生成系统用户在下拉中的显示标签：alias > display_name > username > 用户#ID
function buildUserLabel(u: any): string {
  if (!u) return ''
  const alias = (u.alias && String(u.alias).trim()) ? String(u.alias).trim() : ''
  const dn = (u.display_name && String(u.display_name).trim()) ? String(u.display_name).trim() : ''
  const un = (u.username && String(u.username).trim()) ? String(u.username).trim() : ''
  const id = Number(u.id)
  return alias || dn || un || (Number.isFinite(id) ? `用户#${id}` : '')
}

// 预加载给定或表单中选中的用户ID为下拉选项，确保显示用户名而非原始ID
async function preloadSelectedUsersIntoOptions(idsOverride?: number[]) {
  const ids: number[] = Array.isArray(idsOverride) ? idsOverride.filter(n => typeof n === 'number' && n > 0) : []
  if (!idsOverride) {
    const addIf = (v: any) => { const n = Number(v); if (!Number.isNaN(n) && n > 0) ids.push(n) }
    addIf((form.customer_fee_owner_id as any))
    addIf((form.network_line_fee_owner_id as any))
    addIf((form.general_fee_owner_id as any))
  }
  if (ids.length === 0) return
  try {
    const res: any = await api.system.users.list({ ids: ids.join(',') })
    const list: any[] = Array.isArray(res?.items) ? res.items : []
    for (const u of list) {
      const opt = { id: u.id, label: buildUserLabel(u) }
      const sIdx = ownerUserOptions.value.findIndex(x => x.id === u.id)
      if (sIdx >= 0) ownerUserOptions.value[sIdx] = opt
      else ownerUserOptions.value.push(opt)
      const lIdx = ownerUserLineOptions.value.findIndex(x => x.id === u.id)
      if (lIdx >= 0) ownerUserLineOptions.value[lIdx] = opt
      else ownerUserLineOptions.value.push(opt)
      const nIdx = ownerUserNodeOptions.value.findIndex(x => x.id === u.id)
      if (nIdx >= 0) ownerUserNodeOptions.value[nIdx] = opt
      else ownerUserNodeOptions.value.push(opt)
    }
    // 如果选中的ID不在系统用户返回结果中（历史上可能是业务对象ID），清空以避免显示纯数字
    const returnedIds = new Set<number>(list.map((u: any) => Number(u.id)))
    const salesSel = Number((form.customer_fee_owner_id as any) || 0)
    if (salesSel > 0 && !returnedIds.has(salesSel)) {
      form.customer_fee_owner_id = undefined as any
      try { ElMessage?.warning?.('检测到历史数据不是系统用户，请重新选择“客户费归属（销售）”。') } catch {}
    }
    const lineSel = Number((form.network_line_fee_owner_id as any) || 0)
    if (lineSel > 0 && !returnedIds.has(lineSel)) {
      form.network_line_fee_owner_id = undefined as any
      try { ElMessage?.warning?.('检测到历史数据不是系统用户，请重新选择“线路费归属（线路用户）”。') } catch {}
    }
    const nodeSel = Number((form.general_fee_owner_id as any) || 0)
    if (nodeSel > 0 && !returnedIds.has(nodeSel)) {
      form.general_fee_owner_id = undefined as any
      try { ElMessage?.warning?.('检测到历史数据不是系统用户，请重新选择“节点通用费归属（节点供应商）”。') } catch {}
    }
  } catch {}
}

async function loadRegionsAndCPs() {
  try {
    const [regions, cps] = await Promise.all([
      (api as any).v2.getRegions(),
      (api as any).v2.getCPs(),
    ])
    regionOptions.value = Array.isArray(regions) ? regions.filter((v: any) => v && v !== 'NULL') : []
    cpOptions.value = Array.isArray(cps) ? cps.filter((v: any) => v && v !== 'NULL') : []
  } catch {}
}

// 学校远程搜索（筛选区）
async function remoteSearchSchoolsFilter(q: string) {
  schoolsLoading.value = true
  try {
    const data = await (api as any).v2.getSchools({ region: query.region, cp: query.cp, school_name: q || undefined, limit: 20, offset: 0 })
    const list: any[] = Array.isArray(data?.items) ? data.items : (Array.isArray(data) ? data : [])
    schoolOptions.value = list.map((it: any) => it?.school_name || it?.name || it).filter(Boolean)
  } catch {}
  finally { schoolsLoading.value = false }
}

// 学校远程搜索（弹窗区）
async function remoteSearchSchoolsDialog(q: string) {
  schoolsLoading.value = true
  try {
    const data = await (api as any).v2.getSchools({ region: form.region, cp: form.cp, school_name: q || undefined, limit: 20, offset: 0 })
    const list: any[] = Array.isArray(data?.items) ? data.items : (Array.isArray(data) ? data : [])
    schoolOptions.value = list.map((it: any) => it?.school_name || it?.name || it).filter(Boolean)
  } catch {}
  finally { schoolsLoading.value = false }
}

// 业务对象远程搜索
async function remoteSearchEntitiesForCustomer(q: string) {
  entitiesLoading.value = true
  try {
    const res = await api.settlementEntities.list({ page: 1, page_size: 20, entity_name: q || undefined })
    entityOptionsCustomer.value = Array.isArray((res as any)?.items) ? (res as any).items as BusinessEntity[] : []
  } catch {}
  finally { entitiesLoading.value = false }
}

function onSearch() { page.value = 1; fetchData() }
function onReset() { Object.assign(query, { region: undefined, cp: undefined, school_name: undefined, settlement_ready: '' as any }); settlementTab.value='all'; page.value=1; pageSize.value=10; fetchData() }
function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

function goSyncRules() { router.push({ name: 'settlement-rates-sync-rules' }) }

// 切换“参与结算”分类（表头标签）
function onSettlementTabChange(val: 'all'|'ready'|'not_ready') {
  if (val === 'ready') query.settlement_ready = 'true' as any
  else if (val === 'not_ready') query.settlement_ready = 'false' as any
  else query.settlement_ready = '' as any
  page.value = 1
  fetchData()
}

// Dialog
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive<UpsertRateCustomerRequest>({ region: '', cp: '' })
const extraEditorText = ref<string>('')

async function openDialog(row?: RateCustomer) {
  if (row) {
    // 先提取将要设置的 owner_id
    const salesId = (row.customer_fee_owner_id != null ? Number(row.customer_fee_owner_id) : undefined) as any
    const lineId = (row.network_line_fee_owner_id != null ? Number(row.network_line_fee_owner_id) : undefined) as any
    const nodeId = (row.general_fee_owner_id != null ? Number(row.general_fee_owner_id) : undefined) as any
    // 其他字段先设置；owner_id 先置空，待预加载完 options 后再赋值
    Object.assign(form, {
      region: row.region,
      cp: row.cp,
      school_name: row.school_name ?? undefined,
      customer_fee: row.customer_fee ?? undefined,
      network_line_fee: row.network_line_fee ?? undefined,
      general_fee: row.general_fee ?? undefined,
      customer_fee_owner_id: undefined as any,
      network_line_fee_owner_id: undefined as any,
      general_fee_owner_id: undefined as any,
    })
    extraEditorText.value = stringify(row.extra ?? {})
    // 先加载 options 再赋值，确保下拉能显示 label
    try { await preloadSelectedUsersIntoOptions([Number(salesId||0), Number(lineId||0), Number(nodeId||0)].filter(n => n>0)) } catch {}
    form.customer_fee_owner_id = salesId
    form.network_line_fee_owner_id = lineId
    form.general_fee_owner_id = nodeId
  } else {
    Object.assign(form, { region: '', cp: '', school_name: undefined, customer_fee: undefined, network_line_fee: undefined, general_fee: undefined, customer_fee_owner_id: undefined, network_line_fee_owner_id: undefined, general_fee_owner_id: undefined })
    extraEditorText.value = ''
  }
  // 显示弹窗
  dialogVisible.value = true
  // 异步加载下拉数据（包含合并已选用户项的逻辑）
  try { remoteSearchEntitiesForCustomer('') } catch {}
  try { remoteSearchSystemUsers('') } catch {}
  try { remoteSearchSystemUsersLine('') } catch {}
  try { remoteSearchSystemUsersNode('') } catch {}
}

async function onSave() {
  if (!form.region || !form.cp) { ElMessage.warning('区域与运营商为必填'); return }
  saving.value = true
  try {
    // 解析扩展 JSON（可选）
    const payload: any = { ...form }
    const txt = (extraEditorText.value || '').trim()
    if (txt) {
      try { payload.extra = JSON.parse(txt) } catch (e) { ElMessage.error('扩展JSON格式错误'); saving.value=false; return }
    }
    await api.settlementRates.customer.upsert(payload)

    // 保存后，根据“客户费归属（销售）”下拉的选择，设置用户-院校绑定
    const selectedUserId: number | null | undefined = form.customer_fee_owner_id as any
    if (form.school_name) {
      try {
        // 通过 v2 学校接口解析唯一 school_id（按 region+cp+school_name）
        const res: any = await (api as any).v2.getSchools({ region: form.region, cp: form.cp, school_name: form.school_name, limit: 50, offset: 0 })
        const items: any[] = Array.isArray(res?.items) ? res.items : (Array.isArray(res) ? res : [])
        const matched = items.filter((s: any) => s && s.school_name === form.school_name && s.region === form.region && s.cp === form.cp)
        if (matched.length === 1 && matched[0]?.school_id) {
          const schoolId = matched[0].school_id as string
          await api.system.userSchools.setOwner({ school_id: schoolId, user_id: selectedUserId ?? null })
        } else {
          console.warn('未能唯一定位 school_id，跳过绑定。匹配数量:', matched.length)
        }
      } catch (e) {
        console.warn('绑定用户-院校失败（已忽略）：', e)
      }
    }

    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

// 执行同步
async function onExecuteSync() {
  try {
    await ElMessageBox.confirm('将按启用的同步规则批量更新客户费率的自定义字段，是否继续？', '确认执行', { type: 'warning', confirmButtonText: '执行', cancelButtonText: '取消' })
  } catch {
    return
  }
  syncing.value = true
  try {
    const affected = await api.settlementRates.sync.execute()
    ElMessage.success(`同步完成，受影响行数：${affected}`)
    fetchData()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '同步失败')
  } finally {
    syncing.value = false
  }
}

function extraCount(extra: any): number {
  try {
    if (!extra) return 0
    if (typeof extra === 'string') {
      const obj = JSON.parse(extra)
      return obj && typeof obj === 'object' ? Object.keys(obj).length : 0
    }
    if (typeof extra === 'object') return Object.keys(extra).length
    return 0
  } catch { return 0 }
}

function openExtra(row: RateCustomer) {
  extraDialogContent.value = row.extra ?? {}
  extraDialogVisible.value = true
}

function stringify(obj: any): string {
  try {
    if (typeof obj === 'string') return JSON.stringify(JSON.parse(obj), null, 2)
    return JSON.stringify(obj, null, 2)
  } catch { return String(obj) }
}

function formatTime(s?: string | null): string {
  if (!s) return '-'
  const d = new Date(s)
  if (isNaN(d.getTime())) return String(s)
  return d.toLocaleString()
}

function formatMode(m?: string): string {
  if (!m) return '自动'
  return m === 'configed' ? '手工' : '自动'
}

// 缺失字段提示
function formatMissingFields(m?: string[]): string {
  if (!m || m.length === 0) return ''
  const map: Record<string, string> = {
    school_name: '学校',
    customer_fee: '客户费',
    network_line_fee: '线路费',
    general_fee: '节点通用费',
  }
  return '缺失字段：' + m.map(k => map[k] || k).join('、')
}

// 远程搜索系统用户（单选）
async function remoteSearchSystemUsers(q: string) {
  ownerUserLoading.value = true
  try {
    // 确保已加载允许的角色列表
    if (!allowedBindRoles.value || allowedBindRoles.value.length === 0) {
      try { allowedBindRoles.value = await api.system.binding.getAllowedUserRoles('sales') } catch { allowedBindRoles.value = [] }
    }
    const rolesParam = (allowedBindRoles.value && allowedBindRoles.value.length > 0) ? allowedBindRoles.value.join(',') : undefined
    const res: any = await api.system.users.list({ page: 1, page_size: 20, username: q || undefined, roles: rolesParam })
    const list: any[] = Array.isArray(res?.items) ? res.items : []
    const newOptions = list.map((u: any) => ({ id: u.id, label: buildUserLabel(u) }))
    // 确保当前选中用户在选项中
    const selectedId = Number((form.customer_fee_owner_id as any) || 0)
    if (selectedId > 0 && !newOptions.some(o => o.id === selectedId)) {
      const existing = ownerUserOptions.value.find(o => o.id === selectedId)
      if (existing) newOptions.unshift(existing)
    }
    ownerUserOptions.value = newOptions
  } catch {
    ownerUserOptions.value = []
  } finally {
    ownerUserLoading.value = false
  }
}

// 远程搜索系统用户（线路）
async function remoteSearchSystemUsersLine(q: string) {
  ownerUserLineLoading.value = true
  try {
    // 确保已加载允许的线路角色列表
    if (!allowedLineRoles.value || allowedLineRoles.value.length === 0) {
      try { allowedLineRoles.value = await api.system.binding.getAllowedUserRoles('line') } catch { allowedLineRoles.value = [] }
    }
    const rolesParam = (allowedLineRoles.value && allowedLineRoles.value.length > 0) ? allowedLineRoles.value.join(',') : undefined
    const res: any = await api.system.users.list({ page: 1, page_size: 20, username: q || undefined, roles: rolesParam })
    const list: any[] = Array.isArray(res?.items) ? res.items : []
    const newOptions = list.map((u: any) => ({ id: u.id, label: buildUserLabel(u) }))
    // 确保当前选中用户在选项中
    const selectedId = Number((form.network_line_fee_owner_id as any) || 0)
    if (selectedId > 0 && !newOptions.some(o => o.id === selectedId)) {
      const existing = ownerUserLineOptions.value.find(o => o.id === selectedId)
      if (existing) newOptions.unshift(existing)
    }
    ownerUserLineOptions.value = newOptions
  } catch {
    ownerUserLineOptions.value = []
  } finally {
    ownerUserLineLoading.value = false
  }
}

// 远程搜索系统用户（节点）
async function remoteSearchSystemUsersNode(q: string) {
  ownerUserNodeLoading.value = true
  try {
    // 确保已加载允许的节点角色列表
    if (!allowedNodeRoles.value || allowedNodeRoles.value.length === 0) {
      try { allowedNodeRoles.value = await api.system.binding.getAllowedUserRoles('node') } catch { allowedNodeRoles.value = [] }
    }
    const rolesParam = (allowedNodeRoles.value && allowedNodeRoles.value.length > 0) ? allowedNodeRoles.value.join(',') : undefined
    const res: any = await api.system.users.list({ page: 1, page_size: 20, username: q || undefined, roles: rolesParam })
    const list: any[] = Array.isArray(res?.items) ? res.items : []
    const newOptions = list.map((u: any) => ({ id: u.id, label: buildUserLabel(u) }))
    // 确保当前选中用户在选项中
    const selectedId = Number((form.general_fee_owner_id as any) || 0)
    if (selectedId > 0 && !newOptions.some(o => o.id === selectedId)) {
      const existing = ownerUserNodeOptions.value.find(o => o.id === selectedId)
      if (existing) newOptions.unshift(existing)
    }
    ownerUserNodeOptions.value = newOptions
  } catch {
    ownerUserNodeOptions.value = []
  } finally {
    ownerUserNodeLoading.value = false
  }
}

onMounted(async () => {
  loadRegionsAndCPs();
  fetchData();
  // 预加载允许绑定的角色（销售），供后续用户搜索使用
  try {
    allowedBindRoles.value = await api.system.binding.getAllowedUserRoles('sales')
  } catch { allowedBindRoles.value = [] }
  // 预加载线路用户角色
  try {
    allowedLineRoles.value = await api.system.binding.getAllowedUserRoles('line')
  } catch { allowedLineRoles.value = [] }
  // 预加载节点用户角色
  try {
    allowedNodeRoles.value = await api.system.binding.getAllowedUserRoles('node')
  } catch { allowedNodeRoles.value = [] }
})

// 监听两个归属字段：若为数字但对应选项缺失，则预加载该用户到选项，避免显示为纯数字
watch(() => form.customer_fee_owner_id as any, async (val) => {
  const n = Number(val)
  if (!Number.isNaN(n) && n > 0 && !ownerUserOptions.value.some(o => o.id === n)) {
    try { await preloadSelectedUsersIntoOptions([n]) } catch {}
  }
})

watch(() => form.network_line_fee_owner_id as any, async (val) => {
  const n = Number(val)
  if (!Number.isNaN(n) && n > 0 && !ownerUserLineOptions.value.some(o => o.id === n)) {
    try { await preloadSelectedUsersIntoOptions([n]) } catch {}
  }
})

// 监听“节点通用费归属（节点供应商）”选中值，确保选项中有对应 label
watch(() => form.general_fee_owner_id as any, async (val) => {
  const n = Number(val)
  if (!Number.isNaN(n) && n > 0 && !ownerUserNodeOptions.value.some(o => o.id === n)) {
    try { await preloadSelectedUsersIntoOptions([n]) } catch {}
  }
})
</script>

<style scoped>
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
