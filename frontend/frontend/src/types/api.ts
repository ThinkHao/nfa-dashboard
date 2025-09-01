// API响应通用类型定义

// 通用API响应接口
export interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface RefreshResponse {
  token: string;
  refresh_token: string;
  user: {
    id: number;
    username: string;
    alias?: string;
    display_name?: string;
    status?: number;
  };
  permissions: (PermissionLite | string)[];
}

// 鉴权相关
export interface LoginRequest {
  username: string;
  password: string;
}

export interface PermissionLite {
  id?: number;
  code?: string;
  name: string; // e.g. operation_logs.read
  description?: string | null;
  created_at?: string;
}

export interface LoginResponse {
  token: string;
  refresh_token: string;
  user: {
    id: number;
    username: string;
    alias?: string;
    display_name?: string;
    status?: number;
  };
  permissions: (PermissionLite | string)[];
}

export interface ProfileResponse {
  user: {
    id: number;
    username: string;
    alias?: string;
    display_name?: string;
    status?: number;
  };
  permissions: (PermissionLite | string)[];
}

// 分页数据接口
export interface PaginatedData<T = any> {
  items: T[];
  total: number;
}

// 分页请求参数
export interface PaginationParams {
  page?: number;
  page_size?: number;
  limit?: number;
  offset?: number;
}

// 学校信息接口
export interface School {
  school_id: string;
  school_name: string;
  region?: string;
  cp?: string;
  create_time?: string;
  update_time?: string;
}

// 流量数据接口
export interface TrafficData {
  id?: number;
  school_id: string;
  create_time: string;
  total_recv: number;
  total_send: number;
  [key: string]: any;
}

// 操作日志
export interface OperationLog {
  id: number;
  user_id?: number | null;
  method: string;
  path: string;
  resource?: string | null;
  action?: string | null;
  status_code: number;
  success: number; // 1/0
  latency_ms?: number | null;
  ip?: string | null;
  user_agent?: string | null;
  error_message?: string | null;
  created_at: string;
}

// 系统管理 - 用户/角色/权限
export interface Role {
  id: number;
  name: string;
  description?: string;
  created_at?: string;
}

export interface SystemUser {
  id: number;
  username: string;
  alias?: string;
  display_name?: string;
  status?: number;
  roles?: Role[];
  created_at?: string;
}

export interface UpdateUserStatusRequest {
  status: number;
}

export interface UpdateUserAliasRequest {
  alias?: string | null;
}

export interface SetUserRolesRequest {
  role_ids: number[];
}

// 新建系统用户请求
export interface CreateUserRequest {
  username: string;
  alias?: string;
  password: string;
  email?: string;
  phone?: string;
  status?: number; // 1 启用, 0 禁用
  role_ids?: number[];
}

export interface RoleCreateRequest {
  name: string;
  description?: string;
}

export interface RoleUpdateRequest {
  name?: string;
  description?: string;
}

export interface SetRolePermissionsRequest {
  permission_ids: number[];
}

// ------------------------------
// Settlement Rates & Entities
// ------------------------------

// 业务对象（business_entities）
export interface BusinessEntity {
  id: number;
  entity_type: string;
  entity_name: string;
  contact_info?: string | null;
  created_at?: string;
  updated_at?: string;
}

export interface CreateBusinessEntityRequest {
  entity_type: string;
  entity_name: string;
  contact_info?: string | null;
}

export interface UpdateBusinessEntityRequest {
  entity_type?: string;
  entity_name?: string;
  contact_info?: string | null;
}

// 业务类型（business_types）
export interface BusinessType {
  id: number;
  code: string;
  name: string;
  description?: string | null;
  enabled: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface CreateBusinessTypeRequest {
  code: string;
  name: string;
  description?: string | null;
  enabled?: boolean;
}

export interface UpdateBusinessTypeRequest {
  name?: string;
  description?: string | null;
  enabled?: boolean;
}

// 客户业务费率（rate_customer）
export interface RateCustomer {
  id: number;
  region: string;
  cp: string;
  school_name?: string | null;
  customer_fee?: number | null;
  network_line_fee?: number | null;
  general_fee?: number | null;
  customer_fee_owner_id?: number | null;
  network_line_fee_owner_id?: number | null;
  created_at?: string;
  updated_at?: string;
}

export interface UpsertRateCustomerRequest {
  region: string;
  cp: string;
  school_name?: string | null;
  customer_fee?: number | null;
  network_line_fee?: number | null;
  general_fee?: number | null;
  customer_fee_owner_id?: number | null;
  network_line_fee_owner_id?: number | null;
}

// 节点业务费率（rate_node）
export interface RateNode {
  id: number;
  region: string;
  cp: string;
  settlement_type: string; // IDC/...
  cp_fee?: number | null;
  cp_fee_owner_id?: number | null;
  node_construction_fee?: number | null;
  node_construction_fee_owner_id?: number | null;
  rack_fee?: number | null;
  rack_fee_owner_id?: number | null;
  other_fee?: number | null;
  other_fee_owner_id?: number | null;
  created_at?: string;
  updated_at?: string;
}

export interface UpsertRateNodeRequest {
  region: string;
  cp: string;
  settlement_type: string;
  cp_fee?: number | null;
  cp_fee_owner_id?: number | null;
  node_construction_fee?: number | null;
  node_construction_fee_owner_id?: number | null;
  rack_fee?: number | null;
  rack_fee_owner_id?: number | null;
  other_fee?: number | null;
  other_fee_owner_id?: number | null;
}

// 最终客户费率（rate_final_customer）
export interface RateFinalCustomer {
  id: number;
  region: string;
  cp: string;
  school_name: string;
  fee_type: string; // standard / ...
  final_fee?: number | null;
  customer_fee?: number | null;
  customer_fee_owner_id?: number | null;
  network_line_fee?: number | null;
  network_line_fee_owner_id?: number | null;
  node_deduction_fee?: number | null;
  node_deduction_fee_owner_id?: number | null;
  created_at?: string;
  updated_at?: string;
}

export interface UpsertRateFinalCustomerRequest {
  region: string;
  cp: string;
  school_name: string;
  fee_type: string;
  final_fee?: number | null;
  customer_fee?: number | null;
  customer_fee_owner_id?: number | null;
  network_line_fee?: number | null;
  network_line_fee_owner_id?: number | null;
  node_deduction_fee?: number | null;
  node_deduction_fee_owner_id?: number | null;
}

// ------------------------------
// Settlement Rates - Sync Rules
// ------------------------------

// 同步规则（rate_customer_sync_rules）
export interface SyncRule {
  id: number;
  name: string;
  enabled: boolean;
  priority: number;
  scope_region?: any;
  scope_cp?: any;
  condition_expr?: string | null;
  fields_to_update?: any;
  overwrite_strategy: string;
  actions: any;
  created_by?: number | null;
  updated_by?: number | null;
  created_at?: string;
  updated_at?: string;
}

export interface CreateSyncRuleRequest {
  name: string;
  enabled?: boolean;
  priority?: number;
  scope_region?: any;
  scope_cp?: any;
  condition_expr?: string | null;
  fields_to_update?: any;
  overwrite_strategy: string;
  actions: any;
}

export interface UpdateSyncRuleRequest {
  name?: string;
  enabled?: boolean;
  priority?: number;
  scope_region?: any;
  scope_cp?: any;
  condition_expr?: string | null;
  fields_to_update?: any;
  overwrite_strategy?: string;
  actions?: any;
}
