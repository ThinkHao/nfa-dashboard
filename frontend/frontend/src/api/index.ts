import axios from 'axios'
import type { PaginatedData, OperationLog, LoginRequest, LoginResponse, ProfileResponse, RefreshRequest, RefreshResponse, Role, SystemUser, UpdateUserStatusRequest, SetUserRolesRequest, RoleCreateRequest, RoleUpdateRequest, SetRolePermissionsRequest, PermissionLite, CreateUserRequest } from '@/types/api'

// 获取当前主机名和协议
const getBaseUrl = () => {
  // 在浏览器环境中使用window.location
  if (typeof window !== 'undefined') {
    const { protocol, hostname } = window.location;
    // 使用与当前页面相同的主机名，但端口使用8081
    return `${protocol}//${hostname}:8081`;
  }
  // 默认值，用于SSR或回退
  return 'http://localhost:8081';
};

// 创建axios实例
const api = axios.create({
  baseURL: getBaseUrl(), // 动态设置后端API地址
  timeout: 60000, // 请求超时时间增加到60秒，以处理大量数据
  maxContentLength: 50 * 1024 * 1024, // 最大内容长度50MB
  maxBodyLength: 50 * 1024 * 1024 // 最大请求体长度50MB
})

// 用于刷新令牌的原始实例（避免响应拦截器递归）
const raw = axios.create({
  baseURL: getBaseUrl(),
  timeout: 30000,
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    // 附带本地 token（如存在）
    try {
      const token = localStorage.getItem('token')
      if (token) {
        if (config.headers) {
          ;(config.headers as any)['Authorization'] = `Bearer ${token}`
        } else {
          // 创建带有 Authorization 的 headers，避免类型不匹配
          config.headers = { Authorization: `Bearer ${token}` } as any
        }
      }
    } catch (e) {
      // 忽略本地存储异常
    }
    return config
  },
  (error) => {
    try {
      const status = error?.response?.status
      if (status === 401) {
        // 清理本地凭证并跳转登录
        try {
          localStorage.removeItem('token')
          localStorage.removeItem('auth_user')
          localStorage.removeItem('auth_perms')
        } catch {}
        const redirect = encodeURIComponent(window.location.pathname + window.location.search)
        if (!window.location.pathname.startsWith('/login')) {
          window.location.href = `/login?redirect=${redirect}`
        }
      } else if (status === 403) {
        if (!window.location.pathname.startsWith('/403')) {
          window.location.href = '/403'
        }
      }
    } catch {}
    return Promise.reject(error)
  }
)

// 响应拦截器
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
  (response) => {
    // 只返回响应的data部分
    return response.data
  },
  async (error) => {
    try {
      const status = error?.response?.status
      const cfg = error?.config || {}
      const url: string = cfg?.url || ''
      if (status === 401 && !cfg.__retry && !url.includes('/auth/login') && !url.includes('/auth/refresh')) {
        cfg.__retry = true
        try {
          const newToken = await doRefresh()
          // 续签后重放原请求
          cfg.headers = cfg.headers || {}
          cfg.headers['Authorization'] = `Bearer ${newToken}`
          return api.request(cfg)
        } catch (_) {
          // 刷新失败则清理并跳转登录
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

// API接口
export default {
  // 认证
  auth: {
    login(data: LoginRequest): Promise<LoginResponse> {
      return api.post('/api/v1/auth/login', data).then((d: any) => d as LoginResponse)
    },
    refresh(data: RefreshRequest): Promise<RefreshResponse> {
      // 提供直接调用能力（通常由拦截器处理）
      return raw.post('/api/v1/auth/refresh', data).then((resp) => resp.data as RefreshResponse)
    },
    profile(): Promise<ProfileResponse> {
      return api.get('/api/v1/auth/profile').then((d: any) => d as ProfileResponse)
    }
  },
  // 获取学校列表
  getSchools(params?: any) {
    return api.get('/api/v1/schools', { params })
  },
  
  // 获取地区列表
  getRegions() {
    return api.get('/api/v1/regions')
  },
  
  // 获取运营商列表
  getCPs() {
    return api.get('/api/v1/cps')
  },
  
  // 获取流量数据
  getTrafficData(params?: any) {
    return api.get('/api/v1/traffic', { params })
  },
  
  // 获取流量汇总数据
  getTrafficSummary(params?: any) {
    return api.get('/api/v1/traffic/summary', { params })
  },

  // 结算系统相关API
  settlement: {
    // 获取结算配置
    getConfig() {
      return api.get('/api/v1/settlement/config')
    },

    // 更新结算配置
    updateConfig(config: any) {
      return api.put('/api/v1/settlement/config', config)
    },

    // 获取结算任务列表
    getTasks(params?: any) {
      return api.get('/api/v1/settlement/tasks', { params })
    },

    // 获取结算任务详情
    getTaskById(id: number) {
      return api.get(`/api/v1/settlement/tasks/${id}`)
    },

    // 创建日结算任务
    createDailyTask(params?: any) {
      return api.post('/api/v1/settlement/tasks/daily', params)
    },

    // 创建周结算任务
    createWeeklyTask(params?: any) {
      return api.post('/api/v1/settlement/tasks/weekly', params)
    },

    // 删除结算任务
    deleteTask(id: number) {
      return api.delete(`/api/v1/settlement/tasks/${id}`)
    },

    // 获取结算数据列表
    getSettlements(params?: any) {
      return api.get('/api/v1/settlement/data', { params })
    },

    // 获取日95明细数据列表
    getDailySettlementDetails(params?: any) {
      return api.get('/api/v1/settlement/daily-details', { params })
    }
  }
  ,
  // 操作日志 API
  operationLogs: {
    list(params?: any): Promise<PaginatedData<OperationLog>> {
      // 由于 axios 类型推断与拦截器返回 data 存在差异，这里进行显式断言
      return api
        .get('/api/v1/system/operation-logs', { params })
        .then((d: any) => d as PaginatedData<OperationLog>)
    },
    export(params?: any): Promise<Blob> {
      // 使用后端导出接口，服务端分页并流式生成CSV
      return api.get('/api/v1/system/operation-logs/export', { params, responseType: 'blob' as any })
        .then((d: any) => d as Blob)
    }
  },
  // 系统管理 API
  system: {
    users: {
      list(params?: any): Promise<PaginatedData<SystemUser>> {
        return api.get('/api/v1/system/users', { params }).then((d: any) => d as PaginatedData<SystemUser>)
      },
      create(data: CreateUserRequest): Promise<SystemUser> {
        return api.post('/api/v1/system/users', data).then((d: any) => d as SystemUser)
      },
      updateStatus(id: number, data: UpdateUserStatusRequest) {
        return api.put(`/api/v1/system/users/${id}/status`, data)
      },
      setRoles(id: number, data: SetUserRolesRequest) {
        return api.put(`/api/v1/system/users/${id}/roles`, data)
      },
    },
    roles: {
      list(params?: any): Promise<PaginatedData<Role>> {
        return api.get('/api/v1/system/roles', { params }).then((d: any) => d as PaginatedData<Role>)
      },
      create(data: RoleCreateRequest): Promise<Role> {
        return api.post('/api/v1/system/roles', data).then((d: any) => d as Role)
      },
      update(id: number, data: RoleUpdateRequest): Promise<Role> {
        return api.put(`/api/v1/system/roles/${id}`, data).then((d: any) => d as Role)
      },
      remove(id: number) {
        return api.delete(`/api/v1/system/roles/${id}`)
      },
      getPermissions(id: number): Promise<PermissionLite[]> {
        return api.get(`/api/v1/system/roles/${id}/permissions`).then((d: any) => {
          if (Array.isArray(d)) return d as PermissionLite[]
          if (d && Array.isArray(d.items)) return d.items as PermissionLite[]
          return [] as PermissionLite[]
        })
      },
      setPermissions(id: number, data: SetRolePermissionsRequest) {
        return api.put(`/api/v1/system/roles/${id}/permissions`, data)
      },
    },
    permissions: {
      list(params?: any): Promise<PaginatedData<PermissionLite>> {
        return api.get('/api/v1/system/permissions', { params }).then((d: any) => d as PaginatedData<PermissionLite>)
      },
      create(data: { code: string; name: string; description?: string | null }): Promise<PermissionLite> {
        return api.post('/api/v1/system/permissions', data).then((d: any) => d as PermissionLite)
      },
      get(id: number): Promise<PermissionLite> {
        return api.get(`/api/v1/system/permissions/${id}`).then((d: any) => d as PermissionLite)
      },
      update(id: number, data: { name?: string; description?: string | null }): Promise<PermissionLite> {
        return api.put(`/api/v1/system/permissions/${id}`, data).then((d: any) => d as PermissionLite)
      },
      remove(id: number) {
        return api.delete(`/api/v1/system/permissions/${id}`)
      },
      sync() {
        return api.post('/api/v1/system/permissions/sync', {})
      },
    },
  }
}
