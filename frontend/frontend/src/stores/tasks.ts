import { defineStore } from 'pinia'

export type TaskStatus = 'pending' | 'running' | 'waiting_user_confirm' | 'success' | 'failed'
export type TaskType = 'export' | 'import' | 'settlement' | 'other'

export interface BgTask {
  id: string
  type: TaskType
  title: string
  status: TaskStatus
  progress?: number | null
  info?: string
  downloadUrl?: string
  processed?: number | null
  total?: number | null
  stage?: string
  detailId?: number
  createdAt: number
  updatedAt: number
}

export const useTasksStore = defineStore('bgTasks', {
  state: () => ({
    tasks: [] as BgTask[],
  }),
  getters: {
    active: (s) => s.tasks.filter(t => t.status === 'pending' || t.status === 'running' || t.status === 'waiting_user_confirm'),
  },
  actions: {
    start(task: Omit<BgTask, 'createdAt' | 'updatedAt'>) {
      const now = Date.now()
      const t: BgTask = { ...task, createdAt: now, updatedAt: now }
      const idx = this.tasks.findIndex(x => x.id === t.id)
      if (idx >= 0) this.tasks.splice(idx, 1, t)
      else this.tasks.unshift(t)
      return t.id
    },
    update(id: string, patch: Partial<BgTask>) {
      const idx = this.tasks.findIndex(x => x.id === id)
      if (idx < 0) return
      const now = Date.now()
      const cur = this.tasks[idx]
      this.tasks.splice(idx, 1, { ...cur, ...patch, updatedAt: now })
    },
    complete(id: string, downloadUrl?: string) {
      this.update(id, { status: 'success', progress: 1, downloadUrl })
    },
    fail(id: string, info?: string) {
      this.update(id, { status: 'failed', info })
    },
    remove(id: string) {
      const idx = this.tasks.findIndex(x => x.id === id)
      if (idx >= 0) this.tasks.splice(idx, 1)
    },
    upsertSettlementTask(payload: { id: number; title?: string; status: TaskStatus; processed_count?: number; total_count?: number }) {
      const id = `settlement:${payload.id}`
      const exists = this.tasks.find(x => x.id === id)
      const title = payload.title || `结算任务 #${payload.id}`
      const processed = typeof payload.processed_count === 'number' ? payload.processed_count : null
      const total = typeof payload.total_count === 'number' ? payload.total_count : null
      const info = processed != null ? (total != null && total > 0 ? `已处理 ${processed}/${total}` : `已处理 ${processed}`) : ''
      // 若是已完成/失败的任务且当前浮层不存在，则不再插入，避免历史任务反复刷出
      if (!exists && (payload.status === 'success' || payload.status === 'failed')) {
        return
      }
      if (!exists) {
        const progress = processed != null && total != null && total > 0 ? Math.min(1, Math.max(0, processed / total)) : null
        this.start({ id, type: 'settlement', title, status: payload.status, progress, info, processed, total })
      } else {
        const progress = processed != null && total != null && total > 0 ? Math.min(1, Math.max(0, processed / total)) : undefined
        this.update(id, { status: payload.status, info, processed: processed ?? undefined, total: total ?? undefined, progress })
      }
      if (payload.status === 'success' || payload.status === 'failed') {
        setTimeout(() => this.remove(id), 10000)
      }
    },
  }
})
