<template>
  <div :class="['bg-tasks', inline ? 'bg-tasks--inline' : 'bg-tasks--floating']">
    <el-popover placement="bottom-end" trigger="click" width="360">
      <template #reference>
        <el-badge :value="activeCount" :hidden="activeCount===0" type="primary">
          <el-button circle title="后台任务">
            <i class="el-icon"><svg viewBox="0 0 24 24" width="18" height="18"><path fill="currentColor" d="M12 22a2 2 0 0 0 2-2h-4a2 2 0 0 0 2 2Zm6-6V11a6 6 0 1 0-12 0v5l-2 2v1h16v-1l-2-2Z"/></svg></i>
          </el-button>
        </el-badge>
      </template>
      <div class="tasks-list">
        <div v-if="tasks.length===0" class="empty">暂无后台任务</div>
        <div v-for="t in tasks" :key="t.id" class="task-item">
          <div class="title-row">
            <span class="title">{{ t.title }}</span>
            <el-tag size="small" :type="statusType(t.status)">{{ statusText(t.status) }}</el-tag>
          </div>
          <div v-if="t.progress!=null" class="progress-row">
            <el-progress :percentage="Math.round((t.progress||0)*100)" :status="t.status==='failed'?'exception':(t.status==='success'?'success':undefined)" />
            <div class="meta" v-if="t.total!=null && t.processed!=null">
              <span>进度：{{ t.processed }}/{{ t.total }}</span>
              <span v-if="etaText(t)">剩余：{{ etaText(t) }}</span>
            </div>
          </div>
          <div v-if="t.info" class="info">{{ t.info }}</div>
          <div class="action-row" v-if="t.downloadUrl && t.status==='success'">
            <el-link type="primary" :href="t.downloadUrl" :download="'export.csv'">下载文件</el-link>
          </div>
        </div>
      </div>
    </el-popover>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useTasksStore } from '@/stores/tasks'
withDefaults(defineProps<{ inline?: boolean }>(), { inline: false })

const store = useTasksStore()
const tasks = computed(() => store.tasks)
const activeCount = computed(() => store.active.length)

const statusType = (s: string) => {
  if (s === 'running') return 'warning'
  if (s === 'waiting_user_confirm') return 'info'
  if (s === 'success') return 'success'
  if (s === 'failed') return 'danger'
  if (s === 'partial') return 'warning'
  if (s === 'interrupted') return 'danger'
  return 'info'
}
const statusText = (s: string) => ({ pending: '等待中', running: '执行中', waiting_user_confirm: '待确认', success: '完成', failed: '失败', partial: '部分成功', interrupted: '已中断' } as any)[s] || s

// 计时器用于触发 ETA 重渲染
const now = ref(Date.now())
let timer: number | null = null
onMounted(() => {
  timer = window.setInterval(() => { now.value = Date.now() }, 1000) as unknown as number
})
onUnmounted(() => { if (timer) clearInterval(timer) })

function etaText(t: any): string {
  try {
    const processed = Number(t.processed || 0)
    const total = Number(t.total || 0)
    if (!(total > 0) || processed <= 0 || processed >= total) return ''
    const startedAt = Number(t.createdAt || Date.now())
    const elapsedSec = Math.max(1, Math.round((now.value - startedAt) / 1000))
    const speed = processed / elapsedSec // items/sec
    if (!(speed > 0)) return ''
    const remaining = Math.max(0, total - processed)
    const remainSec = Math.round(remaining / speed)
    const mm = String(Math.floor(remainSec / 60)).padStart(2, '0')
    const ss = String(remainSec % 60).padStart(2, '0')
    return `${mm}:${ss}`
  } catch { return '' }
}
</script>

<style scoped>
.bg-tasks--floating {
  position: fixed;
  top: 64px;
  right: 16px;
  z-index: 3000;
}
.bg-tasks--inline {
  position: static;
  display: inline-flex;
  align-items: center;
}
.tasks-list { max-height: 380px; overflow: auto; }
.task-item { padding: 8px 4px; border-bottom: 1px solid var(--el-border-color-lighter); }
.task-item:last-child { border-bottom: none; }
.title-row { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 6px; }
.title { font-weight: 600; }
.progress-row { margin: 6px 0; }
.info { color: var(--text-muted); font-size: 12px; }
.action-row { margin-top: 6px; }
.meta { display:flex; gap:12px; color: var(--text-muted); font-size:12px; margin-top:4px; }

@media (max-width: 992px) {
  .bg-tasks--floating {
    top: 62px;
    right: 12px;
  }
}

@media (max-width: 768px) {
  .bg-tasks--floating {
    top: 58px;
    right: 10px;
  }
}
</style>


