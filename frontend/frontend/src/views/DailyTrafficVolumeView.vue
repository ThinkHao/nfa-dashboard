<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { use } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'

import api from '@/api'
import FilterPanel from '@/components/ui/FilterPanel.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import SearchSelect from '@/components/ui/SearchSelect.vue'
import SectionCard from '@/components/ui/SectionCard.vue'
import UnifiedDateRange from '@/components/ui/UnifiedDateRange.vue'
import { useSystemTrafficSettings } from '@/composables/useSystemTrafficSettings'
import { normalizeByteUnitBase } from '@/utils/traffic-units'
import {
  defaultDailyTrafficDateRange,
  formatDailyTrafficBytes,
  meanDailyTrafficBytes,
  totalDailyTrafficBytes,
  type DailyTrafficVolumeRow,
} from './daily-traffic-volume'

use([BarChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer])

const queryForm = reactive({
  region: '',
  cp: '',
  school_name: '',
})
const dateRange = ref<[string, string] | null>(defaultDailyTrafficDateRange())
const regions = ref<string[]>([])
const cps = ref<string[]>([])
const schools = ref<Array<{ school_name: string; region: string; cp: string }>>([])
const rows = ref<DailyTrafficVolumeRow[]>([])
const loading = ref(false)
const filterLoading = ref(false)
const trafficSettings = useSystemTrafficSettings()
const unitBase = computed(() => normalizeByteUnitBase(trafficSettings.settings.value.daily_traffic_volume_unit_base, 1000))

const totalBytes = computed(() => totalDailyTrafficBytes(rows.value))
const meanBytes = computed(() => meanDailyTrafficBytes(rows.value))
const chartOption = computed(() => ({
  tooltip: {
    trigger: 'axis',
    valueFormatter: (value: number) => formatDailyTrafficBytes(Number(value), unitBase.value),
  },
  legend: {
    left: 0,
    bottom: 0,
    data: ['日服务流量'],
    formatter: () => `日服务流量  Mean: ${formatDailyTrafficBytes(meanBytes.value, unitBase.value)}  Total: ${formatDailyTrafficBytes(totalBytes.value, unitBase.value)}`,
  },
  grid: { left: 72, right: 28, top: 36, bottom: 88 },
  xAxis: {
    type: 'category',
    data: rows.value.map((row) => row.date),
    axisLabel: { rotate: rows.value.length > 12 ? 45 : 0 },
  },
  yAxis: {
    type: 'value',
    name: '日流量',
    axisLabel: { formatter: (value: number) => formatDailyTrafficBytes(value, unitBase.value) },
  },
  series: [{
    name: '日服务流量',
    type: 'bar',
    data: rows.value.map((row) => row.service_bytes),
    itemStyle: { color: '#409eff' },
    barMaxWidth: 48,
  }],
}))

async function loadRegionsAndCPs() {
  filterLoading.value = true
  try {
    const [regionData, cpData] = await Promise.all([api.v2.getRegions(), api.v2.getCPs()])
    regions.value = Array.isArray(regionData) ? regionData : []
    cps.value = Array.isArray(cpData) ? cpData : []
  } finally {
    filterLoading.value = false
  }
}

async function loadSchools() {
  filterLoading.value = true
  try {
    const data = await api.v2.getSchools({
      region: queryForm.region || undefined,
      cp: queryForm.cp || undefined,
      limit: 10000,
      offset: 0,
      sort: 'school_name',
    })
    schools.value = Array.isArray(data?.items) ? data.items : []
  } finally {
    filterLoading.value = false
  }
}

async function handleRegionChange() {
  queryForm.cp = ''
  queryForm.school_name = ''
  schools.value = []
  filterLoading.value = true
  try {
    const data = await api.v2.getSchools({
      region: queryForm.region || undefined,
      limit: 10000,
      offset: 0,
      sort: 'school_name',
    })
    const items = Array.isArray(data?.items) ? data.items : []
    cps.value = [...new Set<string>(items.map((item: any) => String(item.cp || '')).filter(Boolean))].sort()
  } finally {
    filterLoading.value = false
  }
}

async function handleCPChange() {
  queryForm.school_name = ''
  await loadSchools()
}

async function runQuery() {
  if (loading.value) return
  if (!dateRange.value) {
    ElMessage.warning('请选择日期范围')
    return
  }
  if (!queryForm.region || !queryForm.cp || !queryForm.school_name) {
    ElMessage.warning('请选择区域、CP和院校名称')
    return
  }
  loading.value = true
  try {
    const data = await api.v2.getDailyTrafficVolume({
      start_date: dateRange.value[0],
      end_date: dateRange.value[1],
      region: queryForm.region || undefined,
      cp: queryForm.cp || undefined,
      school_name: queryForm.school_name || undefined,
    })
    rows.value = Array.isArray(data) ? data : []
  } catch (error: any) {
    rows.value = []
    ElMessage.error(error?.message || '查询日流量失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadRegionsAndCPs(), trafficSettings.ensureLoaded()])
})
</script>

<template>
  <div class="page-container">
    <PageHeader title="流量监控" description="按自然日查询院校服务流量，结果已累加 V4 和 V6。" />

    <FilterPanel>
      <el-form :model="queryForm" label-width="80px" inline class="filter-form">
        <el-form-item label="区域">
          <SearchSelect
            v-model="queryForm.region"
            :options="regions"
            :loading="filterLoading"
            placeholder="选择区域"
            class="field-sm"
            @change="handleRegionChange"
          />
        </el-form-item>
        <el-form-item label="CP">
          <SearchSelect
            v-model="queryForm.cp"
            :options="cps"
            :loading="filterLoading"
            placeholder="选择 CP"
            class="field-sm"
            @change="handleCPChange"
          />
        </el-form-item>
        <el-form-item label="院校名称">
          <SearchSelect
            v-model="queryForm.school_name"
            :options="schools"
            label-key="school_name"
            value-key="school_name"
            :loading="filterLoading"
            placeholder="选择院校"
            class="field-lg"
          />
        </el-form-item>
        <el-form-item label="日期范围">
          <UnifiedDateRange v-model="dateRange" type="daterange" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" :disabled="loading" @click="runQuery">查询</el-button>
        </el-form-item>
      </el-form>
    </FilterPanel>

    <SectionCard title="院校日流量">
      <template #extra>
        <span class="total-volume">查询区间合计：{{ formatDailyTrafficBytes(totalBytes, unitBase) }}</span>
      </template>
      <VChart v-if="rows.length" class="daily-volume-chart" :option="chartOption" autoresize />
      <el-empty v-else description="请选择过滤条件并查询" />
    </SectionCard>

    <SectionCard title="日流量明细">
      <el-table v-loading="loading" :data="rows" stripe>
        <el-table-column prop="date" label="日期" width="130" />
        <el-table-column prop="region" label="区域" min-width="120" />
        <el-table-column prop="cp" label="CP" min-width="120" />
        <el-table-column prop="school_name" label="院校名称" min-width="220" />
        <el-table-column label="日服务流量" min-width="150" align="right">
          <template #default="{ row }">
            {{ formatDailyTrafficBytes(Number(row.service_bytes), unitBase) }}
          </template>
        </el-table-column>
      </el-table>
    </SectionCard>
  </div>
</template>

<style scoped>
.daily-volume-chart {
  width: 100%;
  height: 420px;
}

.total-volume {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
</style>
