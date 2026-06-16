import type {
  PaginatedData,
  OperationLog,
  LoginRequest,
  LoginResponse,
  ProfileResponse,
  RefreshRequest,
  RefreshResponse,
  ChangePasswordRequest,
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
  CustomerRateImportResponse,
  CustomerRateImportTask,
  UpsertRateCustomerRequest,
  RateNode,
  UpsertRateNodeRequest,
  EDCNodeSettlementGroup,
  UpsertEDCNodeSettlementGroupRequest,
  RateFinalCustomer,
  RateFinalNode,
  UpsertRateFinalNodeRequest,
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
  FilterRule,
  CreateFilterRuleRequest,
  UpdateFilterRuleRequest,
  SettlementFormulaItem,
  CreateSettlementFormulaRequest,
  UpdateSettlementFormulaRequest,
  DiscountedFinalCustomerRate,
  TrafficScopePreview,
  TrafficScopeOptionItem,
  TrafficScopeOptionParams,
  TrafficScopeRuleGroup,
  TrafficScopeUserLite,
  SystemTrafficSettings,
  SettlementRuleScopeOptions,
} from '@/types/api'
import type { AxiosRequestConfig } from 'axios'
import { api, raw } from './httpClient'

let latestTrafficSettings: SystemTrafficSettings | null = null

function normalizeSettlementResultUnitBase(value: unknown): 1000 | 1024 | null {
  if (value === 1000 || value === 1024) return value
  return null
}

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
    },
    changePassword(data: ChangePasswordRequest): Promise<void> {
      return api.post('/api/v1/auth/change-password', data).then(() => undefined)
    }
  },
  // 获取学校列表
  getSchools(params?: any, config?: AxiosRequestConfig) {
    return api.get('/api/v1/schools', { params, ...(config || {}) }).then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
  },
  
  // 获取地区列表
  getRegions(config?: AxiosRequestConfig) {
    return api.get('/api/v1/regions', { ...(config || {}) }).then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
  },
  
  // 获取运营商列表
  getCPs(config?: AxiosRequestConfig) {
    return api.get('/api/v1/cps', { ...(config || {}) }).then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
  },
  
  // 获取流量数据
  getTrafficData(params?: any, config?: AxiosRequestConfig) {
    return api.get('/api/v1/traffic', { params, ...(config || {}) }).then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
  },
  
  // 获取流量汇总数据
  getTrafficSummary(params?: any, config?: AxiosRequestConfig) {
    return api.get('/api/v1/traffic/summary', { params, ...(config || {}) }).then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
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
        daily_enabled: config.daily_enabled,
        weekly_enabled: config.weekly_enabled,
        node_daily_enabled: config.node_daily_enabled,
        node_daily_time: config.node_daily_time,
        node_monthly_enabled: config.node_monthly_enabled,
        node_monthly_day: config.node_monthly_day,
        node_monthly_time: config.node_monthly_time,
        recalc_after_daily: config.recalc_after_daily,
        recalc_after_weekly: config.recalc_after_weekly,
      }
      return api
        .put('/api/v1/settlement/config', payload)
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },

    // 获取结算任务列表
    getTasks(params?: any, config?: AxiosRequestConfig) {
      // 统一解包 { data: { items, total } } 或直接返回数组/对象
      return api
        .get('/api/v1/settlement/tasks', { params, ...(config || {}) })
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

    createNodeDailyTask(params?: any) {
      return api.post('/api/v1/settlement/tasks/node-daily95', params, { timeout: 180000 })
    },

    createNodeMonthlyTask(params?: any) {
      return api.post('/api/v1/settlement/tasks/node-monthly95', params, { timeout: 180000 })
    },

    // 删除结算任务
    deleteTask(id: number) {
      return api.delete(`/api/v1/settlement/tasks/${id}`)
    },

    // 获取结算数据列表
    getSettlements(params?: any, config?: AxiosRequestConfig) {
      return api
        .get('/api/v1/settlement/data', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },

    // 获取结算结果列表
    getResults(params?: any, config?: AxiosRequestConfig) {
      const normalizedParams = { ...(params || {}) }
      const hasUnitBase = normalizedParams.unit_base === 1000 || normalizedParams.unit_base === 1024
      if (!hasUnitBase) {
        const fallback = normalizeSettlementResultUnitBase(latestTrafficSettings?.settlement_result_unit_base)
        if (fallback != null) normalizedParams.unit_base = fallback
      }
      return api
        .get('/api/v1/settlement/results', { params: normalizedParams, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },

    // 获取渠道维度结算结果列表
    getChannelResults(params?: any, config?: AxiosRequestConfig) {
      return api
        .get('/api/v1/settlement/results/channels', { params, ...(config || {}) })
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
    list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<OperationLog>> {
      // 由于 axios 类型推断与拦截器返回 data 存在差异，这里进行显式断言
      return api
        .get('/api/v1/system/operation-logs', { params, ...(config || {}) })
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
      list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<SystemUser>> {
        return api.get('/api/v1/system/users', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<SystemUser>)
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
      async getAllowedUserRoles(type?: 'sales' | 'line' | 'node' | 'channel'): Promise<string[]> {
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
    trafficScopes: {
      listUsers(params?: any): Promise<PaginatedData<TrafficScopeUserLite>> {
        return api.get('/api/v1/system/traffic-scopes/users', { params }).then((d: any) => d as PaginatedData<TrafficScopeUserLite>)
      },
      options(params: TrafficScopeOptionParams): Promise<PaginatedData<TrafficScopeOptionItem>> {
        return api.get('/api/v1/system/traffic-scopes/options', { params }).then((d: any) => d as PaginatedData<TrafficScopeOptionItem>)
      },
      list(userId: number): Promise<PaginatedData<TrafficScopeRuleGroup>> {
        return api.get(`/api/v1/system/traffic-scopes/${userId}`).then((d: any) => d as PaginatedData<TrafficScopeRuleGroup>)
      },
      replace(userId: number, rules: TrafficScopeRuleGroup[]): Promise<void> {
        return api.put(`/api/v1/system/traffic-scopes/${userId}`, { rules }).then(() => undefined)
      },
      preview(userId: number): Promise<TrafficScopePreview> {
        return api.get(`/api/v1/system/traffic-scopes/${userId}/preview`).then((d: any) => d as TrafficScopePreview)
      },
    },
    settings: {
      getTraffic(): Promise<SystemTrafficSettings> {
        return api.get('/api/v1/system/settings/traffic').then((d: any) => {
          latestTrafficSettings = d as SystemTrafficSettings
          return d as SystemTrafficSettings
        })
      },
      updateTraffic(data: SystemTrafficSettings): Promise<SystemTrafficSettings> {
        return api.put('/api/v1/system/settings/traffic', data).then((d: any) => {
          latestTrafficSettings = d as SystemTrafficSettings
          return d as SystemTrafficSettings
        })
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
      list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<PermissionLite>> {
        return api.get('/api/v1/system/permissions', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<PermissionLite>)
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
      list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<RateCustomer>> {
        return api.get('/api/v1/settlement/rates/customer', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<RateCustomer>)
      },
      upsert(data: UpsertRateCustomerRequest): Promise<void> {
        return api.post('/api/v1/settlement/rates/customer', data).then(() => undefined)
      },
      export(params?: any): Promise<Blob> {
        return api.get('/api/v1/settlement/rates/customer/export', { params, responseType: 'blob' as any }).then((d: any) => d as Blob)
      },
      exportXlsx(params?: any): Promise<Blob> {
        return api.get('/api/v1/settlement/rates/customer/export-xlsx', { params, responseType: 'blob' as any }).then((d: any) => d as Blob)
      },
      template(): Promise<Blob> {
        return api
          .get('/api/v1/settlement/rates/customer/import-template', { responseType: 'blob' as any })
          .then((d: any) => d as Blob)
      },
      createImportTask(form: FormData, opts?: { validateOnly?: boolean }): Promise<{ task_id: number; status: string; task_stage?: string; validate_only?: boolean }> {
        const params: any = {}
        if (opts?.validateOnly) params.validate_only = 1
        return api
          .post('/api/v1/settlement/rates/customer/import/tasks', form, { params })
          .then((d: any) => {
            const taskId = Number((d as any)?.task_id || 0)
            return {
              task_id: taskId,
              status: String((d as any)?.status || 'pending'),
              task_stage: (d as any)?.task_stage ? String((d as any).task_stage) : undefined,
              validate_only: (d as any)?.validate_only != null ? Boolean((d as any).validate_only) : undefined,
            }
          })
      },
      getImportTask(taskId: number): Promise<CustomerRateImportTask> {
        return api
          .get(`/api/v1/settlement/rates/customer/import/tasks/${taskId}`)
          .then((d: any) => d as CustomerRateImportTask)
      },
      continueImportTask(taskId: number): Promise<{ task_id: number; status: string; task_stage?: string }> {
        return api
          .post(`/api/v1/settlement/rates/customer/import/tasks/${taskId}/continue`, {})
          .then((d: any) => ({
            task_id: Number((d as any)?.task_id || taskId),
            status: String((d as any)?.status || 'running'),
            task_stage: (d as any)?.task_stage ? String((d as any).task_stage) : undefined,
          }))
      },
      downloadImportErrorsCsv(taskId: number): Promise<Blob> {
        return api
          .get(`/api/v1/settlement/rates/customer/import/tasks/${taskId}/errors.csv`, { responseType: 'blob' as any })
          .then((d: any) => d as Blob)
      },
      downloadImportCreatedUsersCsv(taskId: number): Promise<Blob> {
        return api
          .get(`/api/v1/settlement/rates/customer/import/tasks/${taskId}/created-users.csv`, { responseType: 'blob' as any })
          .then((d: any) => d as Blob)
      },
      import(form: FormData, opts?: { validateOnly?: boolean }): Promise<CustomerRateImportResponse> {
        const params: any = {}
        if (opts?.validateOnly) params.validate_only = 1
        return api
          .post('/api/v1/settlement/rates/customer/import', form, { params })
          .then((d: any) => {
            const affected = d && typeof d === 'object' && 'affected' in d ? Number((d as any).affected) : 0
            const errors = d && typeof d === 'object' && Array.isArray((d as any).errors) ? (d as any).errors as Array<{ line: number; message: string }> : []
            const validate_only = d && typeof d === 'object' && 'validate_only' in d ? Boolean((d as any).validate_only) : undefined
            const stage = d && typeof d === 'object' && 'stage' in d ? String((d as any).stage) as CustomerRateImportResponse['stage'] : undefined
            const missing_users = d && typeof d === 'object' && Array.isArray((d as any).missing_users) ? (d as any).missing_users as CustomerRateImportResponse['missing_users'] : []
            const created_users = d && typeof d === 'object' && Array.isArray((d as any).created_users) ? (d as any).created_users as CustomerRateImportResponse['created_users'] : []
            const can_auto_create_users = d && typeof d === 'object' && 'can_auto_create_users' in d ? Boolean((d as any).can_auto_create_users) : undefined
            const resumable_token = d && typeof d === 'object' && 'resumable_token' in d ? String((d as any).resumable_token || '') : undefined
            return { affected, errors, validate_only, stage, missing_users, created_users, can_auto_create_users, resumable_token }
          })
      },
    },
    node: {
      list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<RateNode>> {
        return api.get('/api/v1/settlement/rates/node', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<RateNode>)
      },
      upsert(data: UpsertRateNodeRequest): Promise<void> {
        return api.post('/api/v1/settlement/rates/node', data).then(() => undefined)
      },
    },
    nodeGroups: {
      list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<EDCNodeSettlementGroup>> {
        return api.get('/api/v1/settlement/rates/node-groups', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<EDCNodeSettlementGroup>)
      },
      create(data: UpsertEDCNodeSettlementGroupRequest): Promise<EDCNodeSettlementGroup> {
        return api.post('/api/v1/settlement/rates/node-groups', data).then((d: any) => d as EDCNodeSettlementGroup)
      },
      update(id: number, data: UpsertEDCNodeSettlementGroupRequest): Promise<EDCNodeSettlementGroup> {
        return api.put(`/api/v1/settlement/rates/node-groups/${id}`, data).then((d: any) => d as EDCNodeSettlementGroup)
      },
      remove(id: number): Promise<void> {
        return api.delete(`/api/v1/settlement/rates/node-groups/${id}`).then(() => undefined)
      },
    },
    finalNode: {
      list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<RateFinalNode>> {
        return api.get('/api/v1/settlement/rates/final-node', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<RateFinalNode>)
      },
      upsert(data: UpsertRateFinalNodeRequest): Promise<void> {
        return api.post('/api/v1/settlement/rates/final-node', data).then(() => undefined)
      },
      initFromNode(): Promise<number> {
        return api.post('/api/v1/settlement/rates/final-node/init-from-node', {})
          .then((d: any) => (d && typeof d === 'object' && 'affected' in d ? Number((d as any).affected) : 0))
      },
      refresh(): Promise<number> {
        return api.post('/api/v1/settlement/rates/final-node/refresh', {})
          .then((d: any) => (d && typeof d === 'object' && 'affected' in d ? Number((d as any).affected) : 0))
      },
    },
    final: {
      list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<RateFinalCustomer>> {
        return api.get('/api/v1/settlement/rates/final', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<RateFinalCustomer>)
      },
      // 按服务日期获取折损后的最终客户费率视图
      listDiscounted(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<DiscountedFinalCustomerRate>> {
        return api
          .get('/api/v1/settlement/rates/final-discounted', { params, ...(config || {}) })
          .then((d: any) => d as PaginatedData<DiscountedFinalCustomerRate>)
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
          .post('/api/v1/settlement/rates/sync/execute', {}, { timeout: 300000 })
          .then((d: any) => (d && typeof d === 'object' && 'affected' in d ? Number((d as any).affected) : 0))
      },
    },
    syncRules: {
      options(): Promise<SettlementRuleScopeOptions> {
        return api.get('/api/v1/settlement/rates/sync-rules/options').then((d: any) => d as SettlementRuleScopeOptions)
      },
      list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<SyncRule>> {
        return api.get('/api/v1/settlement/rates/sync-rules', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<SyncRule>)
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
    filterRules: {
      options(): Promise<SettlementRuleScopeOptions> {
        return api.get('/api/v1/settlement/rates/filter-rules/options').then((d: any) => d as SettlementRuleScopeOptions)
      },
      list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<FilterRule>> {
        return api.get('/api/v1/settlement/rates/filter-rules', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<FilterRule>)
      },
      create(data: CreateFilterRuleRequest): Promise<FilterRule> {
        return api.post('/api/v1/settlement/rates/filter-rules', data).then((d: any) => d as FilterRule)
      },
      update(id: number, data: UpdateFilterRuleRequest): Promise<void> {
        return api.put(`/api/v1/settlement/rates/filter-rules/${id}`, data).then(() => undefined)
      },
      remove(id: number): Promise<void> {
        return api.delete(`/api/v1/settlement/rates/filter-rules/${id}`).then(() => undefined)
      },
      updatePriority(id: number, priority: number): Promise<void> {
        return api.put(`/api/v1/settlement/rates/filter-rules/${id}/priority`, { priority }).then(() => undefined)
      },
      setEnabled(id: number, enabled: boolean): Promise<void> {
        return api.put(`/api/v1/settlement/rates/filter-rules/${id}/enabled`, { enabled }).then(() => undefined)
      },
    },
    // 折损规则管理
    discountRules: {
      list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<any>> {
        return api.get('/api/v1/settlement/rates/discount-rules', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<any>)
      },
      get(id: number): Promise<{ rule: any; items: any[] }> {
        return api.get(`/api/v1/settlement/rates/discount-rules/${id}`).then((d: any) => d as { rule: any; items: any[] })
      },
      create(data: any): Promise<{ rule: any; items: any[] }> {
        return api.post('/api/v1/settlement/rates/discount-rules', data).then((d: any) => d as { rule: any; items: any[] })
      },
      update(id: number, data: any): Promise<void> {
        return api.put(`/api/v1/settlement/rates/discount-rules/${id}`, data).then(() => undefined)
      },
      remove(id: number): Promise<void> {
        return api.delete(`/api/v1/settlement/rates/discount-rules/${id}`).then(() => undefined)
      },
      replaceItems(id: number, items: any[]): Promise<void> {
        return api.put(`/api/v1/settlement/rates/discount-rules/${id}/items`, items).then(() => undefined)
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
    list(params?: any, config?: AxiosRequestConfig): Promise<PaginatedData<BusinessType>> {
      return api.get('/api/v1/settlement/business-types', { params, ...(config || {}) }).then((d: any) => d as PaginatedData<BusinessType>)
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
  // 结算数据明细 API（settlement_customer）
  settlementData: {
    // 列表
    list(params?: any, config?: AxiosRequestConfig) {
      return api
        .get('/api/v1/settlement/data/customer', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 月度聚合列表
    monthlyList(params?: any, config?: AxiosRequestConfig) {
      return api
        .get('/api/v1/settlement/data/customer/monthly', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    nodeList(params?: any, config?: AxiosRequestConfig) {
      return api
        .get('/api/v1/settlement/data/node', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    nodeMonthlyList(params?: any, config?: AxiosRequestConfig) {
      return api
        .get('/api/v1/settlement/data/node/monthly', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 重建月度快照
    rebuildMonthly(payload: any = {}): Promise<number> {
      return api
        .post('/api/v1/settlement/data/customer/monthly/rebuild', payload)
        .then((d: any) => {
          const data = d && typeof d === 'object' && 'data' in d ? (d as any).data : d
          const affected = data && typeof data === 'object' && 'affected' in data ? Number((data as any).affected) : Number((d as any)?.affected)
          return Number.isFinite(affected) ? affected : 0
        })
    },
    // 统一的费用归属主体（system user）下拉
    ownerSubjects(params?: any, config?: AxiosRequestConfig): Promise<Array<{ type: string; id: number; label: string }>> {
      return api
        .get('/api/v1/settlement/data/customer/owner-subjects', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (((d as any).data?.items) ?? []) : (Array.isArray(d) ? d : [])))
    },
    // 导出 CSV
    export(params?: any): Promise<Blob> {
      return api
        .get('/api/v1/settlement/data/customer/export', { params, responseType: 'blob' as any })
        .then((d: any) => d as Blob)
    },
    // 触发复算
    recalculate(payload: any = {}): Promise<number> {
      return api
        .post('/api/v1/settlement/data/customer/recalculate', payload)
        .then((d: any) => {
          // 兼容 {code,message,data:{task_id}} 或直接 {task_id}
          const data = d && typeof d === 'object' && 'data' in d ? (d as any).data : d
          const taskId = data && typeof data === 'object' && 'task_id' in data ? Number((data as any).task_id) : Number((d as any)?.task_id)
          return Number.isFinite(taskId) ? taskId : 0
        })
    },
    // 已使用过的费用归属主体ID（兼容历史数据）下拉
    usedOwners(): Promise<Array<{ id: number; entity_name: string }>> {
      return api
        .get('/api/v1/settlement/data/customer/owners')
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? ((d as any).data?.items ?? []) : (Array.isArray(d) ? d : [])))
    },
    // 已使用过的渠道归属用户下拉
    usedChannelOwners(): Promise<Array<{ id: number; display_name: string }>> {
      return api
        .get('/api/v1/settlement/data/customer/channel-owners')
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? ((d as any).data?.items ?? []) : (Array.isArray(d) ? d : [])))
    },
  }
  ,
  // v2 接口：启用按用户过滤（后端会在无权限时强制使用当前用户）
  v2: {
    // 学校列表（v2）
    getSchools(params?: any, config?: AxiosRequestConfig) {
      return api.get('/api/v2/schools', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 地区列表（v2，按用户可见范围）
    getRegions(params?: any, config?: AxiosRequestConfig) {
      return api.get('/api/v2/regions', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 运营商列表（v2，按用户可见范围）
    getCPs(params?: any, config?: AxiosRequestConfig) {
      return api.get('/api/v2/cps', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 流量数据（v2）
    getTrafficData(params?: any, config?: AxiosRequestConfig) {
      return api.get('/api/v2/traffic', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    // 流量汇总（v2）
    getTrafficSummary(params?: any, config?: AxiosRequestConfig) {
      return api.get('/api/v2/traffic/summary', { params, ...(config || {}) })
        .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
    },
    edc: {
      getEntities(params?: any, config?: AxiosRequestConfig) {
        return api.get('/api/v2/edc/entities', { params, ...(config || {}) })
          .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
      },
      getRegions(params?: any, config?: AxiosRequestConfig) {
        return api.get('/api/v2/edc/regions', { params, ...(config || {}) })
          .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
      },
      getCPs(params?: any, config?: AxiosRequestConfig) {
        return api.get('/api/v2/edc/cps', { params, ...(config || {}) })
          .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
      },
      getTrafficData(params?: any, config?: AxiosRequestConfig) {
        return api.get('/api/v2/edc/traffic', { params, ...(config || {}) })
          .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
      },
      getTrafficSummary(params?: any, config?: AxiosRequestConfig) {
        return api.get('/api/v2/edc/traffic/summary', { params, ...(config || {}) })
          .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
      },
    },
    // 结算相关（v2）
    settlement: {
      // 获取结算数据列表（v2）
      getSettlements(params?: any, config?: AxiosRequestConfig) {
        return api.get('/api/v2/settlement/data', { params, ...(config || {}) })
          .then((d: any) => (d && typeof d === 'object' && 'data' in d ? (d as any).data : d))
      },
    },
  }
}
