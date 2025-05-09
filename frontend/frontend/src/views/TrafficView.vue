<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from 'vue'
import api from '../api'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent, GridComponent, DataZoomComponent, ToolboxComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { 
  ElCard, 
  ElForm, 
  ElFormItem, 
  ElSelect, 
  ElOption, 
  ElDatePicker, 
  ElButton, 
  ElTable, 
  ElTableColumn,
  ElPagination,
  ElMessage
} from 'element-plus'

// 注册 ECharts 组件
use([
  CanvasRenderer,
  LineChart,
  TooltipComponent,
  LegendComponent,
  GridComponent,
  DataZoomComponent,
  ToolboxComponent
])

// 数据状态
const loading = ref(false)
const chartLoading = ref(false)
const trafficData = ref([])
const regions = ref([])
const cps = ref([])
const schools = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)

// 查询表单
const queryForm = reactive({
  school_name: '',
  region: '',
  cp: '',
  start_time: '',
  end_time: '',
  timeRange: 'last1h' // 默认选择过去1小时
})

// 预设时间范围选项
const timeRangeOptions = [
  { label: '过去1小时', value: 'last1h' },
  { label: '过去3小时', value: 'last3h' },
  { label: '过去6小时', value: 'last6h' },
  { label: '过去12小时', value: 'last12h' },
  { label: '过去24小时', value: 'last24h' },
  { label: '过去2天', value: 'last2d' },
  { label: '过去7天', value: 'last7d' },
  { label: '过去30天', value: 'last30d' },
  { label: '自定义时间', value: 'custom' }
]

// 图表选项
const chartOption = computed(() => {
  // 添加调试信息
  console.log('构建图表选项，数据长度:', trafficData.value.length)
  if (trafficData.value.length > 0) {
    console.log('第一条数据:', trafficData.value[0])
  }
  
  // 检查数据是否为空
  if (trafficData.value.length === 0) {
    console.warn('没有数据可供显示')
    // 返回空图表
    return {
      title: {
        text: '流量监控',
        left: 'center'
      },
      xAxis: {
        type: 'category',
        data: []
      },
      yAxis: {
        type: 'value'
      },
      series: []
    }
  }
  
  // 按时间升序排序数据
  const sortedData = [...trafficData.value].sort((a, b) => {
    return new Date(a.create_time).getTime() - new Date(b.create_time).getTime()
  })
  
  console.log('排序后数据长度:', sortedData.length)
  
  // 提取时间点，保留完整时间信息（包括分钟）
  const times = sortedData.map(item => {
    try {
      const date = new Date(item.create_time)
      if (isNaN(date.getTime())) {
        console.error('无效的时间格式:', item.create_time)
        return 'Invalid Date'
      }
      // 保留完整时间格式，包括分钟，确保显示 5 分钟额度
      const year = date.getFullYear()
      const month = String(date.getMonth() + 1).padStart(2, '0')
      const day = String(date.getDate()).padStart(2, '0')
      const hour = String(date.getHours()).padStart(2, '0')
      const minute = String(date.getMinutes()).padStart(2, '0')
      return `${year}-${month}-${day} ${hour}:${minute}`
    } catch (error) {
      console.error('格式化时间出错:', error, item)
      return 'Error'
    }
  })
  
  console.log('时间点数组:', times)
  
  // 将原始数据转换为 bits/s
  // 服务流速（原下载流速）
  const serviceData = sortedData.map(item => {
    try {
      const bitsPerSecond = convertToBitsPerSecond(item.total_recv)
      return bitsPerSecond // 返回原始数值，不进行格式化
    } catch (error) {
      console.error('计算服务流速出错:', error, item)
      return 0
    }
  })
  
  // 回源流速（原上传流速）
  const backSourceData = sortedData.map(item => {
    try {
      const bitsPerSecond = convertToBitsPerSecond(item.total_send)
      return bitsPerSecond // 返回原始数值，不进行格式化
    } catch (error) {
      console.error('计算回源流速出错:', error, item)
      return 0
    }
  })
  
  console.log('服务流速数组:', serviceData)
  console.log('回源流速数组:', backSourceData)
  
  return {
    title: {
      text: '学校流量监控 (bits/s)',
      left: 'center'
    },
    tooltip: {
      trigger: 'axis',
      formatter: function(params) {
        let result = params[0].name + '<br/>'
        params.forEach(param => {
          result += param.seriesName + ': ' + formatBitRate(param.value) + '<br/>'
        })
        return result
      }
    },
    legend: {
      data: ['服务流速', '回源流速'],
      bottom: 0
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '10%',
      top: '10%',
      containLabel: true
    },
    toolbox: {
      feature: {
        saveAsImage: {}
      }
    },
    dataZoom: [
      {
        type: 'inside',
        start: 0,
        end: 100
      },
      {
        start: 0,
        end: 100
      }
    ],
    xAxis: {
      type: 'category',
      data: times,
      axisLabel: {
        rotate: 45,
        formatter: function(value) {
          return value
        }
      }
    },
    yAxis: {
      type: 'value',
      name: '流速 (bits/s)',
      axisLabel: {
        formatter: function(value) {
          return formatBitRate(value)
        }
      }
    },
    series: [
      {
        name: '服务流速',
        type: 'line',
        data: serviceData,
        smooth: true,
        areaStyle: {
          opacity: 0.3
        }
      },
      {
        name: '回源流速',
        type: 'line',
        data: backSourceData,
        smooth: true,
        areaStyle: {
          opacity: 0.3
        }
      }
    ]
  }
})

// 初始化数据
onMounted(async () => {
  try {
    // 设置默认时间范围为最近1小时（与 timeRange 保持一致）
    const now = new Date()
    const oneHourAgo = new Date(now.getTime() - 1 * 60 * 60 * 1000)
    queryForm.start_time = oneHourAgo.toISOString()
    queryForm.end_time = now.toISOString()
    
    // 加载下拉选项数据
    await Promise.all([
      loadRegions(),
      loadCPs()
    ])
    
    // 加载学校数据（不依赖于地区）
    await loadSchools()
    
    // 加载流量数据
    await loadTrafficData()
  } catch (error) {
    console.error('初始化数据失败:', error)
    ElMessage.error('加载数据失败，请刷新页面重试')
  }
})

// 监听分页变化
watch(currentPage, () => {
  loadTrafficData()
})

// 加载地区数据
async function loadRegions() {
  try {
    const res = await api.getRegions()
    if (res.code === 200) {
      regions.value = res.data
    }
  } catch (error) {
    console.error('加载地区数据失败:', error)
  }
}

// 加载运营商数据
async function loadCPs() {
  try {
    const res = await api.getCPs()
    if (res.code === 200) {
      cps.value = res.data
    }
  } catch (error) {
    console.error('加载运营商数据失败:', error)
  }
}

// 加载学校数据
async function loadSchools(region = '', cp = '') {
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
      
      // 处理学校数据，确保唯一性
      const uniqueSchools = {}
      schoolsList.forEach(school => {
        // 确保 school.cp 存在，如果不存在则设置为空字符串
        if (!school.cp) school.cp = ''
        
        // 使用学校名称+地区+运营商作为唯一标识
        const key = `${school.school_name}_${school.region}_${school.cp}`
        
        // 只保留每个唯一标识的第一个学校
        if (!uniqueSchools[key]) {
          uniqueSchools[key] = school
        }
      })
      
      // 转换回数组
      schools.value = Object.values(uniqueSchools)
      console.log('去重后的学校数据:', schools.value.length, '所学校')
      
      // 打印每个学校的详细信息便于调试
      schools.value.forEach((school, index) => {
        console.log(`学校${index + 1}:`, school.school_name, '运营商:', school.cp, '地区:', school.region)
      })
    } else {
      console.warn('学校数据请求失败:', res)
      schools.value = []
    }
    
    // 如果没有数据，不再使用测试数据，而是显示错误提示
    if (schools.value.length === 0) {
      console.warn('未获取到学校数据')
      ElMessage.warning('未能加载学校数据，请检查网络连接')
    }
  } catch (error) {
    console.error('加载学校数据失败:', error)
    ElMessage.error('加载学校数据失败')
    schools.value = []
  }
}



// 加载流量数据
async function loadTrafficData() {
  try {
    chartLoading.value = true
    loading.value = true
    
    // 构建查询参数
    const params = {
      school_name: queryForm.school_name,
      region: queryForm.region,
      cp: queryForm.cp,
      start_time: queryForm.start_time,
      end_time: queryForm.end_time,
      limit: 1000, // 增加限制，确保获取足够的数据点
      offset: 0 // 不使用分页，获取所有数据
    }
    
    // 打印当前时间范围参数，便于调试
    const startDate = new Date(params.start_time)
    const endDate = new Date(params.end_time)
    const diffHours = (endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60)
    console.log(`查询时间范围: ${startDate.toLocaleString()} 至 ${endDate.toLocaleString()}, 共${diffHours.toFixed(1)}小时`)
    console.log('详细查询参数:', params)
    
    // 使用真实的API调用
    const res = await api.getTrafficData(params)
    if (res.code === 200) {
      // 处理后端返回的数据，处理可能的字段名变化
      const processedData = res.data.map(item => {
        // 兼容新的time_str字段和旧的create_time字段
        if (item.time_str && !item.create_time) {
          item.create_time = item.time_str
        }
        
        // 如果没有create_time字段，尝试使用当前时间
        if (!item.create_time) {
          console.warn('数据缺少create_time字段，使用当前时间代替:', item)
          item.create_time = new Date().toISOString()
        }
        
        return item
      })
      
      // 调试信息
      console.log('原始数据:', JSON.stringify(res.data[0] || {}))
      console.log('处理后数据:', JSON.stringify(processedData[0] || {}))
      
      // 手动过滤数据，确保只显示指定时间范围内的数据
      const filteredData = processedData.filter(item => {
        if (!item.create_time) {
          console.warn('数据缺少时间字段:', item)
          return false
        }
        const itemTime = new Date(item.create_time).getTime()
        return itemTime >= startDate.getTime() && itemTime <= endDate.getTime()
      })
      
      trafficData.value = filteredData
      total.value = filteredData.length
      
      console.log(`加载流量数据成功: 原始${res.data.length}条, 处理后${processedData.length}条, 过滤后${filteredData.length}条`)
      
      // 调试信息，查看数据结构
      if (res.data.length > 0) {
        console.log('数据样例:', res.data[0])
      }
      
      // 如果数据为空，显示提示
      if (filteredData.length === 0) {
        ElMessage.warning(`所选时间范围内没有数据，请尝试其他时间范围`)
      }
    } else {
      console.warn('流量数据请求失败:', res)
      trafficData.value = []
      total.value = 0
    }
  } catch (error) {
    console.error('加载流量数据失败:', error)
    ElMessage.error('加载流量数据失败')
    trafficData.value = []
    total.value = 0
  } finally {
    chartLoading.value = false
    loading.value = false
  }
}

// 查询按钮点击事件
function handleQuery() {
  currentPage.value = 1
  loadTrafficData()
}

// 当选择省份变化时重新加载学校列表
function handleRegionChange(region) {
  queryForm.school_name = ''
  loadSchools(region, queryForm.cp)
  console.log('基于地区筛选学校:', region, queryForm.cp)
}

// 当选择运营商变化时重新加载学校列表
function handleCPChange(cp) {
  queryForm.school_name = ''
  loadSchools(queryForm.region, cp)
  console.log('基于运营商筛选学校:', queryForm.region, cp)
}

// 处理预设时间范围变化
function handleTimeRangeChange(value) {
  console.log('选择时间范围:', value)
  const now = new Date()
  let startTime
  
  // 测试时间范围选择
  ElMessage.info(`已选择时间范围: ${value}`)
  
  switch (value) {
    case 'last1h':
      startTime = new Date(now.getTime() - 1 * 60 * 60 * 1000)
      ElMessage.success('设置为过去1小时')
      break
    case 'last3h':
      startTime = new Date(now.getTime() - 3 * 60 * 60 * 1000)
      ElMessage.success('设置为过去3小时')
      break
    case 'last6h':
      startTime = new Date(now.getTime() - 6 * 60 * 60 * 1000)
      ElMessage.success('设置为过去6小时')
      break
    case 'last12h':
      startTime = new Date(now.getTime() - 12 * 60 * 60 * 1000)
      ElMessage.success('设置为过去12小时')
      break
    case 'last24h':
      startTime = new Date(now.getTime() - 24 * 60 * 60 * 1000)
      ElMessage.success('设置为过去24小时')
      break
    case 'last2d':
      startTime = new Date(now.getTime() - 2 * 24 * 60 * 60 * 1000)
      ElMessage.success('设置为过去2天')
      break
    case 'last7d':
      startTime = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
      ElMessage.success('设置为过去7天')
      break
    case 'last30d':
      startTime = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
      ElMessage.success('设置为过去30天')
      break
    case 'custom':
      // 如果是自定义时间，不自动设置时间范围
      ElMessage.info('请手动选择时间范围')
      return
    default:
      // 默认为最近1小时
      startTime = new Date(now.getTime() - 1 * 60 * 60 * 1000)
      ElMessage.success('默认设置为过去1小时')
  }
  
  // 设置时间范围
  queryForm.start_time = startTime.toISOString()
  queryForm.end_time = now.toISOString()
  
  console.log('设置时间范围:', queryForm.start_time, '至', queryForm.end_time)
  
  // 测试时间范围设置是否生效
  const startDate = new Date(queryForm.start_time)
  const endDate = new Date(queryForm.end_time)
  const diffHours = (endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60)
  ElMessage.info(`时间范围设置成功，共${diffHours.toFixed(1)}小时`)
  
  // 重置分页到第一页
  currentPage.value = 1
  
  // 自动查询
  loadTrafficData()
}

// 重置按钮点击事件
function handleReset() {
  // 重置表单
  queryForm.school_name = ''
  queryForm.region = ''
  queryForm.cp = ''
  queryForm.timeRange = 'last1h'
  
  // 设置默认时间范围为最近1小时
  const now = new Date()
  const oneHourAgo = new Date(now.getTime() - 1 * 60 * 60 * 1000)
  queryForm.start_time = oneHourAgo.toISOString()
  queryForm.end_time = now.toISOString()
  
  // 重新加载数据
  currentPage.value = 1
  loadTrafficData()
}

// 格式化流量数据
function formatTraffic(bytes, withUnit = true) {
  if (bytes === 0) return withUnit ? '0 B' : 0
  
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  if (withUnit) {
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  } else {
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2))
  }
}

// 将原始数据转换为 bits/s
function convertToBitsPerSecond(bytes) {
  // 原始数据需要 *8/60 转换为 bits/s
  // *8 是将字节转换为比特，/60 是将 5 分钟的数据转换为每秒的数据
  return (bytes * 8) / 60
}

// 格式化比特率
function formatBitRate(bitsPerSecond, withUnit = true) {
  if (bitsPerSecond === 0) return withUnit ? '0 bps' : 0
  
  const k = 1000
  const sizes = ['bps', 'Kbps', 'Mbps', 'Gbps', 'Tbps']
  const i = Math.floor(Math.log(bitsPerSecond) / Math.log(k))
  
  if (withUnit) {
    return parseFloat((bitsPerSecond / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  } else {
    return parseFloat((bitsPerSecond / Math.pow(k, i)).toFixed(2))
  }
}

// 格式化日期
function formatDate(date, interval) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  
  switch (interval) {
    case 'hour':
      return `${year}-${month}-${day} ${hour}:00`
    case 'day':
      return `${year}-${month}-${day}`
    case 'week':
      // 获取周数
      const firstDayOfYear = new Date(year, 0, 1)
      const pastDaysOfYear = (date - firstDayOfYear) / 86400000
      const weekNumber = Math.ceil((pastDaysOfYear + firstDayOfYear.getDay() + 1) / 7)
      return `${year}-W${weekNumber}`
    case 'month':
      return `${year}-${month}`
    default:
      return `${year}-${month}-${day} ${hour}:${minute}`
  }
}
</script>

<template>
  <div class="traffic-container">
    <h1 class="page-title">流量监控</h1>
    
    <!-- 查询表单 -->
    <ElCard class="query-card">
      <ElForm :model="queryForm" label-width="80px" inline>
        <ElFormItem label="地区">
          <ElSelect v-model="queryForm.region" placeholder="选择地区" clearable @change="handleRegionChange">
            <ElOption 
              v-for="region in regions" 
              :key="region" 
              :label="region" 
              :value="region" 
            />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="运营商">
          <ElSelect v-model="queryForm.cp" placeholder="选择运营商" clearable @change="handleCPChange">
            <ElOption 
              v-for="cp in cps" 
              :key="cp" 
              :label="cp" 
              :value="cp" 
            />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="学校名称">
          <ElSelect v-model="queryForm.school_name" placeholder="选择学校" clearable style="width: 300px">
            <ElOption 
              v-for="school in schools" 
              :key="school.school_id" 
              :label="school.cp ? `${school.school_name} (${school.cp})` : school.school_name" 
              :value="school.school_name" 
            />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="时间范围">
          <ElSelect v-model="queryForm.timeRange" placeholder="选择时间范围" @change="handleTimeRangeChange" style="width: 150px">
            <ElOption 
              v-for="option in timeRangeOptions" 
              :key="option.value" 
              :label="option.label" 
              :value="option.value" 
            />
          </ElSelect>
          
          <template v-if="queryForm.timeRange === 'custom'">
            <span class="date-separator"></span>
            <ElDatePicker
              v-model="queryForm.start_time"
              type="datetime"
              placeholder="开始时间"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DDTHH:mm:ss.SSSZ"
            />
            <span class="date-separator">至</span>
            <ElDatePicker
              v-model="queryForm.end_time"
              type="datetime"
              placeholder="结束时间"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DDTHH:mm:ss.SSSZ"
            />
          </template>
        </ElFormItem>
        

        
        <ElFormItem>
          <ElButton type="primary" @click="handleQuery" :loading="loading">查询</ElButton>
          <ElButton @click="handleReset">重置</ElButton>
        </ElFormItem>
      </ElForm>
    </ElCard>
    
    <!-- 流量图表 -->
    <ElCard class="chart-card" v-loading="chartLoading">
      <v-chart class="traffic-chart" :option="chartOption" autoresize />
    </ElCard>
    
    <!-- 流量数据表格 -->
    <ElCard class="data-card">
      <ElTable :data="trafficData" border stripe v-loading="loading">
        <ElTableColumn prop="create_time" label="时间" width="180">
          <template #default="scope">
            {{ new Date(scope.row.create_time).toLocaleString() }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="school_name" label="学校名称" />
        <ElTableColumn prop="region" label="地区" />
        <ElTableColumn prop="cp" label="运营商" />
        <ElTableColumn prop="total_recv" label="服务流量">
          <template #default="scope">
            {{ formatTraffic(scope.row.total_recv) }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="total_recv" label="服务流速">
          <template #default="scope">
            {{ formatBitRate(convertToBitsPerSecond(scope.row.total_recv)) }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="total_send" label="回源流量">
          <template #default="scope">
            {{ formatTraffic(scope.row.total_send) }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="total_send" label="回源流速">
          <template #default="scope">
            {{ formatBitRate(convertToBitsPerSecond(scope.row.total_send)) }}
          </template>
        </ElTableColumn>
      </ElTable>
      
      <div class="pagination-container">
        <ElPagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          layout="total, prev, pager, next, jumper"
          :total="total"
          @current-change="currentPage = $event"
        />
      </div>
    </ElCard>
  </div>
</template>

<style scoped>
.traffic-container {
  padding: 1rem 0;
}

.page-title {
  font-size: 1.8rem;
  margin-bottom: 1.5rem;
  color: var(--dark-color);
}

.query-card {
  margin-bottom: 1.5rem;
}

.chart-card {
  margin-bottom: 1.5rem;
}

.traffic-chart {
  height: 400px;
  width: 100%;
}

.data-card {
  margin-bottom: 1.5rem;
}

.pagination-container {
  margin-top: 1rem;
  display: flex;
  justify-content: flex-end;
}

.date-separator {
  margin: 0 10px;
}

:deep(.el-form-item) {
  margin-bottom: 18px;
  margin-right: 18px;
}

:deep(.el-select) {
  width: 180px !important;
}

:deep(.el-date-editor) {
  width: 180px !important;
}
</style>
