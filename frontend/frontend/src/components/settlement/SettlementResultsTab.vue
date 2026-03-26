<template>
  <div class="settlement-results-tab">
    <el-card class="filter-section" shadow="hover">
      <el-form :model="filterForm" inline label-width="84px">
        <el-form-item label="地区">
          <el-input v-model="filterForm.region" placeholder="输入地区" clearable class="field-w-180" />
        </el-form-item>
        <el-form-item label="CP">
          <el-input v-model="filterForm.cp" placeholder="输入 CP" clearable class="field-w-180" />
        </el-form-item>
        <el-form-item label="学校">
          <el-input v-model="filterForm.school_name" placeholder="按名称搜索" clearable class="field-w-200" />
        </el-form-item>
        <el-form-item label="学校ID">
          <el-input v-model="filterForm.school_id" placeholder="精确匹配" clearable class="field-w-180" />
        </el-form-item>
        <el-form-item label="结算公式" required>
          <el-select
            v-model="filterForm.formula_id"
            placeholder="选择结算公式"
            class="field-w-220"
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
            class="field-w-300"
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
        class="field-w-full"
        empty-text="暂无结算结果"
      >
        <el-table-column prop="school_name" label="学校" min-width="180" />
        <el-table-column prop="region" label="地区" width="120" />
        <el-table-column prop="cp" label="CP" width="120" />
        <el-table-column label="周期" min-width="170">
          <template #default="{ row }">
            {{ formatDate(row.start_date) }} 至 {{ formatDate(row.end_date) }}
          </template>
        </el-table-column>
        <el-table-column label="账期天数" width="90" prop="billing_days" />
        <el-table-column label="缺失天数" width="90" prop="missing_days" />
        <el-table-column label="平均95值 (Gbps)" min-width="220">
          <template #default="{ row }">
            <el-tooltip placement="top">
              <template #content>
                <div>
                  <div>计费天数（自然日）：{{ getNaturalDays(row.start_date, row.end_date) }}</div>
                  <div>总95（Σ每日日95）：{{ formatNumberPrec(getTotalGbps(row), 4) }} Gbps</div>
                  <div>
                    平均95 = 总95 / 计费天数 =
                    {{ formatNumberPrec(getTotalGbps(row), 4) }}
                    /
                    {{ getNaturalDays(row.start_date, row.end_date) }}
                    =
                    {{ formatNumberPrec(getAverageGbps(row, getNaturalDays(row.start_date, row.end_date)), 4) }} Gbps
                  </div>
                </div>
              </template>
              <span>{{ formatNumber(getAverageGbps(row, getNaturalDays(row.start_date, row.end_date))) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="结算金额" min-width="120">
          <template #default="{ row }">
            <el-tooltip :content="`${formatNumberPrec(parseDetail(row.calculation_detail)?.amount_raw ?? row.amount, 4)} ${row.currency || 'CNY'}`" placement="top">
              <span>{{ formatCurrency(row.amount, row.currency) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="费率" min-width="220">
          <template #default="{ row }">
            <div class="rate-line">客户：{{ formatNumber(row.customer_fee) }}</div>
            <div class="rate-line">线路：{{ formatNumber(row.network_line_fee) }}</div>
            <div class="rate-line">节点：{{ formatNumber(row.node_deduction_fee) }}</div>
            <div class="rate-line">毛利：{{ formatNumber(row.final_fee) }}</div>
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
                class="mr-4 mb-4"
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
                <el-button link type="primary">查看公式</el-button>
              </template>
              <code class="token-preview">{{ formulaPreview(row.formula_tokens) }}</code>
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
                    <el-tooltip placement="top">
                      <template #content>
                        <div>
                          <div>计费天数（自然日）：{{ getNaturalDays(row.start_date, row.end_date) }}</div>
                          <div>总95（Σ每日日95）：{{ formatNumberPrec(getTotalGbps(row), 4) }} Gbps</div>
                          <div>
                            平均95 = 总95 / 计费天数 =
                            {{ formatNumberPrec(getTotalGbps(row), 4) }}
                            /
                            {{ getNaturalDays(row.start_date, row.end_date) }}
                            =
                            {{ formatNumberPrec(getAverageGbps(row, getNaturalDays(row.start_date, row.end_date)), 4) }} Gbps
                          </div>
                        </div>
                      </template>
                      <span>{{ formatNumber(getAverageGbps(row, getNaturalDays(row.start_date, row.end_date)) ) }} Gbps</span>
                    </el-tooltip>
                  </span>
                </div>
                <!-- 总95值：优先使用 *_converted 与 *_bytes；退化为 total_95 -->
                <div class="detail-line">
                  <span class="detail-key">总95值：</span>
                  <span>
                    <template v-if="parseDetail(row.calculation_detail)?.total_95_gbps !== undefined">
                      <el-tooltip :content="`${formatNumberPrec(parseDetail(row.calculation_detail)!.total_95_gbps, 4)} Gbps`" placement="top">
                        <span>{{ formatNumber(parseDetail(row.calculation_detail)!.total_95_gbps) }} Gbps</span>
                      </el-tooltip>
                    </template>
                    <template v-else>
                      <el-tooltip :content="`${formatGbpsFromBytesPrec(parseDetail(row.calculation_detail)?.total_95_bytes ?? parseDetail(row.calculation_detail)?.total_95, 4)} Gbps`" placement="top">
                        <span>{{ formatGbpsFromBytes(parseDetail(row.calculation_detail)?.total_95_bytes ?? parseDetail(row.calculation_detail)?.total_95) }} Gbps</span>
                      </el-tooltip>
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
                <div class="detail-line"><span class="detail-key">毛利：</span><span>{{ formatNumber(row.final_fee) }}</span></div>
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
import { reactive, ref, onMounted } from 'vue'
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

const formatGbps = (val: number | null | undefined) => {
  const num = val === null || val === undefined ? 0 : Number(val)
  return num.toFixed(2)
}

// 数字保留自定义小数位（默认9位）
const formatNumberPrec = (val: number | null | undefined, digits = 9) => {
  const num = val === null || val === undefined ? 0 : Number(val)
  return num.toFixed(digits)
}

// 从 Byte（每分钟样本）换算为 Gbps：bytes * 8 / 60 / 1e9
const formatGbpsFromBytes = (bytes: number | null | undefined) => {
  const n = bytes === null || bytes === undefined ? 0 : Number(bytes)
  const gbps = (n * 8) / 60 / 1e9
  return gbps.toFixed(2)
}

const formatGbpsFromBytesPrec = (bytes: number | null | undefined, digits = 9) => {
  const n = bytes === null || bytes === undefined ? 0 : Number(bytes)
  const gbps = (n * 8) / 60 / 1e9
  return gbps.toFixed(digits)
}

// 计算总/平均 Gbps（优先行字段 total_95_flow，其次 *_gbps，再次 *_bytes，缺省返回 0）
const getTotalGbpsFromDetail = (detail: any): number => {
  try {
    if (detail && typeof detail.total_95_gbps === 'number') return Number(detail.total_95_gbps)
    if (detail && (typeof detail.total_95_bytes === 'number' || typeof detail.total_95 === 'number')) {
      const b = Number(detail.total_95_bytes ?? detail.total_95 ?? 0)
      return (b * 8) / 60 / 1e9
    }
  } catch {}
  return 0
}

const getTotalGbps = (row: any): number => {
  if (typeof row?.total_95_flow === 'number') return Number(row.total_95_flow)
  return getTotalGbpsFromDetail(parseDetail(row?.calculation_detail))
}

const getAverageGbps = (row: any, billingDays: number): number => {
  try {
    const total = getTotalGbps(row)
    return billingDays > 0 ? total / billingDays : 0
  } catch {}
  return 0
}

// 计算所选范围内的自然日天数（含起止，两端闭区间）
const getNaturalDays = (start: string | Date, end: string | Date): number => {
  if (!start || !end) return 0
  const s = typeof start === 'string' ? new Date(start) : new Date(start)
  const e = typeof end === 'string' ? new Date(end) : new Date(end)
  if (Number.isNaN(s.getTime()) || Number.isNaN(e.getTime())) return 0
  const oneDay = 24 * 60 * 60 * 1000
  return Math.floor((e.getTime() - s.getTime()) / oneDay) + 1
}

const formatCurrency = (val: number | null | undefined, currency?: string) => {
  const num = val === null || val === undefined ? 0 : Number(val)
  const unit = currency || 'CNY'
  return `${num.toFixed(2)} ${unit}`
}

// 将公式 tokens（JSON 字符串或数组）渲染为可读表达式
const formulaPreview = (tokensPayload: any): string => {
  try {
    let tokens: any[] = []
    if (typeof tokensPayload === 'string' && tokensPayload.trim().length) {
      tokens = JSON.parse(tokensPayload)
    } else if (Array.isArray(tokensPayload)) {
      tokens = tokensPayload
    } else {
      return '无'
    }
    const parts = tokens.map((t: any) => {
      if (!t) return ''
      const type = typeof t.type === 'string' ? t.type : (typeof t.value === 'string' && /^[+\-*/()]$/.test(t.value) ? 'operator' : 'field')
      const value = typeof t.value === 'string' ? t.value : ''
      const label = typeof t.label === 'string' && t.label.length ? t.label : ''
      if (type === 'field') return label || `{{${value}}}`
      return label || value
    }).filter(Boolean)
    return parts.join(' ')
  } catch {
    // 解析失败时，回退为原始字符串或提示
    return typeof tokensPayload === 'string' && tokensPayload.trim().length ? tokensPayload : '无'
  }
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


