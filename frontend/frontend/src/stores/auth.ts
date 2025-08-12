import { defineStore } from 'pinia'
import router from '@/router'
import api from '@/api'

export interface AuthUser {
  id: number
  username: string
  display_name?: string
  status?: number
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: '' as string,
    refresh_token: '' as string,
    user: null as AuthUser | null,
    permissions: [] as string[],
    loadingProfile: false as boolean,
  }),
  getters: {
    isAuthenticated: (s) => !!s.token,
    hasPermission: (s) => (perm: string) => s.permissions.includes(perm),
    hasAnyPermission: (s) => (perms?: string[]) => !perms || perms.length === 0 || perms.some(p => s.permissions.includes(p)),
  },
  actions: {
    initFromStorage() {
      try {
        const token = localStorage.getItem('token') || ''
        const refreshToken = localStorage.getItem('refresh_token') || ''
        const user = localStorage.getItem('auth_user')
        const perms = localStorage.getItem('auth_perms')
        this.token = token
        this.refresh_token = refreshToken
        this.user = user ? JSON.parse(user) : null
        this.permissions = perms ? JSON.parse(perms) : []
      } catch {}
    },
    async login(username: string, password: string) {
      const res = await api.auth.login({ username, password })
      // 预期后端返回 { token, user, permissions }
      this.token = res.token
      this.refresh_token = res.refresh_token
      this.user = res.user
      this.permissions = (res.permissions || []).map((p: any) => p.name || p)
      localStorage.setItem('token', this.token)
      localStorage.setItem('refresh_token', this.refresh_token)
      localStorage.setItem('auth_user', JSON.stringify(this.user))
      localStorage.setItem('auth_perms', JSON.stringify(this.permissions))
    },
    async loadProfile() {
      if (!this.token) return
      this.loadingProfile = true
      try {
        const res = await api.auth.profile()
        // 预期返回 { user, permissions }
        this.user = res.user
        this.permissions = (res.permissions || []).map((p: any) => p.name || p)
        localStorage.setItem('auth_user', JSON.stringify(this.user))
        localStorage.setItem('auth_perms', JSON.stringify(this.permissions))
      } finally {
        this.loadingProfile = false
      }
    },
    logout() {
      this.token = ''
      this.refresh_token = ''
      this.user = null
      this.permissions = []
      localStorage.removeItem('token')
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('auth_user')
      localStorage.removeItem('auth_perms')
      const redirect = router.currentRoute.value.fullPath
      if (!redirect.startsWith('/login')) {
        router.push({ path: '/login', query: { redirect } })
      }
    }
  }
})
