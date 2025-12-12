<template>
  <div class="settlement-data-tab">
    <!-- 筛选条件区域 -->
    <el-card class="filter-section" shadow="hover">
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
        <el-form-item label="CP" style="min-width: 200px;">
          <el-select v-model="filterForm.cp" placeholder="选择 CP" clearable style="width: 180px;" @change="handleCPChange">
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
        <el-form-item label="服务时间" style="min-width: 400px;">
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
    </el-card>

    <!-- 数据表格区域 -->
    <el-card class="table-section" shadow="hover">
      <div class="table-header">
        <h3>结算数据列表</h3>
        <div style="display:flex; gap:8px;">
          <el-button type="primary" @click="openExportDialog">导出</el-button>
          <el-button v-if="canRecalc" type="warning" @click="onRecalculate">复算</el-button>
        </div>
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
        <el-table-column prop="school_name" label="学校名称" min-width="160" />
        <el-table-column prop="region" label="地区" width="100" />
        <el-table-column prop="cp" label="CP" width="100" />
        <el-table-column prop="service_date" label="服务日期" width="120">
          <template #default="{ row }">{{ row.service_date ? formatDateDisplay(row.service_date) : '-' }}</template>
        </el-table-column>
        <el-table-column label="日95值(Mbps)" width="150">
          <template #default="{ row }">
            {{ row.settlement_value != null ? formatBitRate(convertToBitsPerSecond(row.settlement_value), false) : '0.00' }}
          </template>
        </el-table-column>
        <el-table-column prop="customer_fee" label="客户费率" width="110" />
        <el-table-column prop="customer_bill" label="客户金额" width="110" />
        <el-table-column label="客户费归属" min-width="160">
          <template #default="{ row }">{{ displayEntity(row.customer_fee_owner_id) }}</template>
        </el-table-column>
        <el-table-column prop="network_line_fee" label="线路费率" width="110" />
        <el-table-column prop="network_line_bill" label="线路金额" width="110" />
        <el-table-column label="线路费归属" min-width="160">
          <template #default="{ row }">{{ displayEntity(row.network_line_fee_owner_id) }}</template>
        </el-table-column>
        <el-table-column prop="node_deduction_fee" label="节点通用费率" width="110" />
        <el-table-column prop="node_deduction_bill" label="节点通用金额" width="120" />
        <el-table-column label="节点通用费归属" min-width="160">
          <template #default="{ row }">{{ displayEntity(row.node_deduction_fee_owner_id) }}</template>
        </el-table-column>
        <el-table-column prop="channel_rate" label="渠道费率" width="110" />
        <el-table-column prop="channel_bill" label="渠道金额" width="110" />
        <el-table-column label="渠道费归属" min-width="160">
          <template #default="{ row }">{{ displayUser(row.channel_owner_user_id) }}</template>
        </el-table-column>
        <el-table-column prop="recalculated" label="是否复算" width="100">
          <template #default="{ row }">{{ row.recalculated ? '是' : '否' }}</template>
        </el-table-column>
        <el-table-column prop="last_recalc_time" label="最近复算时间" width="160">
          <template #default="{ row }">{{ row.last_recalc_time ? row.last_recalc_time : '-' }}</template>
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
    </el-card>
    <!-- 统一导出弹窗 -->
    <el-dialog v-model="exportDialogVisible" title="导出设置" width="720px">
      <div style="display:flex; gap:24px; align-items:flex-start;">
        <div style="flex:1;">
          <div style="font-weight:600; margin-bottom:8px;">选择字段</div>
          <el-checkbox-group v-model="exportForm.selectedFields">
            <div style="margin-bottom:6px;">基础字段</div>
            <el-checkbox v-for="f in baseFields" :key="f.key" :label="f.key">{{ f.label }}</el-checkbox>
            <el-divider style="margin:10px 0" />
            <div style="margin-bottom:6px;">流量/金额字段</div>
            <el-checkbox v-for="f in numericFields" :key="f.key" :label="f.key">{{ f.label }}</el-checkbox>
            <el-divider style="margin:10px 0" />
            <div style="margin-bottom:6px;">归属/其它</div>
            <el-checkbox v-for="f in otherFields" :key="f.key" :label="f.key">{{ f.label }}</el-checkbox>
          </el-checkbox-group>
        </div>
        <div style="width:220px;">
          <div style="font-weight:600; margin-bottom:8px;">选项</div>
          <el-checkbox v-model="exportForm.monthlyAvg" :disabled="monthlyAvgDisabled">按月平均</el-checkbox>
          <div style="color: var(--text-muted); margin-top:8px; font-size:12px;">仅对已勾选的流量/金额字段生效</div>
        </div>
      </div>
      <template #footer>
        <el-button @click="exportDialogVisible=false">取消</el-button>
        <el-button type="primary" @click="doExport">导出</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import api from '../../api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useTasksStore } from '@/stores/tasks'
import type { SettlementListResponse } from '../../types/settlement'
import type { School, PaginationParams, BusinessEntity } from '../../types/api'

// 学校、地区和运营商数据

const schools = ref<School[]>([])
const regions = ref<string[]>([])
const cps = ref<string[]>([])

// 筛选表单
interface FilterForm {
  school_id: string;
  region: string;
  cp: string;
  start_service_date: string;
  end_service_date: string;
  page: number;
  page_size: number;
}

const filterForm = reactive<FilterForm>({
  school_id: '',
  region: '',
  cp: '',
  start_service_date: '',
  end_service_date: '',
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

// 获取基础数据
const fetchBaseData = async () => {
  try {
    // 直接加载 v2 学校（已按用户权限过滤），后派生地区/运营商
    await loadSchools()
    computeRegionCpOptions()
  } catch (error) {
    console.error('获取基础数据失败', error)
    ElMessage.error('获取基础数据失败')
  }
}

// 加载学校数据
const loadSchools = async (region: string = '', cp: string = ''): Promise<number> => {
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
    
    const response = await (api as any).v2.getSchools(params) as any
    console.log('学校列表原始响应:', response)
    const items: School[] = Array.isArray(response)
      ? response
      : Array.isArray(response?.items)
        ? response.items
        : []
    // 过滤掉异常项
    schools.value = items.filter((s: any) => s && s.school_id && s.school_name)
    console.log('学校列表设置为:', schools.value)
    const total: number = typeof response?.total === 'number'
      ? response.total
      : Array.isArray(items)
        ? items.length
        : 0
    // 刷新地区/运营商选项
    computeRegionCpOptions()
    return total
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
    loadSchools(region, filterForm.cp).then(() => computeRegionCpOptions())
  } else {
    loadSchools('', filterForm.cp).then(() => computeRegionCpOptions())
  }
  // 当地区变化时自动刷新数据
  fetchData()
}

// 处理运营商选择变化
const handleCPChange = (cp: string): void => {
  console.log('运营商选择变化:', cp)
  // 当运营商变化时，重新加载学校列表
  if (cp) {
    loadSchools(filterForm.region, cp).then(() => computeRegionCpOptions())
  } else {
    loadSchools(filterForm.region, '').then(() => computeRegionCpOptions())
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
    filterForm.start_service_date = val[0]
    filterForm.end_service_date = val[1]
    console.log('设置日期范围:', val[0], '至', val[1])
  } else {
    filterForm.start_service_date = ''
    filterForm.end_service_date = ''
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
    // 新接口使用 page/page_size 与服务时间
    const params: { 
      region?: string;
      cp?: string;
      school_name?: string;
      start_service_date?: string;
      end_service_date?: string;
      page?: number;
      page_size?: number;
    } = {
      page: currentPage.value,
      page_size: pageSize.value,
      start_service_date: filterForm.start_service_date,
      end_service_date: filterForm.end_service_date,
    }
    
    console.log('分页参数:', { 页码: currentPage.value, 每页条数: pageSize.value, offset: (currentPage.value - 1) * pageSize.value })
    
    // 添加可选参数
    if (filterForm.school_id) {
      const s = (schools.value || []).find(x => x.school_id === filterForm.school_id)
      if (s && s.school_name) params.school_name = s.school_name
    }
    
    if (filterForm.region) {
      params.region = filterForm.region
    }
    
    if (filterForm.cp) {
      params.cp = filterForm.cp
    }
    
    console.log('最终请求参数:', params)
    
    // 发送请求并解析已解包的数据
    const response = await (api as any).settlementData.list(params) as any
    console.log('结算数据响应:', response)
    if (Array.isArray(response)) {
      settlementData.value = { items: response, total: response.length }
    } else if (response && typeof response === 'object') {
      if (Array.isArray((response as any).items)) {
        settlementData.value = { items: (response as any).items, total: Number((response as any).total) || (response as any).items.length }
      } else {
        settlementData.value = { items: [], total: 0 }
      }
    } else {
      settlementData.value = { items: [], total: 0 }
    }
    // 加载映射用于归属显示
    if (!entityMap.value || Object.keys(entityMap.value).length === 0) {
      await loadEntityMap()
    }
    await loadUsersForItems()
    
    // 检查数据结构
    if (settlementData.value.items && Array.isArray(settlementData.value.items)) {
      console.log('结算数据项目数量:', settlementData.value.items.length)
      if (settlementData.value.items.length > 0) {
        console.log('第一个数据项:', JSON.stringify(settlementData.value.items[0]))
      }
    } else {
      console.error('数据结构不符合预期:', settlementData.value)
      // 如果没有数据，显示提示
      if (!Array.isArray(settlementData.value.items) || settlementData.value.items.length === 0) {
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
  filterForm.start_service_date = ''
  filterForm.end_service_date = ''
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

function csvEscape(v: any): string {
  let s = v == null ? '' : String(v)
  if (s.includes('"')) s = s.replace(/"/g, '""')
  if (s.search(/[",\n]/) >= 0) s = `"${s}` + `"`
  return s
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

// 统一导出：弹窗与逻辑
const exportDialogVisible = ref(false)
const DEFAULT_FIELDS = ['school_name','region','cp','service_date','daily_95_mbps']
const exportForm = reactive<{ selectedFields: string[]; monthlyAvg: boolean }>({ selectedFields: [...DEFAULT_FIELDS], monthlyAvg: false })

type FieldType = 'base' | 'traffic' | 'money'
interface FieldDef { key: string; label: string; type: FieldType; getter?: (row: any) => any }

const allFieldDefs: FieldDef[] = [
  { key: 'school_name', label: '学校名称', type: 'base', getter: (r:any)=> r?.school_name ?? '' },
  { key: 'region', label: '地区', type: 'base', getter: (r:any)=> r?.region ?? '' },
  { key: 'cp', label: 'CP', type: 'base', getter: (r:any)=> r?.cp ?? '' },
  { key: 'service_date', label: '服务日期', type: 'base', getter: (r:any)=> r?.service_date ? formatDateDisplay(String(r.service_date)) : '' },
  { key: 'daily_95_mbps', label: '日95(Mbps)', type: 'traffic', getter: (r:any)=> (convertToBitsPerSecond(Number(r?.settlement_value ?? 0)) / 1_000_000).toFixed(2) },
  { key: 'customer_fee', label: '客户费率', type: 'base', getter: (r:any)=> r?.customer_fee },
  { key: 'customer_bill', label: '客户金额', type: 'money', getter: (r:any)=> r?.customer_bill },
  { key: 'customer_fee_owner_name', label: '客户费归属', type: 'base', getter: (r:any)=> displayEntity(r?.customer_fee_owner_id) },
  { key: 'network_line_fee', label: '线路费率', type: 'base', getter: (r:any)=> r?.network_line_fee },
  { key: 'network_line_bill', label: '线路金额', type: 'money', getter: (r:any)=> r?.network_line_bill },
  { key: 'network_line_fee_owner_name', label: '线路费归属', type: 'base', getter: (r:any)=> displayEntity(r?.network_line_fee_owner_id) },
  { key: 'node_deduction_fee', label: '节点通用费率', type: 'base', getter: (r:any)=> r?.node_deduction_fee },
  { key: 'node_deduction_bill', label: '节点通用金额', type: 'money', getter: (r:any)=> r?.node_deduction_bill },
  { key: 'node_deduction_fee_owner_name', label: '节点通用费归属', type: 'base', getter: (r:any)=> displayEntity(r?.node_deduction_fee_owner_id) },
  { key: 'channel_rate', label: '渠道费率', type: 'base', getter: (r:any)=> r?.channel_rate },
  { key: 'channel_bill', label: '渠道金额', type: 'money', getter: (r:any)=> r?.channel_bill },
  { key: 'channel_owner_name', label: '渠道费归属', type: 'base', getter: (r:any)=> displayUser(r?.channel_owner_user_id) },
  { key: 'recalculated', label: '是否复算', type: 'base', getter: (r:any)=> r?.recalculated ? '是' : '否' },
  { key: 'last_recalc_time', label: '最近复算时间', type: 'base', getter: (r:any)=> r?.last_recalc_time ?? '' },
]

const baseFields = computed(() => allFieldDefs.filter(f => ['school_name','region','cp','service_date','customer_fee','network_line_fee','node_deduction_fee','channel_rate','recalculated','last_recalc_time'].includes(f.key)))
const numericFields = computed(() => allFieldDefs.filter(f => f.type === 'traffic' || f.type === 'money'))
const otherFields = computed(() => allFieldDefs.filter(f => ['customer_fee_owner_name','network_line_fee_owner_name','node_deduction_fee_owner_name','channel_owner_name'].includes(f.key)))

const monthlyAvgDisabled = computed(() => {
  const selected = new Set(exportForm.selectedFields)
  return !numericFields.value.some(f => selected.has(f.key))
})

function openExportDialog() {
  exportForm.selectedFields = [...DEFAULT_FIELDS]
  exportForm.monthlyAvg = false
  exportDialogVisible.value = true
}

async function fetchAllDataForExport(onProgress?: (p: number, meta?: { processed: number; total?: number }) => void): Promise<any[]> {
  const params: any = {
    page: 1,
    page_size: 1000,
    start_service_date: filterForm.start_service_date,
    end_service_date: filterForm.end_service_date,
  }
  if (filterForm.region) params.region = filterForm.region
  if (filterForm.cp) params.cp = filterForm.cp
  if (filterForm.school_id) {
    const s = (schools.value || []).find(x => x.school_id === filterForm.school_id)
    if (s && s.school_name) params.school_name = s.school_name
  }
  const all: any[] = []
  let total = 0
  while (true) {
    const res: any = await (api as any).settlementData.list(params)
    let items: any[] = []
    if (Array.isArray(res)) { items = res; total = total || res.length } else if (res && Array.isArray(res.items)) { items = res.items; total = Number(res.total || total || 0) }
    all.push(...items)
    if (typeof total === 'number' && total > 0 && onProgress) {
      const processed = Math.min(all.length, total)
      onProgress(Math.max(0, Math.min(1, processed / total)), { processed, total })
    }
    if (items.length < params.page_size) break
    if (total > 0 && (params.page * params.page_size) >= total) break
    params.page += 1
  }
  return all
}

function monthKey(d: string): string {
  if (!d) return ''
  const s = String(d)
  const dateStr = s.includes('T') ? s.split('T')[0] : (s.includes(' ') ? s.split(' ')[0] : s)
  return dateStr.slice(0, 7)
}

async function doExport() {
  let taskId: string | null = null
  try {
    if (exportForm.selectedFields.length === 0) { ElMessage.warning('请至少选择一个字段'); return }
    if (!filterForm.start_service_date || !filterForm.end_service_date) {
      try { await ElMessageBox.confirm('未选择服务时间范围，将导出全量范围，可能耗时较长。是否继续？', '确认导出', { type: 'warning' }) } catch { return }
    }
    const tasks = useTasksStore()
    taskId = `export:${Date.now()}`
    tasks.start({ id: taskId, type: 'export', title: '结算数据导出', status: 'running', progress: 0 })
    const data = await fetchAllDataForExport((p, meta) => { tasks.update(taskId, { progress: p, status: 'running', processed: meta?.processed ?? null, total: meta?.total ?? null }) })
    const selectedDefs = allFieldDefs.filter(f => exportForm.selectedFields.includes(f.key))
    let header: string[] = []

    let rows: string[] = []

    if (exportForm.monthlyAvg && !monthlyAvgDisabled.value) {
      const metricDefs = selectedDefs.filter(f => f.type === 'traffic' || f.type === 'money')
      const selectedTrafficKeys = new Set(metricDefs.filter(f => f.type === 'traffic').map(f => f.key))
      const stripLabel = (s: string) => String(s).replace(/\(.*?\)/g, '').trim()
      const monthOf = (d: string): number => {
        if (!d) return 0
        const s = String(d)
        const dateStr = s.includes('T') ? s.split('T')[0] : (s.includes(' ') ? s.split(' ')[0] : s)
        const mm = Number(dateStr.slice(5, 7))
        return Number.isFinite(mm) && mm >= 1 && mm <= 12 ? mm : 0
      }
      type Agg = { base: any; traf: Record<string, { sum: number[]; cnt: number[] }>; money: Record<string, number[]> }
      const group = new Map<string, Agg>()
      for (const r of data) {
        const m = monthOf(r?.service_date || '')
        if (m <= 0) continue
        const gk = `${r?.region || ''}__${r?.school_name || ''}__${r?.cp || ''}`
        if (!group.has(gk)) {
          const g: Agg = { base: { school_name: r?.school_name, region: r?.region, cp: r?.cp }, traf: {}, money: {} }
          for (const def of metricDefs) {
            if (def.type === 'traffic') g.traf[def.key] = { sum: Array(12).fill(0), cnt: Array(12).fill(0) }
            else g.money[def.key] = Array(12).fill(0)
          }
          group.set(gk, g)
        }
        const g = group.get(gk)!
        if (selectedTrafficKeys.has('daily_95_mbps')) {
          const v = Number((convertToBitsPerSecond(Number(r?.settlement_value ?? 0)) / 1_000_000).toFixed(9))
          if (!Number.isNaN(v)) { g.traf['daily_95_mbps'].sum[m - 1] += v; g.traf['daily_95_mbps'].cnt[m - 1] += 1 }
        }
        for (const def of metricDefs) {
          if (def.type !== 'money') continue
          const k = def.key
          const v = Number(r?.[k] ?? 0)
          if (!Number.isNaN(v)) g.money[k][m - 1] += v
        }
      }
      header = ['学校名称', '地区', 'CP']
      for (const def of metricDefs) {
        const name = stripLabel(def.label)
        for (let i = 1; i <= 12; i++) header.push(`${i}月${name}`)
      }
      const lines: string[] = []
      for (const [, g] of group) {
        const row: string[] = [csvEscape(g.base.school_name), csvEscape(g.base.region), csvEscape(g.base.cp)]
        for (const def of metricDefs) {
          if (def.type === 'traffic') {
            const s = g.traf[def.key].sum
            const c = g.traf[def.key].cnt
            for (let i = 0; i < 12; i++) {
              const avg = c[i] > 0 ? (s[i] / c[i]) : ''
              row.push(avg === '' ? '' : String(Number(avg).toFixed(2)))
            }
          } else {
            const arr = g.money[def.key]
            for (let i = 0; i < 12; i++) {
              const val = arr[i]
              row.push(val == null ? '' : String(Number(val).toFixed(2)))
            }
          }
        }
        lines.push(row.join(','))
      }
      rows = lines
    } else {
      header = selectedDefs.map(def => def.label)
      const lines: string[] = []
      for (const r of data) {
        const row: string[] = []
        for (const def of selectedDefs) {
          const val = def.getter ? def.getter(r) : r?.[def.key]
          row.push(csvEscape(val))
        }
        lines.push(row.join(','))
      }
      rows = lines
    }

    const content = ['\uFEFF' + header.join(','), ...rows].join('\n')
    const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
    const filename = exportForm.monthlyAvg ? 'settlement_export_monthly.csv' : 'settlement_export.csv'
    const url = URL.createObjectURL(blob)
    tasks.complete(taskId, url)
    // 立即触发下载
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    ElMessage.success('导出成功')
    exportDialogVisible.value = false
  } catch (e:any) {
    try { const tasks = useTasksStore(); if (taskId) tasks.fail(taskId, e?.message) } catch {}
    ElMessage.error(e?.response?.data?.message || e?.message || '导出失败')
  }
}

const auth = useAuthStore()
const canRecalc = computed(() => auth.hasPermission('settlement.data.recalculate'))

const onRecalculate = async () => {
  try {
    await ElMessageBox.confirm('将按筛选条件与服务时间范围触发复算，并覆盖既有数据。是否继续？', '确认复算', { type: 'warning', confirmButtonText: '复算', cancelButtonText: '取消' })
  } catch { return }
  if (!dateRange.value) { ElMessage.warning('请先选择服务时间范围'); return }
  try {
    const body: any = {
      start_service_date: dateRange.value[0],
      end_service_date: dateRange.value[1],
    }
    if (filterForm.region) body.region = filterForm.region
    if (filterForm.cp) body.cp = filterForm.cp
    if (filterForm.school_id) {
      const s = (schools.value || []).find(x => x.school_id === filterForm.school_id)
      if (s && s.school_name) body.school_name = s.school_name
    }
    // 1) 触发后端复算，拿到任务ID
    const taskNumericId: number = await (api as any).settlementData.recalculate(body)
    const taskId = `settlement:${taskNumericId}`
    const tasks = useTasksStore()
    // 2) 预估总量：用 v2 日95明细列表的 total 作为复算任务的总工作量
    let estTotal: number | null = null
    try {
      const params: any = {
        start_date: dateRange.value[0],
        end_date: dateRange.value[1],
        limit: 1,
        offset: 0,
      }
      if (filterForm.region) params.region = filterForm.region
      if (filterForm.cp) params.cp = filterForm.cp
      if (filterForm.school_id) {
        const s = (schools.value || []).find(x => x.school_id === filterForm.school_id)
        if (s && s.school_name) params.school_name = s.school_name
      }
      const res: any = await (api as any).v2.settlement.getDailySettlementDetails(params)
      if (res && typeof res === 'object' && 'total' in res) estTotal = Number((res as any).total) || null
    } catch {}
    // 3) 在全局浮层启动任务展示
    tasks.upsertSettlementTask({ id: taskNumericId, status: 'running', processed_count: 0, total_count: estTotal ?? undefined })
    ElMessage.success('已触发复算，后台执行中')
    // 4) 轮询该任务直至完成/失败，期间更新进度/ETA
    let stopped = false
    const stop = () => { stopped = true }
    const poll = async () => {
      if (stopped) return
      try {
        const t: any = await (api as any).settlement.getTaskById(taskNumericId)
        if (t && typeof t === 'object') {
          const processed = Number((t as any).processed_count ?? 0) || 0
          const total = Number((t as any).total_count ?? (estTotal ?? 0)) || (estTotal ?? undefined)
          tasks.upsertSettlementTask({ id: taskNumericId, status: (t as any).status as any, processed_count: processed, total_count: total as any })
          if ((t as any).status === 'success' || (t as any).status === 'failed') {
            stop()
            // 完成后刷新当前列表
            fetchData()
            return
          }
        }
      } catch {}
      setTimeout(poll, 2000)
    }
    setTimeout(poll, 1500)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '复算失败')
  }
}

// 组件挂载时获取数据
onMounted(() => {
  fetchBaseData()
  fetchData()
})

// 业务对象/系统用户映射，用于归属显示
const entityMap = ref<Record<number, string>>({})
const userMap = ref<Record<number, { id:number; alias?:string; display_name?:string; username:string }>>({})

const loadEntityMap = async () => {
  try {
    const pageSize = 1000
    let page = 1
    const map: Record<number, string> = {}
    while (true) {
      const res = await (api as any).settlementEntities.list({ page, page_size: pageSize })
      const list = (res?.items || []) as BusinessEntity[]
      for (const e of list) { if (e && typeof e.id === 'number') map[e.id] = e.entity_name }
      const total = Number(res?.total || 0)
      if (page * pageSize >= total || list.length === 0) break
      page += 1
    }
    entityMap.value = map
  } catch {}
}

const loadUsersForItems = async () => {
  const ids = new Set<number>()
  for (const r of settlementData.value.items as any[]) {
    if (r?.channel_owner_user_id != null) { const n = Number(r.channel_owner_user_id); if (!Number.isNaN(n) && n>0) ids.add(n) }
  }
  if (ids.size === 0) { userMap.value = {}; return }
  try {
    const res: any = await (api as any).system.users.list({ ids: Array.from(ids).join(',') })
    const list: any[] = Array.isArray(res?.items) ? res.items : []
    const m: Record<number, { id:number; alias?:string; display_name?:string; username:string }> = {}
    for (const u of list) { if (u && typeof u.id === 'number') m[u.id] = { id:u.id, alias:u.alias, display_name:u.display_name, username:u.username } }
    userMap.value = m
  } catch { userMap.value = {} }
}

function displayEntity(id?: number | null): string {
  if (id == null) return '-'
  const key = Number(id)
  return entityMap.value[key] || `#${key}`
}
function displayUser(id?: number | null): string {
  if (!id) return '-'
  const key = Number(id)
  const u = userMap.value[key]
  if (!u) return `#${key}`
  const alias = (u.alias && String(u.alias).trim()) ? String(u.alias).trim() : ''
  const dn = (u.display_name && String(u.display_name).trim()) ? String(u.display_name).trim() : ''
  const un = (u.username && String(u.username).trim()) ? String(u.username).trim() : ''
  return alias || dn || un || `用户#${key}`
}
</script>

<style scoped>
.settlement-data-tab {
  padding: 10px;
}

.filter-section {
  margin-bottom: 20px;
}

/* .table-section 使用全局 .el-card 玻璃化样式，无需局部背景与阴影 */

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
