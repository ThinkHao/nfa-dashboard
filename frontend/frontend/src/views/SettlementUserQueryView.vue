<template>
  <div class="single-user-settlement-view">
    <el-card class="filter-card" shadow="hover">
      <div class="toolbar">
        <div class="title">
          单用户结算查询
      <el-tag v-if="currentDataSourceLabel" size="small" type="info" class="ml-8">来源：{{ currentDataSourceLabel }}</el-tag>
        </div>
        <div class="actions">
          <QueryActionButton :running="queryCtl.running.value" @trigger="handleQuery" />
          <el-button @click="handleReset">重置</el-button>
          <el-button type="success" :loading="exporting" @click="handleExport">导出</el-button>
        </div>
      </div>

      <el-form :inline="true" class="query-form">
        <el-form-item label="用户" required>
          <SearchSelect
            v-model="filter.userId"
            :options="userOptions"
            :loading="userOptionsLoading"
            label-key="label"
            value-key="id"
            clearable
            placeholder="请选择用户"
            class="field-w-240"
            @visible-change="onUserDropdownVisible"
          />
        </el-form-item>
        <el-form-item label="粒度">
          <el-select v-model="filter.granularity" class="field-w-120">
            <el-option label="按月" value="monthly" />
            <el-option label="按日" value="daily" />
          </el-select>
        </el-form-item>
        <el-form-item label="查看视图">
          <el-select v-model="viewMode" class="field-w-160" :disabled="filter.granularity !== 'monthly'">
            <el-option label="明细视图" value="detail" />
            <el-option label="按月列视图" value="monthly_columns" />
          </el-select>
        </el-form-item>
        <el-form-item label="服务月份">
          <UnifiedDateRange
            v-model="monthRange"
            type="monthrange"
            format="YYYY-MM"
            value-format="YYYY-MM"
            class="field-w-260"
            @change="handleMonthRangeChange"
          />
        </el-form-item>
        <el-form-item label="地区">
          <SearchSelect v-model="filter.region" :options="regions" clearable class="field-w-160" @change="onRegionChange" />
        </el-form-item>
        <el-form-item label="CP">
          <SearchSelect v-model="filter.cp" :options="cps" clearable class="field-w-160" @change="onCPChange" />
        </el-form-item>
        <el-form-item label="学校">
          <SearchSelect v-model="filter.schoolName" :options="schools" label-key="school_name" value-key="school_name" clearable class="field-w-280" />
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="table-card" shadow="hover">
      <el-table v-if="!isMonthlyColumnView" v-loading="loading" :data="rows" border stripe class="field-w-full">
        <el-table-column prop="school_name" label="学校名称" min-width="180" />
        <el-table-column prop="region" label="地区" width="110" />
        <el-table-column prop="cp" label="CP" width="110" />
        <el-table-column prop="service_date" :label="serviceDateLabel" width="130" />
        <el-table-column prop="stock_start_at" label="存量起算时间" width="130">
          <template #default="{ row }">{{ row.stock_start_at || '-' }}</template>
        </el-table-column>
        <el-table-column prop="increment_start_at" label="增量起算时间" width="130">
          <template #default="{ row }">{{ row.increment_start_at || '-' }}</template>
        </el-table-column>
        <el-table-column :label="`日95流量值(${singleUserRateUnit})`" width="150">
          <template #default="{ row }">{{ fmtFlowRate(row.settlement_value) }}</template>
        </el-table-column>
        <el-table-column label="客户金额" width="120">
          <template #default="{ row }">{{ fmtMoney(row.customer_bill) }}</template>
        </el-table-column>
        <el-table-column label="线路金额" width="120">
          <template #default="{ row }">{{ fmtMoney(row.network_line_bill) }}</template>
        </el-table-column>
        <el-table-column label="节点金额" width="120">
          <template #default="{ row }">{{ fmtMoney(row.node_deduction_bill) }}</template>
        </el-table-column>
        <el-table-column label="渠道金额" width="120">
          <template #default="{ row }">{{ fmtMoney(row.channel_bill) }}</template>
        </el-table-column>
        <el-table-column label="总归属金额" width="130">
          <template #default="{ row }">{{ fmtTotal(row) }}</template>
        </el-table-column>
      </el-table>

      <el-table
        v-else
        v-loading="columnViewLoading"
        :data="monthlyColumnRows"
        :row-key="(row) => row.id || row.metric"
        :tree-props="monthlyTreeProps"
        :expand-row-keys="monthlyExpandedRowKeys"
        :row-class-name="monthlyColumnRowClassName"
        border
        class="field-w-full"
      >
        <el-table-column prop="metric" label="分组/学校" min-width="220" fixed="left">
          <template #default="{ row }">
            <span>{{ row.metric }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="stockStartAt" label="存量起算时间" min-width="130" />
        <el-table-column prop="incrementStartAt" label="增量起算时间" min-width="130" />
        <el-table-column prop="daily95Rate" :label="`日95均值(${singleUserRateUnit})`" min-width="130" />
        <el-table-column v-for="month in monthlyColumnMonths" :key="month" :label="month" min-width="120">
          <template #default="{ row }">{{ row.values[month] || '-' }}</template>
        </el-table-column>
      </el-table>

      <div v-if="!isMonthlyColumnView" class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="pagination.total"
          @size-change="onSizeChange"
          @current-change="onPageChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { School } from '@/types/api'
import { buildMonthlyAmountColumnView, normalizeDateText, resolveMonthRangeDateTime, type MonthlyMetricRow } from './settlement-user-query-utils'
import { buildCsvContent, formatExportFilename, triggerBlobDownload } from '@/utils/export'
import { EXPORT_FILENAME_PREFIX } from '@/utils/export-standards'
import SearchSelect from '@/components/ui/SearchSelect.vue'
import UnifiedDateRange from '@/components/ui/UnifiedDateRange.vue'
import QueryActionButton from '@/components/ui/QueryActionButton.vue'
import { useCancelableQuery, isAbortError } from '@/composables/useCancelableQuery'
import { usePageRefresh } from '@/composables/usePageRefresh'
import { useSystemTrafficSettings } from '@/composables/useSystemTrafficSettings'
import { settlementValueToRate } from '@/utils/traffic-units'

type Granularity = 'daily' | 'monthly'
type UserOption = { id: number; label: string }
type ViewMode = 'detail' | 'monthly_columns'

const loading = ref(false)
const exporting = ref(false)
const rows = ref<any[]>([])
const columnViewLoading = ref(false)
const viewMode = ref<ViewMode>('detail')
const monthlyColumnMonths = ref<string[]>([])
const monthlyColumnRows = ref<MonthlyMetricRow[]>([])
const queryCtl = useCancelableQuery()
const trafficSettings = useSystemTrafficSettings()
const singleUserRateUnit = computed<'Mbps' | 'Gbps'>(() => (
  trafficSettings.settings.value.settlement_single_user_rate_unit === 'Mbps' ? 'Mbps' : 'Gbps'
))

const userOptions = ref<UserOption[]>([])
const userOptionsLoading = ref(false)
const ownerUsersFetchSeq = ref(0)
const lastOwnerUsersQueryKey = ref('')
const regions = ref<string[]>([])
const cps = ref<string[]>([])
const schools = ref<School[]>([])

const filter = reactive({
  userId: null as number | null,
  granularity: 'monthly' as Granularity,
  region: '',
  cp: '',
  schoolName: '',
})

const monthRange = ref<[string, string] | null>(null)
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const serviceDateLabel = computed(() => (filter.granularity === 'monthly' ? '服务月份' : '服务日期'))
const isMonthlyColumnView = computed(() => filter.granularity === 'monthly' && viewMode.value === 'monthly_columns')
const isMonthlyTreeMode = computed(() => isMonthlyColumnView.value && !filter.region && !filter.cp && !filter.schoolName)
const monthlyTreeProps = computed(() => ({ children: 'children' as const }))
const monthlyExpandedRowKeys = computed(() => {
  if (!isMonthlyTreeMode.value) return [] as string[]
  return monthlyColumnRows.value.filter((row) => row.rowType === 'region' && row.id).map((row) => String(row.id))
})
const currentDataSourceLabel = computed(() => {
  const first = rows.value.length ? rows.value[0] : null
  const src = String(first?.data_source || '').toLowerCase()
  if (src === 'snapshot') return '月快照'
  if (src === 'realtime') return '实时聚合'
  return ''
})

function parseMonth(ym: string): Date {
  const [y, m] = ym.split('-').map((x) => Number(x))
  return new Date(y, (m || 1) - 1, 1)
}

function monthDiffInclusive(startYm: string, endYm: string): number {
  const s = parseMonth(startYm)
  const e = parseMonth(endYm)
  return (e.getFullYear() - s.getFullYear()) * 12 + (e.getMonth() - s.getMonth()) + 1
}

function setDefaultMonthRange() {
  const end = new Date()
  const start = new Date(end.getFullYear(), end.getMonth() - 2, 1)
  const endYm = `${end.getFullYear()}-${String(end.getMonth() + 1).padStart(2, '0')}`
  const startYm = `${start.getFullYear()}-${String(start.getMonth() + 1).padStart(2, '0')}`
  monthRange.value = [startYm, endYm]
}

function fmtMoney(v: any): string {
  if (v == null || v === '') return '-'
  const n = Number(v)
  if (Number.isNaN(n)) return '-'
  return n.toFixed(2)
}

function fmtFlowRate(v: any): string {
  if (v == null || v === '') return '-'
  const n = Number(v)
  if (Number.isNaN(n)) return '-'
  return settlementValueToRate(n, singleUserRateUnit.value).toFixed(2)
}

function fmtTotal(row: any): string {
  const vals = [row?.customer_bill, row?.network_line_bill, row?.node_deduction_bill, row?.channel_bill]
  const nums = vals.map((v) => (v == null ? null : Number(v))).filter((v) => v != null && !Number.isNaN(v)) as number[]
  if (!nums.length) return '-'
  return nums.reduce((a, b) => a + b, 0).toFixed(2)
}

function buildParams(page = pagination.page, pageSize = pagination.pageSize) {
  const { start, end } = resolveMonthRangeDateTime(monthRange.value)
  const params: any = { page, page_size: pageSize, channel_owner_user_id: filter.userId }
  if (start) params.start_service_date = start
  if (end) params.end_service_date = end
  if (filter.region) params.region = filter.region
  if (filter.cp) params.cp = filter.cp
  if (filter.schoolName) params.school_name = filter.schoolName
  return params
}

function parseDateOnly(value: unknown): Date | null {
  const normalized = normalizeDateText(value)
  if (!normalized) return null
  const m = normalized.match(/^(\d{4})-(\d{2})-(\d{2})/)
  if (!m) return null
  const y = Number(m[1]); const mo = Number(m[2]); const d = Number(m[3])
  if (!Number.isFinite(y) || !Number.isFinite(mo) || !Number.isFinite(d)) return null
  return new Date(y, mo - 1, d)
}

function monthRangeBoundary(): { startMonth: string; endMonth: string } | null {
  if (!monthRange.value || !monthRange.value[0] || !monthRange.value[1]) return null
  return { startMonth: String(monthRange.value[0]), endMonth: String(monthRange.value[1]) }
}

function clipRowsBySelectedMonths<T extends Record<string, any>>(inputRows: T[]): T[] {
  const boundary = monthRangeBoundary()
  if (!boundary) return inputRows
  const { startMonth, endMonth } = boundary
  return (inputRows || []).filter((row) => {
    const month = String(row?.service_date || '').slice(0, 7)
    return !!month && month >= startMonth && month <= endMonth
  })
}

function pickEffectiveRate(rates: any[], serviceDateText: string): any | null {
  const serviceDate = parseDateOnly(serviceDateText)
  if (!serviceDate) return null
  const candidates = (rates || []).filter((rate) => {
    const startDate = parseDateOnly(rate?.start_at)
    if (!startDate) return true
    return startDate.getTime() <= serviceDate.getTime()
  })
  if (!candidates.length) return null
  return [...candidates].sort((a, b) => {
    const ad = parseDateOnly(a?.start_at)?.getTime() || 0
    const bd = parseDateOnly(b?.start_at)?.getTime() || 0
    return bd - ad
  })[0]
}

async function enrichRowsWithStartDates(inputRows: any[], signal?: AbortSignal): Promise<any[]> {
  if (!Array.isArray(inputRows) || inputRows.length === 0) return []
  const keySet = new Set<string>()
  for (const row of inputRows) {
    const region = String(row?.region || '')
    const cp = String(row?.cp || '')
    const schoolName = String(row?.school_name || '')
    if (!region || !cp || !schoolName) continue
    keySet.add(`${region}__${cp}__${schoolName}`)
  }

  const rateMap = new Map<string, any[]>()
  await Promise.all(Array.from(keySet).map(async (key) => {
    if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
    const [region, cp, schoolName] = key.split('__')
    try {
      const res: any = await (api as any).settlementRates.customer.list({
        region,
        cp,
        school_name: schoolName,
        page: 1,
        page_size: 300,
      }, { signal })
      const items: any[] = Array.isArray(res?.items) ? res.items : (Array.isArray(res) ? res : [])
      rateMap.set(key, items)
    } catch {
      rateMap.set(key, [])
    }
  }))

  return inputRows.map((row) => {
    const key = `${String(row?.region || '')}__${String(row?.cp || '')}__${String(row?.school_name || '')}`
    const rates = rateMap.get(key) || []
    const effective = pickEffectiveRate(rates, String(row?.service_date || ''))
    return {
      ...row,
      stock_start_at: normalizeDateText(effective?.start_at) || '',
      increment_start_at: normalizeDateText(effective?.increment_start_at) || '',
    }
  })
}

async function fetchAllRowsForCurrentFilter(granularity: Granularity = filter.granularity, signal?: AbortSignal): Promise<any[]> {
  const all: any[] = []
  let page = 1
  const pageSize = 1000
  while (true) {
    if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
    const params = buildParams(page, pageSize)
    const res = granularity === 'monthly'
      ? await (api as any).settlementData.monthlyList(params, { signal })
      : await (api as any).settlementData.list(params, { signal })
    const items = Array.isArray(res) ? res : (Array.isArray(res?.items) ? res.items : [])
    const total = Array.isArray(res) ? items.length : Number(res?.total || 0)
    all.push(...items)
    if (items.length < pageSize) break
    if (total > 0 && page * pageSize >= total) break
    page += 1
  }
  return all
}

async function loadOwnerUsers() {
  ownerUsersFetchSeq.value += 1
  const currentSeq = ownerUsersFetchSeq.value
  try {
    const { start, end } = resolveMonthRangeDateTime(monthRange.value)
    const params: any = {}
    if (filter.region) params.region = filter.region
    if (filter.cp) params.cp = filter.cp
    if (start) params.start_service_date = start
    if (end) params.end_service_date = end
    const queryKey = JSON.stringify({
      region: params.region || '',
      cp: params.cp || '',
      start: params.start_service_date || '',
      end: params.end_service_date || '',
    })
    userOptionsLoading.value = true
    const items: any[] = await (api as any).settlementData.ownerSubjects(params)
    if (currentSeq !== ownerUsersFetchSeq.value) return
    const list = (Array.isArray(items) ? items : [])
      .filter((it: any) => it && String(it.type) === 'user')
      .map((it: any) => ({ id: Number(it.id), label: String(it.label || `用户#${it.id}`) }))
      .filter((it: UserOption) => Number.isFinite(it.id))
      .sort((a, b) => a.label.localeCompare(b.label))
    userOptions.value = list
    lastOwnerUsersQueryKey.value = queryKey
  } catch {
    // 保留已有选项，避免临时失败导致下拉被清空
  } finally {
    if (currentSeq === ownerUsersFetchSeq.value) {
      userOptionsLoading.value = false
    }
  }
}

function onUserDropdownVisible(visible: boolean) {
  if (!visible) return
  const { start, end } = resolveMonthRangeDateTime(monthRange.value)
  const queryKey = JSON.stringify({
    region: filter.region || '',
    cp: filter.cp || '',
    start: start || '',
    end: end || '',
  })
  const needReload = !userOptions.value.length || lastOwnerUsersQueryKey.value !== queryKey
  if (needReload && !userOptionsLoading.value) {
    loadOwnerUsers()
  }
}

async function loadRegionCpSchool() {
  try {
    const rs = await (api as any).getRegions()
    regions.value = Array.isArray(rs) ? rs.filter((x) => typeof x === 'string') : []
  } catch { regions.value = [] }

  try {
    const cs = await (api as any).getCPs()
    cps.value = Array.isArray(cs) ? cs.filter((x) => typeof x === 'string') : []
  } catch { cps.value = [] }

  await loadSchools()
}

async function loadSchools() {
  try {
    const params: any = { limit: 1000, offset: 0 }
    if (filter.region) params.region = filter.region
    if (filter.cp) params.cp = filter.cp
    const res = await (api as any).v2.getSchools(params)
    const list = Array.isArray(res) ? res : (Array.isArray(res?.items) ? res.items : [])
    const dedup = new Map<string, School>()
    for (const s of list) {
      const name = typeof s?.school_name === 'string' ? s.school_name.trim() : ''
      if (!name) continue
      if (!dedup.has(name)) dedup.set(name, s as School)
    }
    schools.value = Array.from(dedup.values()).sort((a, b) => String(a.school_name).localeCompare(String(b.school_name)))
  } catch {
    schools.value = []
  }
}

function validateBeforeQuery(): boolean {
  if (!filter.userId) {
    ElMessage.warning('请先选择用户')
    return false
  }
  if (!monthRange.value || !monthRange.value[0] || !monthRange.value[1]) {
    ElMessage.warning('请先选择服务月份范围')
    return false
  }
  const months = monthDiffInclusive(monthRange.value[0], monthRange.value[1])
  if (months > 12) {
    ElMessage.warning('查询时间跨度最多 12 个月')
    return false
  }
  return true
}

async function fetchRows(signal?: AbortSignal) {
  if (!validateBeforeQuery()) return
  loading.value = true
  try {
    const params = buildParams()
    const res = filter.granularity === 'monthly'
      ? await (api as any).settlementData.monthlyList(params, { signal })
      : await (api as any).settlementData.list(params, { signal })
    if (Array.isArray(res)) {
      const clipped = clipRowsBySelectedMonths(res)
      rows.value = await enrichRowsWithStartDates(clipped, signal)
      pagination.total = clipped.length
    } else {
      const list = Array.isArray(res?.items) ? res.items : []
      const clipped = clipRowsBySelectedMonths(list)
      rows.value = await enrichRowsWithStartDates(clipped, signal)
      pagination.total = clipped.length
    }
    await refreshMonthlyColumnView(signal)
  } catch (e: any) {
    if (isAbortError(e)) return
    ElMessage.error(e?.response?.data?.message || e?.message || '查询失败')
  } finally {
    loading.value = false
  }
}

async function fetchAllForExport(signal?: AbortSignal): Promise<any[]> {
  const data = await fetchAllRowsForCurrentFilter(filter.granularity, signal)
  return clipRowsBySelectedMonths(data)
}

async function refreshMonthlyColumnView(signal?: AbortSignal) {
  if (!isMonthlyColumnView.value) {
    monthlyColumnMonths.value = []
    monthlyColumnRows.value = []
    return
  }
  columnViewLoading.value = true
  try {
    const monthlyRows = await fetchAllRowsForCurrentFilter('monthly', signal)
    const dailyRows = await fetchAllRowsForCurrentFilter('daily', signal)
    const enrichedMonthlyRows = await enrichRowsWithStartDates(monthlyRows, signal)
    const { months, rows: pivotRows } = buildMonthlyAmountColumnView(enrichedMonthlyRows, dailyRows, {
      treeByRegionSchoolCp: isMonthlyTreeMode.value,
      rateUnit: singleUserRateUnit.value,
      allowedMonthRange: monthRangeBoundary(),
    })
    monthlyColumnMonths.value = months
    monthlyColumnRows.value = pivotRows
  } catch (e: any) {
    if (isAbortError(e)) return
    monthlyColumnMonths.value = []
    monthlyColumnRows.value = []
    ElMessage.error(e?.response?.data?.message || e?.message || '按月列视图加载失败')
  } finally {
    columnViewLoading.value = false
  }
}

function monthlyColumnRowClassName({ row }: { row: MonthlyMetricRow }) {
  if (row?.rowType === 'region') return 'monthly-region-row'
  if (row?.rowType === 'cp') return 'monthly-cp-row'
  if (row?.rowType === 'school') return 'monthly-school-row'
  if (row?.isTotal || row?.rowType === 'total') return 'monthly-total-row'
  return ''
}

function handleMonthRangeChange(_v: [string, string] | null) {
  loadOwnerUsers()
}

function handleQuery() {
  pagination.page = 1
  queryCtl.run((signal) => fetchRows(signal), { toggleIfRunning: true })
}

function handleReset() {
  filter.userId = null
  filter.granularity = 'monthly'
  viewMode.value = 'detail'
  filter.region = ''
  filter.cp = ''
  filter.schoolName = ''
  pagination.page = 1
  pagination.pageSize = 10
  setDefaultMonthRange()
  loadSchools()
  loadOwnerUsers()
  rows.value = []
  pagination.total = 0
}

function flattenMonthlyRows(rows: MonthlyMetricRow[]): MonthlyMetricRow[] {
  const flat: MonthlyMetricRow[] = []
  for (const row of rows || []) {
    flat.push(row)
    if (Array.isArray(row.children) && row.children.length) {
      flat.push(...flattenMonthlyRows(row.children))
    }
  }
  return flat
}

async function handleExport() {
  if (!validateBeforeQuery()) return
  exporting.value = true
  try {
    if (isMonthlyColumnView.value) {
      const monthlyRows = await fetchAllRowsForCurrentFilter('monthly')
      const dailyRows = await fetchAllRowsForCurrentFilter('daily')
      const enrichedMonthlyRows = await enrichRowsWithStartDates(monthlyRows)
      const { months, rows: pivotRows } = buildMonthlyAmountColumnView(enrichedMonthlyRows, dailyRows, {
        treeByRegionSchoolCp: isMonthlyTreeMode.value,
        rateUnit: singleUserRateUnit.value,
        allowedMonthRange: monthRangeBoundary(),
      })
      const exportRows = flattenMonthlyRows(pivotRows)
      const header = ['学校', '存量起算时间', '增量起算时间', `日95均值(${singleUserRateUnit.value})`, ...months]
      const rowValues = exportRows.map((row: MonthlyMetricRow) => [
        row.metric,
        row.stockStartAt || '-',
        row.incrementStartAt || '-',
        row.daily95Rate || '0.00',
        ...months.map((m) => row.values[m] || '0.00'),
      ])
      const content = buildCsvContent(header, rowValues)
      const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
      triggerBlobDownload(blob, formatExportFilename(EXPORT_FILENAME_PREFIX.singleUserMonthlyColumn, 'csv'))
    } else {
      const data = await fetchAllForExport()
      const header: string[] = ['学校名称', '地区', 'CP', serviceDateLabel.value, `日95流量值(${singleUserRateUnit.value})`, '客户金额', '线路金额', '节点金额', '渠道金额', '总归属金额']
      const rowValues = data.map((r: any) => [
        r?.school_name,
        r?.region,
        r?.cp,
        r?.service_date,
        fmtFlowRate(r?.settlement_value),
        fmtMoney(r?.customer_bill),
        fmtMoney(r?.network_line_bill),
        fmtMoney(r?.node_deduction_bill),
        fmtMoney(r?.channel_bill),
        fmtTotal(r),
      ])
      const content = buildCsvContent(header, rowValues)
      const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
      const prefix = filter.granularity === 'monthly' ? EXPORT_FILENAME_PREFIX.singleUserMonthly : EXPORT_FILENAME_PREFIX.singleUserDaily
      triggerBlobDownload(blob, formatExportFilename(prefix, 'csv'))
    }
    ElMessage.success('导出成功')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '导出失败')
  } finally {
    exporting.value = false
  }
}

function onPageChange(page: number) {
  pagination.page = page
  queryCtl.run((signal) => fetchRows(signal), { showCancelMessage: false })
}

function onSizeChange(size: number) {
  pagination.pageSize = size
  pagination.page = 1
  queryCtl.run((signal) => fetchRows(signal), { showCancelMessage: false })
}

function onRegionChange() {
  filter.schoolName = ''
  loadSchools()
  loadOwnerUsers()
}

function onCPChange() {
  filter.schoolName = ''
  loadSchools()
  loadOwnerUsers()
}

watch(() => filter.granularity, (val) => {
  if (val !== 'monthly' && viewMode.value === 'monthly_columns') {
    viewMode.value = 'detail'
  }
  if (val !== 'monthly') {
    monthlyColumnMonths.value = []
    monthlyColumnRows.value = []
  }
})

watch(viewMode, () => {
  if (isMonthlyColumnView.value && rows.value.length) {
    queryCtl.run((signal) => refreshMonthlyColumnView(signal), { showCancelMessage: false })
  }
})

onMounted(async () => {
  await trafficSettings.ensureLoaded()
  setDefaultMonthRange()
  await loadRegionCpSchool()
  await loadOwnerUsers()
})

usePageRefresh(() => {
  if (!validateBeforeQuery()) return
  pagination.page = 1
  queryCtl.run((signal) => fetchRows(signal), { showCancelMessage: false })
})
</script>

<style scoped>
.single-user-settlement-view {
  padding: 10px;
}

.filter-card {
  margin-bottom: 16px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.title {
  font-weight: 600;
}

.actions {
  display: flex;
  gap: 8px;
}

.query-form {
  row-gap: 8px;
}

.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

:deep(.monthly-total-row > td) {
  font-weight: 600;
  background: #faf3dd;
}

:deep(.monthly-region-row > td) {
  background: #eef4ff;
}

:deep(.monthly-school-row > td) {
  background: #ffffff;
}

:deep(.monthly-cp-row > td) {
  background: #f7f8fa;
}
</style>
