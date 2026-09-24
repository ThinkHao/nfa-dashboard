<script setup lang="ts">
import { computed, onMounted, ref, shallowRef } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TitleComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { ElCard, ElMessage, ElOption, ElPagination, ElSelect, ElTable, ElTableColumn } from 'element-plus'
import api from '../api'
import QueryActionButton from '../components/ui/QueryActionButton.vue'
import UnifiedDateRange from '../components/ui/UnifiedDateRange.vue'
import { useCancelableQuery } from '../composables/useCancelableQuery'
import { comparisonRatioPercent, sampleComparisonChartPoints } from './edc-nfa-chart-sampling'

use([CanvasRenderer, LineChart, GridComponent, LegendComponent, TitleComponent, TooltipComponent])

interface ComparisonGroup {
  id: number
  group_name: string
  nfa_src_region: string
  nfa_cp: string
  members?: Array<{ entity_id: number; edc_name: string; display_name: string }>
}

interface ComparisonPoint {
  bucket_5m: string
  edc_mbps: number
  nfa_mbps: number
  difference_mbps: number
  ratio?: number | null
  edc_member_count: number
  nfa_school_count: number
  status: string
}

const groups = ref<ComparisonGroup[]>([])
const selectedGroupID = ref<number>()
const points = shallowRef<ComparisonPoint[]>([])
const groupLoading = ref(false)
const currentPage = ref(1)
const pageSize = ref(100)
const queryCtl = useCancelableQuery()

function pad(value: number): string { return String(value).padStart(2, '0') }

function formatLocalDateTime(value: Date): string {
  return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())} ${pad(value.getHours())}:${pad(value.getMinutes())}:${pad(value.getSeconds())}`
}

const today = new Date()
const start = new Date(today.getFullYear(), today.getMonth(), today.getDate())
const end = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 23, 59, 59)
const range = ref<[string, string]>([formatLocalDateTime(start), formatLocalDateTime(end)])

const selectedGroup = computed(() => groups.value.find((group) => group.id === selectedGroupID.value))
const chartPoints = computed(() => sampleComparisonChartPoints(points.value))
const pagedPoints = computed(() => {
  const offset = (currentPage.value - 1) * pageSize.value
  return points.value.slice(offset, offset + pageSize.value)
})

const chartOption = computed(() => ({
  title: { text: selectedGroup.value ? `${selectedGroup.value.group_name} 流速比较` : '请选择比较组', left: 'center' },
  tooltip: {
    trigger: 'axis',
    formatter: (params: any) => {
      const items = Array.isArray(params) ? params : [params]
      const first = items[0]
      const time = first?.axisValueLabel || first?.axisValue || '—'
      const edc = Number(items.find((item: any) => item.seriesName === 'EDC')?.value?.[1] ?? 0)
      const nfa = Number(items.find((item: any) => item.seriesName === 'NFA')?.value?.[1] ?? 0)
      const ratio = edc !== 0 ? `${((nfa / edc) * 100).toFixed(2)}%` : '—'
      return `${time}<br/>EDC：${edc.toFixed(2)} Mbps<br/>NFA：${nfa.toFixed(2)} Mbps<br/>NFA/EDC：${ratio}`
    },
  },
  legend: { top: 32, data: ['EDC', 'NFA', '占比（NFA/EDC）'] },
  grid: { left: 60, right: 72, top: 70, bottom: 42 },
  xAxis: { type: 'time' },
  yAxis: [
    { type: 'value', name: 'Mbps', position: 'left' },
    {
      type: 'value',
      name: '占比',
      position: 'right',
      axisLabel: { formatter: (value: number) => `${value.toFixed(0)}%` },
      splitLine: { show: false },
    },
  ],
  series: [
    {
      name: 'EDC', type: 'line', showSymbol: false,
      smooth: points.value.length <= 5000,
      data: chartPoints.value.map((point) => [new Date(point.bucket_5m).getTime(), point.edc_mbps]),
    },
    {
      name: 'NFA', type: 'line', showSymbol: false,
      smooth: points.value.length <= 5000,
      data: chartPoints.value.map((point) => [new Date(point.bucket_5m).getTime(), point.nfa_mbps]),
    },
    {
      name: '占比（NFA/EDC）', type: 'line', showSymbol: false,
      yAxisIndex: 1,
      smooth: points.value.length <= 5000,
      lineStyle: { type: 'dashed' },
      data: chartPoints.value.map((point) => [
        new Date(point.bucket_5m).getTime(),
        comparisonRatioPercent(point),
      ]),
    },
  ],
}))

async function loadGroups() {
  groupLoading.value = true
  try {
    const data = await (api as any).v2.edcNfa.getComparisonGroups()
    groups.value = Array.isArray(data) ? data : (data?.items || [])
    if (!selectedGroupID.value && groups.value.length > 0) selectedGroupID.value = groups.value[0].id
  } catch (error: any) {
    ElMessage.error(error?.message || '获取 EDC/NFA 比较组失败')
  } finally {
    groupLoading.value = false
  }
}

async function loadComparison() {
  if (!selectedGroupID.value) {
    ElMessage.warning('请先选择比较组')
    return
  }
  try {
    await queryCtl.run(async (signal) => {
      const data = await (api as any).v2.edcNfa.getComparison({
        group_id: selectedGroupID.value,
        start_time: range.value?.[0],
        end_time: range.value?.[1],
      }, { signal })
      points.value = Array.isArray(data) ? data : (data?.points || data?.items || [])
      currentPage.value = 1
    }, { toggleIfRunning: true })
  } catch (error: any) {
    ElMessage.error(error?.message || '获取 EDC/NFA 比较数据失败')
  }
}

function formatRatio(value: number | null | undefined): string {
  return value == null || !Number.isFinite(value) ? '—' : `${(value * 100).toFixed(1)}%`
}

function formatNumber(value: number): string {
  return Number(value || 0).toFixed(3)
}

onMounted(async () => {
  await loadGroups()
  if (selectedGroupID.value) await loadComparison()
})
</script>

<template>
  <div class="comparison-page">
    <ElCard shadow="never">
      <div class="toolbar">
        <ElSelect v-model="selectedGroupID" class="group-select" placeholder="选择比较组" filterable :loading="groupLoading">
          <ElOption
            v-for="group in groups"
            :key="group.id"
            :label="`${group.group_name}（${group.nfa_src_region} / ${group.nfa_cp}）`"
            :value="group.id"
          />
        </ElSelect>
        <UnifiedDateRange
          v-model="range"
          type="datetimerange"
          format="YYYY-MM-DD HH:mm:ss"
          value-format="YYYY-MM-DD HH:mm:ss"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          class="time-range"
        />
        <QueryActionButton :running="queryCtl.running.value" @trigger="loadComparison" />
      </div>
      <div v-if="selectedGroup" class="mapping-hint">
        EDC 成员：{{ selectedGroup.members?.map((member) => member.edc_name || member.display_name).join('、') || '—' }}；
        NFA 节点源区域：{{ selectedGroup.nfa_src_region }} + {{ selectedGroup.nfa_cp }}（按权限过滤）
      </div>
    </ElCard>

    <ElCard shadow="never" class="chart-card">
      <VChart class="chart" :option="chartOption" autoresize />
    </ElCard>

    <ElCard shadow="never">
      <ElTable v-loading="queryCtl.running.value" :data="pagedPoints" stripe empty-text="暂无比较数据">
        <ElTableColumn prop="bucket_5m" label="时间" min-width="170" />
        <ElTableColumn label="EDC Mbps" min-width="120">
          <template #default="scope">{{ formatNumber(scope.row.edc_mbps) }}</template>
        </ElTableColumn>
        <ElTableColumn label="NFA Mbps" min-width="120">
          <template #default="scope">{{ formatNumber(scope.row.nfa_mbps) }}</template>
        </ElTableColumn>
        <ElTableColumn label="差值 Mbps" min-width="120">
          <template #default="scope">{{ formatNumber(scope.row.difference_mbps) }}</template>
        </ElTableColumn>
        <ElTableColumn label="NFA/EDC" min-width="110">
          <template #default="scope">{{ formatRatio(scope.row.ratio) }}</template>
        </ElTableColumn>
        <ElTableColumn prop="edc_member_count" label="EDC 成员数" width="110" />
        <ElTableColumn prop="nfa_school_count" label="NFA 院校数" width="110" />
        <ElTableColumn prop="status" label="状态" width="110" />
      </ElTable>
      <div class="table-footer">
        <span class="table-total">共 {{ points.length }} 条</span>
        <ElPagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[50, 100, 200]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="points.length"
          @size-change="currentPage = 1"
        />
      </div>
    </ElCard>
  </div>
</template>

<style scoped>
.comparison-page { display: flex; flex-direction: column; gap: 16px; }
.toolbar { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; }
.group-select { width: 320px; }
.time-range { width: 380px; }
.mapping-hint { margin-top: 12px; color: var(--el-text-color-secondary); font-size: 13px; }
.chart-card { min-height: 390px; }
.chart { width: 100%; height: 360px; }
.table-footer { display: flex; justify-content: space-between; align-items: center; gap: 16px; margin-top: 16px; }
.table-total { color: var(--el-text-color-secondary); font-size: 13px; }
</style>
