<template>
  <div class="settlement-tasks-tab">
    <!-- 筛选条件区域 -->
    <div class="filter-section">
      <el-form :model="filterForm" inline>
        <el-form-item label="任务类型">
          <el-select v-model="filterForm.task_type" placeholder="选择任务类型" clearable>
            <el-option label="日结算" value="daily" />
            <el-option label="周结算" value="weekly" />
          </el-select>
        </el-form-item>
        <el-form-item label="任务状态">
          <el-select v-model="filterForm.status" placeholder="选择任务状态" clearable>
            <el-option label="等待中" value="pending" />
            <el-option label="执行中" value="running" />
            <el-option label="已完成" value="success" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item label="日期范围">
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchTasks">查询</el-button>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 操作按钮区域 -->
    <div class="action-section">
      <el-button type="primary" @click="createDailyTask">创建日结算任务</el-button>
      <el-button type="success" @click="createWeeklyTask">创建周结算任务</el-button>
    </div>

    <!-- 任务表格区域 -->
    <div class="table-section">
      <h3>结算任务列表</h3>
      <el-table
        v-loading="loading"
        :data="taskData.items"
        border
        stripe
        style="width: 100%"
      >
        <el-table-column prop="id" label="任务ID" width="80" />
        <el-table-column prop="task_type" label="任务类型" width="100">
          <template #default="scope">
            {{ scope.row.task_type === 'daily' ? '日结算' : '周结算' }}
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
    </div>

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
          <span class="value">{{ currentTask.task_type === 'daily' ? '日结算' : '周结算' }}</span>
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
        
        <!-- 周结算任务显示日期范围选择器 -->
        <el-form-item v-else label="周日期范围">
          <el-date-picker
            v-model="taskForm.dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
            :default-time="[new Date(2000, 1, 1, 0, 0, 0), new Date(2000, 1, 1, 23, 59, 59)]"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createTaskVisible = false">取消</el-button>
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
import { ElMessage, ElMessageBox } from 'element-plus'
import type { TaskListResponse, SettlementTask, TaskStatus } from '../../types/settlement'

// 筛选表单
const filterForm = reactive({
  task_type: '',
  status: '',
  start_date: '',
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
  dateRange: [] as string[]
})

// 获取任务列表
const fetchTasks = async () => {
  loading.value = true
  
  // 处理日期范围
  if (dateRange.value) {
    filterForm.start_date = dateRange.value[0]
    filterForm.end_date = dateRange.value[1]
  } else {
    filterForm.start_date = ''
    filterForm.end_date = ''
  }

  // 设置分页参数
  filterForm.page = currentPage.value
  filterForm.page_size = pageSize.value

  try {
    const response = await api.settlement.getTasks(filterForm) as any
    // 统一仅处理数组或 { items, total }
    if (Array.isArray(response)) {
      taskData.value = { items: response, total: response.length }
    } else if (response && Array.isArray(response.items)) {
      taskData.value = { items: response.items, total: Number(response.total) || response.items.length }
    } else {
      taskData.value = { items: [], total: 0 }
    }
    
    // 检查是否有进行中的任务，如果有则启动自动刷新
    if (hasRunningTasks.value && !refreshTimer.value) {
      console.log('发现进行中的任务，启动自动刷新')
      startAutoRefresh()
    }
  } catch (error) {
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
  dateRange.value = null
  currentPage.value = 1
  pageSize.value = 10
  fetchTasks()
}

// 处理页码变化
const handleCurrentChange = (page: number) => {
  currentPage.value = page
  fetchTasks()
}

// 处理每页条数变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  fetchTasks()
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
      fetchTasks() // 刷新任务列表
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
  taskForm.dateRange = []
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
  
  taskForm.dateRange = [
    formatDateToYYYYMMDD(monday),
    formatDateToYYYYMMDD(sunday)
  ]
  
  createTaskVisible.value = true
}

// 提交创建任务
const submitTaskCreate = async () => {
  if (taskForm.type === 'daily' && !taskForm.date) {
    ElMessage.warning('请选择任务日期')
    return
  }
  
  if (taskForm.type === 'weekly' && (!taskForm.dateRange || taskForm.dateRange.length !== 2)) {
    ElMessage.warning('请选择完整的周日期范围')
    return
  }

  submitting.value = true
  try {
    let response
    
    if (taskForm.type === 'daily') {
      // 日结算任务使用单个日期
      const params = { date: taskForm.date }
      response = await api.settlement.createDailyTask(params)
    } else {
      // 周结算任务使用日期范围
      const params = { 
        start_date: taskForm.dateRange[0],
        end_date: taskForm.dateRange[1]
      }
      response = await api.settlement.createWeeklyTask(params)
    }
    
    ElMessage.success(`创建${taskForm.type === 'daily' ? '日' : '周'}结算任务成功`)
    createTaskVisible.value = false
    fetchTasks() // 刷新任务列表
    
    // 创建任务后立即启动自动刷新
    startAutoRefresh()
  } catch (error) {
    console.error('创建任务失败', error)
    ElMessage.error('创建任务失败')
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
      fetchTasks()
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
  fetchTasks()
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
  padding: 15px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.action-section {
  margin-bottom: 20px;
  display: flex;
  gap: 10px;
}

.table-section {
  background-color: #fff;
  padding: 15px;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
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
