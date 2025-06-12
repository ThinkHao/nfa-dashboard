<template>
  <div class="settlement-daily-detail-tab">
    <!-- 筛选条件区域 -->
    <div class="filter-section">
      <el-form :model="filterForm" inline>
        <el-form-item label="地区" style="min-width: 200px;">
          <el-select v-model="filterForm.region" placeholder="选择地区" clearable style="width: 180px;" @change="handleRegionChange">
            <el-option
              v-for="region in regions"
              :key="region"
              :label="region"
              :value="region"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="运营商" style="min-width: 200px;">
          <el-select v-model="filterForm.cp" placeholder="选择运营商" clearable style="width: 180px;" @change="handleCPChange">
            <el-option
              v-for="cp in cps"
              :key="cp"
              :label="cp"
              :value="cp"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="学校" style="min-width: 300px;">
          <el-select v-model="filterForm.school_id" placeholder="选择学校" clearable style="width: 250px;" @change="handleSchoolChange">
            <el-option
              v-for="school in schools"
              :key="school.school_id"
              :label="school.school_name"
              :value="school.school_id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="日期范围" style="min-width: 400px;">
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
            style="width: 300px;"
            @change="handleDateRangeChange"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchData">查询</el-button>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 数据表格区域 -->
    <div class="table-section">
      <div class="table-header">
        <h3>日95明细列表</h3>
        <el-button type="success" @click="exportData">导出数据</el-button>
      </div>
      
      <el-table
        v-loading="loading"
        :data="dailyDetailData.items"
        border
        stripe
        style="width: 100%"
        empty-text="暂无数据"
      >
        <el-table-column prop="daily_date" label="日期" width="150">
          <template #default="scope">
            {{ formatDateDisplay(scope.row.daily_date) }}
          </template>
        </el-table-column>
        <el-table-column prop="school_name" label="学校名称" min-width="180" />
        <el-table-column prop="region" label="地区" width="120" />
        <el-table-column prop="cp" label="运营商" width="120" />
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
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import api from '../../api' // 假设 api/index.ts 中会添加新的接口
import { ElMessage } from 'element-plus'
import type { ApiResponse, School, PaginationParams } from '../../types/api'

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

// 获取基础数据 (地区、运营商、学校)
const fetchBaseData = async () => {
  try {
    const regionsResponse = await api.getRegions() as ApiResponse<string[]>
    if (regionsResponse && (regionsResponse.code === 0 || regionsResponse.code === 200) && regionsResponse.data) {
      regions.value = regionsResponse.data.filter(region => region !== "NULL")
    } else {
      regions.value = []
    }

    const cpsResponse = await api.getCPs() as ApiResponse<string[]>
    if (cpsResponse && (cpsResponse.code === 0 || cpsResponse.code === 200) && cpsResponse.data) {
      cps.value = cpsResponse.data.filter(cp => cp !== "NULL")
    } else {
      cps.value = []
    }
    
    await loadSchools()
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
    
    const response = await api.getSchools(params) as ApiResponse<{ items: School[]; total: number }>
    if (response && (response.code === 0 || response.code === 200) && response.data) {
      schools.value = response.data.items || []
    } else {
      schools.value = []
    }
  } catch (error) {
    console.error('获取学校数据失败', error)
    ElMessage.error('获取学校数据失败')
    schools.value = []
  }
}

// 处理地区选择变化
const handleRegionChange = (region: string): void => {
  loadSchools(region, filterForm.cp)
  fetchData()
}

// 处理运营商选择变化
const handleCPChange = (cp: string): void => {
  loadSchools(filterForm.region, cp)
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

    const response = await api.settlement.getDailySettlementDetails(params) as ApiResponse<DailySettlementDetailListResponse>

    if (response.data) {
      dailyDetailData.value = {
        items: response.data.items || [],
        total: response.data.total || 0
      }
      if (response.data.items.length === 0 && (filterForm.start_date && filterForm.end_date)) {
        ElMessage.warning(`没有找到 ${filterForm.start_date} 至 ${filterForm.end_date} 的日95明细数据`)
      }
    } else {
      dailyDetailData.value = { items: [], total: 0 }
      ElMessage.error('获取日95明细数据失败: ' + (response.message || '未知错误'))
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

// 导出数据 (占位)
const exportData = () => {
  // TODO: 实现导出日95明细数据的逻辑
  ElMessage.info('导出功能待实现')
  console.log('导出日95明细数据，参数:', filterForm)
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
  padding: 15px;
  background-color: #f9f9f9;
  border-radius: 4px;
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
