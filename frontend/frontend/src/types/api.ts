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
    display_name?: string;
    status?: number;
  };
  permissions: (PermissionLite | string)[];
}

export interface ProfileResponse {
  user: {
    id: number;
    username: string;
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
  display_name?: string;
  status?: number;
  roles?: Role[];
  created_at?: string;
}

export interface UpdateUserStatusRequest {
  status: number;
}

export interface SetUserRolesRequest {
  role_ids: number[];
}

// 新建系统用户请求
export interface CreateUserRequest {
  username: string;
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
