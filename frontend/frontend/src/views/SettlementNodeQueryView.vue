<template>
  <div class="single-node-settlement-view">
    <el-card class="filter-card" shadow="hover">
      <div class="toolbar">
        <div class="title">单节点结算查询</div>
        <div class="actions">
          <QueryActionButton :running="queryCtl.running.value" @trigger="handleQuery" />
          <el-button @click="handleReset">重置</el-button>
          <el-button type="success" :loading="exporting" @click="handleExport">导出</el-button>
        </div>
      </div>

      <el-form :inline="true" class="query-form">
        <el-form-item label="地区">
          <SearchSelect v-model="filter.region" :options="regions" clearable placeholder="选择地区" class="field-w-160" @change="onRegionChange" />
        </el-form-item>
        <el-form-item label="CP">
          <SearchSelect v-model="filter.cp" :options="cps" clearable placeholder="选择 CP" class="field-w-160" @change="onCPChange" />
        </el-form-item>
        <el-form-item label="节点">
          <SearchSelect
            v-model="filter.nodeNames"
            :options="nodes"
            label-key="display_name"
            value-key="display_name"
            multiple
            clearable
            collapse-tags
            collapse-tags-tooltip
            placeholder="选择节点（可多选）"
            class="field-w-320"
          />
        </el-form-item>
        <el-form-item label="结算方式">
          <el-select v-model="filter.settlementMode" class="field-w-140">
            <el-option label="月95" value="range_95" />
            <el-option label="日95" value="daily_95_avg" />
          </el-select>
        </el-form-item>
        <el-form-item label="查看视图">
          <el-select v-model="viewMode" class="field-w-160">
            <el-option label="明细视图" value="detail" />
            <el-option label="按月列视图" value="monthly_columns" />
          </el-select>
        </el-form-item>
        <el-form-item label="进制">
          <el-select v-model="filter.unitBase" class="field-w-140">
            <el-option label="1000" :value="1000" />
            <el-option label="1024" :value="1024" />
          </el-select>
        </el-form-item>
        <el-form-item label="服务月份">
          <UnifiedDateRange
            v-model="monthRange"
            type="monthrange"
            format="YYYY-MM"
            value-format="YYYY-MM"
            class="field-w-260"
          />
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="table-card" shadow="hover">
      <el-alert
        v-if="isMonthlyColumnView"
        class="monthly-tip"
        type="info"
        :closable="false"
        :title="`提示：${isMonthly ? '月95值' : '日95均值'}与金额使用同一进制口径，可直接对账。`"
      />

      <!-- 明细视图 -->
      <el-table v-if="!isMonthlyColumnView" v-loading="loading" :data="pagedRows" border stripe class="field-w-full" empty-text="暂无数据">
        <el-table-column prop="display_name" label="节点" min-width="180" />
        <el-table-column prop="region" label="地区" width="110" />
        <el-table-column prop="cp" label="CP" width="110" />
        <el-table-column prop="service_month" label="服务月份" width="130">
          <template #default="{ row }">{{ row.service_month || formatDate(row.settlement_time) }}</template>
        </el-table-column>
        <el-table-column label="结算模式" width="120">
          <template #default="{ row }">{{ row.settlement_mode === 'range_95' ? '月95' : '日95均值' }}</template>
        </el-table-column>
        <el-table-column prop="unit_base" label="进制" width="90" />
        <el-table-column prop="mbps_95" :label="metricColumnLabel" width="140">
          <template #default="{ row }">{{ fmtMbps95(row.mbps_95) }}</template>
        </el-table-column>
        <el-table-column prop="cp_bill" label="CP费" width="110">
          <template #default="{ row }">{{ fmtMoney(row.cp_bill) }}</template>
        </el-table-column>
        <el-table-column prop="traffic_bill" label="流量金额" width="120">
          <template #default="{ row }">{{ fmtMoney(row.traffic_bill) }}</template>
        </el-table-column>
        <el-table-column prop="rack_bill" label="机柜费" width="110">
          <template #default="{ row }">{{ fmtMoney(row.rack_bill) }}</template>
        </el-table-column>
        <el-table-column prop="other_bill" label="其他费" width="110">
          <template #default="{ row }">{{ fmtMoney(row.other_bill) }}</template>
        </el-table-column>
        <el-table-column prop="total_bill" label="总金额" width="120">
          <template #default="{ row }">{{ fmtMoney(row.total_bill) }}</template>
        </el-table-column>
      </el-table>

      <!-- 按月列视图 -->
      <el-table
        v-else
        v-loading="columnViewLoading"
        :data="monthlyColumnRows"
        :row-key="(row: any) => row.id || row.metric"
        :row-class-name="monthlyColumnRowClassName"
        border
        class="field-w-full"
      >
        <el-table-column prop="metric" label="节点" min-width="220" fixed="left" />
        <template v-for="month in monthlyColumnMonths" :key="month">
          <el-table-column :label="`${month} ${metricColumnLabel}`" min-width="150">
            <template #default="{ row }">{{ row.monthlyDaily95Values?.[month] || '-' }}</template>
          </el-table-column>
          <el-table-column :label="`${month} 金额`" min-width="120">
            <template #default="{ row }">{{ row.monthlyAmountValues?.[month] || '-' }}</template>
          </el-table-column>
        </template>
      </el-table>

      <div v-if="!isMonthlyColumnView" class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="rows.length"
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
import { ElMessage } from 'element-plus'
import { buildCsvContent, formatExportFilename, triggerBlobDownload } from '@/utils/export'
import { EXPORT_FILENAME_PREFIX } from '@/utils/export-standards'
import SearchSelect from '@/components/ui/SearchSelect.vue'
import UnifiedDateRange from '@/components/ui/UnifiedDateRange.vue'
import QueryActionButton from '@/components/ui/QueryActionButton.vue'
import { useCancelableQuery, isAbortError } from '@/composables/useCancelableQuery'
import { usePageRefresh } from '@/composables/usePageRefresh'
import { sanitizeScopeOptionValues } from '@/utils/scope-options'
import { aggregateDailyNodeRowsByMonth, buildNodeMonthlyColumnView, enumerateMonthsInRange, type NodeMonthlyMetricRow } from './settlement-node-query-utils'

type SettlementMode = 'daily_95_avg' | 'range_95'
type ViewMode = 'detail' | 'monthly_columns'

const loading = ref(false)
const exporting = ref(false)
const rows = ref<any[]>([])
const queryCtl = useCancelableQuery()

const columnViewLoading = ref(false)
const viewMode = ref<ViewMode>('detail')
const monthlyColumnMonths = ref<string[]>([])
const monthlyColumnRows = ref<NodeMonthlyMetricRow[]>([])

const regions = ref<string[]>([])
const cps = ref<string[]>([])
const nodes = ref<any[]>([])

const filter = reactive({
  region: '',
  cp: '',
  nodeNames: [] as string[],
  settlementMode: 'range_95' as SettlementMode,
  unitBase: 1000 as 1000 | 1024,
})

const monthRange = ref<[string, string] | null>(null)
const pagination = reactive({ page: 1, pageSize: 10 })

const isMonthly = computed(() => filter.settlementMode === 'range_95')
const isMonthlyColumnView = computed(() => viewMode.value === 'monthly_columns')
const metricColumnLabel = computed(() => (isMonthly.value ? '月95值(Mbps)' : '日95均值(Mbps)'))

const pagedRows = computed(() => {
  const start = (pagination.page - 1) * pagination.pageSize
  return rows.value.slice(start, start + pagination.pageSize)
})

// ── 下拉数据加载 ──

async function loadRegionCpOptions() {
  try {
    const [regionResp, cpResp] = await Promise.all([
      (api as any).v2.edc.getRegions(),
      (api as any).v2.edc.getCPs(),
    ])
    regions.value = sanitizeScopeOptionValues(Array.isArray(regionResp) ? regionResp : [])
    cps.value = sanitizeScopeOptionValues(Array.isArray(cpResp) ? cpResp : [])
  } catch {
    regions.value = []
    cps.value = []
  }
}

async function loadNodes(region = '', cp = '') {
  try {
    const params: any = {}
    if (region) params.region = region
    if (cp) params.cp = cp
    const resp = await (api as any).v2.edc.getEntities(params)
    nodes.value = Array.isArray(resp) ? resp : Array.isArray(resp?.items) ? resp.items : []
  } catch {
    nodes.value = []
  }
}

function onRegionChange() {
  filter.cp = ''
  filter.nodeNames = []
  loadNodes(filter.region)
  loadRegionCpOptions()
}

function onCPChange() {
  filter.nodeNames = []
  loadNodes(filter.region, filter.cp)
}

// ── 月份工具 ──

function parseMonth(ym: string): Date {
  const [y, m] = ym.split('-').map(Number)
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

function resolveMonthRange(): { startDate: string; endDate: string; serviceMonth: string } {
  if (!monthRange.value || !monthRange.value[0] || !monthRange.value[1]) {
    return { startDate: '', endDate: '', serviceMonth: '' }
  }
  const [startYm, endYm] = monthRange.value
  const startDate = `${startYm}-01 00:00:00`
  const endDate = new Date(parseMonth(endYm).getFullYear(), parseMonth(endYm).getMonth() + 1, 0, 23, 59, 59)
  const endDateStr = `${endDate.getFullYear()}-${String(endDate.getMonth() + 1).padStart(2, '0')}-${String(endDate.getDate()).padStart(2, '0')} 23:59:59`
  return { startDate, endDate: endDateStr, serviceMonth: endYm }
}

function monthRangeBoundary(): { startMonth: string; endMonth: string } | null {
  if (!monthRange.value || !monthRange.value[0] || !monthRange.value[1]) return null
  return { startMonth: String(monthRange.value[0]), endMonth: String(monthRange.value[1]) }
}

function selectedMonths(): string[] {
  const boundary = monthRangeBoundary()
  if (!boundary) return []
  return enumerateMonthsInRange(boundary.startMonth, boundary.endMonth)
}

// ── 格式化 ──

function formatDate(v: string) {
  if (!v) return '-'
  return String(v).split('T')[0].split(' ')[0]
}

function fmtMbps95(v: unknown) {
  if (v == null || v === '') return '-'
  const n = Number(v)
  return Number.isFinite(n) ? n.toFixed(2) : '-'
}

function fmtMoney(v: unknown) {
  if (v == null || v === '') return '-'
  const n = Number(v)
  return Number.isFinite(n) ? n.toFixed(2) : '-'
}

// ── 校验 ──

function validateBeforeQuery(): boolean {
  if (!filter.nodeNames.length) {
    ElMessage.warning('请先选择节点')
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

// ── 查询 ──

async function fetchNodeData(settlementMode: SettlementMode, signal?: AbortSignal): Promise<any[]> {
  const { startDate, endDate, serviceMonth } = resolveMonthRange()
  const allItems: any[] = []

  for (const nodeName of filter.nodeNames) {
    if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
    const params: any = {
      display_name: nodeName,
      unit_base: filter.unitBase,
      page: 1,
      page_size: 10000,
    }
    if (filter.region) params.region = filter.region
    if (filter.cp) params.cp = filter.cp

    if (settlementMode === 'range_95') {
      const months = selectedMonths()
      const queryMonths = months.length ? months : (serviceMonth ? [serviceMonth] : [])
      for (const month of queryMonths) {
        if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
        const res: any = await api.settlementData.nodeMonthlyList({ ...params, service_month: month, settlement_mode: settlementMode }, { signal })
        const items = Array.isArray(res) ? res : Array.isArray(res?.items) ? res.items : []
        allItems.push(...items)
      }
    } else {
      if (startDate) params.start_date = startDate
      if (endDate) params.end_date = endDate
      params.settlement_mode = settlementMode
      const res: any = await api.settlementData.nodeList(params, { signal })
      const items = Array.isArray(res) ? res : Array.isArray(res?.items) ? res.items : []
      allItems.push(...items)
    }
  }

  allItems.sort((a, b) => {
    const da = String(a?.settlement_time || a?.service_month || '')
    const db = String(b?.settlement_time || b?.service_month || '')
    return db.localeCompare(da)
  })

  return allItems
}

async function fetchDisplayRows(settlementMode: SettlementMode, signal?: AbortSignal): Promise<any[]> {
  const items = await fetchNodeData(settlementMode, signal)
  return settlementMode === 'daily_95_avg' ? aggregateDailyNodeRowsByMonth(items) : items
}

async function fetchRows(signal?: AbortSignal) {
  if (!validateBeforeQuery()) return
  loading.value = true
  try {
    rows.value = await fetchDisplayRows(filter.settlementMode, signal)
    pagination.page = 1
    await refreshMonthlyColumnView(signal)
  } catch (e: any) {
    if (isAbortError(e)) return
    ElMessage.error(e?.response?.data?.message || e?.message || '查询失败')
  } finally {
    loading.value = false
  }
}

async function refreshMonthlyColumnView(signal?: AbortSignal) {
  if (!isMonthlyColumnView.value) {
    monthlyColumnMonths.value = []
    monthlyColumnRows.value = []
    return
  }
  columnViewLoading.value = true
  try {
    const displayRows = await fetchDisplayRows(filter.settlementMode, signal)
    const { months, rows: pivotRows } = buildNodeMonthlyColumnView(displayRows, [], {
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

function monthlyColumnRowClassName({ row }: { row: NodeMonthlyMetricRow }) {
  if (row?.isTotal || row?.rowType === 'total') return 'monthly-total-row'
  return ''
}

function handleQuery() {
  pagination.page = 1
  queryCtl.run((signal) => fetchRows(signal), { toggleIfRunning: true })
}

function handleReset() {
  filter.region = ''
  filter.cp = ''
  filter.nodeNames = []
  filter.settlementMode = 'range_95'
  filter.unitBase = 1000
  viewMode.value = 'detail'
  pagination.page = 1
  pagination.pageSize = 10
  setDefaultMonthRange()
  loadNodes()
  rows.value = []
  monthlyColumnMonths.value = []
  monthlyColumnRows.value = []
}

// ── 分页 ──

function onPageChange(page: number) {
  pagination.page = page
}

function onSizeChange(size: number) {
  pagination.pageSize = size
  pagination.page = 1
}

// ── 导出 ──

function flattenMonthlyRows(rows: NodeMonthlyMetricRow[]): NodeMonthlyMetricRow[] {
  const flat: NodeMonthlyMetricRow[] = []
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
      const displayRows = await fetchDisplayRows(filter.settlementMode)
      const { months, rows: pivotRows } = buildNodeMonthlyColumnView(displayRows, [], {
        allowedMonthRange: monthRangeBoundary(),
      })
      const exportRows = flattenMonthlyRows(pivotRows)
      const monthHeaders = months.flatMap((m) => [`${m} ${metricColumnLabel.value}`, `${m} 金额`])
      const header = ['节点', ...monthHeaders]
      const rowValues = exportRows.map((row: NodeMonthlyMetricRow) => [
        row.metric,
        ...months.flatMap((m) => [
          row.monthlyDaily95Values?.[m] || '0.00',
          row.monthlyAmountValues?.[m] || '0.00',
        ]),
      ])
      const content = buildCsvContent(header, rowValues)
      const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
      triggerBlobDownload(blob, formatExportFilename(EXPORT_FILENAME_PREFIX.singleNodeMonthlyColumn, 'csv'))
    } else {
      const header = ['节点', '地区', 'CP', '服务月份', '结算模式', '进制', metricColumnLabel.value, 'CP费', '流量金额', '机柜费', '其他费', '总金额']
      const rowValues = rows.value.map((r: any) => [
        r?.display_name ?? '',
        r?.region ?? '',
        r?.cp ?? '',
        r?.service_month ?? formatDate(r?.settlement_time),
        r?.settlement_mode === 'range_95' ? '月95' : '日95均值',
        r?.unit_base ?? '',
        fmtMbps95(r?.mbps_95),
        fmtMoney(r?.cp_bill),
        fmtMoney(r?.traffic_bill),
        fmtMoney(r?.rack_bill),
        fmtMoney(r?.other_bill),
        fmtMoney(r?.total_bill),
      ])
      const content = buildCsvContent(header, rowValues)
      const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
      const prefix = isMonthly.value ? EXPORT_FILENAME_PREFIX.singleNodeMonthly : EXPORT_FILENAME_PREFIX.singleNodeDaily
      triggerBlobDownload(blob, formatExportFilename(prefix, 'csv'))
    }
    ElMessage.success('导出成功')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '导出失败')
  } finally {
    exporting.value = false
  }
}

// ── 结算方式/视图切换 ──

watch(() => filter.settlementMode, () => {
  monthlyColumnMonths.value = []
  monthlyColumnRows.value = []
})

watch(viewMode, () => {
  if (isMonthlyColumnView.value && filter.nodeNames.length) {
    queryCtl.run((signal) => refreshMonthlyColumnView(signal), { showCancelMessage: false })
  }
})

// ── 初始化 ──

onMounted(async () => {
  setDefaultMonthRange()
  await Promise.all([loadRegionCpOptions(), loadNodes()])
})

usePageRefresh(() => {
  if (!validateBeforeQuery()) return
  pagination.page = 1
  queryCtl.run((signal) => fetchRows(signal), { showCancelMessage: false })
})
</script>

<style scoped>
.single-node-settlement-view {
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

.monthly-tip {
  margin-bottom: 10px;
}

:deep(.monthly-total-row > td) {
  font-weight: 600;
  background: #faf3dd;
}
</style>
