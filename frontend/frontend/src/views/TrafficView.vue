<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
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

function formatLabel(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const h = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${y}-${m}-${day} ${h}:${mi}`
}

// 从任意 create_time 值生成“到分钟”的稳定字符串键（不做时区换算），示例："2025-11-05 15:45"
function toMinuteKeyStr(val: any): string {
  if (typeof val === 'string') {
    const s = val.trim()
    // ISO: 2025-11-05T15:45[:ss][Z]
    let m = s.match(/^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2})/)
    if (m) return `${m[1]} ${m[2]}`
    // 紧凑: 20251105 1545 或 2025/11/05 15:45
    m = s.match(/^(\d{4})[-\/]?(\d{2})[-\/]?(\d{2}).*?(\d{2}):(\d{2})/)
    if (m) return `${m[1]}-${m[2]}-${m[3]} ${m[4]}:${m[5]}`
    return s.slice(0, 16)
  }
  if (typeof val === 'number') {
    const d = new Date(val < 1e12 ? val * 1000 : val)
    return formatLabel(d)
  }
  if (val && typeof val === 'object' && 'toString' in val) {
    return toMinuteKeyStr(String(val))
  }
  return ''
}

// 时间工具：格式化为 RFC3339（去毫秒，UTC Z），避免时区误差
function toRFC3339Seconds(d: Date): string {
  return d.toISOString().replace(/\.\d{3}Z$/, 'Z')
}

// 将任意时间归一化到其所在的5分钟桶，并返回标准ISO字符串（到分钟，秒固定为00）
function toFiveMinuteKeyISO(d: Date): string {
  const ms = d.getTime()
  const minutes = Math.floor(ms / (60 * 1000))
  const bucketMinutes = minutes - (minutes % 5)
  const bucketMs = bucketMinutes * 60 * 1000
  const dd = new Date(bucketMs)
  // 使用 UTC ISO，去毫秒，保留到分钟
  const iso = dd.toISOString().replace(/\.\d{3}Z$/, 'Z')
  // 规范到“YYYY-MM-DDTHH:mm:00Z”
  return iso.replace(/:\d{2}Z$/, ':00Z')
}

// 安全解析任意时间值为 Date（支持 RFC3339、"YYYY-MM-DD HH:mm:ss"、毫秒时间戳等）
function parseTime(val: any): Date {
  if (val instanceof Date) return val
  if (typeof val === 'number') {
    // 兼容秒/毫秒时间戳：小于 1e12 视为秒
    return new Date(val < 1e12 ? val * 1000 : val)
  }
  if (typeof val === 'string') {
    const s = val.trim()
    // 1) 直接解析
    let dt = new Date(s)
    if (!isNaN(dt.getTime())) return dt
    // 2) 抽取 "YYYY[-/]MM[-/]DD HH:mm[:ss]" 数字并作为本地时间构造
    const m = s.match(/(\d{4})[-\/]?(\d{2})[-\/]?(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?/)
    if (m) {
      const year = Number(m[1])
      const month = Number(m[2]) - 1
      const day = Number(m[3])
      const hour = Number(m[4])
      const minute = Number(m[5])
      const second = m[6] ? Number(m[6]) : 0
      dt = new Date(year, month, day, hour, minute, second)
      if (!isNaN(dt.getTime())) return dt
    }
    // 3) 转 RFC3339（无时区时补 Z）
    const t = s.replace(' ', 'T')
    dt = /Z|[+-]\d{2}:?\d{2}$/.test(t) ? new Date(t) : new Date(`${t}Z`)
    if (!isNaN(dt.getTime())) return dt
  }
  // 兜底：当前时间，避免 NaN 破坏聚合（并输出日志）
  try { console.warn('无法解析时间，使用当前时间兜底:', val) } catch {}
  return new Date()
}

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

// 路由
const route = useRoute()

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
const currentGranularity = ref('5m') // 当前使用的时间粒度

// 查询表单
const queryForm = reactive({
  school_name: '',
  region: '',
  cp: '',
  start_time: '',
  end_time: '',
  timeRange: 'last1h' // 默认选择过去1小时
})

// 是否已选择任一筛选条件（地区/内容方/学校）
const hasFilter = computed(() => {
  return !!(queryForm.school_name || queryForm.region || queryForm.cp)
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
  
  // 无筛选时，不显示图表数据
  if (!hasFilter.value) {
    return {
      title: { text: '流量监控', left: 'center', subtext: '请选择任一筛选条件后再查询' },
      xAxis: { type: 'time' },
      yAxis: { type: 'value', name: '流速 (bits/s)' },
      series: []
    }
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
      xAxis: { type: 'time' },
      yAxis: {
        type: 'value'
      },
      series: []
    }
  }
  
  // 按时间升序排序数据（优先使用我们生成的 minute key 字符串，避免 Date 解析差异）
  const sortedData = [...trafficData.value].sort((a, b) => {
    const ak = String((a as any).time_str || (a as any).create_time || '')
    const bk = String((b as any).time_str || (b as any).create_time || '')
    if (ak === bk) return 0
    return ak < bk ? -1 : 1
  })
  
  console.log('排序后数据长度:', sortedData.length)
  
  // 提取时间点标签：使用我们生成的 minute key 字符串
  const times = sortedData.map(item => {
    const key = String((item as any).time_str || (item as any).create_time || '')
    return key.replace('T', ' ').replace('Z', '').slice(0, 16)
  })
  
  console.log('时间点数组:', times)
  
  // 将原始数据转换为 bits/s，并与时间戳配对，供 time 轴渲染
  const points = sortedData.map(item => {
    const ms = typeof item.create_time === 'number' ? item.create_time : parseTime(item.create_time).getTime()
    return {
      t: ms,
      recv: convertToBitsPerSecond(item.total_recv),
      send: convertToBitsPerSecond(item.total_send),
    }
  }).filter(p => !isNaN(p.t))

  const serviceData = points.map(p => [p.t, p.recv])
  const backSourceData = points.map(p => [p.t, p.send])
  
  console.log('服务流速数组:', serviceData)
  console.log('回源流速数组:', backSourceData)
  
  // 格式化粒度显示
  const formatGranularity = (gran) => {
    switch(gran) {
      case '5m': return '原始数据 (5分钟粒度)'
      case '15m': return '15分钟粒度'
      case 'hour': return '小时粒度'
      case 'day': return '天粒度'
      default: return gran
    }
  }
  
  return {
    title: {
      text: `学校流量监控 (bits/s) - ${formatGranularity(currentGranularity.value)}`,
      left: 'center'
    },
    tooltip: {
      trigger: 'axis',
      formatter: function(params) {
        let result = ''
        if (params && params.length) {
          const first = params[0]
          // time 轴下，axisPointer 的 name 可能是格式化后的时间，也可能为空
          const ts = Array.isArray(first.value) ? first.value[0] : undefined
          const label = ts ? new Date(ts).toLocaleString() : (first.name || '')
          result += label + '<br/>'
        }
        params.forEach(param => {
          const v = Array.isArray(param.value) ? param.value[1] : param.value
          result += param.seriesName + ': ' + formatBitRate(Number(v || 0)) + '<br/>'
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
      type: 'time',
      axisLabel: { rotate: 45 }
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
        showSymbol: false,
        areaStyle: { opacity: 0.1 }
      },
      {
        name: '回源流速',
        type: 'line',
        data: backSourceData,
        smooth: true,
        showSymbol: false,
        areaStyle: { opacity: 0.1 }
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
    queryForm.start_time = toRFC3339Seconds(oneHourAgo)
    queryForm.end_time = toRFC3339Seconds(now)
    
    // 读取路由查询参数作为默认过滤
    const q: any = route.query || {}
    if (typeof q.school_name === 'string' && q.school_name) {
      queryForm.school_name = q.school_name
    }
    if (typeof q.region === 'string' && q.region) {
      queryForm.region = q.region
    }
    if (typeof q.cp === 'string' && q.cp) {
      queryForm.cp = q.cp
    }
    
    // 先加载地区/运营商（v2，按用户可见范围）
    await loadRegionCpOptions()
    // 再加载学校数据（基于 v2，仅返回当前用户可见范围）
    await loadSchools()
    // 若未能从接口获得地区/运营商，则基于学校数据兜底派生
    if ((!regions.value || regions.value.length === 0) || (!cps.value || cps.value.length === 0)) {
      computeRegionCpOptions()
    }
    
    // 默认不加载流量数据，待用户选择筛选条件后点击查询
  } catch (error) {
    console.error('初始化数据失败:', error)
    ElMessage.error('加载数据失败，请刷新页面重试')
  }
})

// 监听分页变化
watch(currentPage, () => {
  loadTrafficData()
})

// 监听路由查询变化（在已处于 /traffic 页面时再次从外部带参跳转也能生效）
watch(
  () => route.query,
  (q: any) => {
    try {
      if (q && typeof q === 'object') {
        queryForm.school_name = typeof q.school_name === 'string' ? q.school_name : ''
        queryForm.region = typeof q.region === 'string' ? q.region : ''
        queryForm.cp = typeof q.cp === 'string' ? q.cp : ''
        loadTrafficData()
      }
    } catch {}
  }
)

// 基于当前 schools 列表动态派生地区/运营商选项（仅限可见院校）
function computeRegionCpOptions() {
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

// 通过 v2 接口加载地区/运营商选项（按用户可见范围过滤）
async function loadRegionCpOptions() {
  try {
    const r = await (api as any).v2.getRegions()
    regions.value = Array.isArray(r) ? r.filter((v: any) => v && v !== 'NULL').sort() : []
  } catch {
    regions.value = []
  }
  try {
    const c = await (api as any).v2.getCPs()
    cps.value = Array.isArray(c) ? c.filter((v: any) => v && v !== 'NULL').sort() : []
  } catch {
    cps.value = []
  }
}

// 加载学校数据
async function loadSchools(region = '', cp = '') {
  try {
    // 清空学校列表，避免显示旧数据
    schools.value = []
    
    // 构建请求参数
    const params: Record<string, any> = { limit: 500 } // 增加限制以获取更多学校
    if (region) {
      params.region = region
    }
    if (cp) {
      params.cp = cp
    }
    
    console.log('请求学校数据参数:', params)
    const res = await (api as any).v2.getSchools(params) as any
    console.log('学校数据原始响应:', res)
    
    let schoolsList: any[] = []
    if (Array.isArray(res)) {
      schoolsList = res
      console.log('加载学校数据成功(直接数组):', schoolsList.length, '所学校')
    } else if (res && Array.isArray(res.items)) {
      schoolsList = res.items
      console.log('加载学校数据成功(items):', schoolsList.length, '所学校')
    } else {
      console.warn('未找到有效的学校数据结构')
      schoolsList = []
    }

    // 处理学校数据，确保唯一性
    const uniqueSchools: Record<string, any> = {}
    schoolsList.forEach((school: any) => {
      if (!school.cp) school.cp = ''
      const key = `${school.school_name}_${school.region}_${school.cp}`
      if (!uniqueSchools[key]) {
        uniqueSchools[key] = school
      }
    })
    schools.value = Object.values(uniqueSchools)
    console.log('去重后的学校数据:', schools.value.length, '所学校')
    schools.value.forEach((school: any, index: number) => {
      console.log(`学校${index + 1}:`, school.school_name, '运营商:', school.cp, '地区:', school.region)
    })
    
    // 根据最新学校数据刷新地区/运营商选项
    computeRegionCpOptions()

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
    
    // 计算时间范围
    const startDate = new Date(queryForm.start_time)
    const endDate = new Date(queryForm.end_time)
    const diffMinutes = (endDate.getTime() - startDate.getTime()) / (1000 * 60)
    const diffHours = diffMinutes / 60
    const diffDays = diffHours / 24
    
    // 始终使用原始5分钟粒度
    const granularity = '5m'

    // 按5分钟粒度精确计算所需点数，并留出缓冲，避免服务端降采样或返回不足
    const expectedPoints = Math.ceil(diffMinutes / 5)
    const limit = expectedPoints + 100
    console.log(`按5分钟粒度查询，预期点数: ${expectedPoints}，limit: ${limit}`)
    
    // 构建查询参数
    const params: Record<string, any> = {
      start_time: queryForm.start_time,
      end_time: queryForm.end_time,
      limit: limit, // 使用计算出的限制
      offset: 0, // 不使用分页
      granularity: granularity // 指定时间粒度
    }
    
    // 处理学校和内容方的过滤逻辑
    if (queryForm.region) {
      params.region = queryForm.region
    }
    
    // 如果选择了学校名称但没有选择内容方，则使用学校名称过滤
    if (queryForm.school_name) {
      params.school_name = queryForm.school_name
      
      // 如果没有选择内容方，则不添加内容方过滤条件
      // 这样后端会返回该学校所有内容方的数据
      if (queryForm.cp) {
        params.cp = queryForm.cp
      }
    } else if (queryForm.cp) {
      // 如果只选择了内容方而没有选择学校，则使用内容方过滤
      params.cp = queryForm.cp
    }
    
    // 在图表上显示当前使用的粒度
    currentGranularity.value = granularity || '5m'
    
    // 打印当前时间范围参数，便于调试
    console.log(`查询时间范围: ${startDate.toLocaleString()} 至 ${endDate.toLocaleString()}, 共${diffDays.toFixed(1)}天 (${diffHours.toFixed(1)}小时)`)
    console.log('详细查询参数:', params, '限制数量:', limit)
    
    // 使用真实的API调用
    const res = await (api as any).v2.getTrafficData(params) as any
    let rawList: any[] = []
    if (Array.isArray(res)) {
      rawList = res
    } else if (res && Array.isArray(res.items)) {
      rawList = res.items
    } else {
      rawList = []
    }

    if (!Array.isArray(rawList)) {
      console.warn('未获取到有效的流量数据列表')
      rawList = []
    }

    // 处理后端返回的数据，处理可能的字段名变化
    const processedData = rawList.map(item => {
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
      console.log('原始数据:', JSON.stringify(rawList[0] || {}))
      console.log('处理后数据:', JSON.stringify(processedData[0] || {}))
      
      // 依赖服务端时间范围过滤，前端不再二次截断，避免时区误差导致数据丢失
      const filteredData = processedData

      // 聚合策略：
      // 1) 无任何筛选（学校/地区/运营商均为空）时，按时间点聚合求和，显示整体“总服务/总回源流速”。
      // 2) 选择了学校但未选择内容方时，按时间点聚合该学校的所有内容方数据。
      let finalData = filteredData
      if (!queryForm.school_name && !queryForm.region && !queryForm.cp) {
        console.log('检测到无任何筛选条件，按时间点聚合全量数据')

        const dataByTimeAll: Record<string, any> = {}
        filteredData.forEach((item: any) => {
          const key = toMinuteKeyStr(item.time_str || item.create_time)
          if (!key) return
          if (!dataByTimeAll[key]) {
            dataByTimeAll[key] = {
              create_time: key,
              total_recv: 0,
              total_send: 0,
              time_str: key,
            }
          }
          dataByTimeAll[key].total_recv += Number(item.total_recv) || 0
          dataByTimeAll[key].total_send += Number(item.total_send) || 0
        })
        finalData = Object.values(dataByTimeAll).sort((a: any, b: any) => (a.create_time as string).localeCompare(b.create_time as string))
        try { console.log('无筛选聚合桶数:', Object.keys(dataByTimeAll).length) } catch {}
        console.log(`无筛选聚合后数据点: ${finalData.length}, 原始: ${filteredData.length}`)
      } else if (queryForm.school_name && !queryForm.cp) {
        console.log('检测到选择了学校但未选择内容方，将进行数据合并处理')
        
        // 按时间点分组数据
        const dataByTime: Record<string, any> = {}
        filteredData.forEach(item => {
          const key = toMinuteKeyStr(item.time_str || item.create_time)
          if (!key) return
          if (!dataByTime[key]) {
            dataByTime[key] = {
              create_time: key,
              school_name: queryForm.school_name,
              region: item.region || '',
              total_recv: 0,
              total_send: 0,
              time_str: key
            }
          }
          
          // 累加流量数据
          dataByTime[key].total_recv += Number(item.total_recv) || 0
          dataByTime[key].total_send += Number(item.total_send) || 0
        })
        
        // 转换回数组形式
        finalData = Object.values(dataByTime).sort((a: any, b: any) => (a.create_time as string).localeCompare(b.create_time as string))
        try { console.log('学校聚合桶数:', Object.keys(dataByTime).length) } catch {}
        console.log(`合并后的数据点数量: ${finalData.length}, 原始数据点数量: ${filteredData.length}`)
      }
      
      trafficData.value = finalData
      total.value = finalData.length
      
      console.log(`加载流量数据成功: 原始${rawList.length}条, 处理后${processedData.length}条, 过滤后${filteredData.length}条`)
      
      // 调试信息，查看数据结构
      if (rawList.length > 0) {
        console.log('数据样例:', rawList[0])
      }
      
      // 如果数据为空，显示提示
      if (filteredData.length === 0) {
        ElMessage.warning(`所选时间范围内没有数据，请尝试其他时间范围`)
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
  if (!hasFilter.value) {
    ElMessage.info('请选择地区、内容方或学校后再查询')
    trafficData.value = []
    total.value = 0
    return
  }
  currentPage.value = 1
  loadTrafficData()
}

// 当选择省份变化时重新加载学校列表
function handleRegionChange(region) {
  queryForm.school_name = ''
  // 先按地区/运营商重新加载学校，然后刷新选项集合
  loadSchools(region, queryForm.cp).then(() => computeRegionCpOptions())
  console.log('基于地区筛选学校:', region, queryForm.cp)
}

// 当选择运营商变化时重新加载学校列表
function handleCPChange(cp) {
  queryForm.school_name = ''
  // 先按地区/运营商重新加载学校，然后刷新选项集合
  loadSchools(queryForm.region, cp).then(() => computeRegionCpOptions())
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
  queryForm.start_time = toRFC3339Seconds(startTime)
  queryForm.end_time = toRFC3339Seconds(now)
  
  console.log('设置时间范围:', queryForm.start_time, '至', queryForm.end_time)
  
  // 测试时间范围设置是否生效
  const startDate = new Date(queryForm.start_time)
  const endDate = new Date(queryForm.end_time)
  const diffHours = (endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60)
  ElMessage.info(`时间范围设置成功，共${diffHours.toFixed(1)}小时`)
  
  // 重置分页到第一页
  currentPage.value = 1
  
  // 只有在存在筛选条件时才查询
  if (hasFilter.value) loadTrafficData()
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
  queryForm.start_time = toRFC3339Seconds(oneHourAgo)
  queryForm.end_time = toRFC3339Seconds(now)
  
  // 清空数据并不自动加载
  currentPage.value = 1
  trafficData.value = []
  total.value = 0
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
  // *8 是将字节转换为比特
  // /60 是将每分钟的数据转换为每秒的数据
  // 我们始终使用原始5分钟粒度，所以因子始终是60
  const factor = 300
  
  // 将字节转换为比特，然后除以时间因子
  return (bytes * 8) / factor
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
function formatDate(date: Date | string, granularity: string) {
  if (!date) return ''
  
  try {
    // 规范为 Date 对象
    const d: Date = typeof date === 'string' ? new Date(date) : date
    
    const year = d.getFullYear()
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    const hour = String(d.getHours()).padStart(2, '0')
    const minute = String(d.getMinutes()).padStart(2, '0')
    
    // 根据粒度格式化日期
    switch (granularity) {
      case 'hour':
        // 小时粒度
        return `${month}-${day} ${hour}:00`
      case 'day':
        // 天粒度
        return `${month}-${day}`
      case 'week': {
        // 周粒度
        const firstDayOfYear = new Date(year, 0, 1)
        const pastDaysOfYear = (d.getTime() - firstDayOfYear.getTime()) / 86400000
        const weekNumber = Math.ceil((pastDaysOfYear + firstDayOfYear.getDay() + 1) / 7)
        return `${year}-W${weekNumber}`
      }
      case 'month':
        // 月粒度
        return `${year}-${month}`
      case '15m': {
        // 15分钟粒度
        // 将分钟调整为15分钟的倍数
        const roundedMinute = Math.floor(d.getMinutes() / 15) * 15
        return `${hour}:${String(roundedMinute).padStart(2, '0')}`
      }
      default:
        // 原始5分钟粒度
        return `${hour}:${minute}`
    }
  } catch (error) {
    console.error('格式化日期出错:', error, date)
    return String(date)
  }
}
</script>

<template>
  <div class="traffic-container">
    <h1 class="page-title">流量监控</h1>
    
    <!-- 查询表单 -->
    <ElCard class="query-card">
      <ElForm :model="queryForm" label-width="80px" inline class="filter-form">
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
        
        <ElFormItem label="CP">
          <ElSelect v-model="queryForm.cp" placeholder="选择 CP" clearable @change="handleCPChange">
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
        <ElTableColumn prop="create_time" label="时间" width="200">
          <template #default="scope">
            {{ typeof scope.row.create_time === 'number' 
              ? new Date(scope.row.create_time).toLocaleString() 
              : (scope.row.time_str || scope.row.create_time) }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="school_name" label="学校名称" />
        <ElTableColumn prop="region" label="地区" />
        <ElTableColumn prop="cp" label="内容方" />
        <ElTableColumn prop="total_recv" label="服务流速">
          <template #default="scope">
            {{ formatBitRate(convertToBitsPerSecond(scope.row.total_recv)) }}
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

.filter-form { row-gap: var(--form-item-gap); }

:deep(.el-select) {
  width: 180px !important;
}

:deep(.el-date-editor) {
  width: 180px !important;
}
</style>
