<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api'
import PageHeader from '@/components/ui/PageHeader.vue'
import FilterPanel from '@/components/ui/FilterPanel.vue'
import SectionCard from '@/components/ui/SectionCard.vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent, GridComponent, DataZoomComponent, ToolboxComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { 
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

function parseQueryTimeInput(value: unknown): Date | null {
  if (value instanceof Date) {
    return Number.isNaN(value.getTime()) ? null : value
  }
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) return null
    const direct = new Date(trimmed)
    if (!Number.isNaN(direct.getTime())) return direct
    const m = trimmed.match(/^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2}):(\d{2})$/)
    if (m) {
      const d = new Date(
        Number(m[1]),
        Number(m[2]) - 1,
        Number(m[3]),
        Number(m[4]),
        Number(m[5]),
        Number(m[6]),
      )
      if (!Number.isNaN(d.getTime())) return d
    }
  }
  return null
}

function normalizeTimeRangeForRequest(startRaw: unknown, endRaw: unknown) {
  const startDate = parseQueryTimeInput(startRaw)
  const endDate = parseQueryTimeInput(endRaw)
  if (!startDate || !endDate) {
    throw new Error('时间格式错误，请重新选择开始/结束时间')
  }
  if (!(startDate.getTime() < endDate.getTime())) {
    throw new Error('开始时间必须早于结束时间')
  }
  return {
    startDate,
    endDate,
    startRFC3339: toRFC3339Seconds(startDate),
    endRFC3339: toRFC3339Seconds(endDate),
  }
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

const pagedTrafficData = computed(() => {
  const list = trafficData.value as any[]
  const size = Number(pageSize.value) || 10
  const page = Math.max(1, Number(currentPage.value) || 1)
  const start = (page - 1) * size
  const end = start + size
  return list.slice(start, end)
})

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

  const sortedData = trafficData.value as any[]

  // 将原始数据转换为 bits/s，并与时间戳配对，供 time 轴渲染
  const points = sortedData.map(item => {
    const ms = typeof item.create_time === 'number' ? item.create_time : parseTime(item.create_time).getTime()
    return {
      t: ms,
      cp: String((item as any).cp || ''),
      recv: (item as any).recv_bps != null ? Number((item as any).recv_bps) : convertToBitsPerSecond((item as any).total_recv),
      send: (item as any).send_bps != null ? Number((item as any).send_bps) : convertToBitsPerSecond((item as any).total_send),
    }
  }).filter(p => !isNaN(p.t))

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

  const pointCount = points.length
  const heavy = pointCount > 2000
  const perSeriesBase = {
    type: 'line',
    smooth: true,
    showSymbol: false,
    hoverAnimation: false,
    emphasis: { disabled: true },
    areaStyle: { opacity: 0.1 },
    large: true,
    largeThreshold: 2000,
    progressive: 4000,
    progressiveThreshold: 10000,
    animation: false,
  }

  const compareByCP = !!(queryForm.school_name && !queryForm.cp)
  let legendData: string[] = []
  let series: any[] = []

  if (compareByCP) {
    const cpBuckets: Record<string, Array<{ t: number; recv: number; send: number }>> = {}
    const totalByTime: Record<number, number> = {}

    points.forEach((p) => {
      const cp = p.cp || '未知CP'
      if (!cpBuckets[cp]) cpBuckets[cp] = []
      cpBuckets[cp].push({ t: p.t, recv: p.recv, send: p.send })
      totalByTime[p.t] = (totalByTime[p.t] || 0) + p.recv
    })

    const cps = Object.keys(cpBuckets).sort()
    legendData = ['各CP总服务流速', ...cps.flatMap(cp => [`${cp}-服务`, `${cp}-回源`])]

    const totalSeries = {
      ...perSeriesBase,
      name: '各CP总服务流速',
      lineStyle: { width: 3, type: 'solid' },
      z: 10,
      data: Object.keys(totalByTime)
        .map((k) => Number(k))
        .sort((a, b) => a - b)
        .map((t) => [t, totalByTime[t]]),
    }

    series = [
      totalSeries,
      ...cps.flatMap((cp) => {
        const rows = cpBuckets[cp]
        const recvSeries = {
          ...perSeriesBase,
          name: `${cp}-服务`,
          data: rows.map(r => [r.t, r.recv]),
        }
        const sendSeries = {
          ...perSeriesBase,
          name: `${cp}-回源`,
          data: rows.map(r => [r.t, r.send]),
        }
        return [recvSeries, sendSeries]
      })
    ]
  } else {
    // 默认双线模式下，先按时间点聚合，避免同一时间多条记录导致折线失真
    const mergedByTime: Record<number, { recv: number; send: number }> = {}
    points.forEach((p) => {
      if (!mergedByTime[p.t]) mergedByTime[p.t] = { recv: 0, send: 0 }
      mergedByTime[p.t].recv += Number(p.recv) || 0
      mergedByTime[p.t].send += Number(p.send) || 0
    })
    const timeline = Object.keys(mergedByTime).map((k) => Number(k)).sort((a, b) => a - b)

    legendData = ['服务流速', '回源流速']
    series = [
      {
        ...perSeriesBase,
        name: '服务流速',
        data: timeline.map((t) => [t, mergedByTime[t].recv]),
      },
      {
        ...perSeriesBase,
        name: '回源流速',
        data: timeline.map((t) => [t, mergedByTime[t].send]),
      }
    ]
  }

  return {
    title: {
      text: `学校流量监控 (bits/s) - ${formatGranularity(currentGranularity.value)}`,
      left: 'center'
    },
    tooltip: {
      trigger: 'axis',
      triggerOn: heavy ? 'click' : 'mousemove|click',
      transitionDuration: 0,
      confine: true,
      axisPointer: {
        type: 'line',
        animation: false,
      },
      formatter: function(params) {
        let result = ''
        if (params && params.length) {
          const first = params[0]
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
      data: legendData,
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
        end: 100,
        throttle: 100,
        realtime: !heavy,
      },
      {
        start: 0,
        end: 100,
        throttle: 100,
        realtime: !heavy,
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
    series,
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
      computeRegionCpOptions(true)
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
function computeRegionCpOptions(forceOverwrite = false) {
  try {
    const rset = new Set<string>()
    const cset = new Set<string>()
    ;(schools.value || []).forEach((s: any) => {
      if (s && typeof s.region === 'string' && s.region && s.region !== 'NULL') rset.add(s.region)
      if (s && typeof s.cp === 'string' && s.cp && s.cp !== 'NULL') cset.add(s.cp)
    })
    const nextRegions = Array.from(rset).sort()
    const nextCPs = Array.from(cset).sort()
    if (forceOverwrite || !Array.isArray(regions.value) || regions.value.length === 0) {
      regions.value = nextRegions
    }
    if (forceOverwrite || !Array.isArray(cps.value) || cps.value.length === 0) {
      cps.value = nextCPs
    }
  } catch (e) {
    console.warn('派生地区/运营商选项失败:', e)
    if (forceOverwrite || !Array.isArray(regions.value) || regions.value.length === 0) regions.value = []
    if (forceOverwrite || !Array.isArray(cps.value) || cps.value.length === 0) cps.value = []
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
    const params: Record<string, any> = { limit: 500 }
    if (region) params.region = region
    if (cp) params.cp = cp

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

    // 地区已选且 CP 未选时，按该地区可见学校动态收敛 CP 选项
    if (region && !cp) {
      const cpSet = new Set<string>()
      schoolsList.forEach((school: any) => {
        const v = String(school?.cp || '').trim()
        if (v && v !== 'NULL') cpSet.add(v)
      })
      cps.value = Array.from(cpSet).sort()
      // 若当前已选 CP 不在该地区范围内，则清空
      if (queryForm.cp && !cpSet.has(queryForm.cp)) {
        queryForm.cp = ''
      }
    }

    // 学校下拉按 school_name 去重（不区分 CP）
    const uniqueSchools: Record<string, any> = {}
    schoolsList.forEach((school: any) => {
      const name = String(school?.school_name || '').trim()
      if (!name) return
      if (!uniqueSchools[name]) {
        uniqueSchools[name] = school
      }
    })
    schools.value = Object.values(uniqueSchools)
    console.log('去重后的学校数据:', schools.value.length, '所学校')

    // 仅在接口未返回地区/运营商时，才基于学校数据兜底填充
    computeRegionCpOptions()

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
    const normalizedRange = normalizeTimeRangeForRequest(queryForm.start_time, queryForm.end_time)
    queryForm.start_time = normalizedRange.startRFC3339
    queryForm.end_time = normalizedRange.endRFC3339
    const startDate = normalizedRange.startDate
    const endDate = normalizedRange.endDate
    const diffMinutes = (endDate.getTime() - startDate.getTime()) / (1000 * 60)
    const diffHours = diffMinutes / 60
    const diffDays = diffHours / 24
    
    // 始终使用原始5分钟粒度
    const granularity = '5m'

    // 估算查询上限：
    // 当库中存在同一时间点多条记录时，按5分钟估算会低估并触发 LIMIT 截断，导致长周期断点。
    // 这里按“每分钟1条”的保守估算，并与5分钟估算取较大值。
    const estimateLimit = (minutes: number) => {
      const safeMinutes = Math.max(0, Number(minutes) || 0)
      const by5m = Math.ceil(safeMinutes / 5) + 100
      const by1m = Math.ceil(safeMinutes) + 200
      return Math.max(by5m, by1m)
    }
    const expectedPoints = Math.ceil(diffMinutes / 5)
    const limit = estimateLimit(diffMinutes)
    console.log(`按5分钟粒度查询，预期点数: ${expectedPoints}，安全limit: ${limit}`)
    
    // 构建查询参数
    const params: Record<string, any> = {
      start_time: normalizedRange.startRFC3339,
      end_time: normalizedRange.endRFC3339,
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
    console.log(`查询时间范围(本地): ${startDate.toLocaleString()} 至 ${endDate.toLocaleString()}, 共${diffDays.toFixed(1)}天 (${diffHours.toFixed(1)}小时)`)
    console.debug(`[traffic.view] request window start=${normalizedRange.startRFC3339} end=${normalizedRange.endRFC3339}`)
    console.log('详细查询参数:', params, '限制数量:', limit)
    
    // 分片加载：大范围拆分为多个窗口，降低单次响应体积与解析成本（仍为5分钟粒度）
    let rawList: any[] = []
    const windowMs = diffDays > 30 ? (10 * 24 * 60 * 60 * 1000) : (diffDays > 7 ? (7 * 24 * 60 * 60 * 1000) : 0)
    if (windowMs > 0) {
      let cursor = new Date(startDate.getTime())
      let idx = 0
      while (cursor.getTime() < endDate.getTime()) {
        const chunkStart = new Date(cursor.getTime())
        const chunkEnd = new Date(Math.min(cursor.getTime() + windowMs, endDate.getTime()))
        const chunkDiffMinutes = (chunkEnd.getTime() - chunkStart.getTime()) / (1000 * 60)
        const chunkExpected = estimateLimit(chunkDiffMinutes)
        const chunkParams: Record<string, any> = {
          ...params,
          start_time: toRFC3339Seconds(chunkStart),
          end_time: toRFC3339Seconds(chunkEnd),
          limit: chunkExpected,
          mode: 'compact',            // 未来后端可返回紧凑格式
          fields: 't,recv_bps,send_bps' // 最小列集请求（后端未实现时会被忽略）
        }
        console.log(`分片请求[${++idx}]`, chunkParams)
        const res = await (api as any).v2.getTrafficData(chunkParams) as any
        let list: any[] = []
        if (Array.isArray(res)) { list = res }
        else if (res && Array.isArray(res.items)) { list = res.items }
        rawList = rawList.concat(list)
        // 与后端半开区间保持一致：[start,end)
        cursor = new Date(chunkEnd.getTime())
      }
      console.log(`分片完成，合并总数: ${rawList.length}`)
    } else {
      // 小范围直接一次性请求
      const res = await (api as any).v2.getTrafficData(params) as any
      if (Array.isArray(res)) { rawList = res }
      else if (res && Array.isArray(res.items)) { rawList = res.items }
      else { rawList = [] }
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
      // 1) 无任何筛选（理论上不会触发查询）时，按时间点聚合。
      // 2) 已选择学校且已指定 CP 时，按时间点聚合，避免同时间桶重复记录。
      // 3) 已选择学校但未指定 CP 时，保留 CP 维度，便于查看同院校不同 CP。
      let finalData = filteredData
      if (!queryForm.school_name && !queryForm.region && !queryForm.cp) {
        const dataByTimeAll: Record<string, any> = {}
        filteredData.forEach((item: any) => {
          const key = toMinuteKeyStr(item.time_str || item.create_time)
          if (!key) return
          if (!dataByTimeAll[key]) {
            dataByTimeAll[key] = { create_time: key, total_recv: 0, total_send: 0, time_str: key }
          }
          dataByTimeAll[key].total_recv += Number(item.total_recv) || 0
          dataByTimeAll[key].total_send += Number(item.total_send) || 0
        })
        finalData = Object.values(dataByTimeAll).sort((a: any, b: any) => (a.create_time as string).localeCompare(b.create_time as string))
      } else if (queryForm.school_name && queryForm.cp) {
        const dataByTime: Record<string, any> = {}
        filteredData.forEach((item: any) => {
          const key = toMinuteKeyStr(item.time_str || item.create_time)
          if (!key) return
          if (!dataByTime[key]) {
            dataByTime[key] = {
              create_time: key,
              school_name: queryForm.school_name,
              region: item.region || '',
              cp: queryForm.cp,
              total_recv: 0,
              total_send: 0,
              time_str: key,
            }
          }
          dataByTime[key].total_recv += Number(item.total_recv) || 0
          dataByTime[key].total_send += Number(item.total_send) || 0
        })
        finalData = Object.values(dataByTime).sort((a: any, b: any) => (a.create_time as string).localeCompare(b.create_time as string))
      }
      
      // 预计算 bps，减少渲染与 tooltip 阶段的重复换算
      const withBps = finalData.map((it: any) => ({
        ...it,
        recv_bps: it && it.recv_bps != null ? Number(it.recv_bps) : convertToBitsPerSecond(it.total_recv),
        send_bps: it && it.send_bps != null ? Number(it.send_bps) : convertToBitsPerSecond(it.total_send),
      }))
      trafficData.value = withBps
      total.value = withBps.length
      
      console.log(`加载流量数据成功: 原始${rawList.length}条, 处理后${processedData.length}条, 过滤后${filteredData.length}条`)
      console.debug(`[traffic.view] response rows=${rawList.length} start=${normalizedRange.startRFC3339} end=${normalizedRange.endRFC3339}`)
      
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
    const status = (error as any)?.response?.status
    const backendMsg = (error as any)?.response?.data?.message || (error as any)?.response?.data?.error
    if (status === 400) {
      ElMessage.error(backendMsg || '时间参数错误，请检查开始/结束时间格式')
    } else if (status === 401 || status === 403) {
      ElMessage.error('无权访问该流量数据，请检查登录状态和权限')
    } else if (backendMsg) {
      ElMessage.error(String(backendMsg))
    } else if (error instanceof Error && error.message) {
      ElMessage.error(error.message)
    } else {
      ElMessage.error('加载流量数据失败')
    }
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
async function handleRegionChange(region) {
  queryForm.school_name = ''
  queryForm.cp = ''
  if (!region) {
    await loadRegionCpOptions()
    await loadSchools('', '')
    return
  }
  await loadSchools(region, '')
  console.log('基于地区筛选学校:', region)
}

// 当选择运营商变化时重新加载学校列表
async function handleCPChange(cp) {
  const prevSchool = queryForm.school_name
  // 按地区/运营商重新加载学校；仅当当前学校在新条件下不可见时才清空
  await loadSchools(queryForm.region, cp)
  if (prevSchool) {
    const stillVisible = (schools.value || []).some((s: any) => String(s?.school_name || '') === String(prevSchool))
    queryForm.school_name = stillVisible ? prevSchool : ''
  }
  console.log('基于运营商筛选学校:', queryForm.region, cp)
}

// 处理预设时间范围变化
function handleTimeRangeChange(value) {
  const now = new Date()
  let startTime

  switch (value) {
    case 'last1h':
      startTime = new Date(now.getTime() - 1 * 60 * 60 * 1000)
      break
    case 'last3h':
      startTime = new Date(now.getTime() - 3 * 60 * 60 * 1000)
      break
    case 'last6h':
      startTime = new Date(now.getTime() - 6 * 60 * 60 * 1000)
      break
    case 'last12h':
      startTime = new Date(now.getTime() - 12 * 60 * 60 * 1000)
      break
    case 'last24h':
      startTime = new Date(now.getTime() - 24 * 60 * 60 * 1000)
      break
    case 'last2d':
      startTime = new Date(now.getTime() - 2 * 24 * 60 * 60 * 1000)
      break
    case 'last7d':
      startTime = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
      break
    case 'last30d':
      startTime = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
      break
    case 'custom':
      return
    default:
      startTime = new Date(now.getTime() - 1 * 60 * 60 * 1000)
  }
  
  // 设置时间范围
  queryForm.start_time = toRFC3339Seconds(startTime)
  queryForm.end_time = toRFC3339Seconds(now)
  
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
  const factor = 60
  
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
  <div class="page-container">
    <PageHeader title="流量监控" description="按地区、CP、学校和时间范围查询流速趋势与明细。" />
    
    <!-- 查询表单 -->
    <FilterPanel>
      <ElForm :model="queryForm" label-width="80px" inline class="filter-form">
        <ElFormItem label="地区">
          <ElSelect v-model="queryForm.region" placeholder="选择地区（可输入）" clearable filterable allow-create default-first-option @change="handleRegionChange" class="field-sm">
            <ElOption 
              v-for="region in regions" 
              :key="region" 
              :label="region" 
              :value="region" 
            />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="CP">
          <ElSelect v-model="queryForm.cp" placeholder="选择 CP（可输入）" clearable filterable allow-create default-first-option @change="handleCPChange" class="field-sm">
            <ElOption 
              v-for="cp in cps" 
              :key="cp" 
              :label="cp" 
              :value="cp" 
            />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="学校名称">
          <ElSelect v-model="queryForm.school_name" placeholder="选择或输入学校" clearable filterable allow-create default-first-option :reserve-keyword="false" class="field-lg">
            <ElOption 
              v-for="school in schools" 
              :key="school.school_name" 
              :label="school.school_name" 
              :value="school.school_name" 
            />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="时间范围">
          <ElSelect v-model="queryForm.timeRange" placeholder="选择时间范围" @change="handleTimeRangeChange" class="field-xs">
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
    </FilterPanel>
    
    <!-- 流量图表 -->
    <SectionCard title="趋势图" v-loading="chartLoading">
      <v-chart class="traffic-chart" :option="chartOption" autoresize />
    </SectionCard>
    
    <!-- 流量数据表格 -->
    <SectionCard title="流量明细">
      <ElTable :data="pagedTrafficData" border stripe v-loading="loading">
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
            {{ formatBitRate(Number(scope.row.recv_bps != null ? scope.row.recv_bps : convertToBitsPerSecond(scope.row.total_recv))) }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="total_send" label="回源流速">
          <template #default="scope">
            {{ formatBitRate(Number(scope.row.send_bps != null ? scope.row.send_bps : convertToBitsPerSecond(scope.row.total_send))) }}
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
    </SectionCard>
  </div>
</template>

<style scoped>
.traffic-chart {
  height: 400px;
  width: 100%;
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

.field-sm:deep(.el-select__wrapper),
.field-sm:deep(.el-input__wrapper),
.field-sm:deep(.el-date-editor) {
  width: 180px !important;
}

.field-lg:deep(.el-select__wrapper),
.field-lg:deep(.el-input__wrapper) {
  width: 300px !important;
}

.field-xs:deep(.el-select__wrapper),
.field-xs:deep(.el-input__wrapper) {
  width: 150px !important;
}
</style>



