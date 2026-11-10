<template>
  <el-popover placement="bottom-end" trigger="click" width="380" @show="loadItems">
    <template #reference>
      <el-badge :value="count" :hidden="count === 0" :max="99" type="danger">
        <el-button circle title="业务待办" class="todo-bell-button">
          <el-icon><Bell /></el-icon>
        </el-button>
      </el-badge>
    </template>
    <div class="todo-popover">
      <div class="todo-popover-head">
        <strong>业务待办</strong>
        <el-button link type="primary" @click="router.push('/todos')">查看全部</el-button>
      </div>
      <el-radio-group v-model="showIgnored" size="small" @change="loadItems">
        <el-radio-button :value="false">待处理</el-radio-button>
        <el-radio-button :value="true">已忽略</el-radio-button>
      </el-radio-group>
      <div v-loading="loading" class="todo-list">
        <div v-if="!loading && items.length === 0" class="todo-empty">{{ showIgnored ? '没有已忽略待办' : '暂无待处理事项' }}</div>
        <div v-for="item in items.slice(0, 5)" :key="item.id" class="todo-row">
          <button class="todo-main" @click="openTodo(item)">
            <span class="todo-school">{{ item.school_name }}</span>
            <span class="todo-meta">{{ item.region }} / {{ item.cp }} · 缺 {{ item.missing_fields.length }} 项单价</span>
          </button>
          <el-button link type="info" @click="setIgnored(item, !item.ignored)">{{ item.ignored ? '恢复' : '忽略' }}</el-button>
        </div>
        <div v-if="items.length > 5" class="todo-more">还有 {{ items.length - 5 }} 项，请打开待办中心查看</div>
      </div>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Bell } from '@element-plus/icons-vue'
import api from '@/api'
import type { CustomerRateTodo } from '@/types/api'

const router = useRouter()
const count = ref(0)
const items = ref<CustomerRateTodo[]>([])
const loading = ref(false)
const showIgnored = ref(false)
let refreshTimer: number | undefined

async function loadCount() {
  try { count.value = (await api.todos.count()).count || 0 } catch { /* keep the last known count */ }
}

async function loadItems() {
  loading.value = true
  try {
    const result = await api.todos.list({ status: 'pending', ignored: showIgnored.value })
    items.value = result.items || []
    await loadCount()
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.message || error?.message || '加载待办失败')
  } finally {
    loading.value = false
  }
}

async function setIgnored(item: CustomerRateTodo, ignored: boolean) {
  try {
    await api.todos.setIgnored(item.id, ignored)
    await loadItems()
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.message || error?.message || '更新待办状态失败')
  }
}

function openTodo(item: CustomerRateTodo) {
  router.push({ name: 'settlement-rates-customer', query: { region: item.region, cp: item.cp, school_name: item.school_name, todo_id: String(item.id) } })
}

function refreshOnFocus() {
  if (document.visibilityState === 'visible') void loadCount()
}

function refreshFromBusinessChange() { void loadCount() }

onMounted(() => {
  void loadCount()
  refreshTimer = window.setInterval(() => void loadCount(), 60_000)
  window.addEventListener('focus', refreshOnFocus)
  document.addEventListener('visibilitychange', refreshOnFocus)
  window.addEventListener('nfa:todos-refresh', refreshFromBusinessChange)
})

onUnmounted(() => {
  if (refreshTimer) window.clearInterval(refreshTimer)
  window.removeEventListener('focus', refreshOnFocus)
  document.removeEventListener('visibilitychange', refreshOnFocus)
  window.removeEventListener('nfa:todos-refresh', refreshFromBusinessChange)
})
</script>

<style scoped>
.todo-bell-button :deep(.el-icon) { font-size: 18px; }
.todo-popover { display: flex; flex-direction: column; gap: 10px; }
.todo-popover-head { display: flex; align-items: center; justify-content: space-between; }
.todo-list { max-height: 300px; min-height: 46px; overflow: auto; }
.todo-row { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 9px 2px; border-bottom: 1px solid var(--el-border-color-lighter); }
.todo-row:last-child { border-bottom: 0; }
.todo-main { min-width: 0; flex: 1; display: flex; flex-direction: column; align-items: flex-start; gap: 3px; border: 0; padding: 0; background: none; color: inherit; text-align: left; cursor: pointer; }
.todo-main:hover .todo-school { color: var(--el-color-primary); }
.todo-school { max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 600; }
.todo-meta, .todo-more, .todo-empty { color: var(--text-muted); font-size: 12px; }
.todo-empty { padding: 14px 2px; text-align: center; }
.todo-more { padding-top: 8px; text-align: center; }
</style>
