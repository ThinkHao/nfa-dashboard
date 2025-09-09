import { defineStore } from 'pinia'
import type { RouteLocationNormalized } from 'vue-router'

export type Tag = { title: string; path: string; name?: string }

export const useTagsViewStore = defineStore('tagsView', {
  state: () => ({
    visited: [] as Tag[],
    activePath: '' as string,
  }),
  actions: {
    addRoute(route: RouteLocationNormalized) {
      const title = (route.meta?.title as string) || (route.name?.toString() || route.path)
      const path = route.fullPath
      if (!this.visited.find(t => t.path === path)) {
        this.visited.push({ title, path, name: route.name?.toString() })
      }
      this.activePath = path
    },
    remove(path: string) {
      const idx = this.visited.findIndex(t => t.path === path)
      if (idx >= 0) this.visited.splice(idx, 1)
      if (this.activePath === path) {
        const next = this.visited[idx] || this.visited[idx - 1]
        this.activePath = next ? next.path : ''
      }
    },
    removeOthers(currentPath: string) {
      const cur = this.visited.find(t => t.path === currentPath)
      this.visited = cur ? [cur] : []
      this.activePath = cur ? cur.path : ''
    },
    removeAll() {
      this.visited = []
      this.activePath = ''
    },
    setActive(path: string) { this.activePath = path },
  },
})
