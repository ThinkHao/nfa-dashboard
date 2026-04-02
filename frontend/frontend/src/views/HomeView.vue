<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import * as Icons from '@element-plus/icons-vue'
import api from '@/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useAuthStore } from '@/stores/auth'
import { useTasksStore } from '@/stores/tasks'
import { buildQuickAccessItems, type QuickAccessItem } from './home-workbench'

type RecentSchool = {
  id: number
  school_id: string
  school_name: string
  region?: string
  cp?: string
  update_time?: string
}

const router = useRouter()
const auth = useAuthStore()
const tasksStore = useTasksStore()
const recentSchools = ref<RecentSchool[]>([])
const recentSchoolsLoading = ref(false)

const quickAccessItems = computed<QuickAccessItem[]>(() => buildQuickAccessItems(auth.permissions))
const activeTaskCount = computed(() => tasksStore.active.length)
const latestTasks = computed(() => tasksStore.tasks.slice(0, 5))
const canReadSchools = computed(() => auth.hasPermission('school.read'))
const hasAnySection = computed(() => quickAccessItems.value.length > 0 || canReadSchools.value || tasksStore.tasks.length > 0)

function resolveIcon(iconName?: string) {
  const fallbackIcon = Icons.Menu
  if (!iconName) return fallbackIcon
  const pack = Icons as Record<string, unknown>
  return pack[iconName] || fallbackIcon
}

function navigateTo(path: string) {
  router.push(path)
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

async function loadRecentSchools() {
  if (!canReadSchools.value) return
  recentSchoolsLoading.value = true
  try {
    const res = await api.v2.getSchools({ limit: 5, offset: 0, sort: 'id_desc' })
    const items = Array.isArray(res?.items) ? res.items : []
    const mapped = items.map((item: any) => ({
      id: Number(item?.id || 0),
      school_id: String(item?.school_id || '-'),
      school_name: String(item?.school_name || '-'),
      region: item?.region || '-',
      cp: item?.cp || '-',
      update_time: item?.update_time || '',
    }))
    // 后端若尚未支持 sort，前端做本地兜底。
    recentSchools.value = mapped.sort((a, b) => b.id - a.id).slice(0, 5)
  } catch (error) {
    console.error('加载最近新增院校失败', error)
    recentSchools.value = []
  } finally {
    recentSchoolsLoading.value = false
  }
}

onMounted(async () => {
  await loadRecentSchools()
})
</script>

<template>
  <div class="page-container home-workbench">
    <PageHeader title="工作台" description="快捷进入核心模块，跟踪待办任务与最近新增院校。" />

    <section v-if="quickAccessItems.length" class="launchpad section-block">
      <div class="launchpad-head">
        <div>
          <h2>快捷入口</h2>
          <p>优先展示高频操作，减少层级跳转。</p>
        </div>
      </div>
      <ul class="launch-list">
        <li v-for="item in quickAccessItems" :key="item.key" class="launch-item" @click="navigateTo(item.path)">
          <div class="launch-main">
            <el-icon class="launch-icon"><component :is="resolveIcon(item.icon)" /></el-icon>
            <div class="launch-copy">
              <h3>{{ item.title }}</h3>
              <p>{{ item.description }}</p>
            </div>
          </div>
          <span class="launch-cta">进入</span>
        </li>
      </ul>
    </section>

    <el-row :gutter="16" class="section-block panel-grid">
      <el-col :xs="24" :lg="12">
        <el-card shadow="never" class="panel-card">
          <template #header>
            <div class="panel-title">
              <span>待办摘要</span>
              <el-tag size="small" type="warning">进行中 {{ activeTaskCount }}</el-tag>
            </div>
          </template>
          <div v-if="latestTasks.length === 0" class="empty-tip">暂无后台任务</div>
          <ul v-else class="task-list">
            <li v-for="task in latestTasks" :key="task.id" class="task-item">
              <div class="task-main">
                <span class="task-name">{{ task.title }}</span>
                <el-tag size="small" :type="task.status === 'failed' ? 'danger' : task.status === 'success' ? 'success' : 'info'">
                  {{ task.status }}
                </el-tag>
              </div>
              <span class="task-sub">{{ task.info || '无附加信息' }}</span>
            </li>
          </ul>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="12">
        <el-card shadow="never" class="panel-card">
          <template #header>
            <div class="panel-title">
              <span>最近新增院校</span>
              <el-button v-if="canReadSchools" type="primary" link @click="navigateTo('/schools')">查看全部</el-button>
            </div>
          </template>
          <div v-if="!canReadSchools" class="empty-tip">无院校查看权限</div>
          <div v-else-if="recentSchoolsLoading" class="empty-tip">加载中...</div>
          <div v-else-if="recentSchools.length === 0" class="empty-tip">暂无院校数据</div>
          <ul v-else class="op-list">
            <li v-for="item in recentSchools" :key="item.id" class="op-item">
              <div class="op-main">
                <span class="task-name">{{ item.school_name }}</span>
                <el-tag size="small" type="info">{{ item.school_id }}</el-tag>
              </div>
              <div class="op-meta">
                <span>{{ item.region || '-' }} / {{ item.cp || '-' }}</span>
                <span>{{ formatDateTime(item.update_time) }}</span>
              </div>
            </li>
          </ul>
        </el-card>
      </el-col>
    </el-row>

    <el-empty
      v-if="!hasAnySection"
      description="当前账号暂无可展示模块，请联系管理员开通权限。"
      class="empty-state"
    />
  </div>
</template>

<style scoped>
.home-workbench {
  --home-accent: #0f6fff;
  --home-accent-soft: rgba(15, 111, 255, 0.08);
  --home-ink: #10213b;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.section-block {
  animation: rise-in 0.45s ease both;
}

.launchpad {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background:
    linear-gradient(120deg, rgba(15, 111, 255, 0.05), rgba(15, 111, 255, 0)),
    var(--bg-card);
  overflow: hidden;
}

.launchpad-head {
  padding: 16px 18px 10px;
  border-bottom: 1px dashed var(--border-color);
}

.launchpad-head h2 {
  margin: 0;
  font-size: 18px;
  color: var(--home-ink);
  letter-spacing: 0.01em;
}

.launchpad-head p {
  margin: 6px 0 0;
  color: var(--text-muted);
  font-size: 13px;
}

.launch-list {
  list-style: none;
  margin: 0;
  padding: 2px 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.launch-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 16px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-color);
  transition: background-color 0.22s ease, transform 0.22s ease;
}

.launch-item:nth-child(odd) {
  border-right: 1px solid var(--border-color);
}

.launch-item:hover {
  background: var(--home-accent-soft);
  transform: translateY(-1px);
}

.launch-main {
  display: flex;
  align-items: center;
  gap: 12px;
}

.launch-copy h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-default);
}

.launch-copy p {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}

.launch-icon {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  font-size: 16px;
  color: var(--home-accent);
  background: rgba(15, 111, 255, 0.12);
}

.launch-cta {
  font-size: 12px;
  color: var(--home-accent);
  font-weight: 600;
  white-space: nowrap;
}

.panel-grid {
  animation-delay: 0.08s;
  align-items: stretch;
}

.panel-grid :deep(.el-col) {
  display: flex;
}

.panel-card {
  min-height: 320px;
  border: 1px solid var(--border-color);
  width: 100%;
  height: 100%;
}

.panel-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.empty-tip {
  color: var(--text-muted);
  font-size: 13px;
  padding: 6px 0;
}

.task-list,
.op-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.task-item,
.op-item {
  border-bottom: 1px solid var(--border-color);
  padding: 10px 0;
}

.task-list li:last-child,
.op-list li:last-child {
  border-bottom: none;
}

.task-main,
.op-main,
.op-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.task-sub {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-muted);
}

.task-name {
  font-weight: 600;
}

.op-meta {
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-muted);
}

.empty-state {
  background: var(--bg-card);
  border-radius: var(--radius-md);
}

@keyframes rise-in {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 992px) {
  .launch-list {
    grid-template-columns: 1fr;
  }

  .launch-item:nth-child(odd) {
    border-right: none;
  }
}
</style>


