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
        <div class="card-header"><span class="card-title">费率列表</span></div>
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
        <el-table-column prop="network_line_fee" label="专线费" width="120" />
        <el-table-column prop="general_fee" label="通用费" width="120" />
        <el-table-column label="客户费归属" width="200">
          <template #default="{ row }">
            <span v-if="row.customer_fee_owner_id">{{ displayOwner(row.customer_fee_owner_id) }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="专线费归属" width="200">
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
        <el-form-item label="专线费">
          <el-input-number v-model="form.network_line_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="通用费">
          <el-input-number v-model="form.general_fee" :min="0" :step="0.01" :precision="2" />
        </el-form-item>
        <el-form-item label="客户费归属对象">
          <el-select
            v-model="form.customer_fee_owner_id"
            filterable
            remote
            clearable
            :remote-method="remoteSearchEntitiesForCustomer"
            :loading="entitiesLoading"
            placeholder="搜索业务对象名称"
            style="width: 300px"
          >
            <el-option
              v-for="e in entityOptionsCustomer"
              :key="e.id"
              :label="`${e.entity_name} (${e.entity_type})`"
              :value="e.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="专线费归属对象">
          <el-select
            v-model="form.network_line_fee_owner_id"
            filterable
            remote
            clearable
            :remote-method="remoteSearchEntitiesForNetwork"
            :loading="entitiesLoading"
            placeholder="搜索业务对象名称"
            style="width: 300px"
          >
            <el-option
              v-for="e in entityOptionsNetwork"
              :key="e.id"
              :label="`${e.entity_name} (${e.entity_type})`"
              :value="e.id"
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
import { computed, onMounted, reactive, ref } from 'vue'
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

const query = reactive<{ region?: string; cp?: string; school_name?: string }>({})
// 下拉与远程搜索状态
const regionOptions = ref<string[]>([])
const cpOptions = ref<string[]>([])
const schoolOptions = ref<string[]>([])
const schoolsLoading = ref(false)
const entitiesLoading = ref(false)
const entityOptionsCustomer = ref<BusinessEntity[]>([])
const entityOptionsNetwork = ref<BusinessEntity[]>([])
// ID -> 实体 映射，用于列表展示
const entityMap = ref<Record<number, BusinessEntity>>({})

// 扩展字段弹窗
const extraDialogVisible = ref(false)
const extraDialogContent = ref<any>(null)

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.region) p.region = query.region
  if (query.cp) p.cp = query.cp
  if (query.school_name) p.school_name = query.school_name
  return p
}

async function fetchData() {
  loading.value = true
  try {
    const res: PaginatedData<RateCustomer> = await api.settlementRates.customer.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
    // 批量加载归属对象信息，构建映射
    loadEntitiesForItems()
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

function displayOwner(id?: number | null): string {
  if (!id) return '-'
  const e = entityMap.value[id as number]
  if (e) return `${e.entity_name}`
  return String(id)
}

async function loadRegionsAndCPs() {
  try {
    const [regions, cps] = await Promise.all([
      api.getRegions(),
      api.getCPs(),
    ])
    regionOptions.value = Array.isArray(regions) ? regions : []
    cpOptions.value = Array.isArray(cps) ? cps : []
  } catch {}
}

// 学校远程搜索（筛选区）
async function remoteSearchSchoolsFilter(q: string) {
  schoolsLoading.value = true
  try {
    const data = await api.getSchools({ region: query.region, cp: query.cp, school_name: q || undefined, page: 1, page_size: 20 })
    const list: any[] = Array.isArray(data?.items) ? data.items : (Array.isArray(data) ? data : [])
    schoolOptions.value = list.map((it: any) => it?.school_name || it?.name || it).filter(Boolean)
  } catch {}
  finally { schoolsLoading.value = false }
}

// 学校远程搜索（弹窗区）
async function remoteSearchSchoolsDialog(q: string) {
  schoolsLoading.value = true
  try {
    const data = await api.getSchools({ region: form.region, cp: form.cp, school_name: q || undefined, page: 1, page_size: 20 })
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

async function remoteSearchEntitiesForNetwork(q: string) {
  entitiesLoading.value = true
  try {
    const res = await api.settlementEntities.list({ page: 1, page_size: 20, entity_name: q || undefined })
    entityOptionsNetwork.value = Array.isArray((res as any)?.items) ? (res as any).items as BusinessEntity[] : []
  } catch {}
  finally { entitiesLoading.value = false }
}

function onSearch() { page.value = 1; fetchData() }
function onReset() { Object.assign(query, { region: undefined, cp: undefined, school_name: undefined }); page.value=1; pageSize.value=10; fetchData() }
function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

function goSyncRules() { router.push({ name: 'settlement-rates-sync-rules' }) }

// Dialog
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive<UpsertRateCustomerRequest>({ region: '', cp: '' })
const extraEditorText = ref<string>('')

function openDialog(row?: RateCustomer) {
  if (row) {
    Object.assign(form, {
      region: row.region,
      cp: row.cp,
      school_name: row.school_name ?? undefined,
      customer_fee: row.customer_fee ?? undefined,
      network_line_fee: row.network_line_fee ?? undefined,
      general_fee: row.general_fee ?? undefined,
      customer_fee_owner_id: row.customer_fee_owner_id ?? undefined,
      network_line_fee_owner_id: row.network_line_fee_owner_id ?? undefined,
    })
    extraEditorText.value = stringify(row.extra ?? {})
  } else {
    Object.assign(form, { region: '', cp: '', school_name: undefined, customer_fee: undefined, network_line_fee: undefined, general_fee: undefined, customer_fee_owner_id: undefined, network_line_fee_owner_id: undefined })
    extraEditorText.value = ''
  }
  dialogVisible.value = true
}

async function onSave() {
  if (!form.region || !form.cp) { ElMessage.warning('区域与运营商为必填'); return }
  saving.value = true
  try {
    // 解析扩展 JSON（可选）
    let payload: any = { ...form }
    const txt = (extraEditorText.value || '').trim()
    if (txt) {
      try { payload.extra = JSON.parse(txt) } catch (e) { ElMessage.error('扩展JSON格式错误'); saving.value=false; return }
    }
    await api.settlementRates.customer.upsert(payload)
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

onMounted(() => { loadRegionsAndCPs(); fetchData() })
</script>

<style scoped>
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
