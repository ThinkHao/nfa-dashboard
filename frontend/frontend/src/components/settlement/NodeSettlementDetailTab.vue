<template>
  <div class="node-settlement-detail-tab">
    <el-card class="filter-section">
      <el-form :model="filterForm" inline>
        <el-form-item label="地区">
          <el-input v-model="filterForm.region" clearable class="field-w-160" />
        </el-form-item>
        <el-form-item label="CP">
          <el-input v-model="filterForm.cp" clearable class="field-w-160" />
        </el-form-item>
        <el-form-item label="节点">
          <el-input v-model="filterForm.display_name" clearable class="field-w-200" />
        </el-form-item>
        <el-form-item v-if="kind === 'monthly'" label="服务月份">
          <el-date-picker v-model="filterForm.service_month" type="month" value-format="YYYY-MM" format="YYYY-MM" />
        </el-form-item>
        <el-form-item v-else label="日期范围">
          <UnifiedDateRange v-model="dateRange" type="daterange" format="YYYY-MM-DD" value-format="YYYY-MM-DD HH:mm:ss" @change="handleDateRangeChange" />
        </el-form-item>
        <el-form-item>
          <QueryActionButton :running="queryCtl.running.value" @trigger="onSearch" />
          <el-button @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="table-section">
      <template #header>
        <div class="table-header">
          <h3 class="card-title">{{ kind === 'monthly' ? '节点月95明细列表' : '节点日95明细列表' }}</h3>
        </div>
      </template>
      <el-table v-loading="loading" :data="items" border stripe class="field-w-full" empty-text="暂无数据">
        <el-table-column prop="display_name" label="节点" min-width="180" />
        <el-table-column prop="region" label="地区" width="110" />
        <el-table-column prop="cp" label="CP" width="110" />
        <el-table-column :prop="kind === 'monthly' ? 'service_month' : 'settlement_time'" :label="kind === 'monthly' ? '服务月份' : '日期'" width="130">
          <template #default="{ row }">{{ kind === 'monthly' ? row.service_month : formatDate(row.settlement_time) }}</template>
        </el-table-column>
        <el-table-column label="结算模式" width="120">
          <template #default="{ row }">{{ row.settlement_mode === 'range_95' ? '月95' : '日95均值' }}</template>
        </el-table-column>
        <el-table-column prop="unit_base" label="进制" width="90" />
        <el-table-column prop="mbps_95" label="95值(Mbps)" width="130">
          <template #default="{ row }">{{ formatMbps95(row.mbps_95) }}</template>
        </el-table-column>
        <el-table-column prop="cp_bill" label="CP费" width="110">
          <template #default="{ row }">{{ formatMoney(row.cp_bill) }}</template>
        </el-table-column>
        <el-table-column prop="traffic_bill" label="流量金额" width="120">
          <template #default="{ row }">{{ formatMoney(row.traffic_bill) }}</template>
        </el-table-column>
        <el-table-column prop="rack_bill" label="机柜费" width="110">
          <template #default="{ row }">{{ formatMoney(row.rack_bill) }}</template>
        </el-table-column>
        <el-table-column prop="other_bill" label="其他费" width="110">
          <template #default="{ row }">{{ formatMoney(row.other_bill) }}</template>
        </el-table-column>
        <el-table-column prop="total_bill" label="总金额" width="120">
          <template #default="{ row }">{{ formatMoney(row.total_bill) }}</template>
        </el-table-column>
      </el-table>
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10,20,50,100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          @size-change="onPageSizeChange"
          @current-change="onPageChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import api from '@/api'
import QueryActionButton from '@/components/ui/QueryActionButton.vue'
import UnifiedDateRange from '@/components/ui/UnifiedDateRange.vue'
import { useCancelableQuery, isAbortError } from '@/composables/useCancelableQuery'
import { splitSettlementDayRange } from './settlement-day-range'
import { ElMessage } from 'element-plus'
import { formatMbps95 } from './settlement-display-utils'

const props = defineProps<{ kind: 'daily' | 'monthly' }>()
const kind = props.kind
const loading = ref(false)
const items = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const dateRange = ref<[string, string] | null>(null)
const queryCtl = useCancelableQuery()
const filterForm = reactive({ region: '', cp: '', display_name: '', service_month: '', start_date: '', end_date: '' })

function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (filterForm.region) p.region = filterForm.region
  if (filterForm.cp) p.cp = filterForm.cp
  if (filterForm.display_name) p.display_name = filterForm.display_name
  if (kind === 'monthly' && filterForm.service_month) p.service_month = filterForm.service_month
  if (kind === 'daily') {
    if (filterForm.start_date) p.start_date = filterForm.start_date
    if (filterForm.end_date) p.end_date = filterForm.end_date
  }
  return p
}
async function fetch(signal?: AbortSignal) {
  loading.value = true
  try {
    const res: any = kind === 'monthly'
      ? await api.settlementData.nodeMonthlyList(buildParams(), { signal })
      : await api.settlementData.nodeList(buildParams(), { signal })
    items.value = Array.isArray(res?.items) ? res.items : []
    total.value = Number(res?.total) || items.value.length
  } catch (e: any) {
    if (!isAbortError(e)) ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}
function onSearch() { page.value = 1; queryCtl.run((signal) => fetch(signal), { toggleIfRunning: true }) }
function onReset() { Object.assign(filterForm, { region: '', cp: '', display_name: '', service_month: '', start_date: '', end_date: '' }); dateRange.value = null; onSearch() }
function onPageChange(nextPage: number) {
  page.value = nextPage
  queryCtl.run((signal) => fetch(signal), { showCancelMessage: false })
}
function onPageSizeChange(nextSize: number) {
  pageSize.value = nextSize
  page.value = 1
  queryCtl.run((signal) => fetch(signal), { showCancelMessage: false })
}
function handleDateRangeChange(val: [string, string] | null) {
  const { start, end } = splitSettlementDayRange(val)
  filterForm.start_date = start
  filterForm.end_date = end
  onSearch()
}
function formatDate(v: string) {
  if (!v) return '-'
  return String(v).split('T')[0].split(' ')[0]
}
function formatMoney(v: unknown) {
  if (v === null || v === undefined || v === '') return '-'
  const n = Number(v)
  return Number.isFinite(n) ? n.toFixed(2) : '-'
}
onMounted(() => queryCtl.run((signal) => fetch(signal), { showCancelMessage: false }))
</script>
