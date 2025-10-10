<template>
  <div class="settlement-results-tab">
    <el-card class="filter-section" shadow="hover">
      <el-form :model="filterForm" inline label-width="84px">
        <el-form-item label="地区">
          <el-input v-model="filterForm.region" placeholder="输入地区" clearable style="width: 180px" />
        </el-form-item>
        <el-form-item label="运营商">
          <el-input v-model="filterForm.cp" placeholder="输入运营商" clearable style="width: 180px" />
        </el-form-item>
        <el-form-item label="学校">
          <el-input v-model="filterForm.school_name" placeholder="按名称搜索" clearable style="width: 200px" />
        </el-form-item>
        <el-form-item label="学校ID">
          <el-input v-model="filterForm.school_id" placeholder="精确匹配" clearable style="width: 180px" />
        </el-form-item>
        <el-form-item label="结算公式" required>
          <el-select
            v-model="filterForm.formula_id"
            placeholder="选择结算公式"
            style="width: 220px"
            :loading="formulasLoading"
          >
            <el-option
              v-for="item in formulas"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="换算系数">
          <el-select v-model="unitBase" style="width: 200px">
            <el-option :value="1000" label="SI (1000, GB)" />
            <el-option :value="1024" label="IEC (1024, GiB)" />
          </el-select>
        </el-form-item>
        <el-form-item label="日期范围" required>
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
            format="YYYY-MM-DD"
            :disabled-date="disableFutureDate"
            style="width: 300px"
            @change="handleDateRangeChange"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleQuery">查询</el-button>
          <el-button @click="resetFilter">重置</el-button>
          <el-button type="success" @click="handleCalculate">计算结算结果</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="table-section" shadow="hover">
      <div class="table-header">
        <h3 class="card-title">结算结果列表</h3>
      </div>
      <el-table
        v-loading="loading"
        :data="results.items"
        border
        stripe
        style="width: 100%"
        empty-text="暂无结算结果"
      >
        <el-table-column prop="school_name" label="学校" min-width="180" />
        <el-table-column prop="region" label="地区" width="120" />
        <el-table-column prop="cp" label="运营商" width="120" />
        <el-table-column label="周期" min-width="170">
          <template #default="{ row }">
            {{ formatDate(row.start_date) }} 至 {{ formatDate(row.end_date) }}
          </template>
        </el-table-column>
        <el-table-column label="账期天数" width="90" prop="billing_days" />
        <el-table-column label="缺失天数" width="90" prop="missing_days" />
        <el-table-column :label="`平均95值 (${unitLabel})`" min-width="180">
          <template #default="{ row }">
            {{ formatGByte(row.average_95_flow) }}
          </template>
        </el-table-column>
        <el-table-column label="结算金额" min-width="120">
          <template #default="{ row }">
            {{ formatCurrency(row.amount, row.currency) }}
          </template>
        </el-table-column>
        <el-table-column label="费率" min-width="220">
          <template #default="{ row }">
            <div class="rate-line">客户：{{ formatNumber(row.customer_fee) }}</div>
            <div class="rate-line">线路：{{ formatNumber(row.network_line_fee) }}</div>
            <div class="rate-line">节点：{{ formatNumber(row.node_deduction_fee) }}</div>
            <div class="rate-line">最终：{{ formatNumber(row.final_fee) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="缺失字段" min-width="160">
          <template #default="{ row }">
            <template v-if="row.missing_fields?.length">
              <el-tag
                v-for="field in row.missing_fields"
                :key="field"
                size="small"
                type="warning"
                effect="dark"
                style="margin-right: 4px; margin-bottom: 4px"
              >
                {{ field }}
              </el-tag>
            </template>
            <span v-else>无</span>
          </template>
        </el-table-column>
        <el-table-column label="公式" min-width="160">
          <template #default="{ row }">
            <div>{{ row.formula_name }} (#{{ row.formula_id }})</div>
            <el-popover trigger="hover" placement="top" width="260">
              <template #reference>
                <el-button link type="primary">查看公式Token</el-button>
              </template>
              <pre class="token-preview">{{ prettyJSON(row.formula_tokens) }}</pre>
            </el-popover>
          </template>
        </el-table-column>
        <el-table-column label="计算明细" min-width="220">
          <template #default="{ row }">
            <el-popover trigger="click" placement="top" width="360">
              <template #reference>
                <el-button link type="primary">查看</el-button>
              </template>
              <div v-if="parseDetail(row.calculation_detail)">
                <!-- 平均95值：优先使用 *_converted 与 *_bytes；退化为 average_95 -->
                <div class="detail-line">
                  <span class="detail-key">平均95值：</span>
                  <span>
                    <template v-if="parseDetail(row.calculation_detail)?.average_95_converted !== undefined">
                      {{ formatNumber(parseDetail(row.calculation_detail)!.average_95_converted) }}
                      {{ parseDetail(row.calculation_detail)!.converted_unit || unitLabel }}
                      ({{ formatBytes(parseDetail(row.calculation_detail)!.average_95_bytes) }})
                    </template>
                    <template v-else>
                      {{ formatGByte(parseDetail(row.calculation_detail)!.average_95) }} {{ unitLabel }}
                      ({{ formatBytes(parseDetail(row.calculation_detail)!.average_95) }})
                    </template>
                  </span>
                </div>
                <!-- 总95值：优先使用 *_converted 与 *_bytes；退化为 total_95 -->
                <div class="detail-line">
                  <span class="detail-key">总95值：</span>
                  <span>
                    <template v-if="parseDetail(row.calculation_detail)?.total_95_converted !== undefined">
                      {{ formatNumber(parseDetail(row.calculation_detail)!.total_95_converted) }}
                      {{ parseDetail(row.calculation_detail)!.converted_unit || unitLabel }}
                      ({{ formatBytes(parseDetail(row.calculation_detail)!.total_95_bytes) }})
                    </template>
                    <template v-else>
                      {{ formatGByte(parseDetail(row.calculation_detail)!.total_95) }} {{ unitLabel }}
                      ({{ formatBytes(parseDetail(row.calculation_detail)!.total_95) }})
                    </template>
                  </span>
                </div>
                <!-- 金额（带四舍五入策略） -->
                <div class="detail-line">
                  <span class="detail-key">金额：</span>
                  <span>
                    {{ formatCurrency(row.amount, row.currency) }}
                    <span class="detail-key">
                      （原始: {{ formatNumber(parseDetail(row.calculation_detail)!.amount_raw) }},
                      舍入: {{ parseDetail(row.calculation_detail)!.rounding_mode || 'HALF_UP' }}/{{ parseDetail(row.calculation_detail)!.rounding_scale ?? 2 }}）
                    </span>
                  </span>
                </div>
                <!-- 费率项 -->
                <div class="detail-line"><span class="detail-key">客户费率：</span><span>{{ formatNumber(row.customer_fee) }}</span></div>
                <div class="detail-line"><span class="detail-key">线路费率：</span><span>{{ formatNumber(row.network_line_fee) }}</span></div>
                <div class="detail-line"><span class="detail-key">节点扣减：</span><span>{{ formatNumber(row.node_deduction_fee) }}</span></div>
                <div class="detail-line"><span class="detail-key">最终费率：</span><span>{{ formatNumber(row.final_fee) }}</span></div>
              </div>
              <div v-else>暂无</div>
            </el-popover>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" min-width="160">
          <template #default="{ row }">
            {{ formatDateTime(row.updated_at) }}
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-container">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="results.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import type { SettlementResultFilter, SettlementResultItem, SettlementResultResponse } from '@/types/settlement'
import type { SettlementFormulaItem } from '@/types/api'

interface FilterForm extends SettlementResultFilter {}

const loading = ref(false)
const results = ref<SettlementResultResponse>({ items: [], total: 0 })
const currentPage = ref(1)
const pageSize = ref(10)
const dateRange = ref<[string, string] | null>(null)
const formulas = ref<SettlementFormulaItem[]>([])
const formulasLoading = ref(false)
const calculating = ref(false)

// 单位换算设置：1000 表示十进制（GB），1024 表示二进制（GiB），默认 1024
const unitBase = ref<number>(1024)
const unitLabel = computed(() => (unitBase.value === 1000 ? 'GB' : 'GiB'))

// 从本地存储读取用户偏好
const loadUnitBaseFromStorage = () => {
  const raw = localStorage.getItem('settlement.unitBase')
  const v = raw ? Number(raw) : 1024
  unitBase.value = v === 1000 || v === 1024 ? v : 1024
}

// 持久化用户选择
watch(unitBase, (v) => {
  try { localStorage.setItem('settlement.unitBase', String(v)) } catch {}
})

const filterForm = reactive<FilterForm>({
  region: '',
  cp: '',
  school_name: '',
  school_id: '',
  start_date: '',
  end_date: '',
  limit: 10,
  offset: 0,
  formula_id: undefined,
})

const disableFutureDate = (date: Date) => date.getTime() > Date.now()

const setDefaultDateRange = () => {
  const today = new Date()
  const start = new Date(today.getTime() - 6 * 24 * 60 * 60 * 1000)
  const format = (d: Date) => d.toISOString().slice(0, 10)
  const startStr = format(start)
  const endStr = format(today)
  dateRange.value = [startStr, endStr]
  filterForm.start_date = startStr
  filterForm.end_date = endStr
}

const parseDetail = (detail: string | null | undefined): Record<string, any> | null => {
  if (!detail) return null
  try {
    const obj = JSON.parse(detail) as Record<string, any>
    return obj && typeof obj === 'object' ? obj : null
  } catch (error) {
    console.warn('解析计算明细失败', error)
    return null
  }
}

const prettyJSON = (payload: string | null | undefined) => {
  if (!payload) return '无'
  try {
    const obj = JSON.parse(payload)
    return JSON.stringify(obj, null, 2)
  } catch {
    return payload
  }
}

const formatNumber = (val: number | null | undefined) => {
  if (val === null || val === undefined || Number.isNaN(val)) return '0.00'
  return Number(val).toFixed(2)
}

// 将后端字节值换算为 GB/GiB 的数值字符串（两位小数），单位由 unitLabel 控制
const formatGByte = (val: number | null | undefined) => {
  const num = val === null || val === undefined ? 0 : Number(val)
  const base = unitBase.value
  const converted = num / Math.pow(base, 3)
  return converted.toFixed(2)
}

// 计算明细中需要按容量单位换算的键
const FLOW_DETAIL_KEYS = new Set<string>(['average_95', 'total_95'])

// 统一格式化计算明细的值：流量相关按 GB/GiB 展示，其余保留两位小数
const formatDetailValue = (key: string, value: number) => {
  if (FLOW_DETAIL_KEYS.has(key)) {
    return `${formatGByte(value)} ${unitLabel.value}`
  }
  return formatNumber(value)
}

const formatCurrency = (val: number | null | undefined, currency?: string) => {
  const num = val === null || val === undefined ? 0 : Number(val)
  const unit = currency || 'CNY'
  return `${num.toFixed(2)} ${unit}`
}

// Byte 原样格式化展示（带千分位）
const formatBytes = (val: number | null | undefined) => {
  const n = val === null || val === undefined ? 0 : Number(val)
  return `${Math.round(n).toLocaleString()} B`
}

const formatDate = (val: string | Date) => {
  if (!val) return '-'
  const str = typeof val === 'string' ? val : val.toISOString()
  return str.slice(0, 10)
}

const formatDateTime = (val: string | null | undefined) => {
  if (!val) return '-'
  const date = new Date(val)
  if (Number.isNaN(date.getTime())) return val
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const buildParams = () => {
  const params: SettlementResultFilter = {
    start_date: filterForm.start_date,
    end_date: filterForm.end_date,
    limit: pageSize.value,
    offset: (currentPage.value - 1) * pageSize.value,
    // 将单位基数传递给后端，配合元/G 的公式计算
    // @ts-ignore 扩展字段
    unit_base: unitBase.value as any,
  }

  if (filterForm.region) params.region = filterForm.region
  if (filterForm.cp) params.cp = filterForm.cp
  if (filterForm.school_name) params.school_name = filterForm.school_name
  if (filterForm.school_id) params.school_id = filterForm.school_id
  if (filterForm.formula_id) params.formula_id = filterForm.formula_id

  return params
}

const fetchResults = async () => {
  if (!filterForm.start_date || !filterForm.end_date) {
    ElMessage.warning('请先选择日期范围')
    return
  }
  loading.value = true
  try {
    const params = buildParams()
    const data = await (api as any).settlement.getResults(params)
    if (Array.isArray(data)) {
      results.value = { items: data as SettlementResultItem[], total: (data as SettlementResultItem[]).length }
    } else if (data && typeof data === 'object') {
      const items = Array.isArray((data as any).items) ? (data as any).items as SettlementResultItem[] : []
      const total = typeof (data as any).total === 'number' ? Number((data as any).total) : items.length
      results.value = { items, total }
    } else {
      results.value = { items: [], total: 0 }
    }
  } catch (error) {
    console.error('获取结算结果失败', error)
    ElMessage.error('获取结算结果失败')
  } finally {
    loading.value = false
  }
}

const handleDateRangeChange = (range: [string, string] | null) => {
  if (range) {
    filterForm.start_date = range[0]
    filterForm.end_date = range[1]
  } else {
    filterForm.start_date = ''
    filterForm.end_date = ''
  }
}

const handleQuery = () => {
  currentPage.value = 1
  fetchResults()
}

const resetFilter = () => {
  filterForm.region = ''
  filterForm.cp = ''
  filterForm.school_name = ''
  filterForm.school_id = ''
  filterForm.formula_id = undefined
  currentPage.value = 1
  pageSize.value = 10
  setDefaultDateRange()
  fetchResults()
}

const loadFormulas = async () => {
  formulasLoading.value = true
  try {
    const res = await (api as any).settlement.formulas.list({ limit: 200, offset: 0 })
    if (res && Array.isArray(res.items)) {
      formulas.value = res.items.filter((item: SettlementFormulaItem) => item.enabled)
    } else if (Array.isArray(res)) {
      formulas.value = (res as SettlementFormulaItem[]).filter(item => item.enabled)
    } else {
      formulas.value = []
    }
    if (!filterForm.formula_id && formulas.value.length) {
      filterForm.formula_id = formulas.value[0].id
    }
  } catch (error) {
    console.error('加载结算公式失败', error)
    ElMessage.error('加载结算公式失败')
  } finally {
    formulasLoading.value = false
  }
}

const handleCalculate = async () => {
  if (!filterForm.start_date || !filterForm.end_date) {
    ElMessage.warning('请先选择日期范围')
    return
  }
  if (!filterForm.formula_id) {
    ElMessage.warning('请选择结算公式')
    return
  }

  const confirmed = await ElMessageBox.confirm('将根据所选公式重新计算结算结果，并写入缓存。确定继续？', '提示', {
    type: 'warning',
    confirmButtonText: '确定',
    cancelButtonText: '取消',
  }).catch(() => false)
  if (!confirmed) return

  calculating.value = true
  try {
    const params = buildParams()
    await (api as any).settlement.getResults(params)
    ElMessage.success('结算计算完成')
    fetchResults()
  } catch (error) {
    console.error('结算计算失败', error)
    ElMessage.error('结算计算失败')
  } finally {
    calculating.value = false
  }
}

const handleCurrentChange = (page: number) => {
  currentPage.value = page
  fetchResults()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  fetchResults()
}

onMounted(async () => {
  loadUnitBaseFromStorage()
  setDefaultDateRange()
  await loadFormulas()
  fetchResults()
})
</script>

<style scoped>
.settlement-results-tab {
  padding: 10px;
}

.filter-section {
  margin-bottom: 20px;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.rate-line {
  line-height: 1.4;
}

.token-preview {
  max-height: 240px;
  overflow: auto;
  white-space: pre-wrap;
}

.detail-line {
  display: flex;
  justify-content: space-between;
  margin-bottom: 4px;
}

.detail-key {
  color: var(--text-muted);
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
