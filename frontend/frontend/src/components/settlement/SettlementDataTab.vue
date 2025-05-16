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
      >
        <el-table-column prop="school_name" label="学校名称" min-width="180" />
        <el-table-column prop="region" label="地区" width="120" />
        <el-table-column prop="cp" label="运营商" width="120" />
        <el-table-column label="95值(Mbps)" width="150">
          <template #default="scope">
            {{ scope.row.settlement_value ? formatBitRate(scope.row.settlement_value, false) : '0.00' }}
          </template>
        </el-table-column>
        <el-table-column v-if="dateRange && dateRange[0] !== dateRange[1]" label="时间范围" width="200">
          <template #default="scope">
            {{ dateRange ? `${dateRange[0]} 至 ${dateRange[1]}` : '-' }}
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

// 学校、地区和运营商数据
const schools = ref<any[]>([])
const regions = ref<string[]>([])
const cps = ref<string[]>([])

// 筛选表单
const filterForm = reactive({
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
const convertToBitsPerSecond = (bytes) => {
  // 原始数据需要 *8/60 转换为 bits/s
  // *8 是将字节转换为比特
  // /60 是将每分钟的数据转换为每秒的数据
  const factor = 60
  
  // 将字节转换为比特，然后除以时间因子
  return (bytes * 8) / factor
}

// 格式化比特率
const formatBitRate = (bitsPerSecond, withUnit = true) => {
  if (bitsPerSecond === null || bitsPerSecond === undefined) {
    return '0.00 Mbps'
  }
  
  // 转换为 Mbps
  const mbps = bitsPerSecond / 1000000
  
  return withUnit ? `${mbps.toFixed(2)} Mbps` : mbps.toFixed(2)
}

// 获取基础数据
const fetchBaseData = async () => {
  try {
    // 获取地区列表
    const regionsResponse = await api.getRegions()
    console.log('地区列表原始响应:', regionsResponse)
    if (regionsResponse && regionsResponse.data) {
      regions.value = regionsResponse.data
      console.log('地区列表设置为:', regions.value)
    } else {
      console.error('地区列表数据为空')
      regions.value = []
    }

    // 获取运营商列表
    const cpsResponse = await api.getCPs()
    console.log('运营商列表原始响应:', cpsResponse)
    if (cpsResponse && cpsResponse.data) {
      cps.value = cpsResponse.data
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
const loadSchools = async (region = '', cp = '') => {
  try {
    // 清空学校列表，避免显示旧数据
    schools.value = []
    
    // 构建请求参数
    const params = { limit: 100 }
    if (region) {
      params.region = region
    }
    if (cp) {
      params.cp = cp
    }
    
    console.log('请求学校数据参数:', params)
    const res = await api.getSchools(params)
    console.log('学校数据原始响应:', res)
    
    let schoolsList = []
    
    if (res.code === 200 && res.data) {
      // 正确的数据结构：data.items
      if (Array.isArray(res.data.items)) {
        schoolsList = res.data.items
        console.log('加载学校数据成功:', schoolsList.length, '所学校')
      } 
      // 兼容旧的数据结构：data.schools
      else if (Array.isArray(res.data.schools)) {
        schoolsList = res.data.schools
        console.log('加载学校数据成功(旧结构):', schoolsList.length, '所学校')
      }
      // 如果数据本身就是数组
      else if (Array.isArray(res.data)) {
        schoolsList = res.data
        console.log('加载学校数据成功(直接数组):', schoolsList.length, '所学校')
      }
      // 如果没有有效数据
      else {
        console.warn('未找到有效的学校数据结构')
        schoolsList = []
      }
      
      schools.value = schoolsList
    } else {
      console.error('获取学校列表失败:', res)
      
      // 尝试直接获取学校列表
      try {
        const queryParams = new URLSearchParams()
        if (region) queryParams.append('region', region)
        if (cp) queryParams.append('cp', cp)
        queryParams.append('limit', '100')
        
        const response = await fetch(`http://localhost:8081/api/v1/schools?${queryParams.toString()}`)
        const data = await response.json()
        console.log('直接获取学校列表响应:', data)
        
        if (data && data.code === 200 && data.data && Array.isArray(data.data.items)) {
          schools.value = data.data.items
          console.log('直接获取学校列表成功:', schools.value.length, '所学校')
        }
      } catch (fetchError) {
        console.error('直接获取学校列表失败:', fetchError)
      }
    }
  } catch (error) {
    console.error('加载学校数据失败:', error)
    ElMessage.error('加载学校数据失败')
  }
}

// 当选择省份变化时重新加载学校列表
const handleRegionChange = (region) => {
  filterForm.school_id = ''
  loadSchools(region, filterForm.cp)
  console.log('基于地区筛选学校:', region, filterForm.cp)
  // 当地区变化时自动刷新数据
  fetchData()
}

// 当选择运营商变化时重新加载学校列表
const handleCPChange = (cp) => {
  filterForm.school_id = ''
  loadSchools(filterForm.region, cp)
  console.log('基于运营商筛选学校:', filterForm.region, cp)
  // 当运营商变化时自动刷新数据
  fetchData()
}

// 处理学校选择变化
const handleSchoolChange = (schoolId) => {
  console.log('选择学校变化:', schoolId)
  // 当学校变化时自动刷新数据
  fetchData()
}

// 处理日期范围变化
const handleDateRangeChange = (val) => {
  if (val) {
    filterForm.start_date = val[0]
    filterForm.end_date = val[1]
    console.log('设置日期范围:', val[0], '至', val[1])
  } else {
    filterForm.start_date = ''
    filterForm.end_date = ''
    console.log('清除日期范围')
  }
}

// 获取结算数据
const fetchData = async () => {
  loading.value = true
  
  // 处理日期范围
  if (dateRange.value) {
    filterForm.start_date = dateRange.value[0]
    filterForm.end_date = dateRange.value[1]
  } else {
    // 如果没有选择日期，默认使用当前日期
    const today = new Date()
    const year = today.getFullYear()
    const month = String(today.getMonth() + 1).padStart(2, '0')
    const day = String(today.getDate()).padStart(2, '0')
    const formattedDate = `${year}-${month}-${day}`
    
    filterForm.start_date = formattedDate
    filterForm.end_date = formattedDate
    console.log('未选择日期，使用默认日期:', formattedDate)
  }

  // 设置分页参数
  filterForm.limit = pageSize.value
  filterForm.offset = (currentPage.value - 1) * pageSize.value

  console.log('发送查询参数:', filterForm)

  try {
    // 构建请求参数
    const params = {
      start_date: filterForm.start_date,
      end_date: filterForm.end_date,
      limit: filterForm.limit,
      offset: filterForm.offset
    }
    
    // 添加可选过滤条件
    if (filterForm.school_id) params.school_id = filterForm.school_id
    if (filterForm.region) params.region = filterForm.region
    if (filterForm.cp) params.cp = filterForm.cp
    
    console.log('学校ID筛选条件:', filterForm.school_id)
    
    console.log('最终请求参数:', params)
    
    // 尝试使用API获取数据
    const response = await api.settlement.getSettlements(params)
    console.log('结算数据原始响应:', response)
    
    // 处理数据
    let items = []
    let total = 0
    
    if (response && response.code === 200) {
      if (response.data) {
        // 正确的数据结构：data.items
        if (Array.isArray(response.data.items)) {
          items = response.data.items
          total = response.data.total || 0
          console.log('使用新结构数据:', items.length, '条记录')
        }
        // 兼容旧的数据结构：data直接是数组
        else if (Array.isArray(response.data)) {
          items = response.data
          total = items.length
          console.log('使用直接数组数据:', items.length, '条记录')
        }
      }
    }
    
    // 如果没有数据，显示提示
    if (items.length === 0) {
      console.log('没有找到结算数据')
      ElMessage.warning(`没有找到${filterForm.start_date}至${filterForm.end_date}的结算数据`)
      
      // 尝试直接使用fetch获取数据
      try {
        const queryParams = new URLSearchParams()
        queryParams.append('start_date', filterForm.start_date)
        queryParams.append('end_date', filterForm.end_date)
        queryParams.append('limit', String(filterForm.limit))
        queryParams.append('offset', String(filterForm.offset))
        
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
            items = fetchData.data.items
            total = fetchData.data.total || 0
            console.log('直接请求获取数据成功:', items.length, '条记录')
          }
        }
      } catch (fetchError) {
        console.error('直接请求获取数据失败:', fetchError)
      }
    }
    
    // 如果是多天数据，需要计算平均值
    if (dateRange.value && dateRange.value[0] !== dateRange.value[1]) {
      // 计算日期范围天数
      const startDate = new Date(dateRange.value[0])
      const endDate = new Date(dateRange.value[1])
      const daysDiff = Math.ceil((endDate - startDate) / (1000 * 60 * 60 * 24)) + 1
      console.log('日期范围天数:', daysDiff)
      
      // 对每个学校的数据进行处理
      items.forEach(item => {
        // 如果是多天，则计算平均值
        if (daysDiff > 1 && item.settlement_value) {
          console.log('处理前的值:', item.school_name, item.settlement_value)
          item.settlement_value = Math.round(item.settlement_value / daysDiff)
          console.log('处理后的值:', item.school_name, item.settlement_value)
        }
      })
    }
    
    // 设置数据和总数
    settlementData.value = {
      items: items,
      total: total
    }
    console.log('最终数据:', settlementData.value)
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
