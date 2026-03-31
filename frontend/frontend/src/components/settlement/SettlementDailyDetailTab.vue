<template>
  <div class="settlement-daily-detail-tab">
    <!-- 筛选条件区域 -->
    <el-card class="filter-section">
      <el-form :model="filterForm" inline>
        <el-form-item label="地区" class="min-w-200">
          <el-select v-model="filterForm.region" placeholder="选择地区" clearable class="field-w-180" @change="handleRegionChange">
            <el-option
              v-for="region in regions"
              :key="region"
              :label="region"
              :value="region"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="CP" class="min-w-200">
          <el-select v-model="filterForm.cp" placeholder="选择 CP" clearable class="field-w-180" @change="handleCPChange">
            <el-option
              v-for="cp in cps"
              :key="cp"
              :label="cp"
              :value="cp"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="学校" class="min-w-300">
          <el-select v-model="filterForm.school_id" placeholder="选择学校" clearable class="field-w-250" @change="handleSchoolChange">
            <el-option
              v-for="school in schools"
              :key="school.school_id"
              :label="school.school_name"
              :value="school.school_id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="日期范围" class="min-w-400">
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
            class="field-w-300"
            @change="handleDateRangeChange"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchData">查询</el-button>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 数据表格区域 -->
    <el-card class="table-section">
      <template #header>
        <div class="table-header">
          <h3 class="card-title">日95明细列表</h3>
          <el-button type="success" @click="exportData">导出数据</el-button>
        </div>
      </template>
      
      <el-table
        v-loading="loading"
        :data="dailyDetailData.items"
        border
        stripe
        class="field-w-full"
        empty-text="暂无数据"
      >
        <el-table-column prop="daily_date" label="日期" width="150">
          <template #default="scope">
            {{ formatDateDisplay(scope.row.daily_date) }}
          </template>
        </el-table-column>
        <el-table-column prop="school_name" label="学校名称" min-width="180" />
        <el-table-column prop="region" label="地区" width="120" />
        <el-table-column prop="cp" label="CP" width="120" />
        <el-table-column label="95值(Mbps)" width="150">
          <template #default="scope">
            {{ scope.row.daily_95_value ? formatBitRate(convertToBitsPerSecond(scope.row.daily_95_value), false) : '0.00' }}
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="dailyDetailData.total"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import api from '../../api' // 假设 api/index.ts 中会添加新的接口
import { ElMessage } from 'element-plus'
import type { School } from '../../types/api'
import { useTasksStore } from '@/stores/tasks'
import { buildCsvContent, formatExportFilename, triggerBlobDownload } from '@/utils/export'
import { EXPORT_FILENAME_PREFIX, EXPORT_HEADERS } from '@/utils/export-standards'

// 定义日95明细数据项接口
interface DailySettlementDetail {
  id: string; // 或其他唯一标识符
  daily_date: string;
  school_id: string;
  school_name: string;
  region: string;
  cp: string;
  daily_95_value: number; // 假设这是原始值，需要转换
}

// 定义日95明细列表响应接口
interface DailySettlementDetailListResponse {
  items: DailySettlementDetail[];
  total: number;
}

// 学校、地区和运营商数据
const schools = ref<School[]>([])
const regions = ref<string[]>([])
const cps = ref<string[]>([])

// 筛选表单
interface FilterForm {
  school_id: string;
  region: string;
  cp: string;
  start_date: string;
  end_date: string;
  page: number;
  page_size: number;
}

const filterForm = reactive<FilterForm>({
  school_id: '',
  region: '',
  cp: '',
  start_date: '',
  end_date: '',
  page: 1,
  page_size: 10
})

// 日期范围选择器
const dateRange = ref<[string, string] | null>(null)

// 分页相关
const currentPage = ref(1)
const pageSize = ref(10)

// 加载状态
const loading = ref(false)

// 日95明细数据
const dailyDetailData = ref<DailySettlementDetailListResponse>({
  items: [],
  total: 0
})

// 将原始数据转换为 bits/s
const convertToBitsPerSecond = (bytes: number | null | undefined): number => {
  if (bytes === null || bytes === undefined) {
    return 0
  }
  const factor = 60 // 假设原始数据单位与 SettlementDataTab 一致
  return (bytes * 8) / factor
}

// 格式化比特率
const formatBitRate = (bitsPerSecond: number | null | undefined, withUnit = true): string => {
  if (bitsPerSecond === null || bitsPerSecond === undefined) {
    return withUnit ? '0.00 Mbps' : '0.00'
  }
  const mbps = bitsPerSecond / 1000000
  return withUnit ? `${mbps.toFixed(2)} Mbps` : mbps.toFixed(2)
}

// 格式化日期显示
const formatDateDisplay = (dateStr: string): string => {
  if (!dateStr) return '-';
  if (dateStr.includes(' ')) {
    return dateStr.split(' ')[0]
  }
  if (dateStr.includes('T')) {
    return dateStr.split('T')[0]
  }
  return dateStr
}

// 基于 schools 动态派生地区/运营商选项，仅限可见院校范围
const computeRegionCpOptions = () => {
  try {
    const rset = new Set<string>()
    const cset = new Set<string>()
    ;(schools.value || []).forEach((s: any) => {
      if (s && typeof s.region === 'string' && s.region && s.region !== 'NULL') rset.add(s.region)
      if (s && typeof s.cp === 'string' && s.cp && s.cp !== 'NULL') cset.add(s.cp)
    })
    regions.value = Array.from(rset).sort()
    cps.value = Array.from(cset).sort()
  } catch (e) {
    console.warn('派生地区/运营商选项失败:', e)
    regions.value = []
    cps.value = []
  }
}

// 获取基础数据 (学校 -> 地区/运营商)
const fetchBaseData = async () => {
  try {
    await loadSchools()
    computeRegionCpOptions()
  } catch (error) {
    console.error('获取基础数据失败', error)
    ElMessage.error('获取基础数据失败')
  }
}

// 加载学校数据
const loadSchools = async (region: string = '', cp: string = ''): Promise<void> => {
  try {
    schools.value = []
    const params: { region?: string; cp?: string; limit?: number; offset?: number } = {}
    if (region) params.region = region
    if (cp) params.cp = cp
    params.limit = 1000 
    params.offset = 0
    
    const response = await (api as any).v2.getSchools(params) as any
    if (Array.isArray(response)) {
      schools.value = response
    } else if (response && Array.isArray(response.items)) {
      schools.value = response.items
    } else {
      schools.value = []
    }
    computeRegionCpOptions()
  } catch (error) {
    console.error('获取学校数据失败', error)
    ElMessage.error('获取学校数据失败')
    schools.value = []
  }
}

// 处理地区选择变化
const handleRegionChange = (region: string): void => {
  loadSchools(region, filterForm.cp).then(() => computeRegionCpOptions())
  fetchData()
}

// 处理运营商选择变化
const handleCPChange = (cp: string): void => {
  loadSchools(filterForm.region, cp).then(() => computeRegionCpOptions())
  fetchData()
}

// 处理学校选择变化
const handleSchoolChange = (): void => {
  fetchData()
}

// 处理日期范围变化
const handleDateRangeChange = (val: [string, string] | null) => {
  if (val) {
    filterForm.start_date = val[0]
    filterForm.end_date = val[1]
  } else {
    filterForm.start_date = ''
    filterForm.end_date = ''
  }
  setTimeout(() => {
    fetchData()
  }, 0)
}
const fetchData = async () => {
  loading.value = true
  try {
    const params = {
      school_id: filterForm.school_id,
      region: filterForm.region,
      cp: filterForm.cp,
      start_date: filterForm.start_date,
      end_date: filterForm.end_date,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value
    }

    console.log('Fetching daily settlement details with params:', params)

    const response = await (api as any).v2.settlement.getDailySettlementDetails(params) as any
    let items: any[] = []
    let total = 0
    if (Array.isArray(response)) {
      items = response
      total = response.length
    } else if (response && Array.isArray(response.items)) {
      items = response.items
      total = typeof response.total === 'number' ? response.total : response.items.length
    }
    dailyDetailData.value = { items, total }
    if (dailyDetailData.value.items.length === 0 && (filterForm.start_date && filterForm.end_date)) {
      ElMessage.warning(`没有找到 ${filterForm.start_date} 至 ${filterForm.end_date} 的日95明细数据`)
    }
  } catch (error) {
    console.error('获取日95明细数据失败:', error)
    ElMessage.error('获取日95明细数据失败')
    dailyDetailData.value = { items: [], total: 0 } // 清空数据
  } finally {
    loading.value = false
  }
}

// 重置筛选条件
const resetFilter = () => {
  filterForm.school_id = ''
  filterForm.region = ''
  filterForm.cp = ''
  filterForm.start_date = ''
  filterForm.end_date = ''
  dateRange.value = null
  currentPage.value = 1 // 重置时回到第一页
  // pageSize.value 不重置，保持用户选择
  loadSchools() // 重置后重新加载所有学校
  fetchData()
}

// 处理页码变化
const handleCurrentChange = (page: number) => {
  currentPage.value = page
  fetchData()
}

// 处理每页条数变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1 // 修改每页条数时回到第一页
  fetchData()
}

async function fetchAllDailyDetailsForExport(onProgress?: (p: number | null, meta?: { processed: number; total?: number | null }) => void): Promise<any[]> {
  const baseParams: any = {
    school_id: filterForm.school_id || undefined,
    region: filterForm.region || undefined,
    cp: filterForm.cp || undefined,
    start_date: filterForm.start_date || undefined,
    end_date: filterForm.end_date || undefined,
  }
  let total: number | null = null
  try {
    const probe: any = await (api as any).v2.settlement.getDailySettlementDetails({ ...baseParams, limit: 1, offset: 0 })
    if (typeof probe?.total === 'number') total = probe.total
    else if (Array.isArray(probe?.items)) total = Number(probe.items.length)
  } catch {}
  const limit = 1000
  let offset = 0
  const all: any[] = []
  while (true) {
    const res: any = await (api as any).v2.settlement.getDailySettlementDetails({ ...baseParams, limit, offset })
    let items: any[] = []
    if (Array.isArray(res)) {
      items = res
      if (total == null) total = res.length
    } else if (res && Array.isArray(res.items)) {
      items = res.items
      if (typeof res.total === 'number') total = Number(res.total)
    }
    all.push(...items)
    const processed = all.length
    if (onProgress) {
      if (typeof total === 'number' && total > 0) onProgress(Math.max(0, Math.min(1, processed / total)), { processed, total })
      else onProgress(null, { processed, total: null })
    }
    if (items.length < limit) break
    if (typeof total === 'number' && processed >= total) break
    offset += limit
  }
  return all
}

const exportData = async () => {
  let taskId: string | null = null
  try {
    const tasks = useTasksStore()
    taskId = `export-daily-95:${Date.now()}`
    tasks.start({ id: taskId, type: 'export', title: '日95明细导出', status: 'running', progress: 0 })
    const data = await fetchAllDailyDetailsForExport((p, meta) => {
      tasks.update(taskId, { progress: p == null ? undefined : p, status: 'running', processed: meta?.processed ?? null, total: (meta?.total as any) ?? null })
    })
    const header = [...EXPORT_HEADERS.daily95Detail]
    const rows: Array<Array<unknown>> = []
    for (const r of data) {
      const date = formatDateDisplay(String(r?.daily_date || ''))
      const mbps = (convertToBitsPerSecond(Number(r?.daily_95_value ?? 0)) / 1_000_000).toFixed(2)
      rows.push([date, r?.school_name ?? '', r?.region ?? '', r?.cp ?? '', mbps])
    }
    const content = buildCsvContent(header, rows)
    const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    tasks.complete(taskId, url)
    triggerBlobDownload(blob, formatExportFilename(EXPORT_FILENAME_PREFIX.daily95Detail, 'csv'))
    ElMessage.success('导出成功')
  } catch (e: any) {
    try { const tasks = useTasksStore(); if (taskId) tasks.fail(taskId, e?.message) } catch {}
    ElMessage.error(e?.response?.data?.message || e?.message || '导出失败')
  }
}

// 组件挂载时获取数据
onMounted(() => {
  fetchBaseData()
  fetchData()
})

</script>

<style scoped>
.settlement-daily-detail-tab {
  padding: 10px;
}

.filter-section {
  margin-bottom: 20px;
}

.table-section {
  margin-top: 20px;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>


