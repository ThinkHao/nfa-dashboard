<template>
  <div class="settlement-tasks-tab">
    <!-- 筛选条件区域 -->
    <el-card class="filter-section">
      <el-form :model="filterForm" inline>
        <el-form-item label="任务类型">
          <el-select v-model="filterForm.task_type" placeholder="选择任务类型" clearable class="field-w-180">
            <el-option label="日结算" value="daily" />
            <el-option label="周结算" value="weekly" />
            <el-option label="节点日95" value="node_daily95" />
            <el-option label="节点月95" value="node_monthly95" />
            <el-option label="初算" value="customer_init" />
            <el-option label="复算" value="customer_recalc" />
          </el-select>
        </el-form-item>
        <el-form-item label="任务状态">
          <el-select v-model="filterForm.status" placeholder="选择任务状态" clearable class="field-w-160">
            <el-option label="等待中" value="pending" />
            <el-option label="执行中" value="running" />
            <el-option label="已完成" value="success" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item label="日期范围">
          <UnifiedDateRange
            v-model="dateRange"
            type="daterange"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </el-form-item>
        <el-form-item>
          <QueryActionButton :running="queryCtl.running.value" @trigger="onSearch" />
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 操作按钮区域 -->
    <div class="action-section">
      <el-button type="primary" @click="createDailyTask">创建日结算任务</el-button>
      <el-button type="success" @click="createWeeklyTask">创建周结算任务</el-button>
      <el-button type="warning" @click="createNodeDailyTask">创建节点日95任务</el-button>
      <el-button type="info" @click="createNodeMonthlyTask">创建节点月95任务</el-button>
    </div>

    <!-- 任务表格区域 -->
    <el-card class="table-section">
      <template #header>
        <div class="table-header">
          <h3 class="card-title">结算任务列表</h3>
        </div>
      </template>
      <el-table
        v-loading="loading"
        :data="taskData.items"
        border
        stripe
        class="field-w-full"
      >
        <el-table-column prop="id" label="任务ID" width="80" />
        <el-table-column prop="task_type" label="任务类型" width="120">
          <template #default="scope">
            {{ taskTypeLabel(scope.row.task_type) }}
          </template>
        </el-table-column>
        <el-table-column prop="task_date" label="任务日期" width="180">
          <template #default="scope">
            {{ formatDate(scope.row.task_date) }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="scope">
            <el-tag :type="getStatusType(scope.row.status)">
              {{ getStatusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="processed_count" label="处理记录数" width="120" />
        <el-table-column prop="start_time" label="开始时间" width="180">
          <template #default="scope">
            {{ formatDateTime(scope.row.start_time) }}
          </template>
        </el-table-column>
        <el-table-column prop="end_time" label="结束时间" width="180">
          <template #default="scope">
            {{ formatDateTime(scope.row.end_time) }}
          </template>
        </el-table-column>
        <el-table-column prop="create_time" label="创建时间" width="180">
          <template #default="scope">
            {{ formatDateTime(scope.row.create_time) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="scope">
            <el-button
              size="small"
              type="primary"
              @click="viewTaskDetail(scope.row)"
              :disabled="loading"
            >
              详情
            </el-button>
            <el-button
              size="small"
              type="danger"
              @click="deleteTask(scope.row)"
              :disabled="scope.row.status === 'running' || loading"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="taskData.total"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 任务详情对话框 -->
    <el-dialog
      v-model="taskDetailVisible"
      title="任务详情"
      width="600px"
    >
      <div v-if="currentTask" class="task-detail">
        <div class="detail-item">
          <span class="label">任务ID:</span>
          <span class="value">{{ currentTask.id }}</span>
        </div>
        <div class="detail-item">
          <span class="label">任务类型:</span>
          <span class="value">{{ taskTypeLabel(currentTask.task_type) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">任务日期:</span>
          <span class="value">{{ formatDate(currentTask.task_date) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">状态:</span>
          <span class="value">
            <el-tag :type="getStatusType(currentTask.status)">
              {{ getStatusText(currentTask.status) }}
            </el-tag>
          </span>
        </div>
        <div class="detail-item">
          <span class="label">处理记录数:</span>
          <span class="value">{{ currentTask.processed_count }}</span>
        </div>
        <div class="detail-item">
          <span class="label">开始时间:</span>
          <span class="value">{{ formatDateTime(currentTask.start_time) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">结束时间:</span>
          <span class="value">{{ formatDateTime(currentTask.end_time) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">创建时间:</span>
          <span class="value">{{ formatDateTime(currentTask.create_time) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">更新时间:</span>
          <span class="value">{{ formatDateTime(currentTask.update_time) }}</span>
        </div>
        <div v-if="currentTask.error_message" class="detail-item">
          <span class="label">错误信息:</span>
          <div class="error-message">{{ currentTask.error_message }}</div>
        </div>
      </div>
    </el-dialog>

    <!-- 创建任务对话框 -->
    <el-dialog
      v-model="createTaskVisible"
      :title="taskDialogTitle"
      width="500px"
      :close-on-click-modal="!submitting"
      :close-on-press-escape="!submitting"
      :show-close="!submitting"
      :before-close="beforeCloseCreateTask"
      @closed="onCreateTaskDialogClosed"
    >
      <el-form :model="taskForm" label-width="100px">
        <!-- 日结算任务显示单日选择器 -->
        <el-form-item v-if="taskForm.type === 'daily'" label="任务日期">
          <el-date-picker
            v-model="taskForm.date"
            type="date"
            placeholder="选择日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
          />
        </el-form-item>

        <el-form-item v-else-if="taskForm.type === 'node_daily95'" label="任务日期范围">
          <UnifiedDateRange
            v-model="taskForm.dateRange"
            type="daterange"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
          />
        </el-form-item>
        
        <!-- 周结算任务显示日期范围选择器 -->
        <el-form-item v-else-if="taskForm.type === 'weekly'" label="周日期范围">
          <UnifiedDateRange
            v-model="taskForm.dateRange"
            type="daterange"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </el-form-item>
        <el-form-item v-else label="服务月份范围">
          <UnifiedDateRange
            v-model="taskForm.monthRange"
            type="monthrange"
            format="YYYY-MM"
            value-format="YYYY-MM"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button :disabled="submitting" @click="createTaskVisible = false">取消</el-button>
          <el-button type="primary" @click="submitTaskCreate" :loading="submitting">
            确认
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import api from '../../api'
import { useTasksStore } from '@/stores/tasks'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { TaskListResponse, SettlementTask, TaskStatus } from '../../types/settlement'
import UnifiedDateRange from '@/components/ui/UnifiedDateRange.vue'
import { buildSettlementDayRange, normalizeSettlementDayRange, splitSettlementDayRange } from './settlement-day-range'
import { expandNodeDailyTaskRange, expandNodeMonthlyTaskRange } from './settlement-task-batch-range'
import QueryActionButton from '@/components/ui/QueryActionButton.vue'
import { useCancelableQuery, isAbortError } from '@/composables/useCancelableQuery'
import { usePageRefresh } from '@/composables/usePageRefresh'
import { cleanupStaleElementOverlays } from '@/utils/overlayCleanup'

// 估算结算任务总工作量：按可见学校的 school_id/region/cp 唯一组合数量
const combosTotal = ref<number | null>(null)
const ensureCombosTotal = async (): Promise<number | null> => {
  if (combosTotal.value != null) return combosTotal.value
  try {
    let offset = 0
    const limit = 1000
    const set = new Set<string>()
    while (true) {
      const res: any = await (api as any).v2.getSchools({ limit, offset })
      const items = Array.isArray(res) ? res : (Array.isArray(res?.items) ? res.items : [])
      for (const s of items) {
        if (!s) continue
        const sid = s.school_id || s.schoolId || ''
        const region = s.region || ''
        const cp = s.cp || ''
        if (sid && region && cp) set.add(`${sid}__${region}__${cp}`)
      }
      if (items.length < limit) break
      offset += limit
    }
    combosTotal.value = set.size
    return combosTotal.value
  } catch {
    combosTotal.value = null
    return null
  }
}

// 筛选表单
const filterForm = reactive({
  task_type: '',
  status: '',
  start_date: '',
  limit: 10,
  offset: 0,
  end_date: '',
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
const submitting = ref(false)

// 任务数据
const taskData = ref<TaskListResponse>({
  items: [],
  total: 0
})
const queryCtl = useCancelableQuery()
const tasksStore = useTasksStore()

// 当前选中的任务
const currentTask = ref<SettlementTask | null>(null)

// 对话框显示状态
const taskDetailVisible = ref(false)
const createTaskVisible = ref(false)
const taskDialogTitle = ref('创建结算任务')

// 任务表单
const taskForm = reactive({
  type: 'daily',
  date: '',
  month: '',
  dateRange: null as [string, string] | null,
  monthRange: null as [string, string] | null
})

function taskTypeLabel(type: string) {
  switch (type) {
    case 'daily': return '院校日结算'
    case 'weekly': return '院校周结算'
    case 'node_daily95': return '节点日95'
    case 'node_monthly95': return '节点月95'
    case 'customer_init': return '初算'
    case 'customer_recalc': return '复算'
    default: return type
  }
}

// 获取任务列表
const syncDateRangeFromFilter = () => {
  dateRange.value = buildSettlementDayRange(filterForm.start_date, filterForm.end_date)
}

const fetchTasks = async (signal?: AbortSignal) => {
  loading.value = true
  
  // 处理日期范围
  if (dateRange.value) {
    const { start, end } = splitSettlementDayRange(dateRange.value)
    filterForm.start_date = start
    filterForm.end_date = end
  } else {
    filterForm.start_date = ''
    filterForm.end_date = ''
  }
  syncDateRangeFromFilter()

  // 设置分页参数
  filterForm.page = currentPage.value
  filterForm.page_size = pageSize.value

  // 后端分页参数使用 limit/offset
  const limit = pageSize.value
  const offset = (currentPage.value - 1) * pageSize.value

  try {
    const response = await api.settlement.getTasks({
      ...filterForm,
      limit,
      offset,
    }, { signal }) as any
    // 统一仅处理数组或 { items, total }
    if (Array.isArray(response)) {
      taskData.value = { items: response, total: response.length }
    } else if (response && Array.isArray(response.items)) {
      taskData.value = { items: response.items, total: Number(response.total) || response.items.length }
    } else {
      taskData.value = { items: [], total: 0 }
    }
    const needTotal = taskData.value.items.some(t => t.status === 'pending' || t.status === 'running')
    let estTotal: number | null = null
    if (needTotal) {
      estTotal = await ensureCombosTotal()
    }
    for (const t of taskData.value.items) {
      // 优先使用后端返回的 precise total_count，其次使用前端估算
      let total_count: number | undefined
      if (typeof (t as any).total_count === 'number' && (t as any).total_count > 0) {
        total_count = Number((t as any).total_count)
      } else if (estTotal && (t.task_type === 'daily' || t.task_type === 'weekly')) {
        total_count = t.task_type === 'daily' ? estTotal : estTotal * 7
      }
      tasksStore.upsertSettlementTask({ id: t.id, status: t.status as any, processed_count: t.processed_count, total_count })
    }
    
    // 检查是否有进行中的任务，如果有则启动自动刷新
    if (hasRunningTasks.value && !refreshTimer.value) {
      console.log('发现进行中的任务，启动自动刷新')
      startAutoRefresh()
    }
  } catch (error) {
    if (isAbortError(error)) return
    console.error('获取结算任务失败', error)
    ElMessage.error('获取结算任务失败')
  } finally {
    loading.value = false
  }
}

// 重置筛选条件
const resetFilter = () => {
  filterForm.task_type = ''
  filterForm.status = ''
  filterForm.start_date = ''
  filterForm.end_date = ''
  syncDateRangeFromFilter()
  currentPage.value = 1
  pageSize.value = 10
  queryCtl.run((signal) => fetchTasks(signal), { showCancelMessage: false })
}

const onSearch = () => {
  currentPage.value = 1
  queryCtl.run((signal) => fetchTasks(signal), { toggleIfRunning: true })
}

// 处理页码变化
const handleCurrentChange = (page: number) => {
  currentPage.value = page
  queryCtl.run((signal) => fetchTasks(signal), { showCancelMessage: false })
}

// 处理每页条数变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  queryCtl.run((signal) => fetchTasks(signal), { showCancelMessage: false })
}

// 查看任务详情
const viewTaskDetail = async (task: SettlementTask) => {
  try {
    const response = await api.settlement.getTaskById(task.id)
    currentTask.value = response as any
    taskDetailVisible.value = true
  } catch (error) {
    console.error('获取任务详情失败', error)
    ElMessage.error('获取任务详情失败')
  }
}

// 删除任务
const deleteTask = (task: SettlementTask) => {
  ElMessageBox.confirm(
    `确定要删除任务 #${task.id} 吗？`,
    '删除确认',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await api.settlement.deleteTask(task.id)
      ElMessage.success('删除任务成功')
      queryCtl.run((signal) => fetchTasks(signal), { showCancelMessage: false }) // 刷新任务列表
    } catch (error: any) {
      console.error('删除任务失败', error)
      ElMessage.error(error.response?.data?.error || '删除任务失败')
    }
  }).catch(() => {
    // 用户取消删除操作
  })
}

// 创建日结算任务
const createDailyTask = () => {
  taskForm.type = 'daily'
  taskDialogTitle.value = '创建日结算任务'
  taskForm.date = formatDateToYYYYMMDD(new Date())
  taskForm.dateRange = null
  taskForm.monthRange = null
  createTaskVisible.value = true
}

// 创建周结算任务
const createWeeklyTask = () => {
  taskForm.type = 'weekly'
  taskDialogTitle.value = '创建周结算任务'
  
  // 计算当前周的开始和结束日期（周一到周日）
  const today = new Date()
  const day = today.getDay() || 7 // 将周日的 0 转换为 7
  const monday = new Date(today)
  monday.setDate(today.getDate() - day + 1) // 设置为当前周的周一
  
  const sunday = new Date(monday)
  sunday.setDate(monday.getDate() + 6) // 设置为当前周的周日
  
  taskForm.dateRange = normalizeSettlementDayRange([
    formatDateToYYYYMMDD(monday),
    formatDateToYYYYMMDD(sunday)
  ])
  taskForm.monthRange = null
  
  createTaskVisible.value = true
}

const createNodeDailyTask = () => {
  taskForm.type = 'node_daily95'
  taskDialogTitle.value = '创建节点日95任务'
  const today = formatDateToYYYYMMDD(new Date())
  taskForm.date = ''
  taskForm.dateRange = [today, today]
  taskForm.month = ''
  taskForm.monthRange = null
  createTaskVisible.value = true
}

const createNodeMonthlyTask = () => {
  taskForm.type = 'node_monthly95'
  taskDialogTitle.value = '创建节点月95任务'
  const today = new Date()
  const currentMonth = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}`
  taskForm.month = ''
  taskForm.dateRange = null
  taskForm.monthRange = [currentMonth, currentMonth]
  createTaskVisible.value = true
}

const beforeCloseCreateTask = (done: () => void) => {
  if (submitting.value) return
  done()
}

const onCreateTaskDialogClosed = () => {
  cleanupStaleElementOverlays(document)
}

// 提交创建任务
const submitTaskCreate = async () => {
  if (submitting.value) return

  if (taskForm.type === 'daily' && !taskForm.date) {
    ElMessage.warning('请选择任务日期')
    return
  }
  
  if (taskForm.type === 'weekly' && (!taskForm.dateRange || taskForm.dateRange.length !== 2)) {
    ElMessage.warning('请选择完整的周日期范围')
    return
  }

  const dailyRange = taskForm.type === 'node_daily95' ? expandNodeDailyTaskRange(taskForm.dateRange) : null
  if (dailyRange?.error) {
    ElMessage.warning(dailyRange.error)
    return
  }

  const monthlyRange = taskForm.type === 'node_monthly95' ? expandNodeMonthlyTaskRange(taskForm.monthRange) : null
  if (monthlyRange?.error) {
    ElMessage.warning(monthlyRange.error)
    return
  }

  let createdCount = 0
  submitting.value = true
  try {
    if (taskForm.type === 'daily') {
      // 日结算任务使用单个日期
      const params = { date: taskForm.date }
      await api.settlement.createDailyTask(params)
      createdCount = 1
    } else if (taskForm.type === 'weekly') {
      // 周结算任务使用日期范围
      const params = { 
        start_date: taskForm.dateRange[0],
        end_date: taskForm.dateRange[1]
      }
      await api.settlement.createWeeklyTask(params)
      createdCount = 1
    } else if (taskForm.type === 'node_daily95') {
      await api.settlement.createNodeDailyTask({
        start_date: dailyRange!.dates[0],
        end_date: dailyRange!.dates[dailyRange!.dates.length - 1],
      })
      createdCount = 1
    } else {
      await api.settlement.createNodeMonthlyTask({
        start_month: monthlyRange!.months[0],
        end_month: monthlyRange!.months[monthlyRange!.months.length - 1],
      })
      createdCount = 1
    }
    
    const periodCount = dailyRange?.dates.length || monthlyRange?.months.length || createdCount
    const countText = periodCount > 1 ? `，覆盖 ${periodCount} 个周期` : ''
    ElMessage.success(`创建${taskTypeLabel(taskForm.type)}任务成功${countText}`)
    createTaskVisible.value = false
    queryCtl.run((signal) => fetchTasks(signal), { showCancelMessage: false }) // 刷新任务列表
    
    // 创建任务后立即启动自动刷新
    startAutoRefresh()
  } catch (error) {
    console.error('创建任务失败', error)
    const message = (error as any)?.response?.data?.message || (error as any)?.response?.data?.error || '创建任务失败'
    if (createdCount > 0) {
      ElMessage.error(`已创建 ${createdCount} 个任务，后续创建失败：${message}`)
      queryCtl.run((signal) => fetchTasks(signal), { showCancelMessage: false })
      startAutoRefresh()
    } else {
      ElMessage.error(message)
    }
  } finally {
    submitting.value = false
  }
}

// 格式化日期时间
const formatDateTime = (dateTimeStr: string) => {
  if (!dateTimeStr || dateTimeStr === '0001-01-01T00:00:00Z') {
    return '未开始'
  }
  const date = new Date(dateTimeStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  })
}

// 格式化日期
const formatDate = (dateStr: string) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  })
}

// 格式化日期为YYYY-MM-DD格式
const formatDateToYYYYMMDD = (date: Date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// 获取状态对应的类型
const getStatusType = (status: TaskStatus) => {
  switch (status) {
    case 'pending': return 'info'
    case 'running': return 'warning'
    case 'success': return 'success'
    case 'failed': return 'danger'
    default: return 'info'
  }
}

// 获取状态对应的文本
const getStatusText = (status: TaskStatus) => {
  switch (status) {
    case 'pending': return '等待中'
    case 'running': return '执行中'
    case 'success': return '已完成'
    case 'failed': return '失败'
    default: return '未知'
  }
}

// 定时器引用
const refreshTimer = ref<number | null>(null)

// 检查是否有进行中的任务
const hasRunningTasks = computed(() => {
  return taskData.value.items.some(task => 
    task.status === 'pending' || task.status === 'running'
  )
})

// 启动定时刷新
const startAutoRefresh = () => {
  // 如果已经有定时器在运行，先清除
  stopAutoRefresh()
  
  // 设置定时器，每5秒刷新一次
  refreshTimer.value = window.setInterval(() => {
    if (hasRunningTasks.value) {
      console.log('有进行中的任务，自动刷新状态')
      queryCtl.run((signal) => fetchTasks(signal), { showCancelMessage: false })
    } else {
      console.log('没有进行中的任务，停止自动刷新')
      stopAutoRefresh()
    }
  }, 5000) // 5秒刷新一次
}

// 停止定时刷新
const stopAutoRefresh = () => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }
}

// 组件挂载时获取数据
onMounted(() => {
  syncDateRangeFromFilter()
  queryCtl.run((signal) => fetchTasks(signal), { showCancelMessage: false })
})
usePageRefresh(() => {
  queryCtl.run((signal) => fetchTasks(signal), { showCancelMessage: false })
})

// 组件卸载时清除定时器
onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.settlement-tasks-tab {
  padding: 10px;
}

.filter-section {
  margin-bottom: 20px;
}

.action-section {
  margin-bottom: 20px;
  display: flex;
  gap: 10px;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.task-detail {
  padding: 10px;
}

.detail-item {
  margin-bottom: 15px;
  display: flex;
}

.detail-item .label {
  width: 120px;
  font-weight: bold;
  color: #606266;
}

.detail-item .value {
  flex: 1;
}

.error-message {
  margin-top: 5px;
  padding: 10px;
  background-color: #fef0f0;
  color: #f56c6c;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
