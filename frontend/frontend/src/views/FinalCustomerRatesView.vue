<template>
  <div class="rates-view">
    <h1 class="page-title">最终客户费率</h1>
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">最终客户费率筛选</span>
          <div>
            <QueryActionButton :running="queryCtl.running.value" @trigger="onSearch" />
            <el-button @click="onReset">重置</el-button>
            <el-button type="info" :loading="exporting" @click="onExport">导出</el-button>
            <el-button v-if="canWrite" type="success" @click="openDialog()">新增/更新</el-button>
            <el-button v-if="canWrite" type="warning" :loading="refreshing" @click="onRefresh">初始化并刷新最终费率</el-button>
            <el-button v-if="canWrite" type="danger" :loading="cleaning" @click="onCleanupInvalid">清理无效数据</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="区域">
          <SearchSelect
            v-model="query.region"
            :options="regionOptions"
            clearable
            placeholder="选择区域"
            class="field-w-160"
          />
        </el-form-item>
        <el-form-item label="CP">
          <SearchSelect
            v-model="query.cp"
            :options="cpOptions"
            clearable
            placeholder="选择 CP"
            class="field-w-160"
          />
        </el-form-item>
        <el-form-item label="学校">
          <SearchSelect
            v-model="query.school_name"
            clearable
            remote
            :remote-method="remoteSearchSchoolsFilter"
            :loading="schoolsLoading"
            :options="schoolOptions"
            label-key="name"
            value-key="name"
            placeholder="搜索学校"
            class="field-w-220"
            @visible-change="(visible) => visible && remoteSearchSchoolsFilter('')"
          >
            <template #option="{ option }">
              <div class="school-option">
                <span>{{ (option as any).name }}</span>
                <span class="school-option-tags">
                  <el-tag v-if="(option as any).inRate" size="small" type="success">已纳入费率</el-tag>
                  <el-tag v-else size="small" type="warning">未纳入费率</el-tag>
                </span>
              </div>
            </template>
          </SearchSelect>
        </el-form-item>
        <el-form-item label="服务日期">
          <el-date-picker
            v-model="query.service_date"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="选择服务日期"
            class="field-w-160"
          />
        </el-form-item>
        <!-- 费率类型筛选暂时隐藏 -->
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card mt-2">
      <template #header>
        <div class="card-header"><span class="card-title">费率列表</span></div>
      </template>

      <el-table :data="itemsCombined" border stripe height="600px" v-loading="loading">
        <el-table-column prop="region" label="区域" width="120" />
        <el-table-column prop="cp" label="CP" width="120" />
        <el-table-column prop="school_name" label="学校" min-width="160" show-overflow-tooltip />
        <el-table-column prop="service_date" label="服务日期" width="140" />
        <el-table-column prop="customer_fee" label="客户费" width="120" />
        <el-table-column prop="customer_fee_discount" label="客户费(折后)" width="140" />
        <el-table-column prop="network_line_fee" label="线路费" width="120" />
        <el-table-column prop="channel_rate" label="渠道费率" width="140" />
        <el-table-column prop="network_line_fee_discount" label="线路费(折后)" width="140" />
        <el-table-column prop="channel_rate_discount" label="渠道费率(折后)" width="160" />
        <el-table-column label="客户费归属" min-width="160">
          <template #default="{ row }">
            <el-tooltip placement="top" :content="`ID: ${row.customer_fee_owner_id ?? '-'}`">
              <span>{{ displayOwner(row.customer_fee_owner_id) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="线路费归属" min-width="160">
          <template #default="{ row }">
            <el-tooltip placement="top" :content="`ID: ${row.network_line_fee_owner_id ?? '-'}`">
              <span>{{ displayOwner(row.network_line_fee_owner_id) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="渠道费归属" min-width="160">
          <template #default="{ row }">
            <el-tooltip placement="top" :content="`ID: ${row.channel_owner_user_id ?? '-'}`">
              <span>{{ displayOwner(row.channel_owner_user_id) }}</span>
            </el-tooltip>
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

    <el-dialog v-model="dialogVisible" title="新增/更新 最终客户费率" width="720px">
      <el-form :model="form" label-width="140px">
        <el-form-item label="区域" required>
          <SearchSelect
            v-model="form.region"
            :options="regionOptions"
            placeholder="选择区域"
            class="field-w-240"
          />
        </el-form-item>
        <el-form-item label="CP" required>
          <SearchSelect
            v-model="form.cp"
            :options="cpOptions"
            placeholder="选择 CP"
            class="field-w-240"
          />
        </el-form-item>
        <el-form-item label="学校" required>
          <SearchSelect
            v-model="form.school_name"
            clearable
            remote
            :remote-method="remoteSearchSchoolsDialog"
            :loading="schoolsLoading"
            :options="schoolOptions"
            label-key="name"
            value-key="name"
            placeholder="搜索学校"
            class="field-w-300"
            @visible-change="(visible) => visible && remoteSearchSchoolsDialog('')"
          >
            <template #option="{ option }">
              <div class="school-option">
                <span>{{ (option as any).name }}</span>
                <span class="school-option-tags">
                  <el-tag v-if="(option as any).inRate" size="small" type="success">已纳入费率</el-tag>
                  <el-tag v-else size="small" type="warning">未纳入费率</el-tag>
                </span>
              </div>
            </template>
          </SearchSelect>
        </el-form-item>

        <el-divider content-position="left">费率</el-divider>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="客户费">
              <el-input-number v-model="form.customer_fee" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="专线费">
              <el-input-number v-model="form.network_line_fee" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="渠道费率">
              <el-input-number v-model="form.channel_rate" :min="0" :step="0.01" :precision="2" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-divider content-position="left">归属</el-divider>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="客户费归属用户ID">
              <el-input-number v-model="form.customer_fee_owner_id" :min="1" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="专线费归属用户ID">
              <el-input-number v-model="form.network_line_fee_owner_id" :min="1" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="渠道费归属用户ID">
              <el-input-number v-model="form.channel_owner_user_id" :min="1" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import type { DiscountedFinalCustomerRate, PaginatedData, UpsertRateFinalCustomerRequest, RateFinalCustomer } from '@/types/api'
import { useAuthStore } from '@/stores/auth'
import { buildCsvContent, formatExportFilename, triggerBlobDownload } from '@/utils/export'
import { EXPORT_FILENAME_PREFIX, EXPORT_HEADERS } from '@/utils/export-standards'
import { loadVisibleRateScopeOptions, searchRateSchoolOptions, type RateSchoolOption } from './rate-filter-options'
import SearchSelect from '@/components/ui/SearchSelect.vue'
import QueryActionButton from '@/components/ui/QueryActionButton.vue'
import { useCancelableQuery, isAbortError } from '@/composables/useCancelableQuery'
import { usePageRefresh } from '@/composables/usePageRefresh'

const auth = useAuthStore()
const canWrite = computed(() => auth.hasPermission('rates.final.write'))

const loading = ref(false)
const refreshing = ref(false)
const cleaning = ref(false)
const exporting = ref(false)
const itemsCombined = ref<Array<RateFinalCustomer & { service_date?: string; customer_fee_discount?: number | null; network_line_fee_discount?: number | null; channel_rate_discount?: number | null }>>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const queryCtl = useCancelableQuery()
const regionOptions = ref<string[]>([])
const cpOptions = ref<string[]>([])
const schoolOptions = ref<RateSchoolOption[]>([])
const schoolsLoading = ref(false)

const query = reactive<{ region?: string; cp?: string; school_name?: string; service_date?: string }>({})

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.region) p.region = query.region
  if (query.cp) p.cp = query.cp
  if (query.school_name) p.school_name = query.school_name
  if (query.service_date) p.service_date = query.service_date
  // fee_type 暂不作为筛选参数
  return p
}

async function fetchData(signal?: AbortSignal) {
  loading.value = true
  try {
    // 默认服务日期为今天（用于折损计算）
    if (!query.service_date) {
      const d = new Date()
      const mm = `${d.getMonth() + 1}`.padStart(2, '0')
      const dd = `${d.getDate()}`.padStart(2, '0')
      query.service_date = `${d.getFullYear()}-${mm}-${dd}`
    }
    const [origRes, discRes]: [PaginatedData<RateFinalCustomer>, PaginatedData<DiscountedFinalCustomerRate>] = await Promise.all([
      api.settlementRates.final.list(buildParams(), { signal }),
      api.settlementRates.final.listDiscounted(buildParams(), { signal }),
    ])
    const orig = (origRes?.items || []) as RateFinalCustomer[]
    total.value = Number(origRes?.total || 0)
    const discList = (discRes?.items || []) as DiscountedFinalCustomerRate[]
    const discMap = new Map<string, DiscountedFinalCustomerRate>()
    for (const d of discList) {
      const key = `${d.region}|${d.cp}|${d.school_name ?? ''}`
      discMap.set(key, d)
    }
    const merged = orig.map(r => {
      const key = `${r.region}|${r.cp}|${r.school_name ?? ''}`
      const d = discMap.get(key)
      return {
        ...r,
        service_date: d?.service_date || query.service_date,
        customer_fee_discount: (d?.customer_fee_discount as any) ?? null,
        network_line_fee_discount: (d?.network_line_fee_discount as any) ?? null,
        channel_rate_discount: (d?.channel_rate_discount as any) ?? null,
      }
    })
    itemsCombined.value = merged
    await loadUsersForItems(signal)
  } catch (e: any) {
    if (isAbortError(e)) return
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadRegionsAndCPs() {
  try {
    const { regions, cps } = await loadVisibleRateScopeOptions()
    regionOptions.value = regions
    cpOptions.value = cps
  } catch {
    regionOptions.value = []
    cpOptions.value = []
  }
}

async function remoteSearchSchoolsFilter(q: string) {
  schoolsLoading.value = true
  try {
    schoolOptions.value = await searchRateSchoolOptions(
      { region: query.region, cp: query.cp, schoolName: q || undefined },
      (params) => api.settlementRates.final.list(params),
    )
  } catch {
    schoolOptions.value = []
  } finally {
    schoolsLoading.value = false
  }
}

async function remoteSearchSchoolsDialog(q: string) {
  schoolsLoading.value = true
  try {
    schoolOptions.value = await searchRateSchoolOptions(
      { region: form.region, cp: form.cp, schoolName: q || undefined },
      (params) => api.settlementRates.final.list(params),
    )
  } catch {
    schoolOptions.value = []
  } finally {
    schoolsLoading.value = false
  }
}

function onSearch() { page.value = 1; queryCtl.run((signal) => fetchData(signal), { toggleIfRunning: true }) }
function onReset() { Object.assign(query, { region: undefined, cp: undefined, school_name: undefined, service_date: undefined }); page.value=1; pageSize.value=10; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }
function onPageChange(p: number) { page.value = p; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }

// 系统用户映射（id -> 用户基本信息），用于优先显示“系统用户别名/名称”
const userMap = ref<Record<number, { id: number; alias?: string; display_name?: string; username: string }>>({})

// 批量按 items 中出现的 owner_id 拉取系统用户，填充 userMap
async function loadUsersForItems(signal?: AbortSignal) {
  const ids = new Set<number>()
  for (const r of itemsCombined.value) {
    if (r?.customer_fee_owner_id != null) { const n = Number(r.customer_fee_owner_id); if (!Number.isNaN(n) && n > 0) ids.add(n) }
    if (r?.network_line_fee_owner_id != null) { const n = Number(r.network_line_fee_owner_id); if (!Number.isNaN(n) && n > 0) ids.add(n) }
    if (r?.channel_owner_user_id != null) { const n = Number(r.channel_owner_user_id); if (!Number.isNaN(n) && n > 0) ids.add(n) }
  }
  if (ids.size === 0) { userMap.value = {}; return }
  try {
    const res: any = await api.system.users.list({ ids: Array.from(ids).join(',') }, { signal })
    const list: any[] = Array.isArray(res?.items) ? res.items : []
    const m: Record<number, { id: number; alias?: string; display_name?: string; username: string }> = {}
    for (const u of list) {
      if (u && typeof u.id === 'number') m[u.id] = { id: u.id, alias: u.alias, display_name: u.display_name, username: u.username }
    }
    userMap.value = m
  } catch { userMap.value = {} }
}

// 统一的 owner 显示：仅系统用户别名/名称
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
  return `无效用户ID#${key}`
}

// Dialog
const dialogVisible = ref(false)
const saving = ref(false)
const DEFAULT_FEE_TYPE = 'auto'
const form = reactive<UpsertRateFinalCustomerRequest>({ region: '', cp: '', school_name: '', fee_type: DEFAULT_FEE_TYPE })

function openDialog() {
  Object.assign(form, { region: '', cp: '', school_name: '', fee_type: DEFAULT_FEE_TYPE, customer_fee: undefined, customer_fee_owner_id: undefined, network_line_fee: undefined, network_line_fee_owner_id: undefined, channel_rate: undefined, channel_owner_user_id: undefined })
  dialogVisible.value = true
}

async function onSave() {
  if (!form.region || !form.cp || !form.school_name) { ElMessage.warning('区域/CP/学校为必填'); return }
  saving.value = true
  try {
    await api.settlementRates.final.upsert(form)
    ElMessage.success('保存成功')
    dialogVisible.value = false
    queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function onRefresh() {
  refreshing.value = true
  try {
    const initAffected = await api.settlementRates.final.initFromCustomer()
    const refreshAffected = await api.settlementRates.final.refresh({})
    ElMessage.success(`初始化 ${initAffected} 条，刷新 ${refreshAffected} 条`)
    queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
  } catch (e: any) {
    const msg = e?.response?.data?.message || e?.message || '初始化/刷新失败'
    ElMessage.error(msg)
  } finally {
    refreshing.value = false
  }
}

async function onCleanupInvalid() {
  try {
    await ElMessageBox.confirm('将删除 fee_type=auto 且任意关键费率字段为空的最终费率记录，是否继续？', '确认清理', { type: 'warning', confirmButtonText: '清理', cancelButtonText: '取消' })
  } catch {
    return
  }
  cleaning.value = true
  try {
    const affected = await api.settlementRates.final.cleanupInvalid()
    ElMessage.success(`已清理 ${affected} 条无效记录`)
    queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
  } catch (e: any) {
    const msg = e?.response?.data?.message || e?.message || '清理失败'
    ElMessage.error(msg)
  } finally {
    cleaning.value = false
  }
}

onMounted(() => {
  loadRegionsAndCPs()
  queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
})
usePageRefresh(() => {
  queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
})

// 导出当前页为 CSV（根据视图类型导出对应列）
async function onExport() {
  try {
    exporting.value = true
    const rows: any[] = itemsCombined.value || []
    const header = [...EXPORT_HEADERS.finalCustomerRates]
    const contentRows: Array<Array<unknown>> = []
    const ownerName = (id?: number | null) => {
      if (!id) return ''
      const key = Number(id)
      const u = userMap.value[key]
      if (u) {
        const alias = (u.alias && String(u.alias).trim()) ? String(u.alias).trim() : ''
        const dn = (u.display_name && String(u.display_name).trim()) ? String(u.display_name).trim() : ''
        const un = (u.username && String(u.username).trim()) ? String(u.username).trim() : ''
        return alias || dn || un || `用户#${key}`
      }
      return `#${key}`
    }
    for (const r of rows) {
      contentRows.push([
        r.region, r.cp, r.school_name ?? '', r.service_date ?? '',
        r.customer_fee ?? '', r.customer_fee_discount ?? '', r.network_line_fee ?? '', r.channel_rate ?? '',
        ownerName(r.customer_fee_owner_id), ownerName(r.network_line_fee_owner_id), ownerName(r.channel_owner_user_id),
      ])
    }
    const content = buildCsvContent(header, contentRows)
    const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
    triggerBlobDownload(blob, formatExportFilename(EXPORT_FILENAME_PREFIX.finalCustomerRates, 'csv'))
  } catch (e: any) {
    ElMessage.error(e?.message || '导出失败')
  } finally {
    exporting.value = false
  }
}
</script>

<style scoped>
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
.school-option { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.school-option-tags { display: inline-flex; gap: 4px; }
</style>

