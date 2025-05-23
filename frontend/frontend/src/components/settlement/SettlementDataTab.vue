<template>
  <div class="settlement-data-tab">
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
        <h3>结算数据列表</h3>
        <el-button type="success" @click="exportData">导出数据</el-button>
      </div>
      
      <el-table
        v-loading="loading"
        :data="settlementData.items"
        border
        stripe
        style="width: 100%"
        empty-text="暂无数据"
      >
        <!-- 调试信息 -->
        <template #empty>
          <div>
            <p>暂无数据</p>
            <p v-if="settlementData.items">数据项数量: {{ settlementData.items.length }}</p>
            <p v-else>数据项为空</p>
          </div>
        </template>
        <el-table-column prop="school_name" label="学校名称" min-width="180" />
        <el-table-column prop="region" label="地区" width="120" />
        <el-table-column prop="cp" label="运营商" width="120" />
        <el-table-column label="95值(Mbps)" width="150">
          <template #default="scope">
            {{ scope.row.settlement_value ? formatBitRate(convertToBitsPerSecond(scope.row.settlement_value), false) : '0.00' }}
          </template>
        </el-table-column>
        <el-table-column label="时间范围" width="200">
          <template #default="scope">
            <span v-if="scope.row.settlement_date && typeof scope.row.settlement_date === 'string' && (!dateRange || dateRange[0] === dateRange[1])">
              {{ formatDateDisplay(scope.row.settlement_date) }}
            </span>
            <span v-else-if="scope.row.settlement_date && typeof scope.row.settlement_date === 'string' && dateRange && dateRange[0] !== dateRange[1]">
              {{ dateRange[0] }} 至 {{ dateRange[1] }}
            </span>
            <span v-else-if="dateRange && dateRange[0] === dateRange[1]">
              {{ dateRange[0] }}
            </span>
            <span v-else-if="dateRange">
              {{ dateRange[0] }} 至 {{ dateRange[1] }}
            </span>
            <span v-else>-</span>
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
          :total="settlementData.total"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import api from '../../api'
import { ElMessage } from 'element-plus'
import type { SettlementListResponse, Settlement } from '../../types/settlement'
import type { ApiResponse, School, PaginationParams } from '../../types/api'

// 学校、地区和运营商数据
interface School {
  school_id: string;
  school_name: string;
  region?: string;
  cp?: string;
}

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
  limit?: number;
  offset?: number;
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

// 结算数据
const settlementData = ref<SettlementListResponse>({
  items: [],
  total: 0
})

// 将原始数据转换为 bits/s
const convertToBitsPerSecond = (bytes: number | null | undefined): number => {
  // 原始数据需要 *8/60 转换为 bits/s
  // *8 是将字节转换为比特
  // /60 是将每分钟的数据转换为每秒的数据
  if (bytes === null || bytes === undefined) {
    return 0
  }
  
  const factor = 60
  
  // 将字节转换为比特，然后除以时间因子
  return (bytes * 8) / factor
}

// 格式化比特率
const formatBitRate = (bitsPerSecond: number | null | undefined, withUnit = true): string => {
  if (bitsPerSecond === null || bitsPerSecond === undefined) {
    return withUnit ? '0.00 Mbps' : '0.00'
  }
  
  // 转换为 Mbps
  const mbps = bitsPerSecond / 1000000
  
  return withUnit ? `${mbps.toFixed(2)} Mbps` : mbps.toFixed(2)
}

// 格式化日期显示
const formatDateDisplay = (dateStr: string): string => {
  // 如果包含时间部分，只返回日期部分
  if (dateStr.includes(' ')) {
    return dateStr.split(' ')[0]
  }
  
  // 如果包含时区信息，去除时区信息
  if (dateStr.includes('T')) {
    // 处理ISO格式日期
    const parts = dateStr.split('T')
    return parts[0]
  }
  
  // 如果是纯日期格式，直接返回
  return dateStr
}

// 获取基础数据
const fetchBaseData = async () => {
  try {
    // 获取地区列表
    const regionsResponse = await api.getRegions() as ApiResponse<string[]>
    console.log('地区列表原始响应:', regionsResponse)
    if (regionsResponse && (regionsResponse.code === 0 || regionsResponse.code === 200) && regionsResponse.data) {
      // 过滤掉 "NULL" 值
      regions.value = regionsResponse.data.filter(region => region !== "NULL")
      console.log('地区列表设置为:', regions.value)
    } else {
      console.error('地区列表数据为空')
      regions.value = []
    }

    // 获取运营商列表
    const cpsResponse = await api.getCPs() as ApiResponse<string[]>
    console.log('运营商列表原始响应:', cpsResponse)
    if (cpsResponse && (cpsResponse.code === 0 || cpsResponse.code === 200) && cpsResponse.data) {
      // 过滤掉 "NULL" 值
      cps.value = cpsResponse.data.filter(cp => cp !== "NULL")
      console.log('运营商列表设置为:', cps.value)
    } else {
      console.error('运营商列表数据为空')
      cps.value = []
    }
    
    // 加载学校列表（不带过滤条件）
    await loadSchools()
  } catch (error) {
    console.error('获取基础数据失败', error)
    ElMessage.error('获取基础数据失败')
  }
}

// 加载学校数据
const loadSchools = async (region: string = '', cp: string = ''): Promise<void> => {
  try {
    // 清空学校列表，避免显示旧数据
    schools.value = []
    
    // 构建请求参数
    const params: { region?: string; cp?: string; limit?: number; offset?: number } = {}
    
    // 添加可选参数
    if (region) {
      params.region = region
    }
    
    if (cp) {
      params.cp = cp
    }
    
    // 分页参数
    params.limit = 1000 // 获取足够多的学校数据
    params.offset = 0
    
    const response = await api.getSchools(params) as ApiResponse<{ items: School[]; total: number }>
    console.log('学校列表原始响应:', response)
    
    // 检查响应状态
    if (response && (response.code === 0 || response.code === 200) && response.data) {
      schools.value = response.data.items || []
      console.log('学校列表设置为:', schools.value)
      return response.data.total || 0
    } else {
      console.error('学校列表数据为空')
      schools.value = []
      return 0
    }
  } catch (error) {
    console.error('获取学校数据失败', error)
    ElMessage.error('获取学校数据失败')
    schools.value = []
    return 0
  }
}

// 处理地区选择变化
const handleRegionChange = (region: string): void => {
  console.log('地区选择变化:', region)
  // 当地区变化时，重新加载学校列表
  if (region) {
    loadSchools(region, filterForm.cp)
  } else {
    loadSchools('', filterForm.cp)
  }
  // 当地区变化时自动刷新数据
  fetchData()
}

// 处理运营商选择变化
const handleCPChange = (cp: string): void => {
  console.log('运营商选择变化:', cp)
  // 当运营商变化时，重新加载学校列表
  if (cp) {
    loadSchools(filterForm.region, cp)
  } else {
    loadSchools(filterForm.region, '')
  }
  // 当运营商变化时自动刷新数据
  fetchData()
}

// 处理学校选择变化
const handleSchoolChange = (schoolId: string): void => {
  console.log('学校选择变化:', schoolId)
  // 当学校变化时，可以在这里添加额外的逻辑
  // 例如，根据学校ID获取更多详细信息等
  // 当学校变化时自动刷新数据
  fetchData()
}

// 处理日期范围变化
const handleDateRangeChange = (val: [string, string] | null) => {
  if (val) {
    filterForm.start_date = val[0]
    filterForm.end_date = val[1]
    console.log('设置日期范围:', val[0], '至', val[1])
  } else {
    filterForm.start_date = ''
    filterForm.end_date = ''
    console.log('清除日期范围')
  }
  
  // 日期范围变化时自动触发数据查询
  // 使用setTimeout确保日期范围已经更新
  setTimeout(() => {
    console.log('日期范围变化，自动触发数据查询')
    fetchData()
  }, 0)
}

// 获取结算数据
const fetchData = async () => {
  loading.value = true
  
  try {
    // 计算分页参数
    const params: PaginationParams & { 
      school_id?: string;
      region?: string;
      cp?: string;
      start_date?: string;
      end_date?: string;
    } = {
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value,
      start_date: filterForm.start_date,
      end_date: filterForm.end_date
    }
    
    console.log('分页参数:', { 页码: currentPage.value, 每页条数: pageSize.value, offset: (currentPage.value - 1) * pageSize.value })
    
    // 添加可选参数
    if (filterForm.school_id) {
      params.school_id = filterForm.school_id
    }
    
    if (filterForm.region) {
      params.region = filterForm.region
    }
    
    if (filterForm.cp) {
      params.cp = filterForm.cp
    }
    
    console.log('最终请求参数:', params)
    
    // 发送请求
    const response = await api.settlement.getSettlements(params) as ApiResponse<SettlementListResponse>
    console.log('结算数据响应:', response)
    
    // 检查响应状态
    if (response && (response.code === 0 || response.code === 200)) {
      console.log('原始响应数据:', JSON.stringify(response.data))
      
      // 直接使用后端返回的数据结构
      if (response.data && typeof response.data === 'object') {
        // 如果数据是数组，将其包装为预期的结构
        if (Array.isArray(response.data)) {
          settlementData.value = { items: response.data, total: response.data.length }
          console.log('将数组转换为预期结构:', settlementData.value)
        } 
        // 如果数据已经有items属性，直接使用
        else if (response.data.items && Array.isArray(response.data.items)) {
          settlementData.value = response.data
          console.log('使用现有items结构:', settlementData.value)
        } 
        // 如果数据是单个对象，将其包装为数组
        else {
          settlementData.value = { items: [response.data], total: 1 }
          console.log('将单个对象转换为预期结构:', settlementData.value)
        }
      } else {
        settlementData.value = { items: [], total: 0 }
        console.log('数据为空或不是对象:', response.data)
      }
      
      // 检查数据结构
      if (settlementData.value.items && Array.isArray(settlementData.value.items)) {
        console.log('结算数据项目数量:', settlementData.value.items.length)
        if (settlementData.value.items.length > 0) {
          console.log('第一个数据项:', JSON.stringify(settlementData.value.items[0]))
        }
      } else {
        console.error('数据结构不符合预期:', settlementData.value)
        // 尝试修复数据结构
        if (typeof settlementData.value === 'object' && !settlementData.value.items) {
          settlementData.value = { items: [settlementData.value], total: 1 }
          console.log('尝试修复后的数据:', settlementData.value)
        }
      }
    } else {
      console.error('获取结算数据失败:', response)
      ElMessage.error('获取结算数据失败: ' + (response?.message || '未知错误'))
      settlementData.value = { items: [], total: 0 }
    }
    
    // 如果没有数据，显示提示
    if (!settlementData.value.items || settlementData.value.items.length === 0) {
      console.log('没有找到结算数据')
      ElMessage.warning(`没有找到${filterForm.start_date}至${filterForm.end_date}的结算数据`)
      
      // 尝试直接使用fetch获取数据
      try {
        const queryParams = new URLSearchParams()
        queryParams.append('start_date', filterForm.start_date)
        queryParams.append('end_date', filterForm.end_date)
        queryParams.append('limit', String(pageSize.value))
        queryParams.append('offset', String((currentPage.value - 1) * pageSize.value))
        
        if (filterForm.school_name) queryParams.append('school_name', filterForm.school_name)
        if (filterForm.region) queryParams.append('region', filterForm.region)
        if (filterForm.cp) queryParams.append('cp', filterForm.cp)
        
        const fullUrl = `http://localhost:8081/api/v1/settlement/data?${queryParams.toString()}`
        console.log('直接请求URL:', fullUrl)
        
        const fetchResponse = await fetch(fullUrl)
        const fetchData = await fetchResponse.json()
        console.log('直接请求响应:', fetchData)
        
        if (fetchData && fetchData.code === 200 && fetchData.data) {
          if (Array.isArray(fetchData.data.items)) {
            settlementData.value.items = fetchData.data.items
            settlementData.value.total = fetchData.data.total || 0
            console.log('直接请求获取数据成功:', settlementData.value.items.length, '条记录')
          }
        }
      } catch (fetchError) {
        console.error('直接请求获取数据失败:', fetchError)
      }
    }
    
    // 多日结算值不需要额外处理，后端已经返回了最大值
    if (dateRange.value && dateRange.value[0] !== dateRange.value[1]) {
      // 计算日期范围天数仅用于日志记录
      const startDate = new Date(dateRange.value[0])
      const endDate = new Date(dateRange.value[1])
      const daysDiff = Math.ceil((endDate - startDate) / (1000 * 60 * 60 * 24)) + 1
      console.log('日期范围天数:', daysDiff)
      
      // 如果没有数据，显示提示
      if (settlementData.value.items.length === 0) {
        console.log('没有找到结算数据')
        ElMessage.warning('没有找到符合条件的结算数据')
      }
    }
  } catch (error) {
    console.error('获取结算数据失败', error)
    ElMessage.error('获取结算数据失败')
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
  currentPage.value = 1
  pageSize.value = 10
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
  currentPage.value = 1
  fetchData()
}

// 导出数据
const exportData = () => {
  ElMessage.info('导出功能待实现')
  // 这里可以调用导出API或者使用前端导出库如xlsx.js
}

// 组件挂载时获取数据
onMounted(() => {
  fetchBaseData()
  fetchData()
})
</script>

<style scoped>
.settlement-data-tab {
  padding: 10px;
}

.filter-section {
  margin-bottom: 20px;
  padding: 15px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.table-section {
  background-color: #fff;
  padding: 15px;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.table-header h3 {
  margin: 0;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
