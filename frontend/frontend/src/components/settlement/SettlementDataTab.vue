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
        <el-form-item label="费用归属" style="min-width: 300px;">
          <el-select v-model="ownerSelect" placeholder="选择费用归属" clearable style="width: 250px;" @change="handleOwnerChange">
            <el-option
              v-for="opt in ownerOptions"
              :key="opt.id"
              :label="opt.label"
              :value="String(opt.id)"
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
        <el-table-column prop="customer_bill" label="客户金额" width="110">
          <template #default="{ row }">
            <el-tooltip placement="top">
              <template #content>
                <pre style="white-space: pre-wrap; max-width: 420px; font-size:12px; line-height:1.3;">{{ amountDetail(row, 'customer_fee', 'customer_bill', '客户费率') }}</pre>
              </template>
              <span>{{ row.customer_bill != null ? row.customer_bill : '-' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="客户费归属" min-width="160">
          <template #default="{ row }">{{ displayUser(row.customer_fee_owner_id) }}</template>
        </el-table-column>
        <el-table-column prop="network_line_fee" label="线路费率" width="110" />
        <el-table-column prop="network_line_bill" label="线路金额" width="110">
          <template #default="{ row }">
            <el-tooltip placement="top">
              <template #content>
                <pre style="white-space: pre-wrap; max-width: 420px; font-size:12px; line-height:1.3;">{{ amountDetail(row, 'network_line_fee', 'network_line_bill', '线路费率') }}</pre>
              </template>
              <span>{{ row.network_line_bill != null ? row.network_line_bill : '-' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="线路费归属" min-width="160">
          <template #default="{ row }">{{ displayUser(row.network_line_fee_owner_id) }}</template>
        </el-table-column>
        <el-table-column prop="node_deduction_fee" label="节点通用费率" width="110" />
        <el-table-column prop="node_deduction_bill" label="节点通用金额" width="120">
          <template #default="{ row }">
            <el-tooltip placement="top">
              <template #content>
                <pre style="white-space: pre-wrap; max-width: 420px; font-size:12px; line-height:1.3;">{{ amountDetail(row, 'node_deduction_fee', 'node_deduction_bill', '节点通用费率') }}</pre>
              </template>
              <span>{{ row.node_deduction_bill != null ? row.node_deduction_bill : '-' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="节点通用费归属" min-width="160">
          <template #default="{ row }">{{ displayUser(row.node_deduction_fee_owner_id) }}</template>
        </el-table-column>
        <el-table-column prop="channel_rate" label="渠道费率" width="110" />
        <el-table-column prop="channel_bill" label="渠道金额" width="110">
          <template #default="{ row }">
            <el-tooltip placement="top">
              <template #content>
                <pre style="white-space: pre-wrap; max-width: 420px; font-size:12px; line-height:1.3;">{{ amountDetail(row, 'channel_rate', 'channel_bill', '渠道费率') }}</pre>
              </template>
              <span>{{ row.channel_bill != null ? row.channel_bill : '-' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
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
          <el-checkbox v-model="exportForm.groupBySchoolCp">按学校+CP聚合</el-checkbox>
          <div style="color: var(--text-muted); margin:6px 0 10px; font-size:12px;">金额字段将累加，流量字段取平均</div>
          <el-checkbox v-model="exportForm.monthlyAvg" :disabled="monthlyAvgDisabled">按月聚合</el-checkbox>
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
import type { School, PaginationParams } from '../../types/api'

// 学校、地区和运营商数据

const schools = ref<School[]>([])
const regions = ref<string[]>([])
const cps = ref<string[]>([])
// 费用归属（统一：仅用户）下拉
type OwnerOption = { id: number; name: string; label: string }
const ownerOptions = ref<OwnerOption[]>([])
const ownerSelect = ref<string | null>(null)

// 筛选表单
interface FilterForm {
  school_id: string;
  region: string;
  cp: string;
  start_service_date: string;
  end_service_date: string;
  channel_owner_user_id: number | null;
  page: number;
  page_size: number;
}

function daysInMonthFrom(dateStr?: string | null): number {
  try {
    if (!dateStr) return 30
    const d = new Date(String(dateStr))
    if (!isNaN(d.getTime())) {
      const y = d.getFullYear()
      const m = d.getMonth()
      return new Date(y, m + 1, 0).getDate()
    }
    const m2 = String(dateStr).match(/^(\d{4})-(\d{2})-(\d{2})/)
    if (m2) {
      const y = Number(m2[1])
      const m = Number(m2[2]) - 1
      return new Date(y, m + 1, 0).getDate()
    }
    return 30
  } catch { return 30 }
}

function toFixedNum(n: number, digits = 2): string {
  try { return Number(n).toFixed(digits) } catch { return String(n) }
}

function amountDetail(row: any, rateField: string, billField: string, rateLabel: string): string {
  try {
    const sv = Number(row?.settlement_value ?? 0)
    const bps = (sv * 8) / 60
    const mbps = bps / 1_000_000
    const gbps = bps / 1_000_000_000
    const rateRaw = row ? row[rateField] : null
    const rate = rateRaw != null ? Number(rateRaw) : NaN
    const days = daysInMonthFrom(row?.service_date)
    const calc = Number.isFinite(rate) ? (gbps * rate) / days : NaN
    const bill = row ? row[billField] : null
    const lines: string[] = []
    lines.push(`学校：${row?.school_name ?? ''}`)
    lines.push(`日期：${row?.service_date ?? ''}`)
    lines.push(`日95原值：${sv}`)
    lines.push(`换算：bits/s = 日95 * 8 / 60 = ${(sv*8).toFixed(2)} / 60 = ${toFixedNum(bps, 2)}`)
    lines.push(`换算：Mbps = bits/s / 1e6 = ${toFixedNum(bps,2)} / 1e6 = ${toFixedNum(mbps, 2)}`)
    lines.push(`换算：Gbps = bits/s / 1e9 = ${toFixedNum(bps,2)} / 1e9 = ${toFixedNum(gbps, 6)}`)
    let extra: string[] = []
    if (rateField === 'customer_fee') {
      const rid = Number(row?.discount_rule_id || 0)
      const yi = Number(row?.service_year_index || 0)
      if (rid > 0 && yi > 0) {
        const det = discountRuleDetailMap.value[rid]
        const rule = det?.rule
        const items = Array.isArray(det?.items) ? det!.items : []
        const used = items.find((it: any) => yi >= Number(it.from_year) && (it.to_year == null || yi <= Number(it.to_year)))
        if (rule) {
          const scope = `${rule.scope_type || 'global'}${rule.scope_type === 'global' ? '' : `/${rule.scope_key ?? '-'}`}`
          extra.push(`折损规则：#${rid} ${rule.name || ''}（范围：${scope}）`)
        } else {
          extra.push(`折损规则：#${rid}`)
        }
        if (used && Number.isFinite(Number(used.discount_rate))) {
          const ratio = Number(used.discount_rate)
          const baseFee = Number.isFinite(rate) ? (rate as number) / ratio : NaN
          extra.push(`生效条目：第${yi}年，区间 ${used.from_year}-${used.to_year ?? '∞'}，比例=${Math.round(ratio*100)}%`)
          if (Number.isFinite(baseFee)) {
            extra.push(`原始费率（未折损）：${toFixedNum(baseFee, 4)}，折后费率：${rate}`)
          }
        } else if (yi > 0) {
          extra.push(`生效条目：第${yi}年（未匹配到条目）`)
        }
      }
    }
    lines.push(`${rateLabel}：${Number.isFinite(rate) ? rate : '-'}`)
    if (extra.length > 0) lines.push(...extra)
    lines.push(`当月天数：${days}`)
    if (Number.isFinite(rate)) {
      lines.push(`公式：金额 = Gbps * 费率 / 当月天数 = ${toFixedNum(gbps,6)} * ${rate} / ${days} = ${toFixedNum(calc, 2)}`)
    } else {
      lines.push('公式：缺少费率，金额不可计算')
    }
    if (bill != null) {
      lines.push(`金额(入库)：${bill}`)
    }
    return lines.join('\n')
  } catch { return '' }
}
const filterForm = reactive<FilterForm>({
  school_id: '',
  region: '',
  cp: '',
  start_service_date: '',
  end_service_date: '',
  channel_owner_user_id: null,
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

const discountRuleDetailMap = ref<Record<number, { rule: any; items: any[] }>>({})

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
    // 加载费用归属下拉
    await loadOwnerOptions()
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
  loadOwnerOptions()
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
  loadOwnerOptions()
}

// 处理学校选择变化
const handleSchoolChange = (schoolId: string): void => {
  console.log('学校选择变化:', schoolId)
  // 当学校变化时，可以在这里添加额外的逻辑
  // 例如，根据学校ID获取更多详细信息等
  // 当学校变化时自动刷新数据
  fetchData()
  loadOwnerOptions()
}

// 处理费用归属选择变化（仅用户）：仅设置 channel_owner_user_id，用后端统一 OR 过滤四个归属字段
const handleOwnerChange = (val: string | null): void => {
  console.log('费用归属选择变化:', val)
  // 重置两个字段
  filterForm.channel_owner_user_id = null
  if (val && typeof val === 'string') {
    const id = Number(val)
    if (Number.isFinite(id) && id > 0) {
      filterForm.channel_owner_user_id = id
    }
  }
  fetchData()
}

// 加载费用归属（用户）：统一从后端 /owner-subjects 获取，并只保留 type==='user'
const loadOwnerOptions = async () => {
  try {
    const params: any = {
      region: filterForm.region || undefined,
      cp: filterForm.cp || undefined,
      start_service_date: filterForm.start_service_date || undefined,
      end_service_date: filterForm.end_service_date || undefined,
    }
    const items: any[] = await (api as any).settlementData.ownerSubjects(params)
    const list = (Array.isArray(items) ? items : [])
      .filter((it: any) => it && String(it.type) === 'user')
      .map((it: any) => ({ id: Number(it.id), name: String(it.label || `用户#${it.id}`), label: String(it.label || `用户#${it.id}`) }))
      .filter((u: OwnerOption) => Number.isFinite(u.id) && !!u.label)
    ownerOptions.value = list.sort((a, b) => String(a.label).localeCompare(String(b.label)))
  } catch (e) {
    console.warn('加载费用归属下拉失败', e)
    ownerOptions.value = []
  }
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
    loadOwnerOptions()
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
      channel_owner_user_id?: number;
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
    if (filterForm.channel_owner_user_id != null) params.channel_owner_user_id = filterForm.channel_owner_user_id
    
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
    // 加载用户映射用于归属显示
    await loadUsersForItems()
    try {
      const ids = new Set<number>()
      for (const r of settlementData.value.items || []) {
        const id = Number((r as any)?.discount_rule_id || 0)
        if (Number.isFinite(id) && id > 0) ids.add(id)
      }
      await Promise.all(Array.from(ids).map(async (id) => {
        if (!discountRuleDetailMap.value[id]) {
          try { discountRuleDetailMap.value[id] = await (api as any).settlementRates.discountRules.get(id) } catch {}
        }
      }))
    } catch {}
    
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
  filterForm.channel_owner_user_id = null
  ownerSelect.value = null
  dateRange.value = null
  currentPage.value = 1
  pageSize.value = 10
  fetchData()
  loadOwnerOptions()
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
const exportForm = reactive<{ selectedFields: string[]; monthlyAvg: boolean; groupBySchoolCp: boolean }>({ selectedFields: [...DEFAULT_FIELDS], monthlyAvg: false, groupBySchoolCp: false })

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
  { key: 'customer_fee_owner_name', label: '客户费归属', type: 'base', getter: (r:any)=> displayUser(r?.customer_fee_owner_id) },
  { key: 'network_line_fee', label: '线路费率', type: 'base', getter: (r:any)=> r?.network_line_fee },
  { key: 'network_line_bill', label: '线路金额', type: 'money', getter: (r:any)=> r?.network_line_bill },
  { key: 'network_line_fee_owner_name', label: '线路费归属', type: 'base', getter: (r:any)=> displayUser(r?.network_line_fee_owner_id) },
  { key: 'node_deduction_fee', label: '节点通用费率', type: 'base', getter: (r:any)=> r?.node_deduction_fee },
  { key: 'node_deduction_bill', label: '节点通用金额', type: 'money', getter: (r:any)=> r?.node_deduction_bill },
  { key: 'node_deduction_fee_owner_name', label: '节点通用费归属', type: 'base', getter: (r:any)=> displayUser(r?.node_deduction_fee_owner_id) },
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
  exportForm.groupBySchoolCp = false
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
  if (filterForm.channel_owner_user_id != null) params.channel_owner_user_id = filterForm.channel_owner_user_id
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
    const selectedDefs = exportForm.selectedFields
      .map(k => allFieldDefs.find(f => f.key === k))
      .filter((x): x is FieldDef => !!x)
    let header: string[] = []

    let rows: string[] = []

    if (exportForm.monthlyAvg && !monthlyAvgDisabled.value) {
      // 校验必须选择时间范围
      if (!filterForm.start_service_date || !filterForm.end_service_date) {
        ElMessage.warning('请先选择服务时间范围，再进行按月聚合导出')
        return
      }
      const metricDefs = selectedDefs.filter(f => f.type === 'traffic' || f.type === 'money')
      const selectedTrafficKeys = new Set(metricDefs.filter(f => f.type === 'traffic').map(f => f.key))
      const stripLabel = (s: string) => String(s).replace(/\(.*?\)/g, '').trim()

      // 计算选择范围内的月份列表（跨年支持），键为 YYYY-MM
      const parseDateOnly = (s: string): Date => {
        const ss = String(s)
        const dstr = ss.includes('T') ? ss.split('T')[0] : (ss.includes(' ') ? ss.split(' ')[0] : ss)
        const [y, m, d] = dstr.split('-').map(n => Number(n))
        return new Date(y, (m || 1) - 1, d || 1)
      }
      const startD = parseDateOnly(filterForm.start_service_date)
      const endD = parseDateOnly(filterForm.end_service_date)
      const months: { key: string; label: string; y: number; m: number }[] = []
      const monthKey2 = (y: number, m: number) => `${y}-${String(m).padStart(2, '0')}`
      // 从起始月到结束月逐月推进
      let cy = startD.getFullYear(), cm = startD.getMonth() + 1
      const ey = endD.getFullYear(), em = endD.getMonth() + 1
      while (cy < ey || (cy === ey && cm <= em)) {
        const key = monthKey2(cy, cm)
        months.push({ key, label: key, y: cy, m: cm })
        cm++
        if (cm > 12) { cm = 1; cy++ }
      }
      const monthIndex = new Map<string, number>()
      months.forEach((it, idx) => monthIndex.set(it.key, idx))

      type Agg = { base: any; traf: Record<string, { sum: number[]; cnt: number[] }>; money: Record<string, number[]> }
      const group = new Map<string, Agg>()
      for (const r of data) {
        const s = String(r?.service_date || '')
        const dateStr = s.includes('T') ? s.split('T')[0] : (s.includes(' ') ? s.split(' ')[0] : s)
        if (!dateStr) continue
        const ym = dateStr.slice(0, 7)
        const idx = monthIndex.get(ym)
        if (idx == null) continue // 不在选择范围内
        const gk = exportForm.groupBySchoolCp
          ? `${r?.school_name || ''}__${r?.cp || ''}`
          : `${r?.region || ''}__${r?.school_name || ''}__${r?.cp || ''}`
        if (!group.has(gk)) {
          const g: Agg = { base: { school_name: r?.school_name, region: r?.region, cp: r?.cp }, traf: {}, money: {} }
          // 基础字段（聚合允许的）按勾选顺序填充一次
          const allowedBase = new Set(['school_name','region','cp','customer_fee_owner_name','network_line_fee_owner_name','node_deduction_fee_owner_name','channel_owner_name'])
          const baseSelectedKeysAll = exportForm.selectedFields.filter(k => allowedBase.has(k))
          for (const k of baseSelectedKeysAll) {
            if (g.base[k] != null && g.base[k] !== '') continue
            const def = allFieldDefs.find(f => f.key === k)
            const val = def && def.getter ? def.getter(r) : (r as any)?.[k]
            g.base[k] = val ?? ''
          }
          for (const def of metricDefs) {
            if (def.type === 'traffic') g.traf[def.key] = { sum: Array(months.length).fill(0), cnt: Array(months.length).fill(0) }
            else g.money[def.key] = Array(months.length).fill(0)
          }
          group.set(gk, g)
        }
        const g = group.get(gk)!
        if (selectedTrafficKeys.has('daily_95_mbps')) {
          const v = Number((convertToBitsPerSecond(Number(r?.settlement_value ?? 0)) / 1_000_000).toFixed(9))
          if (!Number.isNaN(v)) { g.traf['daily_95_mbps'].sum[idx] += v; g.traf['daily_95_mbps'].cnt[idx] += 1 }
        }
        for (const def of metricDefs) {
          if (def.type !== 'money') continue
          const k = def.key
          const v = Number(r?.[k] ?? 0)
          if (!Number.isNaN(v)) g.money[k][idx] += v
        }
      }
      const allowedBase = new Set(['school_name','region','cp','customer_fee_owner_name','network_line_fee_owner_name','node_deduction_fee_owner_name','channel_owner_name'])
      const baseSelectedKeys = exportForm.selectedFields.filter(k => allowedBase.has(k))
      header = baseSelectedKeys.map(k => (allFieldDefs.find(f => f.key === k)?.label || k))
      for (const def of metricDefs) {
        const name = stripLabel(def.label)
        for (const mo of months) header.push(`${mo.label}${name}`)
      }
      const lines: string[] = []
      for (const [, g] of group) {
        const rowBase: string[] = baseSelectedKeys.map(k => csvEscape(g.base[k]))
        const row: string[] = [...rowBase]
        for (const def of metricDefs) {
          if (def.type === 'traffic') {
            const sarr = g.traf[def.key].sum
            const carr = g.traf[def.key].cnt
            for (let i = 0; i < months.length; i++) {
              const avg = carr[i] > 0 ? (sarr[i] / carr[i]) : ''
              row.push(avg === '' ? '' : String(Number(avg).toFixed(2)))
            }
          } else {
            const arr = g.money[def.key]
            for (let i = 0; i < months.length; i++) {
              const val = arr[i]
              row.push(val == null ? '' : String(Number(val).toFixed(2)))
            }
          }
        }
        lines.push(row.join(','))
      }
      rows = lines
    } else if (exportForm.groupBySchoolCp) {
      // 非按月，但按学校+CP聚合：金额字段求和，流量字段取平均
      const metricDefs = selectedDefs.filter(f => f.type === 'traffic' || f.type === 'money')
      const selectedTrafficKeys = new Set(metricDefs.filter(f => f.type === 'traffic').map(f => f.key))
      type Agg2 = { base: { [k:string]: any; school_name: string; region: string; cp: string }; traf: Record<string, { sum: number; cnt: number }>; money: Record<string, number> }
      const group2 = new Map<string, Agg2>()
      for (const r of data) {
        const gk = `${r?.school_name || ''}__${r?.cp || ''}`
        if (!group2.has(gk)) {
          const g: Agg2 = { base: { school_name: r?.school_name || '', region: r?.region || '', cp: r?.cp || '' }, traf: {}, money: {} }
          const allowedBase = new Set(['school_name','region','cp','customer_fee_owner_name','network_line_fee_owner_name','node_deduction_fee_owner_name','channel_owner_name'])
          const baseSelectedKeysAll = exportForm.selectedFields.filter(k => allowedBase.has(k))
          for (const k of baseSelectedKeysAll) {
            if (g.base[k] != null && g.base[k] !== '') continue
            const def = allFieldDefs.find(f => f.key === k)
            const val = def && def.getter ? def.getter(r) : (r as any)?.[k]
            g.base[k] = val ?? ''
          }
          for (const def of metricDefs) {
            if (def.type === 'traffic') g.traf[def.key] = { sum: 0, cnt: 0 }
            else g.money[def.key] = 0
          }
          group2.set(gk, g)
        }
        const g = group2.get(gk)!
        if (selectedTrafficKeys.has('daily_95_mbps')) {
          const v = Number((convertToBitsPerSecond(Number(r?.settlement_value ?? 0)) / 1_000_000).toFixed(9))
          if (!Number.isNaN(v)) { g.traf['daily_95_mbps'].sum += v; g.traf['daily_95_mbps'].cnt += 1 }
        }
        for (const def of metricDefs) {
          if (def.type !== 'money') continue
          const k = def.key
          const v = Number(r?.[k] ?? 0)
          if (!Number.isNaN(v)) g.money[k] += v
        }
      }
      const allowedBase2 = new Set(['school_name','region','cp','customer_fee_owner_name','network_line_fee_owner_name','node_deduction_fee_owner_name','channel_owner_name'])
      const baseSelectedKeys2 = exportForm.selectedFields.filter(k => allowedBase2.has(k))
      header = baseSelectedKeys2.map(k => (allFieldDefs.find(f => f.key === k)?.label || k))
      for (const def of metricDefs) header.push(def.label)
      const lines2: string[] = []
      for (const [, g] of group2) {
        const row: string[] = baseSelectedKeys2.map(k => csvEscape(g.base[k]))
        for (const def of metricDefs) {
          if (def.type === 'traffic') {
            const tt = g.traf[def.key]
            const avg = tt.cnt > 0 ? (tt.sum / tt.cnt) : ''
            row.push(avg === '' ? '' : String(Number(avg).toFixed(2)))
          } else {
            const val = g.money[def.key]
            row.push(val == null ? '' : String(Number(val).toFixed(2)))
          }
        }
        lines2.push(row.join(','))
      }
      rows = lines2
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

// 系统用户映射，用于归属显示
const userMap = ref<Record<number, { id:number; alias?:string; display_name?:string; username:string }>>({})

const loadUsersForItems = async () => {
  const ids = new Set<number>()
  for (const r of settlementData.value.items as any[]) {
    const push = (v:any) => { const n = Number(v); if (!Number.isNaN(n) && n>0) ids.add(n) }
    push(r?.customer_fee_owner_id)
    push(r?.network_line_fee_owner_id)
    push(r?.node_deduction_fee_owner_id)
    push(r?.channel_owner_user_id)
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
