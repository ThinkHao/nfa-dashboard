import axios from 'axios'
import type {
  PaginatedData,
  OperationLog,
  LoginRequest,
  LoginResponse,
  ProfileResponse,
  RefreshRequest,
  RefreshResponse,
  Role,
  SystemUser,
  UpdateUserStatusRequest,
  UpdateUserAliasRequest,
  SetUserRolesRequest,
  RoleCreateRequest,
  RoleUpdateRequest,
  SetRolePermissionsRequest,
  PermissionLite,
  CreateUserRequest,
  RateCustomer,
  UpsertRateCustomerRequest,
  RateNode,
  UpsertRateNodeRequest,
  RateFinalCustomer,
  UpsertRateFinalCustomerRequest,
  BusinessEntity,
  CreateBusinessEntityRequest,
  UpdateBusinessEntityRequest,
  BusinessType,
  CreateBusinessTypeRequest,
  UpdateBusinessTypeRequest,
  SyncRule,
  CreateSyncRuleRequest,
  UpdateSyncRuleRequest,
  SettlementFormulaItem,
  CreateSettlementFormulaRequest,
  UpdateSettlementFormulaRequest,
} from '@/types/api'

// 获取当前 API 基地址（不带路径，形如 https://host:port）
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

// 创建axios实例
const __BASE = getBaseUrl()
try { if ((import.meta as any)?.env?.DEV) console.debug('[API] axios baseURL =', __BASE) } catch {}
const api = axios.create({
  baseURL: __BASE, // 动态设置后端API地址
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
    return api.get('/api/v1/schools', { params }).then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
  },
  
  // 获取地区列表
  getRegions() {
    return api.get('/api/v1/regions').then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
  },
  
  // 获取运营商列表
  getCPs() {
    return api.get('/api/v1/cps').then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
  },
  
  // 获取流量数据
  getTrafficData(params?: any) {
    return api.get('/api/v1/traffic', { params }).then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
  },
  
  // 获取流量汇总数据
  getTrafficSummary(params?: any) {
    return api.get('/api/v1/traffic/summary', { params }).then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
  },

  // 结算系统相关API
  settlement: {
    // 获取结算配置
    getConfig() {
      return api
        .get('/api/v1/settlement/config')
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },

    // 更新结算配置
    updateConfig(config: any) {
      const payload = {
        id: config.id,
        daily_time: config.daily_time,
        weekly_day: config.weekly_day,
        weekly_time: config.weekly_time,
        enabled: config.enabled,
      }
      return api
        .put('/api/v1/settlement/config', payload)
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },

    // 获取结算任务列表
    getTasks(params?: any) {
      // 统一解包 { data: { items, total } } 或直接返回数组/对象
      return api
        .get('/api/v1/settlement/tasks', { params })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },

    // 获取结算任务详情
    getTaskById(id: number) {
      // 解包 { data: task } 以便组件直接使用字段
      return api
        .get(`/api/v1/settlement/tasks/${id}`)
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
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
      return api
        .get('/api/v1/settlement/data', { params })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },

    // 获取结算结果列表
    getResults(params?: any) {
      return api
        .get('/api/v1/settlement/results', { params })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },

    // 获取日95明细数据列表
    getDailySettlementDetails(params?: any) {
      return api
        .get('/api/v1/settlement/daily-details', { params })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },

    // 结算公式管理
    formulas: {
      list(params?: { limit?: number; offset?: number }): Promise<PaginatedData<SettlementFormulaItem>> {
        return api
          .get('/api/v1/settlement/formulas', { params })
          .then((d: any) => {
            const data = d && typeof d === 'object' && 'data' in d ? (d as any).data : d
            if (data && typeof data === 'object' && 'items' in data) {
              const items = Array.isArray((data as any).items) ? (data as any).items : []
              const total = Number((data as any).total ?? items.length)
              return { items: items as SettlementFormulaItem[], total }
            }
            if (Array.isArray(data)) {
              return { items: data as SettlementFormulaItem[], total: (data as SettlementFormulaItem[]).length }
            }
            return { items: [], total: 0 }
          })
      },
      get(id: number): Promise<SettlementFormulaItem | null> {
        return api
          .get(`/api/v1/settlement/formulas/${id}`)
          .then((d: any) => (d && typeof d === 'object' && 'data' in d ? ((d as any).data as SettlementFormulaItem) : (d as SettlementFormulaItem)))
          .catch(() => null)
      },
      create(payload: CreateSettlementFormulaRequest): Promise<SettlementFormulaItem> {
        return api
          .post('/api/v1/settlement/formulas', payload)
          .then((d: any) => (d && typeof d === 'object' && 'data' in d ? ((d as any).data as SettlementFormulaItem) : (d as SettlementFormulaItem)))
      },
      update(id: number, payload: UpdateSettlementFormulaRequest): Promise<void> {
        return api
          .put(`/api/v1/settlement/formulas/${id}`, payload)
          .then(() => undefined)
      },
      remove(id: number): Promise<void> {
        return api
          .delete(`/api/v1/settlement/formulas/${id}`)
          .then(() => undefined)
      },
    },
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
    },
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
      updateAlias(id: number, data: UpdateUserAliasRequest) {
        return api.put(`/api/v1/system/users/${id}/alias`, data)
      },
    },
    binding: {
      // 获取允许被绑定为“院校可见用户”的角色名列表
      async getAllowedUserRoles(type?: 'sales' | 'line' | 'node'): Promise<string[]> {
        const res: any = await api.get('/api/v1/system/binding/allowed-user-roles', { params: type ? { type } : undefined })
        if (res && Array.isArray(res.items)) return res.items as string[]
        if (Array.isArray(res)) return res as string[]
        return []
      },
    },
    userSchools: {
      // 绑定或解绑院校可见用户：传入 user_id 为数字即绑定；不传或为 0/NULL 视为解绑
      setOwner(data: { school_id: string; user_id?: number | null }): Promise<void> {
        return api.post('/api/v1/system/user-schools/owner', data).then(() => undefined)
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
  },

  // 结算 - 费率 API
  settlementRates: {
    customer: {
      list(params?: any): Promise<PaginatedData<RateCustomer>> {
        return api.get('/api/v1/settlement/rates/customer', { params }).then((d: any) => d as PaginatedData<RateCustomer>)
      },
      upsert(data: UpsertRateCustomerRequest): Promise<void> {
        return api.post('/api/v1/settlement/rates/customer', data).then(() => undefined)
      },
    },
    node: {
      list(params?: any): Promise<PaginatedData<RateNode>> {
        return api.get('/api/v1/settlement/rates/node', { params }).then((d: any) => d as PaginatedData<RateNode>)
      },
      upsert(data: UpsertRateNodeRequest): Promise<void> {
        return api.post('/api/v1/settlement/rates/node', data).then(() => undefined)
      },
    },
    final: {
      list(params?: any): Promise<PaginatedData<RateFinalCustomer>> {
        return api.get('/api/v1/settlement/rates/final', { params }).then((d: any) => d as PaginatedData<RateFinalCustomer>)
      },
      upsert(data: UpsertRateFinalCustomerRequest): Promise<void> {
        return api.post('/api/v1/settlement/rates/final', data).then(() => undefined)
      },
      initFromCustomer(): Promise<number> {
        return api.post('/api/v1/settlement/rates/final/init-from-customer', {})
          .then((d: any) => (d && typeof d === 'object' && 'affected' in d ? Number((d as any).affected) : 0))
      },
      refresh(payload: any = {}): Promise<number> {
        return api.post('/api/v1/settlement/rates/final/refresh', payload)
          .then((d: any) => (d && typeof d === 'object' && 'affected' in d ? Number((d as any).affected) : 0))
      },
      cleanupInvalid(): Promise<number> {
        return api.post('/api/v1/settlement/rates/final/cleanup-invalid', {})
          .then((d: any) => (d && typeof d === 'object' && 'affected' in d ? Number((d as any).affected) : 0))
      },
    },
    sync: {
      execute(): Promise<number> {
        return api
          .post('/api/v1/settlement/rates/sync/execute', {})
          .then((d: any) => (d && typeof d === 'object' && 'affected' in d ? Number((d as any).affected) : 0))
      },
    },
    syncRules: {
      list(params?: any): Promise<PaginatedData<SyncRule>> {
        return api.get('/api/v1/settlement/rates/sync-rules', { params }).then((d: any) => d as PaginatedData<SyncRule>)
      },
      create(data: CreateSyncRuleRequest): Promise<SyncRule> {
        return api.post('/api/v1/settlement/rates/sync-rules', data).then((d: any) => d as SyncRule)
      },
      update(id: number, data: UpdateSyncRuleRequest): Promise<void> {
        return api.put(`/api/v1/settlement/rates/sync-rules/${id}`, data).then(() => undefined)
      },
      remove(id: number): Promise<void> {
        return api.delete(`/api/v1/settlement/rates/sync-rules/${id}`).then(() => undefined)
      },
      updatePriority(id: number, priority: number): Promise<void> {
        return api.put(`/api/v1/settlement/rates/sync-rules/${id}/priority`, { priority }).then(() => undefined)
      },
      setEnabled(id: number, enabled: boolean): Promise<void> {
        return api.put(`/api/v1/settlement/rates/sync-rules/${id}/enabled`, { enabled }).then(() => undefined)
      },
    },
  },

  // 结算 - 业务对象 API
  settlementEntities: {
    list(params?: any): Promise<PaginatedData<BusinessEntity>> {
      return api.get('/api/v1/settlement/entities', { params }).then((d: any) => d as PaginatedData<BusinessEntity>)
    },
    create(data: CreateBusinessEntityRequest): Promise<BusinessEntity> {
      return api.post('/api/v1/settlement/entities', data).then((d: any) => d as BusinessEntity)
    },
    update(id: number, data: UpdateBusinessEntityRequest): Promise<void> {
      return api.put(`/api/v1/settlement/entities/${id}`, data).then(() => undefined)
    },
    remove(id: number): Promise<void> {
      return api.delete(`/api/v1/settlement/entities/${id}`).then(() => undefined)
    },
  },

  // 结算 - 业务类型 API
  settlementBusinessTypes: {
    list(params?: any): Promise<PaginatedData<BusinessType>> {
      return api.get('/api/v1/settlement/business-types', { params }).then((d: any) => d as PaginatedData<BusinessType>)
    },
    create(data: CreateBusinessTypeRequest): Promise<BusinessType> {
      return api.post('/api/v1/settlement/business-types', data).then((d: any) => d as BusinessType)
    },
    update(id: number, data: UpdateBusinessTypeRequest): Promise<BusinessType> {
      return api.put(`/api/v1/settlement/business-types/${id}`, data).then((d: any) => d as BusinessType)
    },
    remove(id: number): Promise<void> {
      return api.delete(`/api/v1/settlement/business-types/${id}`).then(() => undefined)
    },
    // 便捷方法：获取全部启用的业务类型（用于下拉）
    async listAllEnabled(): Promise<BusinessType[]> {
      const res = await api.get('/api/v1/settlement/business-types', { params: { enabled: true, page_size: 1000, page: 1 } })
      if (res && typeof res === 'object' && 'items' in res) return (res as any).items as BusinessType[]
      return Array.isArray(res) ? (res as BusinessType[]) : []
    },
  }
  ,
  // v2 接口：启用按用户过滤（后端会在无权限时强制使用当前用户）
  v2: {
    // 学校列表（v2）
    getSchools(params?: any) {
      return api.get('/api/v2/schools', { params })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 地区列表（v2，按用户可见范围）
    getRegions(params?: any) {
      return api.get('/api/v2/regions', { params })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 运营商列表（v2，按用户可见范围）
    getCPs(params?: any) {
      return api.get('/api/v2/cps', { params })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 流量数据（v2）
    getTrafficData(params?: any) {
      return api.get('/api/v2/traffic', { params })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 流量汇总（v2）
    getTrafficSummary(params?: any) {
      return api.get('/api/v2/traffic/summary', { params })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 结算相关（v2）
    settlement: {
      // 获取结算数据列表（v2）
      getSettlements(params?: any) {
        return api.get('/api/v2/settlement/data', { params })
          .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
      },
      // 获取日95明细数据列表（v2）
      getDailySettlementDetails(params?: any) {
        return api.get('/api/v2/settlement/daily-details', { params })
          .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
      },
    },
  }
}
