<template>
  <div class="channel-settlement-view">
    <h1 class="page-title">用户维度结算</h1>

    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">筛选条件</span>
          <div>
            <el-button type="primary" :loading="loading" @click="handleQuery">查询</el-button>
            <el-button @click="resetFilter">重置</el-button>
            <el-button type="success" :loading="calculating" @click="handleCalculate">计算</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="filterForm" label-width="90px" class="filter-form">
        <el-form-item label="用户名称">
          <el-input v-model="filterForm.user_name" placeholder="按用户名称模糊查询" class="field-w-240" />
        </el-form-item>
        <el-form-item label="结算公式">
          <el-select v-model="filterForm.formula_id" filterable placeholder="选择结算公式" class="field-w-260" :loading="formulasLoading">
            <el-option v-for="f in formulas" :key="f.id" :label="f.name" :value="f.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="日期范围">
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
            :disabled-date="disableFutureDate"
            @change="handleDateRangeChange"
          />
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card mt-2">
      <template #header>
        <div class="card-header">
          <span class="card-title">结算结果</span>
          <div class="token-preview" v-if="currentFormula">
            公式：{{ formulaPreview(currentFormula.tokens) }}
          </div>
        </div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="user_name" label="用户" min-width="200" show-overflow-tooltip />
        <el-table-column label="金额" min-width="160">
          <template #default="{ row }">
            <el-tooltip placement="top">
              <template #content>
                <pre class="pre-wrap-zero">{{ renderBreakdownTooltip(row.breakdown_detail) }}</pre>
              </template>
              <span>{{ formatCurrency(row.amount, row.currency) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="formula_name" label="公式" min-width="160" />
        <el-table-column label="账期" width="220">
          <template #default="{ row }">
            {{ row.start_date }} ~ {{ row.end_date }}
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
      </el-table>

      <div class="pagination">
        <el-pagination
          background
          layout="prev, pager, next, sizes, total"
          :total="total"
          :page-size="pageSize"
          :current-page="page"
          :page-sizes="[10,20,50,100]"
          @size-change="onPageSizeChange"
          @current-change="onPageChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import type { SettlementFormulaItem } from '@/types/api'
import type { ChannelSettlementResultFilter, ChannelSettlementResultItem, ChannelSettlementResultResponse } from '@/types/settlement'

const loading = ref(false)
const calculating = ref(false)
const items = ref<ChannelSettlementResultItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const dateRange = ref<[string, string] | null>(null)

const formulas = ref<SettlementFormulaItem[]>([])
const formulasLoading = ref(false)

const filterForm = reactive<ChannelSettlementResultFilter>({
  user_name: '',
  start_date: '',
  end_date: '',
  limit: 10,
  offset: 0,
  formula_id: undefined,
})

const currentFormula = computed(() => formulas.value.find(f => f.id === filterForm.formula_id))

const disableFutureDate = (date: Date) => date.getTime() > Date.now()

const setDefaultDateRange = () => {
  const today = new Date()
  const start = new Date(today.getTime() - 6 * 24 * 60 * 60 * 1000)
  const fmt = (d: Date) => d.toISOString().slice(0, 10)
  dateRange.value = [fmt(start), fmt(today)]
  filterForm.start_date = dateRange.value[0]
  filterForm.end_date = dateRange.value[1]
}

const BREAKDOWN_FIELD_LABELS: Record<string, string> = {
  customer_fee: '客户费',
  network_line_fee: '线路费',
  node_deduction_fee: '节点扣减',
  final_fee: '毛利',
}

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
  } catch { return typeof tokensPayload === 'string' ? tokensPayload : '无' }
}

const renderBreakdownTooltip = (payload?: string) => {
  if (!payload) return '无'
  try {
    const obj = JSON.parse(payload) as Record<string, any>
    const preferredOrder = ['customer_fee', 'network_line_fee', 'node_deduction_fee', 'final_fee']
    const entries = Object.entries(obj)
      .filter(([_, v]) => {
        if (typeof v === 'number') return !Number.isNaN(v)
        if (v && typeof v === 'object') {
          const amount = Number((v as any).amount ?? (v as any).value ?? (v as any).result)
          return Number.isFinite(amount)
        }
        return false
      })
      .sort(([a], [b]) => {
        const ia = preferredOrder.indexOf(a)
        const ib = preferredOrder.indexOf(b)
        if (ia === -1 && ib === -1) return a.localeCompare(b)
        if (ia === -1) return 1
        if (ib === -1) return -1
        return ia - ib
      })
    if (!entries.length) return '无'
    const detailLines: string[] = []
    const formulaParts: string[] = []
    let total = 0

    for (const [k, rawVal] of entries) {
      const label = BREAKDOWN_FIELD_LABELS[k] || k
      let amount = 0
      let explain: string | undefined

      if (typeof rawVal === 'number') {
        amount = rawVal
      } else if (rawVal && typeof rawVal === 'object') {
        const v = rawVal as any
        const parsedAmount = Number(v.amount ?? v.value ?? v.result)
        amount = Number.isFinite(parsedAmount) ? parsedAmount : 0
        explain = typeof v.explain === 'string' && v.explain.trim().length
          ? v.explain.trim()
          : (typeof v.formula === 'string' && v.formula.trim().length ? v.formula.trim() : undefined)
      }

      total += amount
      detailLines.push(`${label}: ${amount.toFixed(4)}`)
      formulaParts.push(`${label}(${amount.toFixed(4)})`)

      if (explain) {
        detailLines.push(`  计算：${explain}`)
      }
    }

    const formulaLine = formulaParts.length
      ? `金额 = ${formulaParts.join(' + ')} = ${total.toFixed(4)}`
      : ''

    return [
      '分项明细：',
      ...detailLines,
      '',
      '计算过程：',
      formulaLine || '金额为各分项加总（含正负号）',
    ].join('\n')
  } catch { return '无' }
}

const formatCurrency = (val: number | null | undefined, currency?: string) => {
  const num = val === null || val === undefined ? 0 : Number(val)
  const unit = currency || 'CNY'
  return `${num.toFixed(2)} ${unit}`
}

const buildParams = (): ChannelSettlementResultFilter => {
  const p: ChannelSettlementResultFilter = {
    start_date: filterForm.start_date,
    end_date: filterForm.end_date,
    limit: pageSize.value,
    offset: (page.value - 1) * pageSize.value,
  }
  if (filterForm.user_name) p.user_name = filterForm.user_name
  if (filterForm.formula_id) p.formula_id = filterForm.formula_id
  return p
}

const fetchResults = async () => {
  if (!filterForm.start_date || !filterForm.end_date) {
    ElMessage.warning('请先选择日期范围')
    return
  }
  loading.value = true
  try {
    const params = buildParams()
    const data = await (api as any).settlement.getChannelResults(params)
    if (Array.isArray(data)) {
      items.value = data as ChannelSettlementResultItem[]
      total.value = (data as ChannelSettlementResultItem[]).length
    } else if (data && typeof data === 'object') {
      const arr = Array.isArray((data as any).items) ? (data as any).items as ChannelSettlementResultItem[] : []
      items.value = arr
      total.value = typeof (data as any).total === 'number' ? Number((data as any).total) : arr.length
    } else {
      items.value = []
      total.value = 0
    }
  } catch (e) {
    console.error('获取渠道结算结果失败', e)
    ElMessage.error('获取渠道结算结果失败')
  } finally {
    loading.value = false
  }
}

const loadFormulas = async () => {
  formulasLoading.value = true
  try {
    const res = await (api as any).settlement.formulas.list({ limit: 200, offset: 0 })
    if (res && Array.isArray(res.items)) {
      formulas.value = res.items.filter((it: SettlementFormulaItem) => it.enabled)
    } else if (Array.isArray(res)) {
      formulas.value = (res as SettlementFormulaItem[]).filter(it => it.enabled)
    } else {
      formulas.value = []
    }
    if (!filterForm.formula_id && formulas.value.length) {
      filterForm.formula_id = formulas.value[0].id
    }
  } catch (e) {
    console.error('加载结算公式失败', e)
    ElMessage.error('加载结算公式失败')
  } finally { formulasLoading.value = false }
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

const handleQuery = () => { page.value = 1; fetchResults() }
const resetFilter = () => {
  ;(filterForm as any).channel_name = ''
  filterForm.user_name = ''
  page.value = 1
  pageSize.value = 10
  setDefaultDateRange()
  fetchResults()
}

const handleCalculate = async () => {
  if (!filterForm.start_date || !filterForm.end_date) { ElMessage.warning('请先选择日期范围'); return }
  if (!filterForm.formula_id) { ElMessage.warning('请选择结算公式'); return }
  const confirmed = await ElMessageBox.confirm('将根据所选公式重新计算用户结算并写入缓存。确定继续？', '提示', { type: 'warning', confirmButtonText: '确定', cancelButtonText: '取消' }).catch(() => false)
  if (!confirmed) return
  calculating.value = true
  try {
    // 直接触发查询，后端按需计算
    await fetchResults()
    ElMessage.success('用户结算计算完成')
  } catch {
    ElMessage.error('用户结算计算失败')
  } finally { calculating.value = false }
}

const onPageChange = (p: number) => { page.value = p; fetchResults() }
const onPageSizeChange = (ps: number) => { pageSize.value = ps; page.value = 1; fetchResults() }

// 输入即过滤（防抖）
function debounce<T extends (...args: any[]) => void>(fn: T, wait = 400) {
  let t: number | undefined
  return (...args: Parameters<T>) => {
    if (t) window.clearTimeout(t)
    t = window.setTimeout(() => fn(...args), wait)
  }
}
const debouncedFilter = debounce(() => { page.value = 1; fetchResults() }, 400)
watch(() => filterForm.user_name, () => {
  debouncedFilter()
})

onMounted(async () => {
  setDefaultDateRange()
  await loadFormulas()
  fetchResults()
})
</script>

<style scoped>
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.token-preview { color: var(--text-muted); font-size: 12px; }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>


