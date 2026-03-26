<template>
  <div class="single-user-settlement-view">
    <el-card class="filter-card" shadow="hover">
      <div class="toolbar">
        <div class="title">
          单用户结算查询
      <el-tag v-if="currentDataSourceLabel" size="small" type="info" class="ml-8">来源：{{ currentDataSourceLabel }}</el-tag>
        </div>
        <div class="actions">
          <el-button type="primary" :loading="loading" @click="handleQuery">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
          <el-button type="success" :loading="exporting" @click="handleExport">导出</el-button>
        </div>
      </div>

      <el-form :inline="true" class="query-form">
        <el-form-item label="用户" required>
          <el-select v-model="filter.userId" filterable clearable placeholder="请选择用户" class="field-w-240">
            <el-option v-for="u in userOptions" :key="u.id" :label="u.label" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="粒度">
          <el-select v-model="filter.granularity" class="field-w-120">
            <el-option label="按月" value="monthly" />
            <el-option label="按日" value="daily" />
          </el-select>
        </el-form-item>
        <el-form-item label="服务月份">
          <el-date-picker
            v-model="monthRange"
            type="monthrange"
            range-separator="至"
            start-placeholder="开始月份"
            end-placeholder="结束月份"
            format="YYYY-MM"
            value-format="YYYY-MM"
            class="field-w-260"
            @change="handleMonthRangeChange"
          />
        </el-form-item>
        <el-form-item label="地区">
          <el-select v-model="filter.region" clearable filterable class="field-w-160" @change="onRegionChange">
            <el-option v-for="r in regions" :key="r" :label="r" :value="r" />
          </el-select>
        </el-form-item>
        <el-form-item label="CP">
          <el-select v-model="filter.cp" clearable filterable class="field-w-160" @change="onCPChange">
            <el-option v-for="c in cps" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="学校">
          <el-select v-model="filter.schoolName" clearable filterable class="field-w-280">
            <el-option v-for="s in schools" :key="s.school_name" :label="s.school_name" :value="s.school_name" />
          </el-select>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="table-card" shadow="hover">
      <el-table v-loading="loading" :data="rows" border stripe class="field-w-full">
        <el-table-column prop="school_name" label="学校名称" min-width="180" />
        <el-table-column prop="region" label="地区" width="110" />
        <el-table-column prop="cp" label="CP" width="110" />
        <el-table-column prop="service_date" :label="serviceDateLabel" width="130" />
        <el-table-column label="日95流量值(Mbps)" width="150">
          <template #default="{ row }">{{ fmtFlowMbps(row.settlement_value) }}</template>
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

      <div class="pagination">
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
import { computed, onMounted, reactive, ref } from 'vue'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { School } from '@/types/api'

type Granularity = 'daily' | 'monthly'
type UserOption = { id: number; label: string }

const loading = ref(false)
const exporting = ref(false)
const rows = ref<any[]>([])

const userOptions = ref<UserOption[]>([])
const regions = ref<string[]>([])
const cps = ref<string[]>([])
const schools = ref<School[]>([])

const filter = reactive({
  userId: null as number | null,
  granularity: 'monthly' as Granularity,
  start: '',
  end: '',
  region: '',
  cp: '',
  schoolName: '',
})

const monthRange = ref<[string, string] | null>(null)
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const serviceDateLabel = computed(() => (filter.granularity === 'monthly' ? '服务月份' : '服务日期'))
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

function monthStartDate(ym: string): string {
  return `${ym}-01`
}

function monthEndDate(ym: string): string {
  const d = parseMonth(ym)
  const y = d.getFullYear()
  const m = d.getMonth()
  const last = new Date(y, m + 1, 0).getDate()
  return `${ym}-${String(last).padStart(2, '0')}`
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
  filter.start = monthStartDate(startYm)
  filter.end = monthEndDate(endYm)
}

function fmtMoney(v: any): string {
  if (v == null || v === '') return '-'
  const n = Number(v)
  if (Number.isNaN(n)) return '-'
  return n.toFixed(2)
}

function fmtFlowMbps(v: any): string {
  if (v == null || v === '') return '-'
  const n = Number(v)
  if (Number.isNaN(n)) return '-'
  const bitsPerSecond = (n * 8) / 60
  return (bitsPerSecond / 1_000_000).toFixed(2)
}

function fmtTotal(row: any): string {
  const vals = [row?.customer_bill, row?.network_line_bill, row?.node_deduction_bill, row?.channel_bill]
  const nums = vals.map((v) => (v == null ? null : Number(v))).filter((v) => v != null && !Number.isNaN(v)) as number[]
  if (!nums.length) return '-'
  return nums.reduce((a, b) => a + b, 0).toFixed(2)
}

function toCsvCell(v: any): string {
  let s = v == null ? '' : String(v)
  if (s.includes('"')) s = s.replace(/"/g, '""')
  if (/[",\n]/.test(s)) s = `"${s}"`
  return s
}

function buildParams(page = pagination.page, pageSize = pagination.pageSize) {
  const params: any = { page, page_size: pageSize, channel_owner_user_id: filter.userId }
  if (filter.start) params.start_service_date = filter.start
  if (filter.end) params.end_service_date = filter.end
  if (filter.region) params.region = filter.region
  if (filter.cp) params.cp = filter.cp
  if (filter.schoolName) params.school_name = filter.schoolName
  return params
}

async function loadOwnerUsers() {
  try {
    const params: any = {}
    if (filter.region) params.region = filter.region
    if (filter.cp) params.cp = filter.cp
    if (filter.start) params.start_service_date = filter.start
    if (filter.end) params.end_service_date = filter.end
    const items: any[] = await (api as any).settlementData.ownerSubjects(params)
    const list = (Array.isArray(items) ? items : [])
      .filter((it: any) => it && String(it.type) === 'user')
      .map((it: any) => ({ id: Number(it.id), label: String(it.label || `用户#${it.id}`) }))
      .filter((it: UserOption) => Number.isFinite(it.id))
      .sort((a, b) => a.label.localeCompare(b.label))
    userOptions.value = list
  } catch {
    userOptions.value = []
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

async function fetchRows() {
  if (!validateBeforeQuery()) return
  loading.value = true
  try {
    const params = buildParams()
    const res = filter.granularity === 'monthly'
      ? await (api as any).settlementData.monthlyList(params)
      : await (api as any).settlementData.list(params)
    if (Array.isArray(res)) {
      rows.value = res
      pagination.total = res.length
    } else {
      rows.value = Array.isArray(res?.items) ? res.items : []
      pagination.total = Number(res?.total || rows.value.length)
    }
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '查询失败')
  } finally {
    loading.value = false
  }
}

async function fetchAllForExport(): Promise<any[]> {
  const all: any[] = []
  let page = 1
  const pageSize = 1000
  while (true) {
    const params = buildParams(page, pageSize)
    const res = filter.granularity === 'monthly'
      ? await (api as any).settlementData.monthlyList(params)
      : await (api as any).settlementData.list(params)
    const items = Array.isArray(res) ? res : (Array.isArray(res?.items) ? res.items : [])
    const total = Array.isArray(res) ? items.length : Number(res?.total || 0)
    all.push(...items)
    if (items.length < pageSize) break
    if (total > 0 && page * pageSize >= total) break
    page += 1
  }
  return all
}

function downloadCsv(filename: string, content: string) {
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

function handleMonthRangeChange(v: [string, string] | null) {
  if (!v) {
    filter.start = ''
    filter.end = ''
  } else {
    filter.start = monthStartDate(v[0])
    filter.end = monthEndDate(v[1])
  }
  loadOwnerUsers()
}

function handleQuery() {
  pagination.page = 1
  fetchRows()
}

function handleReset() {
  filter.userId = null
  filter.granularity = 'monthly'
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

async function handleExport() {
  if (!validateBeforeQuery()) return
  exporting.value = true
  try {
    const data = await fetchAllForExport()
    const header = ['学校名称', '地区', 'CP', serviceDateLabel.value, '日95流量值(Mbps)', '客户金额', '线路金额', '节点金额', '渠道金额', '总归属金额']
    const lines = data.map((r: any) => [
      toCsvCell(r?.school_name),
      toCsvCell(r?.region),
      toCsvCell(r?.cp),
      toCsvCell(r?.service_date),
      toCsvCell(fmtFlowMbps(r?.settlement_value)),
      toCsvCell(fmtMoney(r?.customer_bill)),
      toCsvCell(fmtMoney(r?.network_line_bill)),
      toCsvCell(fmtMoney(r?.node_deduction_bill)),
      toCsvCell(fmtMoney(r?.channel_bill)),
      toCsvCell(fmtTotal(r)),
    ].join(','))
    const content = ['\uFEFF' + header.join(','), ...lines].join('\n')
    downloadCsv(filter.granularity === 'monthly' ? 'single_user_settlement_monthly.csv' : 'single_user_settlement_daily.csv', content)
    ElMessage.success('导出成功')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '导出失败')
  } finally {
    exporting.value = false
  }
}

function onPageChange(page: number) {
  pagination.page = page
  fetchRows()
}

function onSizeChange(size: number) {
  pagination.pageSize = size
  pagination.page = 1
  fetchRows()
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

onMounted(async () => {
  setDefaultMonthRange()
  await loadRegionCpSchool()
  await loadOwnerUsers()
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
</style>


