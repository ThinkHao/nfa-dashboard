<script setup lang="ts">
import { ref, reactive, onMounted, onActivated, nextTick, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api'
import PageHeader from '@/components/ui/PageHeader.vue'
import FilterPanel from '@/components/ui/FilterPanel.vue'
import SectionCard from '@/components/ui/SectionCard.vue'
import SearchSelect from '@/components/ui/SearchSelect.vue'
import UnifiedDateRange from '@/components/ui/UnifiedDateRange.vue'
import QueryActionButton from '@/components/ui/QueryActionButton.vue'
import { buildRangeValue, splitRangeValue } from '@/components/ui/unified-date-range-utils'
import { clearTrafficCustomRange, resolvePresetTrafficRange, type TrafficTimeRangeOption } from './traffic-time-range'
import { calculateTrafficP95, type TrafficP95Result } from './traffic-percentile'
import { useCancelableQuery, isAbortError } from '@/composables/useCancelableQuery'
import { usePageRefresh } from '@/composables/usePageRefresh'
import { useSystemTrafficSettings } from '@/composables/useSystemTrafficSettings'
import { normalizeByteUnitBase } from '@/utils/traffic-units'
import { sanitizeScopeOptionValues } from '@/utils/scope-options'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent, GridComponent, DataZoomComponent, ToolboxComponent, MarkLineComponent } from 'echarts/components'
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

type EDCEntityOption = Record<string, any>

function normalizeEDCText(value: unknown): string {
  const text = String(value ?? '').trim()
  return text && text.toUpperCase() !== 'NULL' ? text : ''
}

function getEDCPrimaryLabel(entity: EDCEntityOption): string {
  return normalizeEDCText(entity?.alias)
    || normalizeEDCText(entity?.display_name)
    || normalizeEDCText(entity?.edc_name)
    || normalizeEDCText(entity?.school_name)
    || (Number(entity?.id) > 0 ? `EDC#${entity.id}` : '未命名 EDC')
}

function getEDCOriginalName(entity: EDCEntityOption): string {
  return normalizeEDCText(entity?.edc_name)
}

function getEDCSearchLabel(entity: EDCEntityOption): string {
  const primary = getEDCPrimaryLabel(entity)
  const names = [normalizeEDCText(entity?.display_name), normalizeEDCText(entity?.edc_name)]
    .filter((name, index, values) => name && name !== primary && values.indexOf(name) === index)
  return names.length > 0 ? `${primary}（${names.join(' / ')}）` : primary
}

function getEDCTypeLabel(value: unknown): string {
  const type = normalizeEDCText(value)
  if (type === 'node') return '节点'
  if (type === 'transmission') return '传输'
  return type
}

function formatEDCOptionMeta(entity: EDCEntityOption): string {
  const originalName = getEDCOriginalName(entity)
  const displayName = normalizeEDCText(entity?.display_name)
  const names = originalName ? [`原名：${originalName}`] : []
  if (displayName && displayName !== originalName) names.push(`显示名：${displayName}`)

  const dimensions = [
    normalizeEDCText(entity?.cp),
    getEDCTypeLabel(entity?.entity_type),
  ].filter(Boolean)
  const srcRegion = normalizeEDCText(entity?.src_region)
  const dstRegion = normalizeEDCText(entity?.dst_region)
  if (srcRegion && dstRegion) dimensions.push(`${srcRegion} → ${dstRegion}`)
  else if (srcRegion) dimensions.push(`源：${srcRegion}`)
  else if (dstRegion) dimensions.push(`目：${dstRegion}`)

  return [...names, ...dimensions].join(' · ')
}

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
  ToolboxComponent,
  MarkLineComponent
])

// 路由
const route = useRoute()

// 数据状态
const loading = ref(false)
const chartLoading = ref(false)
const trafficData = ref([])
const regions = ref([])
const cps = ref([])
const entityTypes = ref([])
const srcRegions = ref([])
const dstRegions = ref([])
const schools = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)
const currentGranularity = ref('5m') // 当前使用的时间粒度
const P95_LINE_COLOR = '#fb7185'
const P95_LINE_COLOR_SOFT = 'rgba(251, 113, 133, 0.72)'
const P95_LABEL_COLOR = '#e11d48'
const P95_LABEL_BG = 'rgba(251, 113, 133, 0.12)'
// 95值图例是否处于选中状态（控制 tooltip 中的 95 信息行）
const p95LegendVisible = ref(true)
const queryCtl = useCancelableQuery()
const trafficSettings = useSystemTrafficSettings()
const trafficByteUnitBase = computed(() => normalizeByteUnitBase(trafficSettings.settings.value.traffic_byte_unit_base, 1024))
const chartRef = ref<InstanceType<typeof VChart> | null>(null)

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
  data_source: 'nfa' as 'nfa' | 'edc',
  school_name: '',
  edc_entity_ids: [] as number[],
  entity_type: '',
  region: '',
  cp: '',
  src_region: '',
  dst_region: '',
  start_time: '',
  end_time: '',
  timeRange: 'last1h' as TrafficTimeRangeOption, // 默认选择过去1小时
})

const selectedEDCEntityIDs = computed(() => {
  if (!Array.isArray(queryForm.edc_entity_ids)) return []
  return queryForm.edc_entity_ids
    .map((id) => Number(id))
    .filter((id) => Number.isFinite(id) && id > 0)
})

const customDateRange = computed<[string, string] | null>({
  get: () => {
    if (queryForm.timeRange !== 'custom') return null
    return buildRangeValue(queryForm.start_time, queryForm.end_time)
  },
  set: (value) => {
    const { start, end } = splitRangeValue(value)
    queryForm.start_time = start
    queryForm.end_time = end
  },
})

// 是否已选择任一筛选条件（地区/内容方/学校）
const hasFilter = computed(() => {
  return !!(
    queryForm.school_name ||
    selectedEDCEntityIDs.value.length > 0 ||
    queryForm.entity_type ||
    (queryForm.data_source !== 'edc' && queryForm.region) ||
    queryForm.cp ||
    queryForm.src_region ||
    queryForm.dst_region
  )
})

const entityLabel = computed(() => queryForm.data_source === 'edc' ? 'EDC名称' : '学校名称')
const dataSourceTitle = computed(() => queryForm.data_source === 'edc' ? 'EDC流速监控' : '学校流速监控')
const dataSourceDescription = computed(() => queryForm.data_source === 'edc'
  ? '按类型、CP、源区域、目区域、EDC名称和时间范围查询流速趋势与明细。'
  : '按地区、CP、学校和时间范围查询流速趋势与明细。')
const selectedEDCEntities = computed<EDCEntityOption[]>(() => {
  const byID = new Map<number, EDCEntityOption>()
  ;(schools.value as EDCEntityOption[]).forEach((entity) => {
    const id = Number(entity?.id)
    if (Number.isFinite(id) && id > 0) byID.set(id, entity)
  })
  return selectedEDCEntityIDs.value.map((id) => byID.get(id) || { id, edc_name: `EDC#${id}` })
})
const selectedEDCAlias = computed(() => selectedEDCEntities.value.map((entity) => normalizeEDCText(entity?.alias)).filter(Boolean).join('、'))
const selectedEDCEntityNames = computed(() => selectedEDCEntities.value.map(getEDCPrimaryLabel))
const selectedEDCOriginalNames = computed(() => selectedEDCEntities.value.map(getEDCOriginalName))
const selectedEDCEntityLabel = computed(() => {
  const ids = selectedEDCEntityIDs.value
  if (queryForm.data_source !== 'edc' || ids.length === 0) return ''
  return selectedEDCEntityNames.value.join('、')
})
const selectedEDCOriginalName = computed(() => {
  if (queryForm.data_source !== 'edc' || selectedEDCEntityIDs.value.length === 0) return ''
  return selectedEDCOriginalNames.value.filter(Boolean).join('、')
})
const selectedEDCSelectionSummary = computed(() => {
  if (queryForm.data_source !== 'edc' || selectedEDCEntityIDs.value.length === 0) return ''
  return `已选 EDC：${selectedEDCEntityLabel.value}`
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
      title: { text: dataSourceTitle.value, left: 'center', subtext: '请选择任一筛选条件后再查询' },
      xAxis: { type: 'time' },
      yAxis: { type: 'value', name: '流速 (bits/s)' },
      series: []
    }
  }

  // 检查数据是否为空
  if (trafficData.value.length === 0) {
    return {
      title: {
        text: dataSourceTitle.value,
        left: 'center',
        subtext: selectedEDCSelectionSummary.value || undefined,
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
      recv: (item as any).recv_bps != null ? Number((item as any).recv_bps) : convertToBitsPerSecond(Number((item as any).total_recv ?? (item as any).service_size) || 0),
      send: (item as any).send_bps != null ? Number((item as any).send_bps) : convertToBitsPerSecond(Number((item as any).total_send ?? (item as any).cache_size) || 0),
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
  let p95Result: TrafficP95Result | null = null
  const p95TooltipMarker = `<span style="display:inline-block;width:16px;height:0;border-top:1px dashed ${P95_LINE_COLOR};vertical-align:middle;margin-right:6px;"></span>`

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
    const totalTimeline = Object.keys(totalByTime).map((k) => Number(k)).sort((a, b) => a - b)
    p95Result = calculateTrafficP95(totalTimeline.map((t) => ({ time: t, recvBps: totalByTime[t] })))

    const totalSeries = {
      ...perSeriesBase,
      name: '各CP总服务流速',
      lineStyle: { width: 3, type: 'solid' },
      z: 10,
      data: totalTimeline.map((t) => [t, totalByTime[t]]),
      ...(p95Result
        ? {
            markLine: {
              symbol: 'none',
              silent: true,
              animation: false,
              label: {
                formatter: () => `95 ${formatBitRate(p95Result?.valueBps || 0)}`,
                position: 'insideEndTop',
                distance: 6,
                color: P95_LABEL_COLOR,
                backgroundColor: P95_LABEL_BG,
                borderRadius: 4,
                padding: [3, 6],
                fontSize: 11,
              },
              lineStyle: {
                color: P95_LINE_COLOR_SOFT,
                width: 1.4,
                type: 'dashed',
                opacity: 0.82,
              },
              data: [{ name: '95值', yAxis: p95Result.valueBps }],
            },
          }
        : {}),
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
    const shouldShowP95 = !!(queryForm.school_name && queryForm.cp)
    p95Result = shouldShowP95
      ? calculateTrafficP95(timeline.map((t) => ({ time: t, recvBps: mergedByTime[t].recv })))
      : null

    legendData = p95Result ? ['服务流速', '回源流速', '95值'] : ['服务流速', '回源流速']
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
      },
      // 95值线挂在自己的 series 上，图例开关才能随之显隐
      ...(p95Result
        ? [{
            ...perSeriesBase,
            name: '95值',
            data: [],
            color: P95_LINE_COLOR,
            lineStyle: { color: P95_LINE_COLOR_SOFT, width: 1.4, type: 'dashed', opacity: 0.82 },
            areaStyle: undefined,
            tooltip: { show: false },
            silent: true,
            markLine: {
              symbol: 'none',
              silent: true,
              animation: false,
              label: {
                formatter: () => `95 ${formatBitRate(p95Result?.valueBps || 0)}`,
                position: 'insideEndTop',
                distance: 6,
                color: P95_LABEL_COLOR,
                backgroundColor: P95_LABEL_BG,
                borderRadius: 4,
                padding: [3, 6],
                fontSize: 11,
              },
              lineStyle: {
                color: P95_LINE_COLOR_SOFT,
                width: 1.4,
                type: 'dashed',
                opacity: 0.82,
              },
              data: [
                {
                  name: '95值',
                  yAxis: p95Result.valueBps,
                },
              ],
            },
          }]
        : [])
    ]
  }

  return {
    title: {
      text: `${dataSourceTitle.value} (bits/s) - ${formatGranularity(currentGranularity.value)}`,
      left: 'center',
      subtext: selectedEDCSelectionSummary.value || undefined,
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
        if (p95Result && (compareByCP || p95LegendVisible.value)) {
          result += `${p95TooltipMarker}95值: ${formatBitRate(p95Result.valueBps)}<br/>`
          result += `${p95TooltipMarker}95时间: ${new Date(p95Result.timeMs).toLocaleString()}<br/>`
        }
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

function isTrafficRouteActive() {
  return route.name === 'traffic' || route.path === '/traffic'
}

type RouteQueryLike = Record<string, unknown>

function applyRouteQueryFilters(q: RouteQueryLike, options: { autoQuery?: boolean } = {}) {
  if (!isTrafficRouteActive()) return

  const hasSchool = Object.prototype.hasOwnProperty.call(q, 'school_name')
  const hasEDCEntityIDs = Object.prototype.hasOwnProperty.call(q, 'entity_ids')
  const hasRegion = Object.prototype.hasOwnProperty.call(q, 'region')
  const hasCp = Object.prototype.hasOwnProperty.call(q, 'cp')
  const hasEntityType = Object.prototype.hasOwnProperty.call(q, 'entity_type')
  const hasSrcRegion = Object.prototype.hasOwnProperty.call(q, 'src_region')
  const hasDstRegion = Object.prototype.hasOwnProperty.call(q, 'dst_region')
  const hasDataSource = Object.prototype.hasOwnProperty.call(q, 'data_source')
  if (hasDataSource && (q.data_source === 'nfa' || q.data_source === 'edc')) queryForm.data_source = q.data_source
  const hasExplicitFilterKeys = hasSchool || hasEDCEntityIDs || (queryForm.data_source !== 'edc' && hasRegion) || hasCp || hasEntityType || hasSrcRegion || hasDstRegion
  if (hasSchool) queryForm.school_name = typeof q.school_name === 'string' ? q.school_name : ''
  if (hasEDCEntityIDs) {
    const raw = Array.isArray(q.entity_ids) ? q.entity_ids.join(',') : String(q.entity_ids || '')
    queryForm.edc_entity_ids = raw
      .split(',')
      .map((v) => Number(String(v).trim()))
      .filter((v) => Number.isFinite(v) && v > 0)
  }
  if (hasRegion) queryForm.region = queryForm.data_source === 'edc' ? '' : (typeof q.region === 'string' ? q.region : '')
  if (hasCp) queryForm.cp = typeof q.cp === 'string' ? q.cp : ''
  if (hasEntityType) queryForm.entity_type = q.entity_type === 'node' || q.entity_type === 'transmission' ? q.entity_type : ''
  if (hasSrcRegion) queryForm.src_region = typeof q.src_region === 'string' ? q.src_region : ''
  if (hasDstRegion) queryForm.dst_region = typeof q.dst_region === 'string' ? q.dst_region : ''

  if (options.autoQuery && hasExplicitFilterKeys && hasFilter.value) {
    currentPage.value = 1
    queryCtl.run((signal) => loadTrafficData(signal), { showCancelMessage: false })
  }
}

// 95值是被服务流速联动隐藏（而非用户手动隐藏）时置位，服务流速恢复时据此回显
let p95AutoHidden = false
// 程序化 dispatchAction 触发的图例事件不算用户手动操作
let syncingP95Legend = false

function toggleP95Legend(show: boolean) {
  syncingP95Legend = true
  try {
    chartRef.value?.dispatchAction({ type: show ? 'legendSelect' : 'legendUnSelect', name: '95值' })
  } finally {
    syncingP95Legend = false
  }
}

function handleLegendSelectChanged(params: any) {
  const selected = params?.selected
  if (!selected || !Object.prototype.hasOwnProperty.call(selected, '95值')) return

  const serviceOn = selected['服务流速'] !== false
  let p95On = selected['95值'] !== false

  if (params?.name === '95值' && !syncingP95Legend) {
    p95AutoHidden = false
  } else if (params?.name === '服务流速') {
    // 95值基于服务流速计算，隐藏服务流速时联动隐藏95值
    if (!serviceOn && p95On) {
      p95AutoHidden = true
      p95On = false
      toggleP95Legend(false)
    } else if (serviceOn && !p95On && p95AutoHidden) {
      p95AutoHidden = false
      p95On = true
      toggleP95Legend(true)
    }
  }

  p95LegendVisible.value = p95On
}

function resizeTrafficChart() {
  const doResize = () => {
    try { chartRef.value?.resize() } catch {}
  }
  nextTick(() => {
    doResize()
    if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
      window.requestAnimationFrame(() => doResize())
    }
  })
}

// 初始化数据
onMounted(async () => {
  try {
    await trafficSettings.ensureLoaded()
    // 设置默认时间范围为最近1小时（与 timeRange 保持一致）
    const initialRange = resolvePresetTrafficRange('last1h')
    queryForm.start_time = initialRange?.[0] || ''
    queryForm.end_time = initialRange?.[1] || ''
    
    // 读取路由查询参数作为默认过滤
    const q = (route.query || {}) as RouteQueryLike
    applyRouteQueryFilters(q, { autoQuery: false })
    
    // 先加载地区/运营商（v2，按用户可见范围）
    await loadRegionCpOptions()
    // 再加载学校数据（基于 v2，仅返回当前用户可见范围）
    await loadSchools()
    if (queryForm.data_source === 'edc') {
      computeEDCFilterOptions(schools.value)
    }
    // 若未能从接口获得地区/运营商，则基于学校数据兜底派生
    if (queryForm.data_source !== 'edc' && ((!regions.value || regions.value.length === 0) || (!cps.value || cps.value.length === 0))) {
      computeRegionCpOptions(true)
    }
    
    // 默认不加载流量数据，待用户选择筛选条件后点击查询
  } catch (error) {
    console.error('初始化数据失败:', error)
    ElMessage.error('加载数据失败，请刷新页面重试')
  }
})

onActivated(() => {
  resizeTrafficChart()
})

// 监听分页变化
watch(currentPage, () => {
  queryCtl.run((signal) => loadTrafficData(signal), { showCancelMessage: false })
})

// 监听路由查询变化（在已处于 /traffic 页面时再次从外部带参跳转也能生效）
watch(
  () => route.query,
  (q: any) => {
    try {
      if (q && typeof q === 'object') {
        applyRouteQueryFilters(q as RouteQueryLike, { autoQuery: true })
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

function computeEDCFilterOptions(items: any[]) {
  const collect = (key: string) => {
    const values = new Set<string>()
    items.forEach((item: any) => {
      const value = String(item?.[key] || '').trim()
      if (value && value !== 'NULL') values.add(value)
    })
    return Array.from(values).sort()
  }
  if (!queryForm.entity_type) {
  const discoveredTypes = collect('entity_type').filter((value) => value === 'node' || value === 'transmission')
  entityTypes.value = discoveredTypes.length > 0 ? discoveredTypes : ['node', 'transmission']
  }
  regions.value = []
  cps.value = collect('cp')
  srcRegions.value = collect('src_region')
  dstRegions.value = collect('dst_region')
}

function updateEDCDownstreamOptions(items: any[], level: 'type' | 'region' | 'cp' | 'src') {
  const collect = (key: string) => {
    const values = new Set<string>()
    items.forEach((item: any) => {
      const value = String(item?.[key] || '').trim()
      if (value && value !== 'NULL') values.add(value)
    })
    return Array.from(values).sort()
  }
  if (level === 'type') {
    regions.value = []
    cps.value = collect('cp')
    srcRegions.value = collect('src_region')
    dstRegions.value = collect('dst_region')
  } else if (level === 'region') {
    cps.value = collect('cp')
    srcRegions.value = collect('src_region')
    dstRegions.value = collect('dst_region')
  } else if (level === 'cp') {
    srcRegions.value = collect('src_region')
    dstRegions.value = collect('dst_region')
  } else {
    dstRegions.value = collect('dst_region')
  }
}

// 通过 v2 接口加载地区/运营商选项（按用户可见范围过滤）
async function loadRegionCpOptions() {
  if (queryForm.data_source === 'edc') {
    try {
      const options = await (api as any).v2.edc.getFilterOptions()
      entityTypes.value = sanitizeScopeOptionValues(Array.isArray(options?.entity_types) ? options.entity_types : []).sort()
      if (entityTypes.value.length === 0) entityTypes.value = ['node', 'transmission']
      regions.value = []
      cps.value = sanitizeScopeOptionValues(Array.isArray(options?.cps) ? options.cps : []).sort()
      srcRegions.value = sanitizeScopeOptionValues(Array.isArray(options?.src_regions) ? options.src_regions : []).sort()
      dstRegions.value = sanitizeScopeOptionValues(Array.isArray(options?.dst_regions) ? options.dst_regions : []).sort()
    } catch {
      entityTypes.value = []
      regions.value = []
      cps.value = []
      srcRegions.value = []
      dstRegions.value = []
    }
    return
  }
  entityTypes.value = []
  srcRegions.value = []
  dstRegions.value = []
  try {
    const r = await (api as any).v2.getRegions()
    regions.value = sanitizeScopeOptionValues(Array.isArray(r) ? r : []).sort()
  } catch {
    regions.value = []
  }
  try {
    const c = await (api as any).v2.getCPs()
    cps.value = sanitizeScopeOptionValues(Array.isArray(c) ? c : []).sort()
  } catch {
    cps.value = []
  }
}

// 加载学校数据
async function loadSchools(region = '', cp = '', entityType = '', srcRegion = '', dstRegion = '') {
  try {
    // 清空学校列表，避免显示旧数据
    schools.value = []

    // 构建请求参数
    const params: Record<string, any> = { limit: 500 }
    if (queryForm.data_source !== 'edc' && region) params.region = region
    if (cp) params.cp = cp
    if (entityType) params.entity_type = entityType
    if (srcRegion) params.src_region = srcRegion
    if (dstRegion) params.dst_region = dstRegion

    console.log('请求实体数据参数:', params)
    const res = queryForm.data_source === 'edc'
      ? await (api as any).v2.edc.getEntities(params)
      : await (api as any).v2.getSchools(params)
    console.log('实体数据原始响应:', res)

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
    if (queryForm.data_source !== 'edc' && region && !cp) {
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

    if (queryForm.data_source === 'edc') {
      schools.value = schoolsList
        .filter((school: any) => Number(school?.id) > 0)
        .map((school: any) => {
          const entity = {
            ...school,
            id: Number(school.id),
          }
          return {
            ...entity,
            school_name: getEDCPrimaryLabel(entity),
            edc_search_label: getEDCSearchLabel(entity),
          }
        })
        .filter((school: any) => school.school_name)
    } else {
      // NFA 下拉按学校名称去重（不区分 CP），保持原有单选行为。
      const uniqueSchools: Record<string, any> = {}
      schoolsList.forEach((school: any) => {
        const name = String(school?.school_name || '').trim()
        if (!name) return
        if (!uniqueSchools[name]) {
          uniqueSchools[name] = {
            ...school,
            school_name: name,
          }
        }
      })
      schools.value = Object.values(uniqueSchools)
    }
    console.log('去重后的实体数据:', schools.value.length)

    // 仅在接口未返回地区/运营商时，才基于学校数据兜底填充
    if (queryForm.data_source !== 'edc') computeRegionCpOptions()

    if (schools.value.length === 0) {
      console.warn('未获取到实体数据')
      ElMessage.warning('未能加载实体数据，请检查网络连接')
    }
  } catch (error) {
    console.error('加载实体数据失败:', error)
    ElMessage.error('加载实体数据失败')
    schools.value = []
  }
}



// 加载流量数据
async function loadTrafficData(signal?: AbortSignal) {
  try {
    chartLoading.value = true
    loading.value = true
    
    // 计算时间范围
    const normalizedRange = normalizeTimeRangeForRequest(queryForm.start_time, queryForm.end_time)
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
    
    // 处理实体和内容方的过滤逻辑
    if (queryForm.data_source !== 'edc' && queryForm.region) {
      params.region = queryForm.region
    }

    if (queryForm.data_source === 'edc') {
      if (queryForm.entity_type) params.entity_type = queryForm.entity_type
      if (queryForm.src_region) params.src_region = queryForm.src_region
      if (queryForm.dst_region) params.dst_region = queryForm.dst_region
    }
    
    const edcEntityIDs = selectedEDCEntityIDs.value
    if (queryForm.data_source === 'edc' && edcEntityIDs.length > 0) {
      params.entity_ids = edcEntityIDs.join(',')
      if (queryForm.cp) {
        params.cp = queryForm.cp
      }
    } else if (queryForm.school_name) {
      if (queryForm.data_source === 'edc') {
        params.display_name = queryForm.school_name
      } else {
        params.school_name = queryForm.school_name
      }
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
        if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
        const res = queryForm.data_source === 'edc'
          ? await (api as any).v2.edc.getTrafficData(chunkParams, { signal })
          : await (api as any).v2.getTrafficData(chunkParams, { signal })
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
      if (signal?.aborted) throw new DOMException('aborted', 'AbortError')
      const res = queryForm.data_source === 'edc'
        ? await (api as any).v2.edc.getTrafficData(params, { signal })
        : await (api as any).v2.getTrafficData(params, { signal })
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
            dataByTimeAll[key] = { create_time: key, total_recv: 0, total_send: 0, service_size: 0, cache_size: 0, time_str: key }
          }
          dataByTimeAll[key].total_recv += Number(item.total_recv ?? item.service_size) || 0
          dataByTimeAll[key].total_send += Number(item.total_send ?? item.cache_size) || 0
          dataByTimeAll[key].service_size = dataByTimeAll[key].total_recv
          dataByTimeAll[key].cache_size = dataByTimeAll[key].total_send
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
              display_name: queryForm.school_name,
              region: item.region || '',
              cp: queryForm.cp,
              total_recv: 0,
              total_send: 0,
              service_size: 0,
              cache_size: 0,
              time_str: key,
            }
          }
          dataByTime[key].total_recv += Number(item.total_recv ?? item.service_size) || 0
          dataByTime[key].total_send += Number(item.total_send ?? item.cache_size) || 0
          dataByTime[key].service_size = dataByTime[key].total_recv
          dataByTime[key].cache_size = dataByTime[key].total_send
        })
        finalData = Object.values(dataByTime).sort((a: any, b: any) => (a.create_time as string).localeCompare(b.create_time as string))
      }
      
      // 预计算 bps，减少渲染与 tooltip 阶段的重复换算
      const withBps = finalData.map((it: any) => {
        const isEDC = queryForm.data_source === 'edc'
        const alias = isEDC ? (selectedEDCAlias.value || normalizeEDCText(it.alias)) : ''
        const technicalName = isEDC ? (selectedEDCOriginalName.value || getEDCOriginalName(it)) : ''
        return {
          ...it,
          edc_alias: alias,
          edc_name: technicalName,
          school_name: it.school_name || alias || it.alias || it.display_name || selectedEDCEntityLabel.value || '',
          total_recv: Number(it.total_recv ?? it.service_size) || 0,
          total_send: Number(it.total_send ?? it.cache_size) || 0,
          recv_bps: it && it.recv_bps != null ? Number(it.recv_bps) : convertToBitsPerSecond(Number(it.total_recv ?? it.service_size) || 0),
          send_bps: it && it.send_bps != null ? Number(it.send_bps) : convertToBitsPerSecond(Number(it.total_send ?? it.cache_size) || 0),
        }
      })
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
    if (isAbortError(error)) {
      return
    }
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
    ElMessage.info('请选择筛选条件后再查询')
    trafficData.value = []
    total.value = 0
    return
  }
  currentPage.value = 1
  queryCtl.run((signal) => loadTrafficData(signal), { toggleIfRunning: true })
}

// 当选择省份变化时重新加载学校列表
async function handleRegionChange(region) {
  queryForm.school_name = ''
  queryForm.edc_entity_ids = []
  if (queryForm.data_source === 'edc') {
    return
  }
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
  if (queryForm.data_source === 'edc') {
    queryForm.school_name = ''
    queryForm.edc_entity_ids = []
    queryForm.src_region = ''
    queryForm.dst_region = ''
    await loadSchools(queryForm.region, cp, queryForm.entity_type)
    updateEDCDownstreamOptions(schools.value, 'cp')
    return
  }
  const prevSchool = queryForm.school_name
  // 按地区/运营商重新加载学校；仅当当前学校在新条件下不可见时才清空
  await loadSchools(queryForm.region, cp)
  if (prevSchool) {
    const stillVisible = (schools.value || []).some((s: any) => String(s?.school_name || '') === String(prevSchool))
    queryForm.school_name = stillVisible ? prevSchool : ''
  }
  console.log('基于运营商筛选学校:', queryForm.region, cp)
}

async function handleEntityTypeChange(entityType) {
  if (queryForm.data_source !== 'edc') return
  queryForm.school_name = ''
  queryForm.edc_entity_ids = []
  queryForm.region = ''
  queryForm.cp = ''
  queryForm.src_region = ''
  queryForm.dst_region = ''
  await loadSchools('', '', entityType || '')
  updateEDCDownstreamOptions(schools.value, 'type')
}

async function handleSrcRegionChange(srcRegion) {
  if (queryForm.data_source !== 'edc') return
  queryForm.school_name = ''
  queryForm.edc_entity_ids = []
  queryForm.dst_region = ''
  await loadSchools(queryForm.region, queryForm.cp, queryForm.entity_type, srcRegion || '')
  updateEDCDownstreamOptions(schools.value, 'src')
}

async function handleDstRegionChange(dstRegion) {
  if (queryForm.data_source !== 'edc') return
  queryForm.school_name = ''
  queryForm.edc_entity_ids = []
  await loadSchools(queryForm.region, queryForm.cp, queryForm.entity_type, queryForm.src_region, dstRegion || '')
}

async function handleDataSourceChange() {
  queryForm.school_name = ''
  queryForm.edc_entity_ids = []
  queryForm.entity_type = ''
  queryForm.region = ''
  queryForm.cp = ''
  queryForm.src_region = ''
  queryForm.dst_region = ''
  trafficData.value = []
  total.value = 0
  currentPage.value = 1
  await loadRegionCpOptions()
  await loadSchools('', '')
  if (queryForm.data_source === 'edc') {
    computeEDCFilterOptions(schools.value)
  }
}

// 处理预设时间范围变化
function handleTimeRangeChange(value: TrafficTimeRangeOption) {
  if (value === 'custom') {
    const clearedRange = clearTrafficCustomRange()
    queryForm.start_time = clearedRange?.[0] || ''
    queryForm.end_time = clearedRange?.[1] || ''
    return
  }

  const presetRange = resolvePresetTrafficRange(value)
  queryForm.start_time = presetRange?.[0] || ''
  queryForm.end_time = presetRange?.[1] || ''
  
  // 重置分页到第一页
  currentPage.value = 1
  
  // 只有在存在筛选条件时才查询
  if (hasFilter.value) queryCtl.run((signal) => loadTrafficData(signal), { showCancelMessage: false })
}

// 重置按钮点击事件
function handleReset() {
  // 重置表单
  queryForm.school_name = ''
  queryForm.edc_entity_ids = []
  queryForm.entity_type = ''
  queryForm.region = ''
  queryForm.cp = ''
  queryForm.src_region = ''
  queryForm.dst_region = ''
  queryForm.timeRange = 'last1h'
  
  // 设置默认时间范围为最近1小时
  const resetRange = resolvePresetTrafficRange('last1h')
  queryForm.start_time = resetRange?.[0] || ''
  queryForm.end_time = resetRange?.[1] || ''
  
  // 清空数据并不自动加载
  currentPage.value = 1
  trafficData.value = []
  total.value = 0
}

// 格式化流量数据
function formatTraffic(bytes, withUnit = true) {
  if (bytes === 0) return withUnit ? '0 B' : 0
  
  const k = trafficByteUnitBase.value
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
  // NFA 原始点按 60 秒口径；EDC 原始点是 5 分钟聚合，按 300 秒口径。
  // *8 是将字节转换为比特
  const factor = queryForm.data_source === 'edc' ? 300 : 60
  
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

usePageRefresh(() => {
  if (!hasFilter.value) return
  currentPage.value = 1
  queryCtl.run((signal) => loadTrafficData(signal), { showCancelMessage: false })
})
</script>

<template>
  <div class="page-container">
    <PageHeader title="流速监控" :description="dataSourceDescription" />
    
    <!-- 查询表单 -->
    <FilterPanel>
      <ElForm :model="queryForm" label-width="80px" inline class="filter-form">
        <ElFormItem label="数据源">
          <ElSegmented
            v-model="queryForm.data_source"
            :options="[
              { label: 'NFA', value: 'nfa' },
              { label: 'EDC', value: 'edc' },
            ]"
            @change="handleDataSourceChange"
          />
        </ElFormItem>

        <ElFormItem v-if="queryForm.data_source === 'edc'" label="类型">
          <SearchSelect
            v-model="queryForm.entity_type"
            :options="entityTypes"
            placeholder="选择类型"
            clearable
            class="field-sm"
            @change="handleEntityTypeChange"
          />
        </ElFormItem>

        <ElFormItem v-if="queryForm.data_source !== 'edc'" label="地区">
          <SearchSelect v-model="queryForm.region" :options="regions" placeholder="选择地区" clearable @change="handleRegionChange" class="field-sm" />
        </ElFormItem>
        
        <ElFormItem label="CP">
          <SearchSelect v-model="queryForm.cp" :options="cps" placeholder="选择 CP" clearable @change="handleCPChange" class="field-sm" />
        </ElFormItem>

        <ElFormItem v-if="queryForm.data_source === 'edc'" label="源区域">
          <SearchSelect v-model="queryForm.src_region" :options="srcRegions" placeholder="选择源区域" clearable @change="handleSrcRegionChange" class="field-sm" />
        </ElFormItem>

        <ElFormItem v-if="queryForm.data_source === 'edc'" label="目区域">
          <SearchSelect v-model="queryForm.dst_region" :options="dstRegions" placeholder="选择目区域" clearable @change="handleDstRegionChange" class="field-sm" />
        </ElFormItem>
        
        <ElFormItem :label="entityLabel">
          <SearchSelect
            v-if="queryForm.data_source === 'edc'"
            v-model="queryForm.edc_entity_ids"
            :options="schools"
            label-key="edc_search_label"
            value-key="id"
            :placeholder="`选择${entityLabel}`"
            clearable
            multiple
            collapse-tags
            collapse-tags-tooltip
            class="field-lg"
            @change="queryForm.school_name = ''"
          >
            <template #option="{ option }">
              <span class="edc-option-main">{{ getEDCPrimaryLabel(option as EDCEntityOption) }}</span>
              <span class="edc-option-meta">{{ formatEDCOptionMeta(option as EDCEntityOption) }}</span>
            </template>
          </SearchSelect>
          <SearchSelect
            v-else
            v-model="queryForm.school_name"
            :options="schools"
            label-key="school_name"
            value-key="school_name"
            :placeholder="`选择${entityLabel}`"
            clearable
            class="field-lg"
          />
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
            <UnifiedDateRange
              v-model="customDateRange"
              type="datetimerange"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DD HH:mm:ss"
            />
          </template>
        </ElFormItem>
        

        
        <ElFormItem>
          <QueryActionButton :running="queryCtl.running.value" @trigger="handleQuery" />
          <ElButton @click="handleReset">重置</ElButton>
        </ElFormItem>
      </ElForm>
    </FilterPanel>
    
    <!-- 流量图表 -->
    <SectionCard title="趋势图" v-loading="chartLoading">
      <v-chart
        ref="chartRef"
        class="traffic-chart"
        :option="chartOption"
        :update-options="{ replaceMerge: ['series'] }"
        autoresize
        @legendselectchanged="handleLegendSelectChanged"
      />
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
        <template v-if="queryForm.data_source === 'edc'">
          <ElTableColumn prop="edc_alias" label="EDC别名" min-width="160">
            <template #default="scope">
              {{ scope.row.edc_alias || '—' }}
            </template>
          </ElTableColumn>
          <ElTableColumn prop="edc_name" label="EDC原名" min-width="180">
            <template #default="scope">
              {{ scope.row.edc_name || '—' }}
            </template>
          </ElTableColumn>
        </template>
        <ElTableColumn v-else prop="school_name" :label="entityLabel" />
        <ElTableColumn v-if="queryForm.data_source !== 'edc'" prop="region" label="地区" />
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

.edc-option-meta {
  display: block;
  margin-left: 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.45;
  white-space: normal;
}

.edc-option-main {
  display: block;
  font-weight: 500;
}
</style>
