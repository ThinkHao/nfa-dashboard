<template>
  <div class="page-container report-page">
    <div class="report-heading">
      <PageHeader title="流量报表" description="统一管理一次性导出、周期生成和预算结果。" />
      <el-button v-if="canWrite" type="primary" @click="openCreate">新建报表</el-button>
    </div>

    <section class="report-toolbar" aria-label="报表筛选">
      <el-input v-model="searchText" clearable class="search-field" placeholder="搜索报表名称或对象" :prefix-icon="Search" />
      <el-select v-model="sourceFilter" clearable placeholder="数据源" class="filter-field">
        <el-option label="NFA" value="nfa" />
        <el-option label="EDC" value="edc" />
      </el-select>
      <el-select v-model="kindFilter" clearable placeholder="任务类型" class="filter-field">
        <el-option label="一次性" value="one_off" />
        <el-option label="周期" value="periodic" />
      </el-select>
      <el-select v-model="stateFilter" clearable placeholder="状态" class="filter-field">
        <el-option label="启用" value="active" />
        <el-option label="暂停" value="paused" />
      </el-select>
      <el-checkbox v-model="budgetOnly">仅看启用预算</el-checkbox>
      <el-button v-if="hasFilters" text @click="resetFilters">清除筛选</el-button>
      <span class="toolbar-spacer" />
      <span class="toolbar-count">共 {{ filteredTasks.length }} 个报表</span>
      <el-button text :loading="loading" @click="loadTasks">刷新</el-button>
    </section>

    <section class="download-section" aria-label="统一报表下载">
      <div class="download-heading">
        <div>
          <div class="eyebrow">统一下载</div>
          <h2>按月下载全部报表</h2>
          <p>将当月成功运行的 CSV、XLSX 和元数据一次打包下载。</p>
        </div>
        <el-button text :loading="downloadsLoading" @click="loadDownloadMonths">刷新月份</el-button>
      </div>
      <div v-loading="downloadsLoading" class="download-month-list">
        <template v-if="latestDownloadMonth">
          <div class="download-month-row download-month-latest">
            <div class="download-month-main">
              <div class="download-month-label"><strong>{{ formatMonthLabel(latestDownloadMonth.month) }}</strong><el-tag size="small" type="primary" effect="plain">最新月份</el-tag></div>
              <span>{{ latestDownloadMonth.run_count }} 次成功运行 · {{ latestDownloadMonth.artifact_count }} 个文件 · {{ formatBytes(latestDownloadMonth.total_size) }}</span>
            </div>
            <el-button type="primary" plain :loading="monthlyDownload === latestDownloadMonth.month" :disabled="latestDownloadMonth.artifact_count === 0" @click="downloadMonth(latestDownloadMonth)">下载整月 ZIP</el-button>
          </div>
          <el-collapse v-if="historicalDownloadMonths.length" v-model="historicalDownloadsExpanded" class="historical-downloads">
            <el-collapse-item name="history">
              <template #title><span class="download-history-title">历史月份下载 <span class="download-history-count">{{ historicalDownloadMonths.length }} 个月</span></span></template>
              <div class="download-month-history-list">
                <div v-for="month in historicalDownloadMonths" :key="month.month" class="download-month-row">
                  <div class="download-month-main">
                    <strong>{{ formatMonthLabel(month.month) }}</strong>
                    <span>{{ month.run_count }} 次成功运行 · {{ month.artifact_count }} 个文件 · {{ formatBytes(month.total_size) }}</span>
                  </div>
                  <el-button type="primary" plain :loading="monthlyDownload === month.month" :disabled="month.artifact_count === 0" @click="downloadMonth(month)">下载整月 ZIP</el-button>
                </div>
              </div>
            </el-collapse-item>
          </el-collapse>
        </template>
        <div v-else-if="!downloadsLoading" class="download-empty">完成一次成功运行后，这里会按月份汇总可下载报表。</div>
      </div>
    </section>

    <div class="report-workspace">
      <section class="task-pane" aria-label="报表任务列表">
        <div class="pane-header">
          <div>
            <div class="eyebrow">报表计划</div>
            <h2>任务列表</h2>
          </div>
          <span class="pane-count">{{ visibleTasks.length }} / {{ filteredTasks.length }}</span>
        </div>

        <div v-loading="loading" class="task-list">
          <div
            v-for="task in visibleTasks"
            :key="task.id"
            class="task-row"
            :class="{ 'is-selected': selectedTask?.id === task.id }"
            role="button"
            tabindex="0"
            @click="selectTask(task)"
            @keydown.enter="selectTask(task)"
          >
            <div class="task-row-main">
              <div class="task-row-title">{{ task.name }}</div>
              <div class="task-row-meta">{{ taskObjectLabel(task) }} · {{ windowLabel(task) }}</div>
            </div>
            <div class="task-row-side">
              <div class="task-tags">
                <el-tag size="small" effect="plain">{{ sourceLabel(task) }}</el-tag>
                <el-tag size="small" effect="plain" :type="task.kind === 'periodic' ? 'warning' : 'info'">{{ kindLabel(task) }}</el-tag>
              </div>
              <div class="task-row-time">{{ task.kind === 'periodic' && task.next_run_at ? `下次 ${formatTime(task.next_run_at)}` : task.active ? '可执行' : '已暂停' }}</div>
            </div>
          </div>
          <div v-if="!loading && !visibleTasks.length" class="empty-state">
            <div class="empty-title">没有匹配的报表</div>
            <div class="empty-desc">调整筛选条件，或新建一个流量报表任务。</div>
            <el-button v-if="hasFilters" text type="primary" @click="resetFilters">清除筛选</el-button>
          </div>
        </div>

        <div v-if="filteredTasks.length > taskPageSize" class="task-pagination">
          <el-pagination v-model:current-page="taskPage" background layout="prev, pager, next" :page-size="taskPageSize" :total="filteredTasks.length" />
        </div>
      </section>

      <aside v-if="selectedTask" class="task-inspector" aria-label="报表详情">
        <div class="inspector-head">
          <div class="inspector-title-wrap">
            <div class="eyebrow">当前报表</div>
            <h2>{{ selectedTask.name }}</h2>
            <p>{{ taskObjectLabel(selectedTask) }} · {{ windowLabel(selectedTask) }}</p>
          </div>
          <div class="inspector-actions">
            <el-button v-if="canWrite" type="primary" plain :disabled="selectedTask.kind === 'periodic' && !selectedTask.active && !isMigratedGoV1Task(selectedTask)" @click="runTask(selectedTask)">立即执行</el-button>
            <el-button v-if="canWrite && isLegacyTask(selectedTask)" type="warning" plain @click="migrateTask(selectedTask)">迁移到 go-v1</el-button>
            <el-button text @click="showRuns(selectedTask)">运行记录</el-button>
          </div>
        </div>

        <div class="inspector-status">
          <el-tag :type="selectedTask.active ? 'success' : 'info'">{{ selectedTask.active ? '启用' : '暂停' }}</el-tag>
          <span>{{ sourceLabel(selectedTask) }} · {{ kindLabel(selectedTask) }}</span>
          <span v-if="selectedTask.next_run_at">下次执行 {{ formatTime(selectedTask.next_run_at) }}</span>
          <el-button v-if="canWrite && selectedTask.kind === 'periodic'" text size="small" @click="toggleTask(selectedTask)">{{ selectedTask.active ? '暂停计划' : '启用计划' }}</el-button>
        </div>

        <section class="inspector-section latest-section">
          <div class="section-heading"><div><div class="eyebrow">最近一次运行</div><h3>{{ latestRun ? runText(latestRun.status) : '尚未运行' }}</h3></div><span v-if="latestRun" class="section-time">{{ formatTime(latestRun.finished_at || latestRun.created_at) }}</span></div>
          <template v-if="latestRun">
            <div class="latest-metrics">
              <div><span>导出行数</span><strong>{{ latestRun.row_count || 0 }}</strong></div>
              <div><span>计算引擎</span><strong>{{ latestRun.engine_version || '—' }}</strong></div>
              <div><span>执行阶段</span><strong>{{ latestRun.progress_stage || (latestRun.status === 'success' ? '已完成' : '—') }}</strong></div>
            </div>
            <div class="run-summary-line">{{ summaryText(latestRun) }}</div>
            <el-alert v-if="latestRun.status === 'failed'" :title="latestRun.error_message || '报表运行失败'" type="error" :closable="false" class="latest-error" />
          </template>
          <div v-else class="section-empty">执行任务后，最近一次导出结果会显示在这里。</div>
        </section>

        <section class="inspector-section budget-section">
          <div class="section-heading"><div><div class="eyebrow">预算结果</div><h3>直接查看预算结算</h3></div><el-tag v-if="budgetSummary.enabled" type="success" effect="plain">已启用</el-tag><el-tag v-else type="info" effect="plain">未启用</el-tag></div>
          <template v-if="budgetSummary.enabled && budgetSummary.hasValues">
            <div class="budget-grid">
              <div class="budget-cell"><span>日95月平均 (1000)</span><strong>{{ formatBudgetValue(budgetSummary.daily1000, 1000) }}</strong></div>
              <div class="budget-cell"><span>月95 (1000)</span><strong>{{ formatBudgetValue(budgetSummary.range1000, 1000) }}</strong></div>
              <div class="budget-cell"><span>日95月平均 (1024)</span><strong>{{ formatBudgetValue(budgetSummary.daily1024, 1024) }}</strong></div>
              <div class="budget-cell"><span>月95 (1024)</span><strong>{{ formatBudgetValue(budgetSummary.range1024, 1024) }}</strong></div>
            </div>
            <div class="budget-footnote">{{ budgetSummary.yearMonth ? `结算月份：${budgetSummary.yearMonth} · ` : '' }}结果来自最近一次成功运行</div>
            <el-collapse class="budget-details">
              <el-collapse-item title="查看计算明细" name="details">
                <div class="detail-list">
                  <div><span>公式</span><code>{{ budgetSummary.formula }}</code></div>
                  <div><span>原始日95月平均</span><strong>{{ formatNumber(budgetSummary.rawDaily) }}</strong></div>
                  <div><span>原始月95</span><strong>{{ formatNumber(budgetSummary.rawRange) }}</strong></div>
                  <div><span>样本</span><strong>{{ budgetSummary.dailyDays || 0 }} 天 / {{ budgetSummary.rangePoints || 0 }} 点</strong></div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </template>
          <div v-else-if="budgetSummary.enabled" class="section-empty">预算已启用，完成一次成功运行后会显示 1000 / 1024 两套结果。</div>
          <div v-else class="section-empty">本任务没有启用数据预算。编辑任务时可设置预算换算参数。</div>
        </section>

        <section class="inspector-section task-details">
          <div class="eyebrow">任务配置</div>
          <div class="detail-list compact">
            <div><span>统计窗口</span><strong>{{ windowLabel(selectedTask) }}</strong></div>
            <div><span>输出格式</span><strong>{{ (selectedTask.export_formats || []).join('、') || '—' }}</strong></div>
            <div><span>时区</span><strong>{{ selectedTask.timezone || 'Asia/Shanghai' }}</strong></div>
            <div><span>更新时间</span><strong>{{ formatTime(selectedTask.updated_at) }}</strong></div>
          </div>
        </section>
      </aside>

      <aside v-else class="task-inspector inspector-empty">
        <div class="empty-title">选择一个报表任务</div>
        <div class="empty-desc">任务的预算结果、最近运行和导出配置会显示在这里。</div>
      </aside>
    </div>

    <el-drawer v-model="runsVisible" :title="`运行记录${selectedTask ? ` · ${selectedTask.name}` : ''}`" direction="rtl" size="min(820px, 100%)" destroy-on-close>
      <div class="drawer-intro">保留最近 50 次运行，可从这里下载对应的 CSV 或 XLSX 文件。</div>
      <el-table :data="runs" border stripe v-loading="runsLoading" class="runs-table">
        <el-table-column prop="created_at" label="创建时间" width="154"><template #default="{ row }">{{ formatTime(row.created_at) }}</template></el-table-column>
        <el-table-column prop="status" label="状态" width="78"><template #default="{ row }"><el-tag :type="runTag(row.status)" size="small">{{ runText(row.status) }}</el-tag></template></el-table-column>
        <el-table-column prop="progress_stage" label="阶段" min-width="96" />
        <el-table-column prop="row_count" label="行数" width="62" />
        <el-table-column label="摘要" min-width="170"><template #default="{ row }">{{ summaryText(row) }}</template></el-table-column>
        <el-table-column label="文件" min-width="210"><template #default="{ row }"><div v-for="artifact in (row.artifacts || [])" :key="artifact.id" class="artifact-link"><el-tooltip :content="artifact.file_name" placement="top" :show-after="250"><el-button link type="primary" class="artifact-download" :title="artifact.file_name" :aria-label="`下载 ${artifact.file_name}`" @click="download(artifact)"><span class="artifact-name">{{ artifact.file_name }}</span></el-button></el-tooltip></div><span v-if="!(row.artifacts || []).length">—</span></template></el-table-column>
      </el-table>
      <div v-if="!runsLoading && !runs.length" class="empty-state drawer-empty"><div class="empty-title">暂无运行记录</div><div class="empty-desc">点击“立即执行”生成第一份报表。</div></div>
    </el-drawer>

    <el-dialog v-model="dialogVisible" title="新建流量报表" width="720px" :close-on-click-modal="false">
      <el-form :model="form" label-width="105px">
        <el-form-item label="报表名称"><el-input v-model="form.name" placeholder="如：上月重点院校流量" /></el-form-item>
        <el-form-item label="数据源"><el-radio-group v-model="form.data_source_type"><el-radio-button label="nfa">NFA</el-radio-button><el-radio-button label="edc">EDC</el-radio-button></el-radio-group></el-form-item>
        <el-form-item label="执行方式"><el-radio-group v-model="form.kind"><el-radio-button label="one_off">立即导出</el-radio-button><el-radio-button label="periodic">周期计划</el-radio-button></el-radio-group></el-form-item>
        <el-form-item v-if="form.kind === 'periodic'" label="周期"><el-select v-model="form.schedule_type" class="field-w-160"><el-option label="每天" value="daily" /><el-option label="每周" value="weekly" /><el-option label="每月" value="monthly" /></el-select><el-input v-model="form.schedule_expr" class="schedule-input" placeholder="每天填 HH:MM；每周填 1 HH:MM；每月填 1 HH:MM" /></el-form-item>
        <el-form-item label="统计窗口"><el-select v-model="form.window_selector" class="field-w-180"><el-option label="自定义" value="custom" /><el-option label="上周" value="last_week" /><el-option label="上月" value="last_month" /><el-option label="最近N天" value="last_n_days" /></el-select></el-form-item>
        <el-form-item v-if="form.window_selector === 'custom'" label="时间范围"><el-date-picker v-model="customRange" type="datetimerange" value-format="YYYY-MM-DD HH:mm:ss" format="YYYY-MM-DD HH:mm:ss" /></el-form-item>
        <el-form-item v-if="form.window_selector === 'last_n_days'" label="天数"><el-input-number v-model="lastDays" :min="1" :max="365" /></el-form-item>
        <el-form-item label="对象筛选"><el-input v-model="form.params.school_name" :placeholder="form.data_source_type === 'nfa' ? '学校名称（可选）' : 'EDC对象ID，逗号分隔（可选）'" /></el-form-item>
        <el-form-item v-if="form.data_source_type === 'nfa'" label="地区/CP"><el-input v-model="form.params.region" placeholder="地区（可选）" class="field-w-180" /><el-input v-model="form.params.cp" placeholder="CP，逗号分隔（可选）" class="field-w-220" /></el-form-item>
        <el-form-item label="方向"><el-select v-model="form.params.direction" class="field-w-140"><el-option label="收发合计" value="both" /><el-option label="接收" value="recv" /><el-option label="发送" value="send" /></el-select><el-select v-model="form.params.unit_base" class="field-w-140 unit-select"><el-option label="1000 进制" :value="1000" /><el-option label="1024 进制" :value="1024" /></el-select></el-form-item>
        <el-form-item label="流量预算"><el-checkbox v-model="form.params.data_budget_enabled">启用预算换算</el-checkbox><el-input-number v-if="form.params.data_budget_enabled" v-model="form.params.data_budget_mul" :min="0.0001" :step="1" class="budget-input" /><span v-if="form.params.data_budget_enabled" class="formula-text">÷</span><el-input-number v-if="form.params.data_budget_enabled" v-model="form.params.data_budget_div" :min="0.0001" :step="1" /></el-form-item>
        <el-form-item label="导出格式"><el-checkbox-group v-model="form.export_formats"><el-checkbox label="csv">CSV</el-checkbox><el-checkbox label="xlsx">XLSX</el-checkbox></el-checkbox-group></el-form-item>
        <el-alert title="每次运行会保存实际时间窗口、对象范围、原始单位和计算公式；预算结果不会写入正式结算表。" type="info" :closable="false" />
      </el-form>
      <template #footer><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" :loading="submitting" @click="submit">创建</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '@/components/ui/PageHeader.vue'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useTasksStore } from '@/stores/tasks'
import type { TrafficReportArtifact, TrafficReportDownloadMonth, TrafficReportRun, TrafficReportTask } from '@/types/api'
import { triggerBlobDownload } from '@/utils/export'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const taskStore = useTasksStore()
const canWrite = computed(() => auth.hasPermission('traffic.report.write'))
const loading = ref(false)
const runsLoading = ref(false)
const submitting = ref(false)
const downloadsLoading = ref(false)
const monthlyDownload = ref('')
const dialogVisible = ref(false)
const runsVisible = ref(false)
const tasks = ref<TrafficReportTask[]>([])
const selectedTask = ref<TrafficReportTask | null>(null)
const runs = ref<TrafficReportRun[]>([])
const downloadMonths = ref<TrafficReportDownloadMonth[]>([])
const historicalDownloadsExpanded = ref<string[]>([])
const searchText = ref('')
const sourceFilter = ref('')
const kindFilter = ref('')
const stateFilter = ref('')
const budgetOnly = ref(false)
const taskPage = ref(1)
const taskPageSize = 12
const customRange = ref<[string, string] | null>(null)
const lastDays = ref(7)
const form = reactive<any>({ name: '', data_source_type: 'nfa', kind: 'one_off', schedule_type: 'daily', schedule_expr: '02:00', window_selector: 'last_month', params: { school_name: '', region: '', cp: '', direction: 'both', unit_base: 1024, data_budget_enabled: false, data_budget_mul: 8, data_budget_div: 300 }, export_formats: ['csv'] })
const pollingRuns = new Set<string>()

const hasFilters = computed(() => Boolean(searchText.value || sourceFilter.value || kindFilter.value || stateFilter.value || budgetOnly.value))
const filteredTasks = computed(() => {
  const keyword = searchText.value.trim().toLowerCase()
  return tasks.value.filter((task) => {
    const params = task.params || {}
    const objectText = String(params.school_name || params.edc_name || (Array.isArray(params.entity_ids) ? params.entity_ids.join(',') : '')).toLowerCase()
    if (keyword && !task.name.toLowerCase().includes(keyword) && !objectText.includes(keyword)) return false
    if (sourceFilter.value && task.data_source_type !== sourceFilter.value) return false
    if (kindFilter.value && task.kind !== kindFilter.value) return false
    if (stateFilter.value === 'active' && !task.active) return false
    if (stateFilter.value === 'paused' && task.active) return false
    if (budgetOnly.value && !isBudgetEnabled(task)) return false
    return true
  })
})
const visibleTasks = computed(() => filteredTasks.value.slice((taskPage.value - 1) * taskPageSize, taskPage.value * taskPageSize))
const sortedDownloadMonths = computed(() => downloadMonths.value.slice().sort((a, b) => b.month.localeCompare(a.month)))
const latestDownloadMonth = computed(() => sortedDownloadMonths.value[0] || null)
const historicalDownloadMonths = computed(() => sortedDownloadMonths.value.slice(1))
const latestRun = computed(() => runs.value.slice().sort(runSort)[0] || null)
const latestSuccessfulRun = computed(() => runs.value.filter((run) => run.status === 'success').slice().sort(runSort)[0] || null)
const budgetSummary = computed(() => {
  const run = latestSuccessfulRun.value
  const summary = run?.summary || {}
  const params = selectedTask.value?.params || {}
  const enabled = summary.budget_enabled != null ? Boolean(summary.budget_enabled) : isBudgetEnabled(selectedTask.value)
  const base = Number(summary.unit_base || params.unit_base || 1024)
  const oldDaily = summary.budget_daily_avg_value
  const oldRange = summary.budget_range_value
  const daily1000 = numberOrNull(summary.daily_95_avg_1000) ?? (base === 1000 ? numberOrNull(oldDaily) : null)
  const range1000 = numberOrNull(summary.range_95_1000) ?? (base === 1000 ? numberOrNull(oldRange) : null)
  const daily1024 = numberOrNull(summary.daily_95_avg_1024) ?? (base === 1024 ? numberOrNull(oldDaily) : null)
  const range1024 = numberOrNull(summary.range_95_1024) ?? (base === 1024 ? numberOrNull(oldRange) : null)
  return { enabled, hasValues: [daily1000, range1000, daily1024, range1024].some((value) => value != null), daily1000, range1000, daily1024, range1024, formula: summary.budget_formula || summary.formula || budgetFormula(params), rawDaily: numberOrNull(summary.raw_daily_95_avg), rawRange: numberOrNull(summary.raw_range_95), dailyDays: summary.budget_daily_days || summary.daily_days, rangePoints: summary.budget_range_points || summary.range_points, yearMonth: summary.year_month || String(summary.window_end || '').slice(0, 7) }
})

function numberOrNull(value: unknown): number | null { const number = Number(value); return value == null || Number.isNaN(number) ? null : number }
function isBudgetEnabled(task: TrafficReportTask | null) { const value = task?.params?.data_budget_enabled; return value === true || value === 1 || value === '1' || value === 'true' }
function budgetFormula(params: Record<string, any>) { return `raw*${params.data_budget_mul || 8}/${params.data_budget_div || 300}/base/base` }
function runSort(a: TrafficReportRun, b: TrafficReportRun) { return Date.parse(b.created_at || b.started_at || '') - Date.parse(a.created_at || a.started_at || '') }
function formatTime(value?: string) { if (!value) return '—'; const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleString() }
function formatNumber(value: unknown) { const number = numberOrNull(value); return number == null ? '—' : number.toFixed(3) }
function formatBudgetValue(value: unknown, base: number) { const number = numberOrNull(value); return number == null ? '—' : `${(number / base).toFixed(4)} Gbps` }
function formatMonthLabel(month: string) { const [year, rawMonth] = month.split('-'); return year && rawMonth ? `${year}年${rawMonth}月` : month }
function formatBytes(value: number) { if (!Number.isFinite(value) || value < 1024) return `${Math.max(0, Number(value) || 0)} B`; const units = ['KB', 'MB', 'GB', 'TB']; let size = value; let index = -1; do { size /= 1024; index += 1 } while (size >= 1024 && index < units.length - 1); return `${size.toFixed(size >= 10 ? 0 : 1)} ${units[index]}` }
function sourceLabel(task: TrafficReportTask) { return task.data_source_type === 'edc' ? 'EDC' : 'NFA' }
function kindLabel(task: TrafficReportTask) { return task.kind === 'periodic' ? '周期计划' : '一次性' }
function windowLabel(task: TrafficReportTask) { const labels: Record<string, string> = { last_week: '上周', last_month: '上月', last_n_days: `最近${task.window_params?.n || 'N'}天`, custom: '自定义窗口' }; return labels[task.window_selector] || task.window_selector || '—' }
function taskObjectLabel(task: TrafficReportTask) { const params = task.params || {}; const objectText = params.school_name || params.edc_name || (Array.isArray(params.entity_ids) ? `${params.entity_ids.length} 个对象` : '全部对象'); return String(objectText) }
function isLegacyTask(task: TrafficReportTask | null) { const marker = task?.params?.legacy_nfatool; return Boolean(marker && (marker.source_task_id != null || marker.original_params)) }
function isMigratedGoV1Task(task: TrafficReportTask | null) { return Boolean(task?.params?._traffic_report_migration) }
function runTag(status: string) { return status === 'success' ? 'success' : status === 'failed' ? 'danger' : status === 'running' ? 'warning' : 'info' }
function runText(status: string) { return ({ pending: '等待中', running: '执行中', success: '完成', failed: '失败' } as Record<string, string>)[status] || status }
function summaryText(run: TrafficReportRun) { const summary = run.summary || {}; if (summary.range_95_mbps != null) { const budget = summary.budget_range_value != null ? ` · 预算95 ${formatNumber(summary.budget_range_value)}` : ''; return `区间95 ${formatNumber(summary.range_95_mbps)} Mbps · 日95平均 ${formatNumber(summary.daily_95_avg_mbps)} Mbps${budget}` }; if (summary.raw_rows != null) return `原始行 ${summary.raw_rows} · ${summary.instance || summary.source || '导出'}`; if (summary.range_points != null) return `${summary.range_points} 个点位 · ${summary.daily_days || 0} 天`; return run.error_message || '—' }

async function loadTasks() {
  loading.value = true
  try {
    const all: TrafficReportTask[] = []
    let page = 1
    let total = 0
    do { const res = await api.trafficReports.listTasks({ page, page_size: 200 }); all.push(...(res.items || [])); total = Number(res.total || all.length); page += 1 } while (all.length < total && page < 20)
    tasks.value = all
    if (selectedTask.value) selectedTask.value = all.find((task) => task.id === selectedTask.value?.id) || null
    if (!selectedTask.value && all.length) selectTask(all[0])
  } catch (error: any) { ElMessage.error(error?.response?.data?.message || error?.message || '加载报表计划失败') } finally { loading.value = false }
}
async function loadDownloadMonths() {
  downloadsLoading.value = true
  try { const res = await api.trafficReports.listDownloadMonths(); downloadMonths.value = res.items || [] } catch (error: any) { ElMessage.error(error?.response?.data?.message || error?.message || '加载统一下载列表失败') } finally { downloadsLoading.value = false }
}
async function loadRuns() {
  if (!selectedTask.value) return
  runsLoading.value = true
  try { const res = await api.trafficReports.listRuns({ task_id: selectedTask.value.id, page: 1, page_size: 50 }); runs.value = res.items || []; for (const run of runs.value) if (run.status === 'pending' || run.status === 'running') trackRun(run.id, selectedTask.value.name, selectedTask.value.id) } catch (error: any) { ElMessage.error(error?.response?.data?.message || error?.message || '加载运行记录失败') } finally { runsLoading.value = false }
}
function selectTask(task: TrafficReportTask) { selectedTask.value = task; void loadRuns() }
function showRuns(task: TrafficReportTask) { selectTask(task); runsVisible.value = true }
function resetFilters() { searchText.value = ''; sourceFilter.value = ''; kindFilter.value = ''; stateFilter.value = ''; budgetOnly.value = false }
function openCreate() { const query = route.query; if (query.data_source_type === 'edc' || query.data_source_type === 'nfa') form.data_source_type = String(query.data_source_type); if (query.entity_ids) form.params.school_name = String(query.entity_ids); else if (query.school_name) form.params.school_name = String(query.school_name); if (query.region) form.params.region = String(query.region); if (query.cp) form.params.cp = String(query.cp); if (query.start_time && query.end_time) { form.window_selector = 'custom'; customRange.value = [String(query.start_time), String(query.end_time)] }; dialogVisible.value = true }
function reportPayload() { const windowParams: any = {}; if (form.window_selector === 'custom' && customRange.value) { windowParams.start_time = customRange.value[0]; windowParams.end_time = customRange.value[1] }; if (form.window_selector === 'last_n_days') windowParams.n = lastDays.value; const params = { ...form.params }; if (form.data_source_type === 'edc' && params.school_name) { params.entity_ids = String(params.school_name).split(',').map(Number).filter((number: number) => number > 0); delete params.school_name }; return { name: form.name, kind: form.kind, data_source_type: form.data_source_type, schedule_type: form.kind === 'periodic' ? form.schedule_type : undefined, schedule_expr: form.kind === 'periodic' ? form.schedule_expr : undefined, window_selector: form.window_selector, window_params: windowParams, params, export_formats: form.export_formats, timezone: 'Asia/Shanghai' } }
function trackRun(runId: string, title: string, taskId?: number) {
  if (!runId || pollingRuns.has(runId)) return
  pollingRuns.add(runId)
  const id = `traffic-report:${runId}`
  taskStore.start({ id, type: 'export', title, status: 'pending', progress: 0, info: '已提交报表任务' })
  const poll = async () => {
    try { const res = await api.trafficReports.getRun(runId); const run = res.run; const status = run.status === 'success' ? 'success' : run.status === 'failed' ? 'failed' : 'running'; taskStore.update(id, { status, progress: Math.max(0, Math.min(1, Number(run.progress_pct || 0) / 100)), stage: run.progress_stage || undefined, info: run.error_message || run.progress_stage || '执行中' }); if (run.status === 'success' || run.status === 'failed') { pollingRuns.delete(runId); if (run.status === 'success') void loadDownloadMonths(); if (selectedTask.value?.id === taskId) void loadRuns(); window.setTimeout(() => taskStore.remove(id), 15000); return } } catch { /* 运行记录会显示最终错误 */ }
    window.setTimeout(poll, 1600)
  }
  void poll()
}
async function submit() { if (!form.name.trim()) { ElMessage.warning('请填写报表名称'); return }; if (form.window_selector === 'custom' && (!customRange.value || customRange.value.length !== 2)) { ElMessage.warning('请选择自定义时间范围'); return }; if (!form.export_formats.length) { ElMessage.warning('至少选择一种导出格式'); return }; submitting.value = true; try { const res = await api.trafficReports.createTask(reportPayload()); ElMessage.success(form.kind === 'one_off' ? '报表任务已创建' : '周期计划已创建'); dialogVisible.value = false; await loadTasks(); if (res.run_id) { trackRun(res.run_id, form.name, res.task?.id); if (res.task) selectTask(res.task) } } catch (error: any) { ElMessage.error(error?.response?.data?.message || error?.message || '创建报表失败') } finally { submitting.value = false } }
async function runTask(task: TrafficReportTask) { try { const res = await api.trafficReports.runTask(task.id); trackRun(res.run_id, task.name, task.id); selectedTask.value = task; await loadRuns(); ElMessage.success('已提交运行') } catch (error: any) { ElMessage.error(error?.response?.data?.message || error?.message || '提交运行失败') } }
async function toggleTask(task: TrafficReportTask) { try { await api.trafficReports.updateTask(task.id, { active: !task.active }); await loadTasks(); ElMessage.success(task.active ? '已暂停计划' : '已启用计划') } catch (error: any) { ElMessage.error(error?.response?.data?.message || error?.message || '更新计划失败') } }
async function migrateTask(task: TrafficReportTask) {
  try {
    await ElMessageBox.confirm('原 nfatool 任务会保留不变，并创建一个默认暂停的 go-v1 新任务。完成结果比对后，再启用新任务。', '迁移到 go-v1', { confirmButtonText: '创建迁移任务', cancelButtonText: '取消', type: 'warning' })
    const res = await api.trafficReports.migrateTaskToGoV1(task.id)
    await loadTasks()
    const migrated = tasks.value.find((item) => item.id === res.task.id)
    if (migrated) selectTask(migrated)
    const warnings = res.warnings || []
    if (warnings.length) await ElMessageBox.alert(warnings.join('\n'), '迁移任务已创建', { confirmButtonText: '知道了', type: 'success' })
    else ElMessage.success('迁移任务已创建，当前处于暂停状态')
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error?.response?.data?.message || error?.message || '创建迁移任务失败')
  }
}
async function download(artifact: TrafficReportArtifact) { try { const blob = await api.trafficReports.downloadArtifact(artifact.id); triggerBlobDownload(blob, artifact.file_name) } catch (error: any) { ElMessage.error(error?.response?.data?.message || error?.message || '下载失败') } }
async function downloadMonth(month: TrafficReportDownloadMonth) { if (!month.artifact_count || monthlyDownload.value) return; monthlyDownload.value = month.month; try { const blob = await api.trafficReports.downloadMonthlyArchive(month.month); triggerBlobDownload(blob, `traffic-reports-${month.month}.zip`); ElMessage.success(`${formatMonthLabel(month.month)}报表已打包`) } catch (error: any) { ElMessage.error(error?.response?.data?.message || error?.message || '整月报表下载失败') } finally { monthlyDownload.value = '' } }
watch([searchText, sourceFilter, kindFilter, stateFilter, budgetOnly], () => { taskPage.value = 1 })
onMounted(() => { void loadTasks(); void loadDownloadMonths(); if (route.query.create === '1') openCreate(); if (route.query.create) router.replace({ path: '/traffic-reports', query: {} }) })
</script>

<style scoped>
.report-heading { display: flex; justify-content: space-between; align-items: flex-start; gap: 20px; }
.report-toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; padding: 12px 14px; margin-bottom: var(--space-4); background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 12px; }
.search-field { width: 250px; }
.filter-field { width: 126px; }
.toolbar-spacer { flex: 1 1 auto; }
.toolbar-count, .pane-count { color: var(--text-muted); font-size: 12px; }
.download-section { margin-bottom: var(--space-4); padding: 18px 20px 8px; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 14px; }
.download-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding-bottom: 12px; }
.download-heading h2 { margin: 3px 0 0; color: var(--text-default); font-size: 18px; line-height: 1.35; }
.download-heading p { margin: 6px 0 0; color: var(--text-muted); font-size: 12px; }
.download-month-list { min-height: 46px; }
.download-month-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 12px 0; border-top: 1px solid var(--border-color); }
.download-month-main { display: flex; min-width: 0; flex-direction: column; gap: 5px; }
.download-month-main strong { color: var(--text-default); font-size: 15px; font-variant-numeric: tabular-nums; }
.download-month-label { display: flex; align-items: center; gap: 8px; }
.download-month-main span { color: var(--text-muted); font-size: 12px; }
.download-month-latest { padding-top: 14px; padding-bottom: 16px; }
.historical-downloads { border-top: 1px solid var(--border-color); border-bottom: 0; }
.historical-downloads :deep(.el-collapse-item__header) { height: 44px; color: var(--text-default); font-size: 13px; font-weight: 600; }
.historical-downloads :deep(.el-collapse-item__wrap) { border-bottom: 0; }
.historical-downloads :deep(.el-collapse-item__content) { padding-bottom: 4px; }
.download-history-title { display: flex; align-items: center; gap: 8px; }
.download-history-count { color: var(--text-muted); font-size: 12px; font-weight: 400; }
.download-empty { padding: 14px 0; color: var(--text-muted); font-size: 12px; }
.report-workspace { display: grid; grid-template-columns: minmax(0, 1.15fr) minmax(360px, .85fr); gap: 16px; align-items: stretch; }
.task-pane, .task-inspector { min-width: 0; background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 14px; }
.task-pane { display: flex; flex-direction: column; min-height: 650px; overflow: hidden; }
.pane-header, .inspector-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding: 18px 20px; border-bottom: 1px solid var(--border-color); }
.pane-header h2, .inspector-head h2 { margin: 3px 0 0; color: var(--text-default); font-size: 18px; line-height: 1.35; }
.eyebrow { color: var(--text-muted); font-size: 11px; font-weight: 600; letter-spacing: .08em; text-transform: uppercase; }
.task-list { flex: 1; min-height: 0; padding: 6px 8px; }
.task-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 14px 12px; border-left: 3px solid transparent; border-bottom: 1px solid var(--border-color); cursor: pointer; transition: background-color .16s ease, border-color .16s ease, transform .16s ease; }
.task-row:hover { background: color-mix(in srgb, var(--color-primary) 5%, transparent); transform: translateX(2px); }
.task-row.is-selected { background: color-mix(in srgb, var(--color-primary) 9%, var(--bg-card)); border-left-color: var(--color-primary); }
.task-row-main { min-width: 0; }
.task-row-title { overflow: hidden; color: var(--text-default); font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.task-row-meta, .task-row-time { margin-top: 6px; color: var(--text-muted); font-size: 12px; }
.task-row-side { display: flex; flex: 0 0 auto; flex-direction: column; align-items: flex-end; }
.task-tags { display: flex; gap: 6px; }
.task-row-time { max-width: 190px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.task-pagination { display: flex; justify-content: flex-end; padding: 12px 16px; border-top: 1px solid var(--border-color); }
.task-inspector { min-height: 650px; overflow: hidden; }
.inspector-title-wrap { min-width: 0; }
.inspector-title-wrap h2 { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.inspector-title-wrap p { margin: 7px 0 0; color: var(--text-muted); font-size: 12px; }
.inspector-actions { display: flex; flex: 0 0 auto; flex-wrap: wrap; gap: 4px; align-items: center; justify-content: flex-end; }
.inspector-status { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; padding: 12px 20px; color: var(--text-muted); font-size: 12px; border-bottom: 1px solid var(--border-color); }
.inspector-status .el-button { margin-left: auto; }
.inspector-section { padding: 18px 20px; border-bottom: 1px solid var(--border-color); }
.section-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.section-heading h3 { margin: 4px 0 0; color: var(--text-default); font-size: 15px; }
.section-time { color: var(--text-muted); font-size: 12px; white-space: nowrap; }
.latest-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; margin-top: 16px; }
.latest-metrics div, .budget-cell { padding: 11px 12px; background: color-mix(in srgb, var(--bg-page) 66%, var(--bg-card)); border-radius: 8px; }
.latest-metrics span, .budget-cell span { display: block; color: var(--text-muted); font-size: 11px; }
.latest-metrics strong { display: block; margin-top: 5px; overflow: hidden; color: var(--text-default); font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
.run-summary-line { margin-top: 13px; color: var(--text-muted); font-size: 12px; line-height: 1.6; }
.latest-error { margin-top: 12px; }
.budget-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin-top: 16px; }
.budget-cell strong { display: block; margin-top: 6px; color: var(--color-primary); font-size: 20px; font-variant-numeric: tabular-nums; }
.budget-footnote { margin-top: 10px; color: var(--text-muted); font-size: 11px; }
.budget-details { margin-top: 12px; border-top: 0; border-bottom: 0; }
.detail-list { display: grid; gap: 9px; margin-top: 12px; }
.detail-list div { display: flex; justify-content: space-between; gap: 14px; color: var(--text-muted); font-size: 12px; }
.detail-list strong { color: var(--text-default); font-weight: 500; text-align: right; }
.detail-list code { color: var(--text-default); font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 11px; }
.detail-list.compact { margin-top: 14px; }
.section-empty, .empty-desc { margin-top: 13px; color: var(--text-muted); font-size: 12px; line-height: 1.6; }
.inspector-empty { display: flex; flex-direction: column; justify-content: center; align-items: center; padding: 30px; text-align: center; }
.empty-title { color: var(--text-default); font-weight: 600; }
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 220px; padding: 24px; text-align: center; }
.empty-state .el-button { margin-top: 12px; }
.drawer-intro { margin-bottom: 14px; color: var(--text-muted); font-size: 12px; line-height: 1.6; }
.drawer-empty { min-height: 240px; }
.runs-table { width: 100%; }
.artifact-link { max-width: 100%; min-width: 0; }
.artifact-download { display: flex; width: 100%; max-width: 100%; height: auto; min-height: 24px; padding-right: 0; padding-left: 0; text-align: left; white-space: normal !important; }
.artifact-name { display: block; max-width: 100%; overflow-wrap: anywhere; white-space: normal !important; word-break: break-word; }
.schedule-input { margin-left: 10px; width: 350px; }
.unit-select { margin-left: 10px; }
.budget-input { margin-left: 10px; }
.formula-text { margin: 0 8px; color: var(--text-muted); }
@media (max-width: 1050px) { .report-workspace { grid-template-columns: 1fr; } .task-pane, .task-inspector { min-height: auto; } .task-pane { max-height: 660px; } }
@media (max-width: 680px) { .report-heading { align-items: stretch; flex-direction: column; } .report-toolbar > * { flex: 1 1 140px; } .report-toolbar .toolbar-spacer { display: none; } .search-field, .filter-field { width: auto; } .download-heading, .download-month-row { align-items: flex-start; flex-direction: column; } .download-month-row .el-button { width: 100%; } .task-row { align-items: flex-start; flex-direction: column; gap: 8px; } .task-row-side { align-items: flex-start; } .latest-metrics, .budget-grid { grid-template-columns: 1fr 1fr; } .inspector-head { flex-direction: column; } }
</style>
