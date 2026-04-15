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

export interface TrafficScopeCondition {
  id?: number;
  dimension_type: 'region' | 'cp' | 'school';
  dimension_value: string;
  created_at?: string;
}

export interface TrafficScopeRuleGroup {
  id?: number;
  user_id?: number;
  rule_type: 'allow' | 'deny';
  conditions: TrafficScopeCondition[];
  created_at?: string;
  updated_at?: string;
}

export interface TrafficScopeUserLite {
  id: number;
  username: string;
  alias?: string;
  display_name: string;
  status: number;
}

export interface TrafficScopeOptionItem {
  value: string;
  label: string;
  dimension: 'region' | 'cp' | 'school';
  school_id?: string;
  school_name?: string;
  region?: string;
  cp?: string;
}

export interface TrafficScopeOptionParams {
  dimension: 'region' | 'cp' | 'school';
  q?: string;
  region?: string;
  cp?: string;
  limit?: number;
}

export interface TrafficScopePreview {
  user_id: number;
  source: 'policy_rule' | 'legacy_user_school' | 'default_admin_role' | 'none';
  rules: TrafficScopeRuleGroup[];
  legacy_school_ids: string[];
  allowed_schools: School[];
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
  general_fee_owner_id?: number | null;
  channel_rate?: number | null;
  channel_owner_user_id?: number | null;
  start_at?: string | null;
  increment_start_at?: string | null;
  stock_ratio?: number | null;
  increment_ratio?: number | null;
  daily_increment_value?: number | null;
  fee_mode?: 'auto' | 'configed';
  last_sync_time?: string | null;
  last_sync_rule_id?: number | null;
  extra?: any;
  created_at?: string;
  updated_at?: string;
  // backend-added classification fields
  settlement_ready?: boolean;
  missing_fields?: string[];
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
  general_fee_owner_id?: number | null;
  channel_rate?: number | null;
  channel_owner_user_id?: number | null;
  start_at?: string | null;
  increment_start_at?: string | null;
  stock_ratio?: number | null;
  increment_ratio?: number | null;
  extra?: any;
}

export interface CustomerRateImportError {
  line: number;
  message: string;
}

export interface MissingImportUser {
  alias: string;
  suggested_username: string;
  fields?: string[];
  lines?: number[];
}

export interface CreatedImportUser {
  alias: string;
  username: string;
  password: string;
}

export interface CustomerRateImportResponse {
  affected: number;
  errors: CustomerRateImportError[];
  validate_only?: boolean;
  stage?: 'needs_user_creation' | 'completed';
  missing_users?: MissingImportUser[];
  created_users?: CreatedImportUser[];
  can_auto_create_users?: boolean;
  resumable_token?: string;
}

export interface CustomerRateImportTaskResult {
  validate_only?: boolean;
  affected?: number;
  error_count?: number;
  created_count?: number;
  errors_preview?: CustomerRateImportError[];
  missing_users_preview?: MissingImportUser[];
  created_users_preview?: CreatedImportUser[];
  can_auto_create_users?: boolean;
  can_continue?: boolean;
  errors_csv_url?: string;
  created_users_csv_url?: string;
}

export interface CustomerRateImportTask {
  id: number;
  task_type: string;
  task_date: string;
  status: 'pending' | 'running' | 'waiting_user_confirm' | 'success' | 'failed';
  task_stage?: string;
  start_time?: string | null;
  end_time?: string | null;
  processed_count?: number;
  total_count?: number;
  error_message?: string;
  create_time?: string;
  update_time?: string;
  result?: CustomerRateImportTaskResult;
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
  channel_rate?: number | null;
  channel_owner_user_id?: number | null;
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
  // 节点扣减费相关字段保留以兼容后端，但前端页面不再使用
  node_deduction_fee?: number | null;
  node_deduction_fee_owner_id?: number | null;
  // 新增：渠道费率与渠道归属
  channel_rate?: number | null;
  channel_owner_user_id?: number | null;
}

// 指定服务日期下、按折损规则计算的最终客户费率视图
export interface DiscountedFinalCustomerRate {
  region: string;
  cp: string;
  school_name?: string | null;
  service_date: string; // YYYY-MM-DD
  customer_fee_base?: number | null;
  customer_fee_discount?: number | null;
  channel_rate_base?: number | null;
  channel_rate_discount?: number | null;
  network_line_fee_base?: number | null;
  network_line_fee_discount?: number | null;
  general_fee_base?: number | null;
  general_fee_discount?: number | null;
  customer_fee_owner_id?: number | null;
  channel_owner_user_id?: number | null;
  discount_rule_id?: number | null;
  discount_rule_name?: string | null;
  service_year_index?: number | null;
}

// ------------------------------
// Settlement Formulas
// ------------------------------

export type SettlementFormulaTokenType = 'field' | 'operator' | 'number';

export interface SettlementFormulaToken {
  id: string;
  type: SettlementFormulaTokenType;
  value: string;
  label: string;
}

export interface SettlementFormulaItem {
  id: number;
  name: string;
  description?: string | null;
  tokens: string | SettlementFormulaToken[];
  enabled: boolean;
  updated_by?: string | null;
  create_time?: string;
  update_time?: string;
}

export interface CreateSettlementFormulaRequest {
  name: string;
  description?: string;
  tokens: SettlementFormulaToken[];
  enabled?: boolean;
}

export interface UpdateSettlementFormulaRequest {
  name: string;
  description?: string;
  tokens?: SettlementFormulaToken[];
  enabled?: boolean;
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

export interface FilterRule {
  id: number;
  name: string;
  enabled: boolean;
  priority: number;
  scope_region?: any;
  scope_cp?: any;
  school_name_match_type?: '' | 'exact' | 'contains';
  school_name_values?: any;
  created_by?: number | null;
  updated_by?: number | null;
  created_at?: string;
  updated_at?: string;
  match_count?: number;
  matched_school_names?: string[];
  matched_summary?: string;
}

export interface CreateFilterRuleRequest {
  name: string;
  enabled?: boolean;
  priority?: number;
  scope_region?: any;
  scope_cp?: any;
  school_name_match_type?: '' | 'exact' | 'contains';
  school_name_values?: any;
}

export interface UpdateFilterRuleRequest {
  name?: string;
  enabled?: boolean;
  priority?: number;
  scope_region?: any;
  scope_cp?: any;
  school_name_match_type?: '' | 'exact' | 'contains';
  school_name_values?: any;
}

export interface SettlementRuleScopeOptions {
  regions: string[];
  cps: string[];
}
