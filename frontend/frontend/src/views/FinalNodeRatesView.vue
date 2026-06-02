<template>
  <div class="rates-view">
    <h1 class="page-title">最终节点费率</h1>
    <el-card shadow="never" class="box-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">最终节点费率筛选</span>
          <div>
            <QueryActionButton :running="queryCtl.running.value" @trigger="onSearch" />
            <el-button @click="onReset">重置</el-button>
            <el-button v-if="canWrite" @click="onInit">从节点业务费率初始化</el-button>
            <el-button v-if="canWrite" @click="onRefresh">刷新自动费率</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="区域">
          <el-input v-model="query.region" clearable class="field-w-160" />
        </el-form-item>
        <el-form-item label="CP">
          <el-input v-model="query.cp" clearable class="field-w-160" />
        </el-form-item>
        <el-form-item label="节点">
          <el-input v-model="query.display_name" clearable class="field-w-180" />
        </el-form-item>
        <el-form-item label="结算模式">
          <el-select v-model="query.settlement_mode" clearable class="field-w-180">
            <el-option label="日95均值" value="daily_95_avg" />
            <el-option label="月95" value="range_95" />
          </el-select>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box-card mt-2">
      <template #header>
        <div class="card-header"><span class="card-title">费率列表</span></div>
      </template>
      <el-table :data="displayItems" border stripe height="600px" v-loading="loading">
        <el-table-column prop="display_name" label="节点" min-width="180">
          <template #default="{ row }">{{ row.display_name || '区域+CP默认' }}</template>
        </el-table-column>
        <el-table-column prop="entity_id" label="实体ID" width="90" />
        <el-table-column prop="region" label="区域" width="120" />
        <el-table-column prop="cp" label="CP" width="120" />
        <el-table-column label="已配置模式" width="170">
          <template #default="{ row }">
            <el-tag v-if="hasMode(row, 'daily_95_avg')" size="small" type="primary" effect="plain">日95均值</el-tag>
            <el-tag v-if="hasMode(row, 'range_95')" size="small" type="success" effect="plain" class="mode-tag">月95</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="日95最终单价" width="130">
          <template #default="{ row }">{{ modeRateValue(row, 'daily_95_avg', 'final_fee') }}</template>
        </el-table-column>
        <el-table-column label="月95最终单价" width="130">
          <template #default="{ row }">{{ modeRateValue(row, 'range_95', 'final_fee') }}</template>
        </el-table-column>
        <el-table-column label="费率详情" min-width="240">
          <template #default="{ row }">
            <el-tooltip placement="top" effect="light" :disabled="rateSummary(row) === '-'" popper-class="final-node-rate-tooltip">
              <template #content>
                <div class="rate-tooltip">
                  <div v-for="mode in rateTooltipModes(row)" :key="mode.mode" class="rate-tooltip-mode">
                    <div class="rate-tooltip-title">{{ mode.label }}</div>
                    <div v-for="item in mode.items" :key="item.label" class="rate-tooltip-row">
                      <span>{{ item.label }}</span>
                      <strong>{{ item.value }}</strong>
                    </div>
                  </div>
                </div>
              </template>
              <span class="rate-summary">{{ rateSummary(row) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="100">
          <template #default="{ row }">{{ feeTypeSummary(row) }}</template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
      </el-table>
      <div class="pagination">
        <el-pagination background layout="prev, pager, next, sizes, total" :total="total" :page-size="pageSize" :current-page="page" :page-sizes="[10,20,50,100]" @size-change="onPageSizeChange" @current-change="onPageChange" />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'
import type { PaginatedData, RateFinalNode } from '@/types/api'
import { useAuthStore } from '@/stores/auth'
import QueryActionButton from '@/components/ui/QueryActionButton.vue'
import { useCancelableQuery, isAbortError } from '@/composables/useCancelableQuery'
import { usePageRefresh } from '@/composables/usePageRefresh'
import { normalizeNodeRateMode, type NodeRateMode } from './node-rates-dual-mode'

const auth = useAuthStore()
const canWrite = computed(() => auth.hasPermission('rates.node.write'))
const loading = ref(false)
const items = ref<RateFinalNode[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const queryCtl = useCancelableQuery()
const query = reactive<{ region?: string; cp?: string; display_name?: string; settlement_mode?: string }>({})
type AggregatedFinalNodeRateRow = RateFinalNode & {
  configured_modes: NodeRateMode[]
  mode_rates: Partial<Record<NodeRateMode, RateFinalNode>>
}

const displayItems = computed(() => aggregateFinalNodeRateRows(items.value))

function modeLabel(mode?: string) { return mode === 'range_95' ? '月95' : '日95均值' }
function finalNodeRateScopeKey(row: Pick<RateFinalNode, 'entity_id' | 'region' | 'cp'>): string {
  const entityID = Number(row.entity_id || 0)
  if (entityID > 0) return `entity:${entityID}`
  return `default:${row.region}:${row.cp}`
}
function newerUpdatedAt(a?: string, b?: string): string | undefined {
  if (!a) return b
  if (!b) return a
  return a >= b ? a : b
}
function aggregateFinalNodeRateRows(rows: RateFinalNode[]): AggregatedFinalNodeRateRow[] {
  const grouped = new Map<string, AggregatedFinalNodeRateRow>()
  for (const row of rows) {
    const mode = normalizeNodeRateMode(row.settlement_mode)
    const key = finalNodeRateScopeKey(row)
    const existing = grouped.get(key)
    if (!existing) {
      grouped.set(key, { ...row, configured_modes: [mode], mode_rates: { [mode]: row } })
      continue
    }
    existing.mode_rates[mode] = row
    if (!existing.configured_modes.includes(mode)) existing.configured_modes.push(mode)
    existing.updated_at = newerUpdatedAt(existing.updated_at, row.updated_at)
  }
  return Array.from(grouped.values())
}
function hasMode(row: AggregatedFinalNodeRateRow, mode: NodeRateMode) {
  return Boolean(row.mode_rates?.[mode])
}
function displayValue(value: unknown) {
  return value === undefined || value === null || value === '' ? '-' : value
}
function modeRateValue(row: AggregatedFinalNodeRateRow, mode: NodeRateMode, key: keyof RateFinalNode) {
  return displayValue(row.mode_rates?.[mode]?.[key])
}
function rateSummary(row: AggregatedFinalNodeRateRow) {
  const parts = (['daily_95_avg', 'range_95'] as NodeRateMode[])
    .map((mode) => {
      const rate = row.mode_rates?.[mode]
      if (!rate) return ''
      return `${modeLabel(mode)}：单价 ${displayValue(rate.final_fee)}`
    })
    .filter(Boolean)
  return parts.length ? parts.join('；') : '-'
}
function feeTypeSummary(row: AggregatedFinalNodeRateRow) {
  const types = new Set<string>()
  for (const mode of ['daily_95_avg', 'range_95'] as NodeRateMode[]) {
    const value = row.mode_rates?.[mode]?.fee_type
    if (value) types.add(value)
  }
  return types.size ? Array.from(types).join(' / ') : '-'
}
function rateTooltipModes(row: AggregatedFinalNodeRateRow) {
  return (['daily_95_avg', 'range_95'] as NodeRateMode[])
    .map((mode) => {
      const rate = row.mode_rates?.[mode]
      if (!rate) return null
      return {
        mode,
        label: modeLabel(mode),
        items: [
          { label: '最终单价', value: displayValue(rate.final_fee) },
          { label: 'CP费', value: displayValue(rate.cp_fee) },
          { label: '月机柜费', value: displayValue(rate.rack_fee) },
          { label: '其他费', value: displayValue(rate.other_fee) },
          { label: '进制偏好', value: displayValue(rate.unit_base) },
          { label: '类型', value: displayValue(rate.fee_type) },
          { label: '更新时间', value: displayValue(rate.updated_at) },
        ],
      }
    })
    .filter(Boolean) as Array<{ mode: NodeRateMode; label: string; items: Array<{ label: string; value: unknown }> }>
}
function buildParams() {
  const p: any = { page: page.value, page_size: pageSize.value }
  if (query.region) p.region = query.region
  if (query.cp) p.cp = query.cp
  if (query.display_name) p.display_name = query.display_name
  if (query.settlement_mode) p.settlement_mode = query.settlement_mode
  return p
}
async function fetchData(signal?: AbortSignal) {
  loading.value = true
  try {
    const res: PaginatedData<RateFinalNode> = await api.settlementRates.finalNode.list(buildParams(), { signal })
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    if (!isAbortError(e)) ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}
function onSearch() { page.value = 1; queryCtl.run((signal) => fetchData(signal), { toggleIfRunning: true }) }
function onReset() { Object.assign(query, { region: undefined, cp: undefined, display_name: undefined, settlement_mode: undefined }); page.value = 1; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }
function onPageChange(p: number) { page.value = p; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) }
async function onInit() {
  const n = await api.settlementRates.finalNode.initFromNode()
  ElMessage.success(`初始化 ${n} 条`)
  queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
}
async function onRefresh() {
  const n = await api.settlementRates.finalNode.refresh()
  ElMessage.success(`刷新 ${n} 条`)
  queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false })
}
onMounted(() => { queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) })
usePageRefresh(() => { queryCtl.run((signal) => fetchData(signal), { showCancelMessage: false }) })
</script>

<style scoped>
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: var(--form-item-gap); }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
.mode-tag { margin-left: 6px; }
.rate-summary { display: inline-block; max-width: 100%; overflow: hidden; text-overflow: ellipsis; vertical-align: bottom; white-space: nowrap; cursor: default; }
.rate-tooltip { min-width: 260px; display: grid; gap: 10px; }
.rate-tooltip-mode + .rate-tooltip-mode { padding-top: 8px; border-top: 1px solid var(--el-border-color-lighter); }
.rate-tooltip-title { margin-bottom: 6px; font-weight: 600; color: var(--el-text-color-primary); }
.rate-tooltip-row { display: flex; justify-content: space-between; gap: 18px; line-height: 22px; color: var(--el-text-color-regular); }
.rate-tooltip-row strong { font-weight: 500; color: var(--el-text-color-primary); }
</style>
