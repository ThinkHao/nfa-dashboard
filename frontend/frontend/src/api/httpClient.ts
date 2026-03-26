import axios from 'axios'
import type { RefreshRequest, RefreshResponse } from '@/types/api'

const getBaseUrl = () => {
  try {
    const raw = (import.meta as any)?.env?.VITE_API_BASE as string | undefined
    const envBase = typeof raw === 'string' ? raw.trim() : ''
    const isDev = (import.meta as any)?.env?.DEV
    if (isDev) {
      try { console.debug('[API] VITE_API_BASE(raw)=', raw, 'trimmed=', envBase) } catch {}
    }
    if (envBase) {
      return envBase.replace(/\/+$/, '')
    }
  } catch {}
  if (typeof window !== 'undefined') {
    return `${window.location.origin}`
  }
  return 'http://localhost:8091'
}

const __BASE = getBaseUrl()
try { if ((import.meta as any)?.env?.DEV) console.debug('[API] axios baseURL =', __BASE) } catch {}

export const api = axios.create({
  baseURL: __BASE,
  timeout: 60000,
  maxContentLength: 50 * 1024 * 1024,
  maxBodyLength: 50 * 1024 * 1024
})

export const raw = axios.create({
  baseURL: getBaseUrl(),
  timeout: 30000,
})

api.interceptors.request.use(
  (config) => {
    try {
      const token = localStorage.getItem('token')
      if (token) {
        if (config.headers) {
          ;(config.headers as any)['Authorization'] = `Bearer ${token}`
        } else {
          config.headers = { Authorization: `Bearer ${token}` } as any
        }
      }
    } catch {}
    return config
  },
  (error) => Promise.reject(error)
)

let refreshing: Promise<string> | null = null

async function doRefresh(): Promise<string> {
  if (refreshing) return refreshing
  const rt = localStorage.getItem('refresh_token')
  if (!rt) return Promise.reject(new Error('no refresh token'))
  const payload: RefreshRequest = { refresh_token: rt }
  refreshing = raw.post('/api/v1/auth/refresh', payload)
    .then((resp) => resp.data as RefreshResponse)
    .then((res) => {
      const perms = (res.permissions || []).map((p: any) => p?.name || p)
      localStorage.setItem('token', res.token)
      localStorage.setItem('refresh_token', res.refresh_token)
      localStorage.setItem('auth_user', JSON.stringify(res.user))
      localStorage.setItem('auth_perms', JSON.stringify(perms))
      return res.token
    })
    .finally(() => { refreshing = null })
  return refreshing
}

api.interceptors.response.use(
  (response) => response.data,
  async (error) => {
    try {
      const status = error?.response?.status
      const cfg = error?.config || {}
      const url: string = cfg?.url || ''
      if (status === 401 && !cfg.__retry && !url.includes('/auth/login') && !url.includes('/auth/refresh')) {
        cfg.__retry = true
        try {
          const newToken = await doRefresh()
          cfg.headers = cfg.headers || {}
          cfg.headers['Authorization'] = `Bearer ${newToken}`
          return api.request(cfg)
        } catch {
          try {
            localStorage.removeItem('token')
            localStorage.removeItem('refresh_token')
            localStorage.removeItem('auth_user')
            localStorage.removeItem('auth_perms')
          } catch {}
          const redirect = encodeURIComponent(window.location.pathname + window.location.search)
          if (!window.location.pathname.startsWith('/login')) {
            window.location.href = `/login?redirect=${redirect}`
          }
        }
      }
    } catch {}
    return Promise.reject(error)
  }
)
