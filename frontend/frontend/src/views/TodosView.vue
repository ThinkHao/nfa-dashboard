<template>
  <div class="page-container">
    <PageHeader title="业务待办" description="待办会在目标费率的客户费、线路费和节点通用费全部填写后自动完成。" />
    <el-card shadow="never" v-loading="loading">
      <div class="toolbar">
        <el-radio-group v-model="view" @change="load">
          <el-radio-button value="pending">待处理</el-radio-button>
          <el-radio-button value="ignored">已忽略</el-radio-button>
          <el-radio-button value="completed">已完成</el-radio-button>
        </el-radio-group>
        <el-button @click="load">刷新</el-button>
      </div>
      <el-empty v-if="items.length === 0" :description="emptyDescription" />
      <el-table v-else :data="items" row-key="id" @row-click="openTodo">
        <el-table-column prop="school_name" label="院校" min-width="200" />
        <el-table-column label="区域 / CP" min-width="150">
          <template #default="{ row }">{{ row.region }} / {{ row.cp }}</template>
        </el-table-column>
        <el-table-column label="待填写单价" min-width="250">
          <template #default="{ row }">
            <el-tag v-for="field in row.missing_fields" :key="field" size="small" type="warning" class="missing-tag">{{ priceFieldLabel(field) }}</el-tag>
            <el-tag v-if="row.status === 'completed'" size="small" type="success">全部已填写</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="170" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click.stop="showEvents(row)">操作记录</el-button>
            <el-button v-if="view !== 'completed'" link type="primary" @click.stop="setIgnored(row, !row.ignored)">{{ row.ignored ? '恢复提醒' : '忽略' }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="eventsVisible" :title="eventTitle" width="560px">
      <el-empty v-if="events.length === 0" description="暂无操作记录" />
      <el-timeline v-else>
        <el-timeline-item v-for="event in events" :key="event.id" :timestamp="formatDateTime(event.created_at)" placement="top">
          <div class="event-title">{{ eventLabel(event.event_type) }} · {{ event.actor_name }}</div>
          <div v-if="event.detail" class="event-detail">{{ event.detail }}</div>
        </el-timeline-item>
      </el-timeline>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '@/api'
import type { CustomerRateTodo, CustomerRateTodoEvent } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'

const router = useRouter()
const loading = ref(false)
const view = ref<'pending' | 'ignored' | 'completed'>('pending')
const items = ref<CustomerRateTodo[]>([])
const emptyDescription = computed(() => view.value === 'ignored' ? '暂无已忽略待办' : view.value === 'completed' ? '暂无已完成待办' : '暂无待处理事项')
const eventsVisible = ref(false)
const events = ref<CustomerRateTodoEvent[]>([])
const eventTitle = ref('待办操作记录')

async function load() {
  loading.value = true
  try {
    const result = await api.todos.list({
      status: view.value === 'completed' ? 'completed' : 'pending',
      ...(view.value === 'ignored' ? { ignored: true } : view.value === 'pending' ? { ignored: false } : {}),
    })
    items.value = result.items || []
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.message || error?.message || '加载待办失败')
  } finally {
    loading.value = false
  }
}

async function setIgnored(item: CustomerRateTodo, ignored: boolean) {
  try {
    await api.todos.setIgnored(item.id, ignored)
    window.dispatchEvent(new Event('nfa:todos-refresh'))
    await load()
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.message || error?.message || '更新待办状态失败')
  }
}

function openTodo(item: CustomerRateTodo) {
  router.push({ name: 'settlement-rates-customer', query: { region: item.region, cp: item.cp, school_name: item.school_name, todo_id: String(item.id) } })
}

async function showEvents(item: CustomerRateTodo) {
  eventTitle.value = `${item.school_name} · 操作记录`
  eventsVisible.value = true
  events.value = []
  try {
    events.value = (await api.todos.events(item.id)).items || []
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.message || error?.message || '加载操作记录失败')
  }
}

function eventLabel(type: string) {
  return ({ created: '创建待办', assigned: '分配接收人', ignored: '忽略提醒', unignored: '恢复提醒', auto_completed: '自动完成', reopened: '重新打开' } as Record<string, string>)[type] || type
}

function priceFieldLabel(field: string) {
  return ({ customer_fee: '客户费', network_line_fee: '线路费', general_fee: '节点通用费' } as Record<string, string>)[field] || field
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 16px; }
.missing-tag { margin: 2px 4px 2px 0; }
.event-title { font-weight: 600; }
.event-detail { margin-top: 4px; color: var(--text-muted); font-size: 13px; }
:deep(.el-table__row) { cursor: pointer; }
</style>
